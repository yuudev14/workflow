package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

const (
	StatusInProgress = "in_progress"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusSkipped    = "skipped"

	// startNode is a no-op node that only reports success, parity with the
	// Python executor's special case.
	startNode = "start"
)

// Executor walks a playbook graph with Kahn's algorithm: nodes whose
// dependencies completed run concurrently (bounded by maxParallel), each node
// is dispatched to its runtime, outputs accumulate in a StepStore, and any
// failure cancels the remaining run (parity with the Python executor).
type Executor struct {
	logger          logger.Logger
	taskService     tasks.TaskService
	playbookService playbooks.PlaybookService
	resolver        RuntimeResolver
	status          StatusPublisher
	maxParallel     int
	nodeTimeout     time.Duration
}

func NewExecutor(
	log logger.Logger,
	taskService tasks.TaskService,
	playbookService playbooks.PlaybookService,
	resolver RuntimeResolver,
	status StatusPublisher,
	maxParallel int,
	nodeTimeout time.Duration,
) *Executor {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &Executor{
		logger:          log,
		taskService:     taskService,
		playbookService: playbookService,
		resolver:        resolver,
		status:          status,
		maxParallel:     maxParallel,
		nodeTimeout:     nodeTimeout,
	}
}

type nodeResult struct {
	node    string
	output  any
	skipped bool
	err     error
}

func (e *Executor) Run(ctx context.Context, msg domain.TaskMessage) error {
	// domain.IsAcyclicGraph returns true when a cycle EXISTS and mutates its
	// input, so it gets a copy.
	if domain.IsAcyclicGraph(copyGraph(msg.Graph)) {
		return e.finishPlaybook(ctx, msg, StatusFailed, nil, fmt.Errorf("graph contains a cycle"))
	}

	indegree, children := buildIndegree(msg.Graph)
	handles := buildEdgeHandles(msg.Edges)
	// followedCount counts, per node, how many completed incoming edges chose
	// to follow it. A node whose dependencies all completed without a single
	// follow is skipped (and propagates the skip).
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
		// ctx, not the cancelled runCtx: the final status must still persist.
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

// processSkippedNode completes a node without executing it: no incoming edge
// chose to follow it (condition branch not taken, or all parents skipped).
func (e *Executor) processSkippedNode(ctx context.Context, msg domain.TaskMessage, node string) error {
	e.logger.Infow("skipping playbook node", "node", node, "playbook_history_id", msg.PlaybookHistoryId)

	task, ok := msg.Tasks[node]
	if !ok {
		return fmt.Errorf("operation (%s) does not exist in task_information", node)
	}
	return e.setTaskStatus(ctx, msg, task, StatusSkipped, nil, nil)
}

// setTaskStatus persists the task history row and fans it out to the API's WS
// hub, replacing the Python send_task_status -> gRPC HandleTask path.
func (e *Executor) setTaskStatus(
	ctx context.Context,
	msg domain.TaskMessage,
	task domain.Tasks,
	status string,
	result any,
	execErr error,
) error {
	// Detach from cancellation: after a fail-fast cancel() the draining nodes
	// still have to persist their failed status, or they'd sit at in_progress
	// forever (same reason finishPlaybook receives the parent ctx).
	ctx = context.WithoutCancel(ctx)
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
	if err := e.status.Publish("task_status", res); err != nil {
		e.logger.Errorw("failed to publish task status event", "task", task.Name, "error", err)
	}
	return nil
}

func (e *Executor) finishPlaybook(
	ctx context.Context,
	msg domain.TaskMessage,
	status string,
	result any,
	execErr error,
) error {
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
		return err
	}
	if err := e.status.Publish("playbook_status", res); err != nil {
		e.logger.Errorw("failed to publish playbook status event",
			"playbook_history_id", msg.PlaybookHistoryId, "error", err)
	}
	return execErr
}

type edgeKey struct {
	source      string
	destination string
}

// buildEdgeHandles indexes the wire edges' source_handles per (source,
// destination) pair — two nodes can be linked by several edges (e.g. a
// condition's true AND false handle both pointing at the same join node).
func buildEdgeHandles(edges []domain.EdgeRef) map[edgeKey][]*string {
	handles := map[edgeKey][]*string{}
	for _, edge := range edges {
		key := edgeKey{source: edge.Source, destination: edge.Destination}
		handles[key] = append(handles[key], edge.SourceHandle)
	}
	return handles
}

// edgeFollowed decides whether a completed node follows the edge to child. A
// branch handle (a condition's "true"/"false", or a switch case id / "else") is
// only followed when it matches the node's {"result": ...}. Everything else always
// follows: directional editor handles ("source-*"/"target-*"), unlabeled edges,
// and output that isn't a condition result.
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

// conditionResult pulls the branch selector out of a node's output. A condition
// returns {"result": ...} — a bool for true/false, or a case id / "else" for a
// switch — which becomes the handle string an edge must match to be followed.
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

// isDirectionalHandle reports whether a source_handle is one of React Flow's
// positional editor handles (source-top, target-left, ...) rather than a
// semantic condition branch — positional handles never gate an edge.
func isDirectionalHandle(handle string) bool {
	return strings.HasPrefix(handle, "source-") || strings.HasPrefix(handle, "target-")
}

// buildIndegree mirrors the Python executor's invert_graph: nodes that appear
// only as edge targets still get an indegree entry.
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
