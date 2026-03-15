package workflow_application

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	mock_edges "github.com/yuudev14-workflow/workflow-service/internal/edges/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	mock_mq "github.com/yuudev14-workflow/workflow-service/internal/infra/mq/mock"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	mock_tasks "github.com/yuudev14-workflow/workflow-service/internal/tasks/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/utils"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
	mock_workflows "github.com/yuudev14-workflow/workflow-service/internal/workflows/mocks"
)

type testEnv struct {
	service *WorkflowApplicationServiceImpl

	mockWorkflow *mock_workflows.MockWorkflowService
	mockTask     *mock_tasks.MockTaskService
	mockEdge     *mock_edges.MockEdgeService
	mockTaskSub  *mock_mq.MockTaskPubSub
}

func setupDBMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error creating sqlmock: %v", err)
	}

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	return sqlxDB, mock
}

func setupTest(t *testing.T) *testEnv {
	logging.Setup("Debug")
	ctrl := gomock.NewController(t)

	mockWorkflow := mock_workflows.NewMockWorkflowService(ctrl)
	mockTask := mock_tasks.NewMockTaskService(ctrl)
	mockEdge := mock_edges.NewMockEdgeService(ctrl)
	mockTaskPubSub := mock_mq.NewMockTaskPubSub(ctrl)

	service := &WorkflowApplicationServiceImpl{
		WorkflowService: mockWorkflow,
		TaskService:     mockTask,
		EdgeService:     mockEdge,
		DB:              &sqlx.DB{},
		TaskPubSUb:      mockTaskPubSub,
	}

	return &testEnv{
		service:      service,
		mockWorkflow: mockWorkflow,
		mockTask:     mockTask,
		mockEdge:     mockEdge,
		mockTaskSub:  mockTaskPubSub,
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
	tests := []struct {
		name      string
		edges     map[string][]string
		nodes     []tasks.TaskPayload
		withError bool
	}{
		{
			name: "with no error",
			edges: map[string][]string{
				"start": {"task1"},
			},
			nodes: []tasks.TaskPayload{
				{Name: "start"},
				{Name: "task1"},
			},
			withError: false,
		},

		{
			name:  "no start in edges",
			edges: map[string][]string{},
			nodes: []tasks.TaskPayload{
				{Name: "start"},
				{Name: "task1"},
			},
			withError: true,
		},
		{
			name: "no start in edges",
			edges: map[string][]string{
				"start": {"task1"},
			},
			nodes:     []tasks.TaskPayload{},
			withError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tasks.UpdateWorkflowtasks{
				Edges: tt.edges,
				Nodes: tt.nodes,
			}

			err := validateWorkflowTaskPayload(body)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}

func TestDeleteEdges(t *testing.T) {
	tests := []struct {
		name      string
		payload   map[string][]string
		mockSetup func(testEnv *testEnv, workflowId string)
		withError bool
	}{
		{
			name:    "delete all edges",
			payload: map[string][]string{},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockEdge.
					EXPECT().
					DeleteAllWorkflowEdges(gomock.Any(), workflowId).
					Return(nil)
			},
			withError: false,
		},
		{
			name: "delete no edges",
			payload: map[string][]string{
				"start": {"task1"},
				"task2": {"task5"},
			},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowId).
					Return([]edges.ResponseEdges{}, nil)
			},
			withError: false,
		},
		{
			name: "delete some edges",
			payload: map[string][]string{
				"start": {"task1"},
				"task2": {"task5"},
			},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowId).
					Return([]edges.ResponseEdges{
						{
							ID:                  uuid.New(),
							SourceTaskName:      "start",
							DestinationTaskName: "task1",
						},
						{
							ID:                  uuid.New(),
							SourceTaskName:      "start",
							DestinationTaskName: "task2",
						},
					}, nil)

				testEnv.mockEdge.
					EXPECT().
					DeleteEdges(gomock.Any(), gomock.Len(1)).
					Return(nil)
			},
			withError: false,
		},
		{
			name: "with error",
			payload: map[string][]string{
				"start": {"task1"},
				"task2": {"task5"},
			},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowId).
					Return([]edges.ResponseEdges{}, fmt.Errorf("error occured"))

				// testEnv.mockEdge.
				// 	EXPECT().
				// 	DeleteEdges(gomock.Any(), gomock.Len(1)).
				// 	Return(nil)
			},
			withError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			env := setupTest(t)

			workflowID := uuid.New()

			tt.mockSetup(env, workflowID.String())

			err := env.service.DeleteEdges(
				nil,
				workflowID,
				tt.payload,
			)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})

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

	handles := &map[string]map[string]edges.EdgeHandle{
		"start": {
			"start": {
				SourceHandle:      utils.StrPtr("start"),
				DestinationHandle: utils.StrPtr("start"),
			},
			"task1": {
				SourceHandle:      utils.StrPtr("task1"),
				DestinationHandle: utils.StrPtr("task1"),
			},
		},
	}

	err := env.service.InsertEdges(
		nil,
		workflowID,
		edgesPayload,
		tasksList,
		handles,
	)

	if err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestDeleteTasks(t *testing.T) {

	tests := []struct {
		name      string
		payload   []tasks.TaskPayload
		mockSetup func(testEnv *testEnv, workflowId string)
		withError bool
	}{

		{
			name:    "delete successfully",
			payload: []tasks.TaskPayload{},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowId).
					Return([]tasks.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				testEnv.mockTask.
					EXPECT().
					DeleteTasks(gomock.Any(), gomock.Any()).
					Return(nil)
			},
			withError: false,
		},
		{
			name: "noting to delete",
			payload: []tasks.TaskPayload{
				{
					Name: "task1",
				},
			},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowId).
					Return([]tasks.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)
			},
			withError: false,
		},
		{
			name: "error occured",
			payload: []tasks.TaskPayload{
				{
					Name: "task1",
				},
			},
			mockSetup: func(testEnv *testEnv, workflowId string) {
				testEnv.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowId).
					Return([]tasks.Tasks{}, fmt.Errorf("error occured"))
			},
			withError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTest(t)

			workflowID := uuid.New()

			tt.mockSetup(env, workflowID.String())

			err := env.service.DeleteTasks(nil, workflowID, tt.payload)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}

}

