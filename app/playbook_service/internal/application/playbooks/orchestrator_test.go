package playbooks_test

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_contracts "github.com/yuudev14/ytsoar/internal/application/contracts/mocks"
	mock_edges "github.com/yuudev14/ytsoar/internal/application/edges/mocks"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	mock_workflows "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	mock_tasks "github.com/yuudev14/ytsoar/internal/application/tasks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
	"github.com/yuudev14/ytsoar/internal/utils"
)

type testEnv struct {
	service *playbooks.PlaybookApplicationServiceImpl

	mockPlaybook    *mock_workflows.MockPlaybookService
	mockTask        *mock_tasks.MockTaskService
	mockEdge        *mock_edges.MockEdgeService
	mockTaskSub     *mock_contracts.MockTaskPublisher
	mockBroadcaster *mock_contracts.MockStatusBroadcaster
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

	mockPlaybook := mock_workflows.NewMockPlaybookService(ctrl)
	mockTask := mock_tasks.NewMockTaskService(ctrl)
	mockEdge := mock_edges.NewMockEdgeService(ctrl)
	mockTaskPubSub := mock_contracts.NewMockTaskPublisher(ctrl)
	mockBroadcaster := mock_contracts.NewMockStatusBroadcaster(ctrl)
	mockBroadcaster.EXPECT().Broadcast(gomock.Any()).AnyTimes()

	service := &playbooks.PlaybookApplicationServiceImpl{
		PlaybookService:   mockPlaybook,
		TaskService:       mockTask,
		EdgeService:       mockEdge,
		DB:                &sqlx.DB{},
		TaskPublisher:     mockTaskPubSub,
		StatusBroadcaster: mockBroadcaster,
	}

	return &testEnv{
		service:         service,
		mockPlaybook:    mockPlaybook,
		mockTask:        mockTask,
		mockEdge:        mockEdge,
		mockTaskSub:     mockTaskPubSub,
		mockBroadcaster: mockBroadcaster,
	}
}

