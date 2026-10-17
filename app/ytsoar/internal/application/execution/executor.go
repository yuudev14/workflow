package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
	"github.com/yuudev14/ytsoar/internal/utils"
)

const (
	StatusInProgress = domain.StatusInProgress
	StatusSuccess    = domain.StatusSuccess
	StatusFailed     = domain.StatusFailed
	StatusSkipped    = domain.StatusSkipped

	// startNode only reports success.
	startNode = "start"
)

// Executor runs a playbook graph (Kahn's algorithm). Ready nodes run in parallel,
// and one failure cancels the run.
type Executor struct {
	logger          logger.Logger
	taskService     tasks.TaskService
	playbookService playbooks.PlaybookService
	resolver        RuntimeResolver
	status          StatusPublisher
	maxParallel     int
	nodeTimeout     time.Duration
	// wait before each retry to save the final status
	finalWriteDelays []time.Duration
}

// Option changes a setting of the Executor.
type Option func(*Executor)

// WithFinalWriteDelays sets the waits between final status saves. Tests use short ones.
func WithFinalWriteDelays(delays ...time.Duration) Option {
	return func(e *Executor) { e.finalWriteDelays = delays }
}

// defaultFinalWriteDelays: about 30 seconds in total.
var defaultFinalWriteDelays = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 15 * time.Second}

func NewExecutor(
	log logger.Logger,
	taskService tasks.TaskService,
	playbookService playbooks.PlaybookService,
	resolver RuntimeResolver,
	status StatusPublisher,
	maxParallel int,
	nodeTimeout time.Duration,
	opts ...Option,
) *Executor {
	if maxParallel < 1 {
		maxParallel = 1
	}
	// zero or negative would time out every node, so use the default
	if nodeTimeout <= 0 {
		nodeTimeout = 5 * time.Minute
	}
	e := &Executor{
		logger:           log,
		taskService:      taskService,
		playbookService:  playbookService,
		resolver:         resolver,
		status:           status,
		maxParallel:      maxParallel,
		nodeTimeout:      nodeTimeout,
		finalWriteDelays: defaultFinalWriteDelays,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// statusWriteTimeout limits one try to save a status.
const statusWriteTimeout = 10 * time.Second

// detach keeps the values of ctx but ignores its cancel, so the last status write
// still runs. It has its own time limit.
func detach(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), statusWriteTimeout)
}

var errWorkerStopped = errors.New("an earlier attempt started this run and did not finish it (the worker may have stopped)")

// ErrNotStarted: the database failed before the run became in_progress. Nothing ran,
// so it is safe to mark the run failed.
var ErrNotStarted = errors.New("run did not start")

// ErrNotFinished: the run ran, but the final status was not saved after several tries.
// The run stays in_progress, so the queue retries and recoverRun marks it failed.
var ErrNotFinished = errors.New("run status was not saved")

type nodeResult struct {
	node    string
	output  any
	skipped bool
	err     error
}

