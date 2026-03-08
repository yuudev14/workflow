package workflow_application

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	mock_edges "github.com/yuudev14-workflow/workflow-service/internal/edges/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	mock_tasks "github.com/yuudev14-workflow/workflow-service/internal/tasks/mocks"
	mock_workflows "github.com/yuudev14-workflow/workflow-service/internal/workflows/mocks"
)

type testEnv struct {
	service *WorkflowApplicationServiceImpl

	mockWorkflow *mock_workflows.MockWorkflowService
	mockTask     *mock_tasks.MockTaskService
	mockEdge     *mock_edges.MockEdgeService
}

func setupTest(t *testing.T) *testEnv {
	logging.Setup("Debug")
	ctrl := gomock.NewController(t)

	mockWorkflow := mock_workflows.NewMockWorkflowService(ctrl)
	mockTask := mock_tasks.NewMockTaskService(ctrl)
	mockEdge := mock_edges.NewMockEdgeService(ctrl)

	service := &WorkflowApplicationServiceImpl{
		WorkflowService: mockWorkflow,
		TaskService:     mockTask,
		EdgeService:     mockEdge,
		DB:              &sqlx.DB{}, // not used in pure unit tests
	}

	return &testEnv{
		service:      service,
		mockWorkflow: mockWorkflow,
		mockTask:     mockTask,
		mockEdge:     mockEdge,
	}
}

func TestPrepareWorkflowMessage(t *testing.T) {

	service := &WorkflowApplicationServiceImpl{}

	tasksData := []tasks.Tasks{
		{Name: "start"},
		{Name: "task1"},
		{Name: "task2"},
	}

	edgesData := []edges.ResponseEdges{
		{
			SourceTaskName:      "start",
			DestinationTaskName: "task1",
		},
		{
			SourceTaskName:      "task1",
			DestinationTaskName: "task2",
		},
	}

	tasksMap, graph := service.PrepareWorkflowMessage(tasksData, edgesData)

	if len(tasksMap) != 3 {
		t.Fatalf("expected 3 tasks")
	}

	if len(graph["start"]) != 1 {
		t.Fatalf("expected start to have 1 child")
	}

	if graph["task1"][0] != "task2" {
		t.Fatalf("graph incorrect")
	}
}

func TestValidateWorkflowTaskPayload(t *testing.T) {

	body := tasks.UpdateWorkflowtasks{
		Edges: map[string][]string{
			"start": {"task1"},
		},
		Nodes: []tasks.TaskPayload{
			{Name: "start"},
			{Name: "task1"},
		},
	}

	err := validateWorkflowTaskPayload(body)

	if err != nil {
		t.Fatalf("expected no error")
	}
}

func TestDeleteEdges_DeleteAll(t *testing.T) {

	env := setupTest(t)

	workflowID := uuid.New()

	env.mockEdge.
		EXPECT().
		DeleteAllWorkflowEdges(gomock.Any(), workflowID.String()).
		Return(nil)

	err := env.service.DeleteEdges(
		nil,
		workflowID,
		map[string][]string{},
	)

	if err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestUpsertTasks(t *testing.T) {

	env := setupTest(t)

	workflowID := uuid.New()

	nodes := []tasks.TaskPayload{
		{
			Name: "task1",
		},
	}

	expected := []tasks.Tasks{
		{
			Name: "task1",
		},
	}

	env.mockTask.
		EXPECT().
		UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
		Return(expected, nil)

	result, err := env.service.UpsertTasks(nil, workflowID, nodes)

	if err != nil {
		t.Fatalf("unexpected error")
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 task")
	}
}

func TestInsertEdges(t *testing.T) {

	env := setupTest(t)

	workflowID := uuid.New()

	taskID1 := uuid.New()
	taskID2 := uuid.New()

	tasksList := []tasks.Tasks{
		{Name: "start", ID: taskID1},
		{Name: "task1", ID: taskID2},
	}

	edgesPayload := map[string][]string{
		"start": {"task1"},
	}

	env.mockEdge.
		EXPECT().
		InsertEdges(gomock.Any(), gomock.Any()).
		Return([]edges.Edges{}, nil)

	err := env.service.InsertEdges(
		nil,
		workflowID,
		edgesPayload,
		tasksList,
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestDeleteTasks(t *testing.T) {

	env := setupTest(t)

	workflowID := uuid.New()

	existingTasks := []tasks.Tasks{
		{
			ID:   uuid.New(),
			Name: "task1",
		},
	}

	env.mockTask.
		EXPECT().
		GetTasksByWorkflowId(workflowID.String()).
		Return(existingTasks, nil)

	env.mockTask.
		EXPECT().
		DeleteTasks(gomock.Any(), gomock.Any()).
		Return(nil)

	nodes := []tasks.TaskPayload{}

	err := env.service.DeleteTasks(nil, workflowID, nodes)

	if err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestGetGraphUUIDS(t *testing.T) {

	source := uuid.New()
	dest := uuid.New()

	edgesData := []edges.ResponseEdges{
		{
			SourceID:      source,
			DestinationID: dest,
		},
	}

	graph := GetGraphUUIDS(edgesData)

	if len(graph[source]) != 1 {
		t.Fatalf("expected edge")
	}
}