func TestPreparePlaybookMessage(t *testing.T) {

	service := &playbooks.PlaybookApplicationServiceImpl{}

	tasksData := []domain.Tasks{
		{Name: "start"},
		{Name: "task1"},
		{Name: "task2"},
	}

	edgesData := []domain.ResponseEdges{
		{
			SourceTaskName:      "start",
			DestinationTaskName: "task1",
		},
		{
			SourceTaskName:      "task1",
			DestinationTaskName: "task2",
		},
	}

	tasksMap, graph := service.PreparePlaybookMessage(tasksData, edgesData)

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
					DeleteAllPlaybookEdges(gomock.Any(), workflowId).
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
					GetEdgesByPlaybookId(workflowId).
					Return([]domain.ResponseEdges{}, nil)
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
					GetEdgesByPlaybookId(workflowId).
					Return([]domain.ResponseEdges{
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
					GetEdgesByPlaybookId(workflowId).
					Return([]domain.ResponseEdges{}, fmt.Errorf("error occured"))

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

	expected := []domain.Tasks{
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

	tasksList := []domain.Tasks{
		{Name: "start", ID: taskID1},
		{Name: "task1", ID: taskID2},
	}

	edgesPayload := map[string][]string{
		"start": {"task1"},
	}

	env.mockEdge.
		EXPECT().
		InsertEdges(gomock.Any(), gomock.Any()).
		Return([]domain.Edges{}, nil)

	handles := &map[string]map[string]domain.EdgeHandle{
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
					GetTasksByPlaybookId(workflowId).
					Return([]domain.Tasks{
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
					GetTasksByPlaybookId(workflowId).
					Return([]domain.Tasks{
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
					GetTasksByPlaybookId(workflowId).
					Return([]domain.Tasks{}, fmt.Errorf("error occured"))
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

func TestUpdatePlaybookTasks(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload)
		withError bool
	}{
		{
			name:      "no error",
			withError: false,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybookTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]domain.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID.String()).
					Return([]domain.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				env.mockPlaybook.
					EXPECT().
					GetPlaybookGraphById(gomock.Any()).
					Return(&domain.PlaybookGraph{}, nil)
			},
		},
		{
			name:      "error in the finale",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybookTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]domain.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID.String()).
					Return([]domain.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				env.mockPlaybook.
					EXPECT().
					GetPlaybookGraphById(gomock.Any()).
					Return(nil, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "error in update workflow",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybookTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(nil, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "error in GetEdgesByPlaybookId",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybookTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID.String()).
					Return([]domain.ResponseEdges{}, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "no error in UpsertTasks",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybookTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]domain.Tasks{}, fmt.Errorf("error occured"))

			},
		},
		{
			name:      "error in GetTasksByPlaybookId",
			withError: true,
			mockSetup: func(env *testEnv, workflowID uuid.UUID, dbMock sqlmock.Sqlmock, payload playbooks.UpdatePlaybookTasksPayload) {

				dbMock.ExpectBegin()
				dbMock.ExpectCommit()

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybookTx(gomock.Any(), workflowID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), workflowID, gomock.Any()).
					Return([]domain.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID.String()).
					Return([]domain.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, fmt.Errorf("error occured"))

				// env.mockPlaybook.
				// 	EXPECT().
				// 	GetPlaybookGraphById(gomock.Any()).
				// 	Return(&domain.PlaybookGraph{}, nil)
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

			payload := playbooks.UpdatePlaybookTasksPayload{
				Task:  &playbooks.UpdatePlaybookData{},
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
			_, err := env.service.UpdatePlaybookTasks(workflowID.String(), payload)
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

	edgesData := []domain.ResponseEdges{
		{
			SourceID:      source,
			DestinationID: dest,
		},
	}

	graph := playbooks.GetGraphUUIDS(edgesData)

	if len(graph[source]) != 1 {
		t.Fatalf("expected edge")
	}
}

func TestTriggerPlaybook(t *testing.T) {

	tests := []struct {
		name      string
		mockSetup func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock)
		withError bool
	}{
		{
			name:      "workflow error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(nil, fmt.Errorf("workflow error"))
			},
		},
		{
			name:      "task error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID).
					Return(nil, fmt.Errorf("task error"))
			},
		},
		{
			name:      "edge error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID).
					Return([]domain.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID).
					Return(nil, fmt.Errorf("edge error"))
			},
		},
		{
			name:      "create workflow history error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				dbMock.ExpectBegin()
				dbMock.ExpectRollback()

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID).
					Return([]domain.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID).
					Return([]domain.ResponseEdges{}, nil)

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(nil, fmt.Errorf("history error"))

			},
		},
		{
			name:      "create task history error",
			withError: true,
			mockSetup: func(env *testEnv, workflowID string, dbMock sqlmock.Sqlmock) {

				dbMock.ExpectBegin()
				dbMock.ExpectRollback()

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID).
					Return([]domain.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID).
					Return([]domain.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(&domain.PlaybookHistory{ID: historyID}, nil)

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

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID).
					Return([]domain.Tasks{
						{ID: uuid.New(), Name: "task1"},
					}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID).
					Return([]domain.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(&domain.PlaybookHistory{ID: historyID}, nil)

				env.mockTask.
					EXPECT().
					CreateTaskHistory(gomock.Any(), historyID.String(), gomock.Any(), gomock.Any()).
					Return([]domain.TaskHistory{}, nil)

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

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(workflowID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(workflowID).
					Return([]domain.Tasks{
						{ID: uuid.New(), Name: "task1"},
					}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(workflowID).
					Return([]domain.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), workflowID, gomock.Any()).
					Return(&domain.PlaybookHistory{ID: historyID}, nil)

				env.mockTask.
					EXPECT().
					CreateTaskHistory(gomock.Any(), historyID.String(), gomock.Any(), gomock.Any()).
					Return([]domain.TaskHistory{}, nil)

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

			_, err := env.service.TriggerPlaybook(workflowID)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