func TestUpdateWorkflowTasks(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks)
		withError bool
	}{
		{
			name:      "no error",
			withError: false,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					UpdateWorkflowTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&workflows.Workflows{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID.String()).
					Return([]edges.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]tasks.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID.String()).
					Return([]tasks.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				env.mockWorkflow.
					EXPECT().
					GetWorkflowGraphById(gomock.Any()).
					Return(&workflows.WorkflowsGraph{}, nil)
			},
		},
		{
			name:      "error in the finale",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					UpdateWorkflowTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&workflows.Workflows{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID.String()).
					Return([]edges.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]tasks.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID.String()).
					Return([]tasks.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				env.mockWorkflow.
					EXPECT().
					GetWorkflowGraphById(gomock.Any()).
					Return(nil, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "error in update workflow",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					UpdateWorkflowTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(nil, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "error in GetEdgesByWorkflowId",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					UpdateWorkflowTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&workflows.Workflows{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID.String()).
					Return([]edges.ResponseEdges{}, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "no error in UpsertTasks",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					UpdateWorkflowTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&workflows.Workflows{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID.String()).
					Return([]edges.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]tasks.Tasks{}, fmt.Errorf("error occured"))

			},
		},
		{
			name:      "error in GetTasksByWorkflowId",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload tasks.UpdateWorkflowtasks) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					UpdateWorkflowTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&workflows.Workflows{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID.String()).
					Return([]edges.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]tasks.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID.String()).
					Return([]tasks.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, fmt.Errorf("error occured"))

				// env.mockWorkflow.
				// 	EXPECT().
				// 	GetWorkflowGraphById(gomock.Any()).
				// 	Return(&workflows.WorkflowsGraph{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, dbMock := setupDBMock(t)
			nodes := []tasks.TaskPayload{
				{
					Name: "start",
				},
				{
					Name: "task1",
				},
			}
			edges_ := map[string][]string{
				"start": {"task1"},
			}

			payload := tasks.UpdateWorkflowtasks{
				Task:  &workflows.UpdateWorkflowData{},
				Nodes: nodes,
				Edges: edges_,
			}

			env := setupTest(t)
			env.service.DB = db

			workflowID := uuid.New()

			tt.mockSetup(
				env,
				workflowID,
				dbMock,
				payload,
			)
			_, err := env.service.UpdateWorkflowTasks(workflowID.String(), payload)
			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

			}

		})
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

