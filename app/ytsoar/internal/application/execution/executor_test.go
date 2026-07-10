package execution_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/execution"
	execmocks "github.com/yuudev14/ytsoar/internal/application/execution/mocks"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	playbookmocks "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	taskmocks "github.com/yuudev14/ytsoar/internal/application/tasks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func sampleTask(name string) domain.Tasks {
	connectorID := "sample_connector"
	return domain.Tasks{
		ID:          uuid.New(),
		Name:        name,
		ConnectorID: &connectorID,
		Operation:   "sample_operation",
		Parameters:  json.RawMessage(`{}`),
	}
}

func tasksFor(graph map[string][]string) map[string]domain.Tasks {
	all := map[string]domain.Tasks{}
	for node, targets := range graph {
		if _, ok := all[node]; !ok {
			all[node] = sampleTask(node)
		}
		for _, target := range targets {
			if _, ok := all[target]; !ok {
				all[target] = sampleTask(target)
			}
		}
	}
	return all
}

type executorFixture struct {
	runtime         *execmocks.MockNodeRuntime
	taskService     *taskmocks.MockTaskService
	playbookService *playbookmocks.MockPlaybookService
	status          *execmocks.MockStatusPublisher
	executor        *execution.Executor

	mu             sync.Mutex
	taskStatuses   map[string][]string
	playbookFinal  string
	playbookErrors []string
}

func newExecutorFixture(t *testing.T, maxParallel int) *executorFixture {
	ctrl := gomock.NewController(t)
	f := &executorFixture{
		runtime:         execmocks.NewMockNodeRuntime(ctrl),
		taskService:     taskmocks.NewMockTaskService(ctrl),
		playbookService: playbookmocks.NewMockPlaybookService(ctrl),
		status:          execmocks.NewMockStatusPublisher(ctrl),
		taskStatuses:    map[string][]string{},
	}

	f.taskService.EXPECT().
		UpdateTaskHistory(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, _ string, data tasks.UpdateTaskHistoryData) (*domain.TaskHistory, error) {
			// behave like a real DB: a cancelled ctx fails the write
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if data.Status.Value != nil {
				f.taskStatuses[data.Name] = append(f.taskStatuses[data.Name], *data.Status.Value)
			}
			return &domain.TaskHistory{}, nil
		}).AnyTimes()

	f.playbookService.EXPECT().
		UpdatePlaybookHistory(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, data playbooks.UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if data.Status.Value != nil {
				f.playbookFinal = *data.Status.Value
			}
			if data.Error.Value != nil {
				f.playbookErrors = append(f.playbookErrors, *data.Error.Value)
			}
			return &domain.PlaybookHistory{}, nil
		}).AnyTimes()

	f.status.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	f.executor = execution.NewExecutor(
		logger.NewNop(),
		f.taskService,
		f.playbookService,
		execution.NewStaticResolver(f.runtime, nil),
		f.status,
		maxParallel,
		time.Minute,
	)
	return f
}

func (f *executorFixture) run(t *testing.T, graph map[string][]string, taskInfo map[string]domain.Tasks) error {
	t.Helper()
	return f.runWithEdges(t, graph, taskInfo, nil)
}

func (f *executorFixture) runWithEdges(t *testing.T, graph map[string][]string, taskInfo map[string]domain.Tasks, edges []domain.EdgeRef) error {
	t.Helper()
	return f.executor.Run(context.Background(), domain.TaskMessage{
		Graph:             graph,
		Tasks:             taskInfo,
		PlaybookHistoryId: uuid.New(),
		Edges:             edges,
	})
}

// edgeRef builds a wire edge; handle "" means a plain edge (nil source_handle).
func edgeRef(source, destination, handle string) domain.EdgeRef {
	ref := domain.EdgeRef{Source: source, Destination: destination}
	if handle != "" {
		ref.SourceHandle = &handle
	}
	return ref
}