func (e *Executor) Run(ctx context.Context, msg domain.TaskMessage) error {
	if _, err := e.setRunStatus(ctx, msg, StatusInProgress, nil, nil); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			return e.recoverRun(ctx, msg)
		}
		return fmt.Errorf("%w: %w", ErrNotStarted, err)
	}

	// IsAcyclicGraph returns true when a cycle exists and changes its input, so pass a copy
	utils.DebugJSON(e.logger, "TaskMessage", msg, " ")
	if domain.IsAcyclicGraph(copyGraph(msg.Graph)) {
		return e.finishPlaybook(ctx, msg, StatusFailed, nil, fmt.Errorf("graph contains a cycle"))
	}

	// an empty graph would succeed and hide a broken message builder
	if len(msg.Graph) == 0 {
		return e.finishPlaybook(ctx, msg, StatusFailed, nil, fmt.Errorf("playbook message has no graph nodes"))
	}

	indegree, children := buildIndegree(msg.Graph)
	handles := buildEdgeHandles(msg.Edges)
	// per node: how many finished parents chose to follow it. Zero means skipped.
	followedCount := map[string]int{}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	store := NewStepStore()
	sem := make(chan struct{}, e.maxParallel)
	done := make(chan nodeResult)

	launch := func(node string, skip bool) {
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			// recover here, River catches panics only in the job goroutine
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("node %s panicked: %v", node, r)
					e.logger.Errorw("node panicked", "node", node, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
					if task, ok := msg.Tasks[node]; ok {
						if statusErr := e.setTaskStatus(runCtx, msg, task, StatusFailed, nil, err); statusErr != nil {
							e.logger.Errorw("failed to persist task failure", "node", node, "error", statusErr)
						}
					}
					done <- nodeResult{node: node, err: err}
				}
			}()
			if skip {
				done <- nodeResult{node: node, skipped: true, err: e.processSkippedNode(runCtx, msg, node)}
				return
			}
			output, err := e.processNode(runCtx, msg, node, store)
			done <- nodeResult{node: node, output: output, err: err}
		}()
	}

	inFlight := 0
	for node, degree := range indegree {
		if degree == 0 {
			launch(node, false)
			inFlight++
		}
	}

	var runErr error
	for inFlight > 0 {
		result := <-done
		inFlight--
		if result.err != nil {
			if runErr == nil {
				runErr = result.err
				cancel() // fail-fast: parity with the Python executor
			}
			continue
		}
		if runErr != nil {
			continue // draining in-flight nodes; nothing new is scheduled
		}
		for _, child := range children[result.node] {
			indegree[child]--
			if !result.skipped && edgeFollowed(handles, result.node, child, result.output) {
				followedCount[child]++
			}
			if indegree[child] == 0 {
				launch(child, followedCount[child] == 0)
				inFlight++
			}
		}
	}

	if runErr != nil {
		// use ctx, not the cancelled runCtx, so the final status still saves
		return e.finishPlaybook(ctx, msg, StatusFailed, nil, runErr)
	}
	return e.finishPlaybook(ctx, msg, StatusSuccess, store.Snapshot(), nil)
}

func (e *Executor) processNode(ctx context.Context, msg domain.TaskMessage, node string, store *StepStore) (any, error) {
	e.logger.Infow("executing playbook node", "node", node, "playbook_history_id", msg.PlaybookHistoryId)

	task, ok := msg.Tasks[node]
	if !ok {
		return nil, fmt.Errorf("operation (%s) does not exist in task_information", node)
	}

	if node == startNode {
		return nil, e.setTaskStatus(ctx, msg, task, StatusSuccess, nil, nil)
	}

	if err := e.setTaskStatus(ctx, msg, task, StatusInProgress, nil, nil); err != nil {
		return nil, err
	}

	if task.ConnectorID == nil {
		err := fmt.Errorf("connector id is none for %s", node)
		if statusErr := e.setTaskStatus(ctx, msg, task, StatusFailed, nil, err); statusErr != nil {
			e.logger.Errorw("failed to persist task failure", "node", node, "error", statusErr)
		}
		return nil, err
	}

	runtime, err := e.resolver.Resolve(task)
	if err != nil {
		if statusErr := e.setTaskStatus(ctx, msg, task, StatusFailed, nil, err); statusErr != nil {
			e.logger.Errorw("failed to persist task failure", "node", node, "error", statusErr)
		}
		return nil, err
	}

	nodeCtx, cancel := context.WithTimeout(ctx, e.nodeTimeout)
	defer cancel()
	raw, err := runtime.Execute(nodeCtx, ExecutionRequest{
		Task:              task,
		Steps:             store.Snapshot(),
		PlaybookHistoryID: msg.PlaybookHistoryId,
		Timeout:           e.nodeTimeout,
		Input:             msg.Input.Resolved(),
	})
	if err != nil {
		if statusErr := e.setTaskStatus(ctx, msg, task, StatusFailed, nil, err); statusErr != nil {
			e.logger.Errorw("failed to persist task failure", "node", node, "error", statusErr)
		}
		return nil, err
	}

	var decoded any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			e.logger.Warnw("runtime returned non-JSON result, storing as string", "node", node, "error", err)
			decoded = string(raw)
		}
	}
	store.Set(node, decoded)
	return decoded, e.setTaskStatus(ctx, msg, task, StatusSuccess, decoded, nil)
}

// processSkippedNode finishes a node without running it, because no parent followed it.
func (e *Executor) processSkippedNode(ctx context.Context, msg domain.TaskMessage, node string) error {
	e.logger.Infow("skipping playbook node", "node", node, "playbook_history_id", msg.PlaybookHistoryId)

	task, ok := msg.Tasks[node]
	if !ok {
		return fmt.Errorf("operation (%s) does not exist in task_information", node)
	}
	return e.setTaskStatus(ctx, msg, task, StatusSkipped, nil, nil)
}

