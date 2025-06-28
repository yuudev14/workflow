package consumer

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/mq"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
	"github.com/yuudev14-workflow/workflow-service/pkg/types"
	"github.com/yuudev14-workflow/workflow-service/service"
)

type MessageBody struct {
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params"`
}

type TaskStatusPayload struct {
	WorkflowHistoryId string                 `json:"workflow_history_id"`
	TaskId            string                 `json:"task_id"`
	Status            types.Nullable[string] `json:"status,omitempty"`
	Error             types.Nullable[string] `json:"error,omitempty"`
	Result            interface{}            `json:"result,omitempty"`
}

type WorkflowStatusPayload struct {
	WorkflowHistoryId string                 `json:"workflow_history_id"`
	Status            types.Nullable[string] `json:"status,omitempty"`
	Error             types.Nullable[string] `json:"error,omitempty"`
	Result            interface{}            `json:"result,omitempty"`
}

type ConsumeMessage struct {
	WorkflowService service.WorkflowService
	TaskService     service.TaskService
}

func NewConsumeMessage(
	WorkflowService service.WorkflowService,
	TaskService service.TaskService,
) *ConsumeMessage {
	return &ConsumeMessage{
		WorkflowService: WorkflowService,
		TaskService:     TaskService,
	}
}

// Example handler functions for different message types
func (c *ConsumeMessage) handleTask(params []byte) {
	var taskParams TaskStatusPayload
	if err := json.Unmarshal(params, &taskParams); err != nil {
		logging.Sugar.Error("Error unmarshalling task params:", err)
		return
	}
	c.TaskService.UpdateTaskHistory(taskParams.WorkflowHistoryId, taskParams.TaskId, dto.UpdateTaskHistoryData{
		Status: taskParams.Status,
		Error:  taskParams.Error,
		Result: taskParams.Result,
	})
}

func (c *ConsumeMessage) handleWorkflow(params []byte) {
	var workflowParams WorkflowStatusPayload
	if err := json.Unmarshal(params, &workflowParams); err != nil {
		logging.Sugar.Error("Error unmarshalling workflow params:", err)
		return
	}
	c.WorkflowService.UpdateWorkflowHistory(workflowParams.WorkflowHistoryId, dto.UpdateWorkflowHistoryData{
		Status: workflowParams.Status,
		Error:  workflowParams.Error,
		Result: workflowParams.Result,
	})

}

func (c *ConsumeMessage) PrepareMessage(data MessageBody) {

	jsonData, err := json.Marshal(data.Params)
	if err != nil {
		fmt.Println("Error marshalling map to JSON:", err)
		return
	}

	switch data.Action {
	case "workflow_status":
		c.handleWorkflow(jsonData)
	case "task_status":
		c.handleTask(jsonData)

	}

}

func Listen() {
	msgs, err := mq.MQChannel.Consume(
		mq.ReceiverQueue.Name, // queue
		"",                    // consumer
		false,                 // auto-acknowledge (changed to false for manual ack)
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // arguments
	)

	if err != nil {
		panic("error in consuming a queue")
	}

	// Number of worker goroutines
	workerCount := 10

	// Use a WaitGroup to manage goroutines
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			workflowRepository := repository.NewWorkflowRepository(db.DB)
			taskRepository := repository.NewTaskRepositoryImpl(db.DB)
			workflowService := service.NewWorkflowService(workflowRepository)
			taskService := service.NewTaskServiceImpl(taskRepository, workflowService)
			consumeMessageService := NewConsumeMessage(workflowService, taskService)

			for d := range msgs {
				var message MessageBody

				err := json.Unmarshal(d.Body, &message)
				if err != nil {
					logging.Sugar.Warnf("Error decoding JSON: %v", err)
					d.Nack(false, true) // Negative acknowledgement, requeue the message
					continue
				}
				logging.Sugar.Infof("Received a message: %s", d.Body)
				consumeMessageService.PrepareMessage(message)
				d.Ack(false) // Acknowledge the message after processing
			}
		}()
	}

	logging.Sugar.Info("Listening to message queue with ", workerCount, "workers")
	wg.Wait() // Wait for all goroutines to finish
}