func TestTriggerWorkflow(t *testing.T) {

	tests := []struct {
		name      string
		mockSetup func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock)
		withError bool
	}{
		{
			name:      "workflow error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(nil, fmt.Errorf("workflow error"))
			},
		},
		{
			name:      "task error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(&workflows.Workflows{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID).
					Return(nil, fmt.Errorf("task error"))
			},
		},
		{
			name:      "edge error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(&workflows.Workflows{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID).
					Return([]tasks.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID).
					Return(nil, fmt.Errorf("edge error"))
			},
		},
		{
			name:      "create workflow history error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				dbMock.ExpectBegin()
				dbMock.ExpectRollback()

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(&workflows.Workflows{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID).
					Return([]tasks.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID).
					Return([]edges.ResponseEdges{}, nil)

				env.mockWorkflow.
					EXPECT().
					CreateWorkflowHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(nil, fmt.Errorf("history error"))
			},
		},
		{
			name:      "create task history error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				dbMock.ExpectBegin()
				dbMock.ExpectRollback()

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(&workflows.Workflows{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID).
					Return([]tasks.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID).
					Return([]edges.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockWorkflow.
					EXPECT().
					CreateWorkflowHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(&workflows.WorkflowHistory{ID: historyID}, nil)

				env.mockTask.
					EXPECT().
					CreateTaskHistory(gomock.Any(), historyID.String(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("task history error"))
			},
		},
		{
			name:      "mq error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(&workflows.Workflows{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID).
					Return([]tasks.Tasks{
						{ID: uuid.New(), Name: "task1"},
					}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID).
					Return([]edges.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockWorkflow.
					EXPECT().
					CreateWorkflowHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(&workflows.WorkflowHistory{ID: historyID}, nil)

				env.mockTask.
					EXPECT().
					CreateTaskHistory(gomock.Any(), historyID.String(), gomock.Any(), gomock.Any()).
					Return([]tasks.TaskHistory{}, nil)

				env.mockTaskSub.
					EXPECT().
					SendMessage(gomock.Any()).
					Return(fmt.Errorf("mq error"))
			},
		},
		{
			name:      "success",
			withError: false,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockWorkflow.
					EXPECT().
					GetWorkflowById(workflowID).
					Return(&workflows.Workflows{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByWorkflowId(workflowID).
					Return([]tasks.Tasks{
						{ID: uuid.New(), Name: "task1"},
					}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByWorkflowId(workflowID).
					Return([]edges.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockWorkflow.
					EXPECT().
					CreateWorkflowHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(&workflows.WorkflowHistory{ID: historyID}, nil)

				env.mockTask.
					EXPECT().
					CreateTaskHistory(gomock.Any(), historyID.String(), gomock.Any(), gomock.Any()).
					Return([]tasks.TaskHistory{}, nil)

				env.mockTaskSub.
					EXPECT().
					SendMessage(gomock.Any()).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			db, dbMock := setupDBMock(t)

			env := setupTest(t)
			env.service.DB = db

			workflowID := uuid.New().String()

			tt.mockSetup(env, workflowID, dbMock)

			_, err := env.service.TriggerWorkflow(workflowID)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
