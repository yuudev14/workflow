package workflow_http_v1

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/mq"
	workflow_application "github.com/yuudev14-workflow/workflow-service/internal/orchestrator/workflows"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
)

func SetupWorkflowController(db *sqlx.DB, mqInstance mq.MQStruct, route *gin.RouterGroup) {
	workflowRepository := workflows.NewWorkflowRepository(db)
	edgeRepository := edges.NewEdgeRepositoryImpl(db)
	taskRepository := tasks.NewTaskRepositoryImpl(db)
	workflowService := workflows.NewWorkflowService(workflowRepository)
	edgeService := edges.NewEdgeServiceImpl(edgeRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)
	workflowApplicationService := workflow_application.NewWorkflowApplicationService(workflowService, taskService, edgeService, db, mqInstance)
	workflowController := NewWorkflowController(workflowService, taskService, edgeService, workflowApplicationService, db)

	r := route.Group("workflows/v1")
	{
		r.GET("/health", workflowController.HealthCheck)
		r.GET("", workflowController.GetWorkflows)
		r.GET("/history", workflowController.GetWorkflowHistory)
		r.GET("/history/:workflow_history_id/tasks", workflowController.GetTaskHistoryByWorkflowHistoryId)
		r.GET("/:workflow_id", workflowController.GetWorkflowGraphById)
		r.GET("/triggers", workflowController.GetWorkflowTriggerTypes)
		r.POST("/trigger/:workflow_id", workflowController.Trigger)
		r.POST("", workflowController.CreateWorkflow)
		r.GET("/:workflow_id/tasks", workflowController.GetTasksByWorkflowId)
		r.PUT("/:workflow_id", workflowController.UpdateWorkflow)
		r.PUT("/tasks/:workflow_id", workflowController.UpdateWorkflowTasks)
		// r.PUT("/trigger/status/:workflow_history_id", workflowController.UpdateWorkflowStatus)
		// r.PUT("/trigger/status/:workflow_history_id/tasks/:task_id", workflowController.UpdateTaskStatus)
	}
}