// conditionRuntime returns runtime outputs per node name; the "cond" node
// yields the condition builtin's {"result": bool} shape.
func conditionRuntime(f *executorFixture, condResult bool, executed *[]string, mu *sync.Mutex) {
	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			mu.Lock()
			*executed = append(*executed, req.Task.Name)
			mu.Unlock()
			if req.Task.Name == "cond" {
				return json.RawMessage(fmt.Sprintf(`{"result":%t}`, condResult)), nil
			}
			return json.RawMessage(`{}`), nil
		}).AnyTimes()
}

func TestExecutorDiamondOrderingAndStepsThreading(t *testing.T) {
	graph := map[string][]string{
		"start": {"A"},
		"A":     {"B", "C"},
		"B":     {"D"},
		"C":     {"D"},
		"D":     {},
	}
	f := newExecutorFixture(t, 4)

	var mu sync.Mutex
	var order []string
	stepsSeenByD := map[string]any{}

	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			mu.Lock()
			order = append(order, req.Task.Name)
			if req.Task.Name == "D" {
				stepsSeenByD = req.Steps
			}
			mu.Unlock()
			return json.RawMessage(fmt.Sprintf(`{"out":%q}`, req.Task.Name)), nil
		}).Times(4) // A, B, C, D — start never reaches the runtime

	err := f.run(t, graph, tasksFor(graph))

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, "A", order[0])
	assert.Equal(t, "D", order[len(order)-1])
	// D's templating snapshot must include every finished predecessor output.
	assert.Contains(t, stepsSeenByD, "A")
	assert.Contains(t, stepsSeenByD, "B")
	assert.Contains(t, stepsSeenByD, "C")
	// start is a no-op: one success write, no in_progress round-trip
	assert.Equal(t, []string{"success"}, f.taskStatuses["start"])
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["D"])
}

func TestExecutorRunsReadyNodesInParallel(t *testing.T) {
	graph := map[string][]string{
		"start": {"B", "C"},
		"B":     {},
		"C":     {},
	}
	f := newExecutorFixture(t, 4)

	bEntered := make(chan struct{})
	cEntered := make(chan struct{})
	waitFor := func(own chan struct{}, other <-chan struct{}) error {
		close(own)
		select {
		case <-other:
			return nil
		case <-time.After(2 * time.Second):
			return errors.New("nodes did not run concurrently")
		}
	}

	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			switch req.Task.Name {
			case "B":
				if err := waitFor(bEntered, cEntered); err != nil {
					return nil, err
				}
			case "C":
				if err := waitFor(cEntered, bEntered); err != nil {
					return nil, err
				}
			}
			return json.RawMessage(`{}`), nil
		}).Times(2)

	err := f.run(t, graph, tasksFor(graph))

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
}

func TestExecutorFailFast(t *testing.T) {
	graph := map[string][]string{
		"start": {"A"},
		"A":     {"B"},
		"B":     {"C"},
	}
	f := newExecutorFixture(t, 4)

	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			if req.Task.Name == "A" {
				return nil, errors.New("boom")
			}
			t.Errorf("node %s must not execute after the failure", req.Task.Name)
			return json.RawMessage(`{}`), nil
		}).Times(1)

	err := f.run(t, graph, tasksFor(graph))

	assert.ErrorContains(t, err, "boom")
	assert.Equal(t, "failed", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "failed"}, f.taskStatuses["A"])
	assert.Empty(t, f.taskStatuses["B"])
	assert.Empty(t, f.taskStatuses["C"])
}

// A fail-fast cancel must not strand the other in-flight nodes at in_progress:
// their failed status is persisted with a cancellation-detached context.
func TestExecutorPersistsFailureStatusOfCancelledNodes(t *testing.T) {
	graph := map[string][]string{
		"start": {"A", "B"},
		"A":     {},
		"B":     {},
	}
	f := newExecutorFixture(t, 4)

	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			if req.Task.Name == "A" {
				return nil, errors.New("boom")
			}
			// B: block until A's failure cancels the run
			<-ctx.Done()
			return nil, ctx.Err()
		}).Times(2)

	err := f.run(t, graph, tasksFor(graph))

	assert.ErrorContains(t, err, "boom")
	assert.Equal(t, "failed", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "failed"}, f.taskStatuses["A"])
	assert.Equal(t, []string{"in_progress", "failed"}, f.taskStatuses["B"])
}

