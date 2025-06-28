package service

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/mq"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
)

type WorkflowTriggerService interface {
	TriggerWorkflow(workflowId string) error
	PrepareWorkflowMessage(tasks []models.Tasks, edges []repository.Edges) (map[string]models.Tasks, map[string][]string)
}

type TaskMessage struct {
	Graph             map[string][]string     `json:"graph"`
	Tasks             map[string]models.Tasks `json:"tasks"`
	WorkflowHistoryId uuid.UUID               `json:"workflow_history_id"`
}

type WorkflowTriggerServiceImpl struct {
	WorkflowService WorkflowService
	TaskService     TaskService
	EdgeService     EdgeService
}

func NewWorflowTriggerService(WorkflowService WorkflowService, TaskService TaskService, EdgeService EdgeService) WorkflowTriggerService {
	return &WorkflowTriggerServiceImpl{
		WorkflowService: WorkflowService,
		TaskService:     TaskService,
		EdgeService:     EdgeService,
	}
}

func SendTaskMessage(graph TaskMessage) error {
	jsonData, jsonErr := json.Marshal(graph)

	if jsonErr != nil {
		return jsonErr
	}
	err := mq.MQChannel.Publish(
		"",                  // exchange
		mq.SenderQueue.Name, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(jsonData),
		})
	if err != nil {
		return err
	}

	logging.Sugar.Infow("successfully pushed the message", "jsonData", string(jsonData))
	return nil

}

// TriggerWorkflow implements WorkflowTriggerService.
func (w *WorkflowTriggerServiceImpl) TriggerWorkflow(workflowId string) error {
	_, workflowErr := w.WorkflowService.GetWorkflowById(workflowId)

	if workflowErr != nil {
		logging.Sugar.Error(workflowErr)
		return workflowErr
	}
	tasks, tasksErr := w.TaskService.GetTasksByWorkflowId(workflowId)
	if tasksErr != nil {
		logging.Sugar.Errorf("error: ", tasksErr)
		return tasksErr
	}

	edges, edgesErr := w.EdgeService.GetEdgesByWorkflowId(workflowId)

	if edgesErr != nil {
		logging.Sugar.Errorf("error: ", edgesErr)
		return edgesErr
	}
	tasksMap, graph := w.PrepareWorkflowMessage(tasks, edges)

	// create transacton

	tx, txErr := db.DB.Beginx()
	if txErr != nil {
		tx.Rollback()
		return txErr
	}

	workflowHistory, workflowHistoryErr := w.WorkflowService.CreateWorkflowHistory(tx, workflowId)
	if workflowHistoryErr != nil {
		tx.Rollback()
		return workflowHistoryErr
	}

	// Log the ID to verify it's correct
	logging.Sugar.Infof("Created workflow history with ID: %v", workflowHistory.ID)
	_, createTaskHistoryErr := w.TaskService.CreateTaskHistory(tx, workflowHistory.ID.String(), tasks)

	if createTaskHistoryErr != nil {
		tx.Rollback()
		return createTaskHistoryErr
	}

	commitErr := tx.Commit()

	if commitErr != nil {
		logging.Sugar.Error(commitErr)
		tx.Rollback()
		return commitErr

	}

	mqErr := SendTaskMessage(TaskMessage{
		Graph:             graph,
		Tasks:             tasksMap,
		WorkflowHistoryId: workflowHistory.ID,
	})

	if mqErr != nil {
		logging.Sugar.Errorf("error when sending the message to queue", mqErr)
		return mqErr
	}
	return nil
}

// PrepareWorkflowMessage implements WorkflowTriggerService.
func (w *WorkflowTriggerServiceImpl) PrepareWorkflowMessage(tasks []models.Tasks, edges []repository.Edges) (map[string]models.Tasks, map[string][]string) {
	tasksMap := make(map[string]models.Tasks)
	graph := map[string][]string{}

	for _, task := range tasks {
		tasksMap[task.Name] = task
	}

	for _, edge := range edges {
		children, ok := graph[edge.SourceTaskName]
		if ok {
			graph[edge.SourceTaskName] = append(children, edge.DestinationTaskName)
		} else {
			graph[edge.SourceTaskName] = []string{edge.DestinationTaskName}
		}

		_, taskNameOk := graph[edge.DestinationTaskName]

		if !taskNameOk {
			graph[edge.DestinationTaskName] = []string{}
		}
	}

	return tasksMap, graph
}
