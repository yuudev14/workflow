package tasks_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	mock_tasks "github.com/yuudev14-workflow/workflow-service/internal/tasks/mocks"
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

func TestServiceGetTasksByWorkflowIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	returnedTasks := []tasks.Tasks{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		GetTasksByWorkflowId(workflowID).
		Return(returnedTasks)

	result, err := service.GetTasksByWorkflowId(workflowID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceUpsertTasksSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()

	tasksData := []tasks.Tasks{
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		UpsertTasks(nil, workflowID, tasksData).
		Return(tasksData, nil)

	result, err := service.UpsertTasks(nil, workflowID, tasksData)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestServiceUpsertTasksFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()

	tasksData := []tasks.Tasks{
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		UpsertTasks(nil, workflowID, tasksData).
		Return(nil, fmt.Errorf("db error"))

	result, err := service.UpsertTasks(nil, workflowID, tasksData)

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
		DeleteTasks(nil, taskIDs).
		Return(nil)

	err := service.DeleteTasks(nil, taskIDs)

	assert.NoError(t, err)
}

func TestServiceDeleteTasksFail(t *testing.T) {
	service, mockRepo := setupService(t)

	taskIDs := []uuid.UUID{
		uuid.New(),
	}

	mockRepo.
		EXPECT().
		DeleteTasks(nil, taskIDs).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteTasks(nil, taskIDs)

	assert.Error(t, err)
}

func TestServiceCreateTaskHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New().String()

	taskList := []tasks.Tasks{
		{ID: uuid.New()},
	}

	graph := map[uuid.UUID][]uuid.UUID{}

	returnedHistory := []tasks.TaskHistory{
		{
			ID: uuid.New(),
		},
	}

	mockRepo.
		EXPECT().
		CreateTaskHistory(nil, workflowHistoryID, taskList, graph).
		Return(returnedHistory, nil)

	result, err := service.CreateTaskHistory(nil, workflowHistoryID, taskList, graph)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestServiceUpdateTaskStatusSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New().String()
	taskID := uuid.New().String()
	status := "completed"

	returnedHistory := &tasks.TaskHistory{
		ID: uuid.New(),
	}

	mockRepo.
		EXPECT().
		UpdateTaskStatus(workflowHistoryID, taskID, status).
		Return(returnedHistory, nil)

	result, err := service.UpdateTaskStatus(workflowHistoryID, taskID, status)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestServiceUpdateTaskStatusFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New().String()
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
				UpdateTaskStatus(workflowHistoryID, taskID, status).
				Return(nil, tt.err)

			result, err := service.UpdateTaskStatus(workflowHistoryID, taskID, status)

			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestServiceUpdateTaskHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New().String()
	taskID := uuid.New().String()

	updateData := tasks.UpdateTaskHistoryData{}

	returnedHistory := &tasks.TaskHistory{
		ID: uuid.New(),
	}

	mockRepo.
		EXPECT().
		UpdateTaskHistory(workflowHistoryID, taskID, updateData).
		Return(returnedHistory, nil)

	result, err := service.UpdateTaskHistory(workflowHistoryID, taskID, updateData)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestServiceUpdateTaskHistoryFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New().String()
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
				UpdateTaskHistory(workflowHistoryID, taskID, updateData).
				Return(nil, tt.err)

			result, err := service.UpdateTaskHistory(workflowHistoryID, taskID, updateData)

			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestServiceGetTaskHistoryByWorkflowHistoryIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New().String()

	returnedHistory := []tasks.TaskHistory{
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	mockRepo.
		EXPECT().
		GetTaskHistoryByWorkflowHistoryId(workflowHistoryID, tasks.TaskHistoryFilter{}).
		Return(returnedHistory, nil)

	result, err := service.GetTaskHistoryByWorkflowHistoryId(workflowHistoryID, tasks.TaskHistoryFilter{})

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceGetTaskHistoryCountSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	mockRepo.
		EXPECT().
		GetTaskHistoryCount(tasks.TaskHistoryFilter{}).
		Return(5, nil)

	count, err := service.GetTaskHistoryCount(tasks.TaskHistoryFilter{})

	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}