func TestExecutorStartNodeIsNoOpSuccess(t *testing.T) {
	graph := map[string][]string{"start": {}}
	f := newExecutorFixture(t, 1)
	// no runtime.Execute expectation: calling it would fail the test

	err := f.run(t, graph, tasksFor(graph))

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, []string{"success"}, f.taskStatuses["start"])
}

func TestExecutorMissingTaskFails(t *testing.T) {
	graph := map[string][]string{"start": {"ghost"}}
	taskInfo := map[string]domain.Tasks{"start": sampleTask("start")}
	f := newExecutorFixture(t, 2)

	err := f.run(t, graph, taskInfo)

	assert.ErrorContains(t, err, "ghost")
	assert.Equal(t, "failed", f.playbookFinal)
}

func TestExecutorRejectsCyclicGraph(t *testing.T) {
	graph := map[string][]string{
		"A": {"B"},
		"B": {"A"},
	}
	f := newExecutorFixture(t, 2)
	// no runtime.Execute expectation: nothing may run on a cyclic graph

	err := f.run(t, graph, tasksFor(graph))

	assert.ErrorContains(t, err, "cycle")
	assert.Equal(t, "failed", f.playbookFinal)
	assert.Empty(t, f.taskStatuses)
}

func TestExecutorConditionFollowsTruePathSkipsFalse(t *testing.T) {
	graph := map[string][]string{
		"start": {"cond"},
		"cond":  {"yes", "no"},
		"yes":   {},
		"no":    {"after"},
		"after": {},
	}
	edges := []domain.EdgeRef{
		edgeRef("cond", "yes", "true"),
		edgeRef("cond", "no", "false"),
	}
	f := newExecutorFixture(t, 4)

	var mu sync.Mutex
	var executed []string
	conditionRuntime(f, true, &executed, &mu)

	err := f.runWithEdges(t, graph, tasksFor(graph), edges)

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["yes"])
	// false branch and everything downstream of it is skipped, not executed.
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["no"])
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["after"])
	mu.Lock()
	assert.NotContains(t, executed, "no")
	assert.NotContains(t, executed, "after")
	mu.Unlock()
}

func TestExecutorConditionFalsePathSkipsTrueSubtree(t *testing.T) {
	graph := map[string][]string{
		"start": {"cond"},
		"cond":  {"yes", "no"},
		"yes":   {"child"},
		"child": {},
		"no":    {},
	}
	edges := []domain.EdgeRef{
		edgeRef("cond", "yes", "true"),
		edgeRef("cond", "no", "false"),
	}
	f := newExecutorFixture(t, 4)

	var mu sync.Mutex
	var executed []string
	conditionRuntime(f, false, &executed, &mu)

	err := f.runWithEdges(t, graph, tasksFor(graph), edges)

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["no"])
	// true branch skips, and the skip propagates to its whole subtree.
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["yes"])
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["child"])
	mu.Lock()
	assert.NotContains(t, executed, "yes")
	assert.NotContains(t, executed, "child")
	mu.Unlock()
}

func TestExecutorJoinRunsWhenOneParentFollowed(t *testing.T) {
	// join has two incoming edges: the taken (true) branch and the skipped
	// (false) branch both point at it. One followed parent is enough to run it.
	graph := map[string][]string{
		"start": {"cond"},
		"cond":  {"yes", "no"},
		"yes":   {"join"},
		"no":    {"join"},
		"join":  {},
	}
	edges := []domain.EdgeRef{
		edgeRef("cond", "yes", "true"),
		edgeRef("cond", "no", "false"),
	}
	f := newExecutorFixture(t, 4)

	var mu sync.Mutex
	var executed []string
	conditionRuntime(f, true, &executed, &mu)

	err := f.runWithEdges(t, graph, tasksFor(graph), edges)

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["yes"])
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["no"])
	// join runs because the true branch reached it, even though "no" skipped.
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["join"])
	mu.Lock()
	assert.Contains(t, executed, "join")
	mu.Unlock()
}