// setTaskStatus saves the task row and sends it to the WS hub.
func (e *Executor) setTaskStatus(
	ctx context.Context,
	msg domain.TaskMessage,
	task domain.Tasks,
	status string,
	result any,
	execErr error,
) error {
	// detach from the cancel: after a fail-fast, the draining nodes must still save failed,
	// or they stay in_progress
	ctx, cancel := detach(ctx)
	defer cancel()
	var errStr *string
	if execErr != nil {
		s := execErr.Error()
		errStr = &s
	}
	var parameters any
	if len(task.Parameters) > 0 {
		if err := json.Unmarshal(task.Parameters, &parameters); err != nil {
			e.logger.Warnw("could not decode task parameters", "task", task.Name, "error", err)
		}
	}

	res, err := e.taskService.UpdateTaskHistory(ctx, msg.PlaybookHistoryId.String(), task.ID.String(),
		tasks.UpdateTaskHistoryData{
			Name:          task.Name,
			Description:   task.Description,
			Parameters:    parameters,
			ConnectorName: types.Nullable[string]{Value: task.ConnectorName, Set: true},
			ConnectorID:   types.Nullable[string]{Value: task.ConnectorID, Set: true},
			Operation:     task.Operation,
			Config:        types.Nullable[string]{Value: task.Config, Set: true},
			X:             task.X,
			Y:             task.Y,
			Status:        types.Nullable[string]{Value: &status, Set: true},
			Error:         types.Nullable[string]{Value: errStr, Set: true},
			Result:        result,
		})
	if err != nil {
		return err
	}
	if err := e.status.Publish(ctx, "task_status", res.ID); err != nil {
		e.logger.Errorw("failed to publish task status event", "task", task.Name, "error", err)
	}
	return nil
}

// recoverRun handles a run that the state rules refused to start (in_progress or finished).
// In_progress means a worker died and the queue gave the job again. Running it again would
// repeat the side effects of the connectors, so mark it failed. Do not touch a finished run.
func (e *Executor) recoverRun(ctx context.Context, msg domain.TaskMessage) error {
	e.logger.Warnw("run was already started, not running it again", "playbook_history_id", msg.PlaybookHistoryId)
	err := e.finishPlaybook(ctx, msg, StatusFailed, nil, errWorkerStopped)
	if errors.Is(err, domain.ErrInvalidTransition) {
		return nil // already finished
	}
	// after the state rules accepted the change, so a finished run keeps its tasks
	writeCtx, cancel := detach(ctx)
	defer cancel()
	failed, failErr := e.taskService.FailUnfinishedTaskHistory(writeCtx, msg.PlaybookHistoryId.String(), errWorkerStopped.Error())
	if failErr != nil {
		e.logger.Errorw("failed to fail unfinished tasks", "playbook_history_id", msg.PlaybookHistoryId, "error", failErr)
	}
	// tell the open pages, so they do not show the tasks as in_progress
	for _, task := range failed {
		if pubErr := e.status.Publish(writeCtx, "task_status", task.ID); pubErr != nil {
			e.logger.Errorw("failed to publish task status event", "task", task.Name, "error", pubErr)
		}
	}
	return err
}

func (e *Executor) finishPlaybook(
	ctx context.Context,
	msg domain.TaskMessage,
	status string,
	result any,
	execErr error,
) error {
	if err := e.saveFinalStatus(ctx, msg, status, result, execErr); err != nil {
		return err
	}
	return execErr
}

// saveFinalStatus tries again after each delay in finalWriteDelays (about 30 seconds), so a
// short database problem does not leave the run in_progress. If all tries fail, it wraps
// ErrNotFinished. ErrInvalidTransition is returned as it is.
func (e *Executor) saveFinalStatus(ctx context.Context, msg domain.TaskMessage, status string, result any, execErr error) error {
	var err error
	for try := 0; ; try++ {
		_, err = e.setRunStatus(ctx, msg, status, result, execErr)
		if err == nil || errors.Is(err, domain.ErrInvalidTransition) {
			return err
		}
		if try >= len(e.finalWriteDelays) {
			return fmt.Errorf("%w: %w", ErrNotFinished, err)
		}
		e.logger.Warnw("could not save the run status, trying again",
			"playbook_history_id", msg.PlaybookHistoryId, "status", status, "error", err, "wait", e.finalWriteDelays[try])
		time.Sleep(e.finalWriteDelays[try])
	}
}

