package tasks_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	mock_tasks "github.com/yuudev14/ytsoar/internal/application/tasks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logging.Setup("DEBUG")
	os.Exit(m.Run())
}

func setupService(t *testing.T) (tasks.TaskService, *mock_tasks.MockTaskRepository) {
	ctrl := gomock.NewController(t)

	mockRepo := mock_tasks.NewMockTaskRepository(ctrl)
	service := tasks.NewTaskServiceImpl(mockRepo)

	t.Cleanup(ctrl.Finish)

	return service, mockRepo
}

func TestServiceGetTasksByPlaybookIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New().String()

	returnedTasks := []domain.Tasks{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		GetTasksByPlaybookId(gomock.Any(), playbookID).
		Return(returnedTasks)

	result, err := service.GetTasksByPlaybookId(context.Background(), playbookID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceUpsertTasksSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New()

	tasksData := []domain.Tasks{
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		UpsertTasks(gomock.Any(), playbookID, tasksData).
		Return(tasksData, nil)

	result, err := service.UpsertTasks(context.Background(), playbookID, tasksData)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestServiceUpsertTasksFail(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New()

	tasksData := []domain.Tasks{
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		UpsertTasks(gomock.Any(), playbookID, tasksData).
		Return(nil, fmt.Errorf("db error"))

	result, err := service.UpsertTasks(context.Background(), playbookID, tasksData)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestServiceDeleteTasksSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	taskIDs := []uuid.UUID{
		uuid.New(),
		uuid.New(),
	}

	mockRepo.
		EXPECT().
		DeleteTasks(gomock.Any(), taskIDs).
		Return(nil)

	err := service.DeleteTasks(context.Background(), taskIDs)

	assert.NoError(t, err)
}

func TestServiceDeleteTasksFail(t *testing.T) {
	service, mockRepo := setupService(t)

	taskIDs := []uuid.UUID{
		uuid.New(),
	}

	mockRepo.
		EXPECT().
		DeleteTasks(gomock.Any(), taskIDs).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteTasks(context.Background(), taskIDs)

	assert.Error(t, err)
}

func TestServiceCreateTaskHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New().String()

	taskList := []domain.Tasks{
		{ID: uuid.New()},
	}

	graph := map[uuid.UUID][]uuid.UUID{}

	returnedHistory := []domain.TaskHistory{
		{
			ID: uuid.New(),
		},
	}

	mockRepo.
		EXPECT().
		CreateTaskHistory(gomock.Any(), playbookHistoryID, taskList, graph).
		Return(returnedHistory, nil)

	result, err := service.CreateTaskHistory(context.Background(), playbookHistoryID, taskList, graph)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestServiceUpdateTaskStatusSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New().String()
	taskID := uuid.New().String()
	status := "completed"

	returnedHistory := &domain.TaskHistory{
		ID: uuid.New(),
	}

	mockRepo.
		EXPECT().
		UpdateTaskStatus(gomock.Any(), playbookHistoryID, taskID, status).
		Return(returnedHistory, nil)

	result, err := service.UpdateTaskStatus(context.Background(), playbookHistoryID, taskID, status)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestServiceUpdateTaskStatusFail(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New().String()
	taskID := uuid.New().String()
	status := "completed"

	tests := []struct {
		name string
		err  error
	}{
		{"repo error", fmt.Errorf("db error")},
		{"nil response", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockRepo.
				EXPECT().
				UpdateTaskStatus(gomock.Any(), playbookHistoryID, taskID, status).
				Return(nil, tt.err)

			result, err := service.UpdateTaskStatus(context.Background(), playbookHistoryID, taskID, status)

			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestServiceUpdateTaskHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New().String()
	taskID := uuid.New().String()

	updateData := tasks.UpdateTaskHistoryData{}

	returnedHistory := &domain.TaskHistory{
		ID: uuid.New(),
	}

	mockRepo.
		EXPECT().
		UpdateTaskHistory(gomock.Any(), playbookHistoryID, taskID, updateData).
		Return(returnedHistory, nil)

	result, err := service.UpdateTaskHistory(context.Background(), playbookHistoryID, taskID, updateData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestServiceUpdateTaskHistoryFail(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New().String()
	taskID := uuid.New().String()

	updateData := tasks.UpdateTaskHistoryData{}

	tests := []struct {
		name string
		err  error
	}{
		{"repo error", fmt.Errorf("db error")},
		{"nil response", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockRepo.
				EXPECT().
				UpdateTaskHistory(gomock.Any(), playbookHistoryID, taskID, updateData).
				Return(nil, tt.err)

			result, err := service.UpdateTaskHistory(context.Background(), playbookHistoryID, taskID, updateData)

			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestServiceGetTaskHistoryByPlaybookHistoryIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New().String()

	returnedHistory := []domain.TaskHistory{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		GetTaskHistoryByPlaybookHistoryId(gomock.Any(), playbookHistoryID, tasks.TaskHistoryFilter{}).
		Return(returnedHistory, nil)

	result, err := service.GetTaskHistoryByPlaybookHistoryId(context.Background(), playbookHistoryID, tasks.TaskHistoryFilter{})

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceGetTaskHistoryCountSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	mockRepo.
		EXPECT().
		GetTaskHistoryCount(gomock.Any(), tasks.TaskHistoryFilter{}).
		Return(5, nil)

	count, err := service.GetTaskHistoryCount(context.Background(), tasks.TaskHistoryFilter{})

	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}
