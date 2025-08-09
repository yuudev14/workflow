package workflow_api

import (
	"github.com/gin-gonic/gin"
	workflow_controller_v1 "github.com/yuudev14-workflow/workflow-service/api/workflow/v1"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/pkg/repository"
	"github.com/yuudev14-workflow/workflow-service/service"
)

func SetupWorkflowController(route *gin.RouterGroup) {
	workflowRepository := repository.NewWorkflowRepository(db.DB)
	edgeRepository := repository.NewEdgeRepositoryImpl(db.DB)
	taskRepository := repository.NewTaskRepositoryImpl(db.DB)
	workflowService := service.NewWorkflowService(workflowRepository)
	edgeService := service.NewEdgeServiceImpl(edgeRepository, workflowService)
	taskService := service.NewTaskServiceImpl(taskRepository, workflowService)
	workflowTriggerService := service.NewWorflowTriggerService(workflowService, taskService, edgeService)
	workflowController := workflow_controller_v1.NewWorkflowController(workflowService, taskService, edgeService, workflowTriggerService)

	r := route.Group("workflows/v1")
	{
		r.GET("", workflowController.GetWorkflows)
		r.GET("/history", workflowController.GetWorkflowHistory)
		r.GET("/history/:workflow_history_id/tasks", workflowController.GetTaskHistoryByWorkflowId)
		r.GET("/:workflow_id", workflowController.GetWorkflowGraphById)
		r.GET("/triggers", workflowController.GetWorkflowTriggerTypes)
		r.POST("/trigger/:workflow_id", workflowController.Trigger)
		r.POST("", workflowController.CreateWorkflow)
		r.GET("/:workflow_id/tasks", workflowController.GetTasksByWorkflowId)
		r.PUT("/:workflow_id", workflowController.UpdateWorkflow)
		r.PUT("/tasks/:workflow_id", workflowController.UpdateWorkflowTasks)
		r.PUT("/trigger/status/:workflow_history_id", workflowController.UpdateWorkflowStatus)
		r.PUT("/trigger/status/:workflow_history_id/tasks/:task_id", workflowController.UpdateTaskStatus)
	}
}
