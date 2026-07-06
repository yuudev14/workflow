package execution

import (
	"context"
	"encoding/json"
	"fmt"
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
	node string
	err  error
}

func (e *Executor) Run(ctx context.Context, msg domain.TaskMessage) error {
	// domain.IsAcyclicGraph returns true when a cycle EXISTS and mutates its
	// input, so it gets a copy.
	if domain.IsAcyclicGraph(copyGraph(msg.Graph)) {
		return e.finishPlaybook(ctx, msg, StatusFailed, nil, fmt.Errorf("graph contains a cycle"))
	}

	indegree, children := buildIndegree(msg.Graph)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	store := NewStepStore()
	sem := make(chan struct{}, e.maxParallel)
	done := make(chan nodeResult)

	launch := func(node string) {
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			done <- nodeResult{node: node, err: e.processNode(runCtx, msg, node, store)}
		}()
	}

	inFlight := 0
	for node, degree := range indegree {
		if degree == 0 {
			launch(node)
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
			if indegree[child] == 0 {
				launch(child)
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

func (e *Executor) processNode(ctx context.Context, msg domain.TaskMessage, node string, store *StepStore) error {
	e.logger.Infow("executing playbook node", "node", node, "playbook_history_id", msg.PlaybookHistoryId)

	task, ok := msg.Tasks[node]
	if !ok {
		return fmt.Errorf("operation (%s) does not exist in task_information", node)
	}

	if err := e.setTaskStatus(ctx, msg, task, StatusInProgress, nil, nil); err != nil {
		return err
	}

	if node == startNode {
		return e.setTaskStatus(ctx, msg, task, StatusSuccess, nil, nil)
	}

	if task.ConnectorID == nil {
		err := fmt.Errorf("connector id is none for %s", node)
		if statusErr := e.setTaskStatus(ctx, msg, task, StatusFailed, nil, err); statusErr != nil {
			e.logger.Errorw("failed to persist task failure", "node", node, "error", statusErr)
		}
		return err
	}

	runtime, err := e.resolver.Resolve(task)
	if err != nil {
		if statusErr := e.setTaskStatus(ctx, msg, task, StatusFailed, nil, err); statusErr != nil {
			e.logger.Errorw("failed to persist task failure", "node", node, "error", statusErr)
		}
		return err
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
		return err
	}

	var decoded any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			e.logger.Warnw("runtime returned non-JSON result, storing as string", "node", node, "error", err)
			decoded = string(raw)
		}
	}
	store.Set(node, decoded)
	return e.setTaskStatus(ctx, msg, task, StatusSuccess, decoded, nil)
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
