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
		DoAndReturn(func(_ context.Context, _ string, _ string, data tasks.UpdateTaskHistoryData) (*domain.TaskHistory, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if data.Status.Value != nil {
				f.taskStatuses[data.Name] = append(f.taskStatuses[data.Name], *data.Status.Value)
			}
			return &domain.TaskHistory{}, nil
		}).AnyTimes()

	f.playbookService.EXPECT().
		UpdatePlaybookHistory(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, data playbooks.UpdatePlaybookHistoryData) (*domain.PlaybookHistory, error) {
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
	return f.executor.Run(context.Background(), domain.TaskMessage{
		Graph:             graph,
		Tasks:             taskInfo,
		PlaybookHistoryId: uuid.New(),
	})
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
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["start"])
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

func TestExecutorStartNodeIsNoOpSuccess(t *testing.T) {
	graph := map[string][]string{"start": {}}
	f := newExecutorFixture(t, 1)
	// no runtime.Execute expectation: calling it would fail the test

	err := f.run(t, graph, tasksFor(graph))

	assert.NoError(t, err)
	assert.Equal(t, "success", f.playbookFinal)
	assert.Equal(t, []string{"in_progress", "success"}, f.taskStatuses["start"])
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
