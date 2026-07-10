package playbooks_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mock_contracts "github.com/yuudev14/ytsoar/internal/application/contracts/mocks"
	mock_edges "github.com/yuudev14/ytsoar/internal/application/edges/mocks"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	mock_playbooks "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	mock_tasks "github.com/yuudev14/ytsoar/internal/application/tasks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/utils"
)

type testEnv struct {
	service *playbooks.PlaybookApplicationServiceImpl

	mockPlaybook    *mock_playbooks.MockPlaybookService
	mockTask        *mock_tasks.MockTaskService
	mockEdge        *mock_edges.MockEdgeService
	mockTaskSub     *mock_contracts.MockTaskPublisher
	mockBroadcaster *mock_contracts.MockStatusBroadcaster
}

func setupTest(t *testing.T) *testEnv {

	ctrl := gomock.NewController(t)

	mockPlaybook := mock_playbooks.NewMockPlaybookService(ctrl)
	mockTask := mock_tasks.NewMockTaskService(ctrl)
	mockEdge := mock_edges.NewMockEdgeService(ctrl)
	mockTaskPubSub := mock_contracts.NewMockTaskPublisher(ctrl)
	mockBroadcaster := mock_contracts.NewMockStatusBroadcaster(ctrl)
	mockBroadcaster.EXPECT().Broadcast(gomock.Any()).AnyTimes()

	// transactions pass straight through so the closure body runs against the mocks
	mockTx := mock_contracts.NewMockTxManager(ctrl)
	mockTx.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		AnyTimes()

	service := &playbooks.PlaybookApplicationServiceImpl{
		Logger:            logger.NewNop(),
		PlaybookService:   mockPlaybook,
		TaskService:       mockTask,
		EdgeService:       mockEdge,
		Tx:                mockTx,
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

	service := &playbooks.PlaybookApplicationServiceImpl{Logger: logger.NewNop()}

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
			SourceHandle:        sql.NullString{String: "true", Valid: true},
		},
	}

	tasksMap, graph, edgeRefs := service.PreparePlaybookMessage(tasksData, edgesData)

	if len(tasksMap) != 3 {
		t.Fatalf("expected 3 tasks")
	}

	if len(graph["start"]) != 1 {
		t.Fatalf("expected start to have 1 child")
	}

	if graph["task1"][0] != "task2" {
		t.Fatalf("graph incorrect")
	}

	if len(edgeRefs) != 2 {
		t.Fatalf("expected 2 edge refs")
	}
	if edgeRefs[0].SourceHandle != nil {
		t.Fatalf("edge without handle must carry nil source_handle")
	}
	if edgeRefs[1].SourceHandle == nil || *edgeRefs[1].SourceHandle != "true" {
		t.Fatalf("source_handle not propagated onto the wire")
	}
}

func TestDeleteEdges(t *testing.T) {
	tests := []struct {
		name      string
		payload   map[string][]string
		mockSetup func(testEnv *testEnv, playbookId string)
		withError bool
	}{
		{
			name:    "delete all edges",
			payload: map[string][]string{},
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockEdge.
					EXPECT().
					DeleteAllPlaybookEdges(gomock.Any(), playbookId).
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
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookId).
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
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookId).
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
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookId).
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

			playbookID := uuid.New()

			tt.mockSetup(env, playbookID.String())

			err := env.service.DeleteEdges(
				context.Background(),
				playbookID,
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

	playbookID := uuid.New()

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
		UpsertTasks(gomock.Any(), playbookID, gomock.Any()).
		Return(expected, nil)

	result, err := env.service.UpsertTasks(context.Background(), playbookID, nodes)

	if err != nil {
		t.Fatalf("unexpected error")
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 task")
	}
}

