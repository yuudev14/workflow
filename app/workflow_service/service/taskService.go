package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
)

type TaskService interface {
	GetTasksByWorkflowId(workflowId string) ([]models.Tasks, error)
	UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []models.Tasks) ([]models.Tasks, error)
	DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error
	CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []models.Tasks) ([]models.TaskHistory, error)
	UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*models.TaskHistory, error)
	UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory dto.UpdateTaskHistoryData) (*models.TaskHistory, error)
}

type TaskServiceImpl struct {
	TaskRepository  repository.TaskRepository
	WorkflowService WorkflowService
}

func NewTaskServiceImpl(TaskService repository.TaskRepository, WorkflowService WorkflowService) TaskService {
	return &TaskServiceImpl{
		TaskRepository:  TaskService,
		WorkflowService: WorkflowService,
	}
}

// CreateTaskHistory implements TaskService.
func (t *TaskServiceImpl) CreateTaskHistory(tx *sqlx.Tx, workflowHistoryId string, tasks []models.Tasks) ([]models.TaskHistory, error) {
	return t.TaskRepository.CreateTaskHistory(tx, workflowHistoryId, tasks)
}

// get tasks by workflow id
func (t *TaskServiceImpl) GetTasksByWorkflowId(workflowId string) ([]models.Tasks, error) {
	_, err := t.WorkflowService.GetWorkflowById(workflowId)
	if err != nil {
		return nil, err
	}
	return t.TaskRepository.GetTasksByWorkflowId(workflowId), nil
}

// upsert tasks. insert multiple tasks.
// if task does not exist yet add the task in the database
// else update the content of the task
func (t *TaskServiceImpl) UpsertTasks(tx *sqlx.Tx, workflowId uuid.UUID, tasks []models.Tasks) ([]models.Tasks, error) {
	return t.TaskRepository.UpsertTasks(tx, workflowId, tasks)
}

// Delete multiple tasks based on the taskIds
func (t *TaskServiceImpl) DeleteTasks(tx *sqlx.Tx, taskIds []uuid.UUID) error {
	return t.TaskRepository.DeleteTasks(tx, taskIds)
}

// UpdateTaskStatus implements TaskService.
func (t *TaskServiceImpl) UpdateTaskStatus(workflowHistoryId string, taskId string, status string) (*models.TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskStatus(workflowHistoryId, taskId, status)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}

// UpdateTaskStatus implements TaskService.
func (t *TaskServiceImpl) UpdateTaskHistory(workflowHistoryId string, taskId string, taskHistory dto.UpdateTaskHistoryData) (*models.TaskHistory, error) {
	res, err := t.TaskRepository.UpdateTaskHistory(workflowHistoryId, taskId, taskHistory)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no task history was updated")
	}

	return res, nil
}