// Abort marks a run as failed when it cannot start (for example, the message cannot be
// built). It does not touch a finished run.
func (e *Executor) Abort(ctx context.Context, historyID uuid.UUID, cause error) error {
	err := e.saveFinalStatus(ctx, domain.TaskMessage{PlaybookHistoryId: historyID}, StatusFailed, nil, cause)
	if errors.Is(err, domain.ErrInvalidTransition) {
		return nil
	}
	return err
}

// setRunStatus changes the run status through the state rules and sends the row to clients.
func (e *Executor) setRunStatus(
	ctx context.Context,
	msg domain.TaskMessage,
	status string,
	result any,
	execErr error,
) (*domain.PlaybookHistory, error) {
	// a timeout or shutdown must still save the status, but one try must not hang when the
	// database drops packets
	ctx, cancel := detach(ctx)
	defer cancel()
	var errStr *string
	if execErr != nil {
		s := execErr.Error()
		errStr = &s
	}
	res, err := e.playbookService.UpdatePlaybookHistory(ctx, msg.PlaybookHistoryId.String(),
		playbooks.UpdatePlaybookHistoryData{
			Status: types.Nullable[string]{Value: &status, Set: true},
			Error:  types.Nullable[string]{Value: errStr, Set: true},
			Result: result,
		})
	if err != nil {
		return nil, err
	}
	if err := e.status.Publish(ctx, "playbook_status", res.ID); err != nil {
		e.logger.Errorw("failed to publish playbook status event",
			"playbook_history_id", msg.PlaybookHistoryId, "error", err)
	}
	return res, nil
}

type edgeKey struct {
	source      string
	destination string
}

// buildEdgeHandles maps (source, destination) to the source handles. Two nodes can have
// several edges (a condition's true and false to the same join node).
func buildEdgeHandles(edges []domain.EdgeRef) map[edgeKey][]*string {
	handles := map[edgeKey][]*string{}
	for _, edge := range edges {
		key := edgeKey{source: edge.Source, destination: edge.Destination}
		handles[key] = append(handles[key], edge.SourceHandle)
	}
	return handles
}

// edgeFollowed says if a finished node follows the edge to child. A branch handle (condition
// true/false, switch case id or "else") follows only when it matches the node's
// {"result": ...}. All other edges always follow.
func edgeFollowed(handles map[edgeKey][]*string, node string, child string, output any) bool {
	entries, ok := handles[edgeKey{source: node, destination: child}]
	if !ok || len(entries) == 0 {
		return true
	}
	selector, hasSelector := conditionResult(output)
	for _, handle := range entries {
		if handle == nil || isDirectionalHandle(*handle) {
			return true
		}
		if !hasSelector {
			return true
		}
		if *handle == selector {
			return true
		}
	}
	return false
}

// conditionResult gets the branch handle from the output of a node: {"result": ...}
// (bool, case id or "else"). It checks the shape, not the connector id, so any connector or
// snippet can be a condition. Normal nodes are not gated, because their edges use
// directional or nil handles.
func conditionResult(output any) (string, bool) {
	outMap, ok := output.(map[string]any)
	if !ok {
		return "", false
	}
	switch result := outMap["result"].(type) {
	case bool:
		if result {
			return "true", true
		}
		return "false", true
	case string:
		return result, true
	default:
		return "", false
	}
}

// isDirectionalHandle: React Flow position handles (source-top, target-left, ...) never
// gate an edge
func isDirectionalHandle(handle string) bool {
	return strings.HasPrefix(handle, "source-") || strings.HasPrefix(handle, "target-")
}

// buildIndegree is like the Python invert_graph. Nodes that are only edge targets also get
// an entry.
func buildIndegree(graph map[string][]string) (indegree map[string]int, children map[string][]string) {
	indegree = map[string]int{}
	children = map[string][]string{}
	for node, targets := range graph {
		if _, ok := indegree[node]; !ok {
			indegree[node] = 0
		}
		for _, target := range targets {
			indegree[target]++
			children[node] = append(children[node], target)
		}
	}
	return indegree, children
}

func copyGraph(graph map[string][]string) map[string][]string {
	out := make(map[string][]string, len(graph))
	for node, targets := range graph {
		out[node] = append([]string(nil), targets...)
	}
	return out
}