func TestExecutorSwitchFollowsMatchingCaseSkipsRest(t *testing.T) {
	// if/else-if/else from one node: three labeled handles, one taken.
	graph := map[string][]string{
		"start": {"cond"},
		"cond":  {"a", "b", "c"},
		"a":     {},
		"b":     {},
		"c":     {},
	}
	edges := []domain.EdgeRef{
		edgeRef("cond", "a", "case-0"),
		edgeRef("cond", "b", "case-1"),
		edgeRef("cond", "c", "else"),
	}
	f := newExecutorFixture(t, 4)

	var mu sync.Mutex
	var executed []string
	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			mu.Lock()
			executed = append(executed, req.Task.Name)
			mu.Unlock()
			if req.Task.Name == "cond" {
				return json.RawMessage(`{"result":"case-1"}`), nil
			}
			return json.RawMessage(`{}`), nil
		}).AnyTimes()

	err := f.runWithEdges(t, graph, tasksFor(graph), edges)

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["b"])
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["a"])
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["c"])
	mu.Lock()
	assert.NotContains(t, executed, "a")
	assert.NotContains(t, executed, "c")
	mu.Unlock()
}

func TestExecutorSwitchFallsThroughToElse(t *testing.T) {
	graph := map[string][]string{
		"start":    {"cond"},
		"cond":     {"a", "fallback"},
		"a":        {},
		"fallback": {},
	}
	edges := []domain.EdgeRef{
		edgeRef("cond", "a", "case-0"),
		edgeRef("cond", "fallback", "else"),
	}
	f := newExecutorFixture(t, 4)

	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			if req.Task.Name == "cond" {
				return json.RawMessage(`{"result":"else"}`), nil
			}
			return json.RawMessage(`{}`), nil
		}).AnyTimes()

	err := f.runWithEdges(t, graph, tasksFor(graph), edges)

	assert.NoError(t, err)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["fallback"])
	assert.Equal(t, []string{"skipped"}, f.taskStatuses["a"])
}

func TestExecutorDirectionalHandlesAlwaysFollow(t *testing.T) {
	// a normal node whose output happens to carry a "result" field must not
	// have its positional editor edges gated.
	graph := map[string][]string{
		"start": {"A"},
		"A":     {"B"},
		"B":     {},
	}
	edges := []domain.EdgeRef{
		edgeRef("start", "A", "source-bottom"),
		edgeRef("A", "B", "source-right"),
	}
	f := newExecutorFixture(t, 4)

	f.runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			return json.RawMessage(`{"result":"anything"}`), nil
		}).AnyTimes()

	err := f.runWithEdges(t, graph, tasksFor(graph), edges)

	assert.NoError(t, err)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["A"])
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["B"])
}

func TestStaticResolverRouting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defaultRuntime := execmocks.NewMockNodeRuntime(ctrl)
	special := execmocks.NewMockNodeRuntime(ctrl)
	resolver := execution.NewStaticResolver(defaultRuntime, map[string]execution.NodeRuntime{
		"code_snippet": special,
	})

	mapped, err := resolver.Resolve(sampleTaskWithConnector("code_snippet"))
	assert.NoError(t, err)
	assert.Same(t, special, mapped)

	fallback, err := resolver.Resolve(sampleTaskWithConnector("anything_else"))
	assert.NoError(t, err)
	assert.Same(t, defaultRuntime, fallback)

	empty := execution.NewStaticResolver(nil, nil)
	_, err = empty.Resolve(sampleTaskWithConnector("anything"))
	assert.Error(t, err)
}

func sampleTaskWithConnector(connectorID string) domain.Tasks {
	task := sampleTask("node")
	task.ConnectorID = &connectorID
	return task
}