func TestInsertEdges(t *testing.T) {

	env := setupTest(t)

	playbookID := uuid.New()

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

	handles := map[string]map[string]domain.EdgeHandle{
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
		context.Background(),
		playbookID,
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
		mockSetup func(testEnv *testEnv, playbookId string)
		withError bool
	}{

		{
			name:    "delete successfully",
			payload: []tasks.TaskPayload{},
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookId).
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
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookId).
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
			mockSetup: func(testEnv *testEnv, playbookId string) {
				testEnv.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookId).
					Return([]domain.Tasks{}, fmt.Errorf("error occured"))
			},
			withError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := setupTest(t)

			playbookID := uuid.New()

			tt.mockSetup(env, playbookID.String())

			err := env.service.DeleteTasks(context.Background(), playbookID, tt.payload)

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
		mockSetup func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload)
		withError bool
	}{
		{
			name:      "no error",
			withError: false,
			mockSetup: func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload) {

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybook(gomock.Any(), playbookID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), playbookID, gomock.Any()).
					Return([]domain.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				env.mockPlaybook.
					EXPECT().
					GetPlaybookGraphById(gomock.Any(), gomock.Any()).
					Return(&domain.PlaybookGraph{}, nil)
			},
		},
		{
			name:      "error in the finale",
			withError: true,
			mockSetup: func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload) {

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybook(gomock.Any(), playbookID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), playbookID, gomock.Any()).
					Return([]domain.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.Tasks{
						{
							ID:   uuid.New(),
							Name: "task1",
						},
					}, nil)

				env.mockPlaybook.
					EXPECT().
					GetPlaybookGraphById(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "error in update playbook",
			withError: true,
			mockSetup: func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload) {

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybook(gomock.Any(), playbookID.String(), *payload.Task).
					Return(nil, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "error in GetEdgesByPlaybookId",
			withError: true,
			mockSetup: func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload) {

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybook(gomock.Any(), playbookID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.ResponseEdges{}, fmt.Errorf("error occured"))
			},
		},
		{
			name:      "no error in UpsertTasks",
			withError: true,
			mockSetup: func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload) {

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybook(gomock.Any(), playbookID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), playbookID, gomock.Any()).
					Return([]domain.Tasks{}, fmt.Errorf("error occured"))

			},
		},
		{
			name:      "error in GetTasksByPlaybookId",
			withError: true,
			mockSetup: func(env *testEnv, playbookID uuid.UUID, payload playbooks.UpdatePlaybookTasksPayload) {

				env.mockPlaybook.
					EXPECT().
					UpdatePlaybook(gomock.Any(), playbookID.String(), *payload.Task).
					Return(&domain.Playbooks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID.String()).
					Return([]domain.ResponseEdges{}, nil)

				env.mockTask.
					EXPECT().
					UpsertTasks(gomock.Any(), playbookID, gomock.Any()).
					Return([]domain.Tasks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID.String()).
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

			playbookID := uuid.New()

			tt.mockSetup(
				env,
				playbookID,
				payload,
			)
			_, err := env.service.UpdatePlaybookTasks(context.Background(), playbookID.String(), payload)
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
		mockSetup func(env *testEnv, playbookID string)
		withError bool
	}{
		{
			name:      "playbook error",
			withError: true,
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(nil, fmt.Errorf("playbook error"))
			},
		},
		{
			name:      "task error",
			withError: true,
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID).
					Return(nil, fmt.Errorf("task error"))
			},
		},
		{
			name:      "edge error",
			withError: true,
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID).
					Return(nil, fmt.Errorf("edge error"))
			},
		},
		{
			name:      "create playbook history error",
			withError: true,
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.ResponseEdges{}, nil)

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), playbookID, gomock.Any()).
					Return(nil, fmt.Errorf("history error"))

			},
		},
		{
			name:      "create task history error",
			withError: true,
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.Tasks{}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), playbookID, gomock.Any()).
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
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.Tasks{
						{ID: uuid.New(), Name: "task1"},
					}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), playbookID, gomock.Any()).
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
			mockSetup: func(env *testEnv, playbookID string) {

				env.mockPlaybook.
					EXPECT().
					GetPlaybookById(gomock.Any(), playbookID).
					Return(&domain.Playbooks{}, nil)

				env.mockTask.
					EXPECT().
					GetTasksByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.Tasks{
						{ID: uuid.New(), Name: "task1"},
					}, nil)

				env.mockEdge.
					EXPECT().
					GetEdgesByPlaybookId(gomock.Any(), playbookID).
					Return([]domain.ResponseEdges{}, nil)

				historyID := uuid.New()

				env.mockPlaybook.
					EXPECT().
					CreatePlaybookHistory(gomock.Any(), playbookID, gomock.Any()).
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

			env := setupTest(t)

			playbookID := uuid.New().String()

			tt.mockSetup(env, playbookID)

			_, err := env.service.TriggerPlaybook(context.Background(), playbookID)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
