package workflow_http_v1_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock_edges "github.com/yuudev14-workflow/workflow-service/internal/edges/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	workflow_http_v1 "github.com/yuudev14-workflow/workflow-service/internal/interface/api/workflow/v1"
	mock_workflow_application "github.com/yuudev14-workflow/workflow-service/internal/orchestrator/workflows/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	mock_tasks "github.com/yuudev14-workflow/workflow-service/internal/tasks/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
	mock_workflows "github.com/yuudev14-workflow/workflow-service/internal/workflows/mocks"
	"go.uber.org/mock/gomock"
)

type setupMockServices struct {
	WorkflowService            *mock_workflows.MockWorkflowService
	TaskService                *mock_tasks.MockTaskService
	EdgeService                *mock_edges.MockEdgeService
	WorkflowApplicationService *mock_workflow_application.MockWorkflowApplicationService
}

func setupController(t *testing.T) (
	*workflow_http_v1.WorkflowController,
	*setupMockServices,
	*gin.Context,
	*httptest.ResponseRecorder,
) {

	logging.Setup("Debug")

	ctrl := gomock.NewController(t)

	mockWorkflowService := mock_workflows.NewMockWorkflowService(ctrl)
	mockTaskService := mock_tasks.NewMockTaskService(ctrl)
	mockEdgeService := mock_edges.NewMockEdgeService(ctrl)
	mockWorkflowAppService := mock_workflow_application.NewMockWorkflowApplicationService(ctrl)

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	controller := &workflow_http_v1.WorkflowController{
		WorkflowService:            mockWorkflowService,
		TaskService:                mockTaskService,
		EdgeService:                mockEdgeService,
		WorkflowApplicationService: mockWorkflowAppService,
	}

	return controller,
		&setupMockServices{
			WorkflowService:            mockWorkflowService,
			TaskService:                mockTaskService,
			EdgeService:                mockEdgeService,
			WorkflowApplicationService: mockWorkflowAppService,
		},
		c,
		recorder
}

func TestControllerGetWorkflowsSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows/v1?offset=0&limit=10",
		nil,
	)

	c.Request = req

	expected := types.Entries[workflows.Workflows]{
		Entries: []workflows.Workflows{
			{
				Name: "Workflow 1",
			},
			{
				Name: "Workflow 2",
			},
		},
		Total: 2,
	}

	mockServices.
		WorkflowService.
		EXPECT().
		GetWorkflowsData(0, 10, workflows.WorkflowFilter{}).
		Return(expected, nil)

	controller.GetWorkflows(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetWorkflowsError(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows/v1?offset=0&limit=10",
		nil,
	)

	c.Request = req

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowsData(0, 10, workflows.WorkflowFilter{}).
		Return(types.Entries[workflows.Workflows]{}, fmt.Errorf("service error"))

	controller.GetWorkflows(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestControllerGetWorkflowsInvalidQuery(t *testing.T) {

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "invalid offset",
			query:          "/workflows/v1?offset=invalid&limit=10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid workflow_id",
			query:          "/workflows/v1?offset=0&limit=10&workflow_id=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, c, recorder := setupController(t)

			req := httptest.NewRequest(
				http.MethodGet,
				tt.query,
				nil,
			)

			c.Request = req

			controller.GetWorkflows(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerGetWorkflowGraphByIdSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)
	uuid := uuid.New()

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows/v1/"+uuid.String(),
		nil,
	)

	c.Request = req
	c.Params = []gin.Param{
		{
			Key:   "workflow_id",
			Value: uuid.String(),
		},
	}

	expected := &workflows.WorkflowsGraph{
		ID:   uuid,
		Name: "Workflow 1",
	}

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowGraphById(uuid.String()).
		Return(expected, nil)

	controller.GetWorkflowGraphById(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetWorkflowGraphByIdError(t *testing.T) {

	tests := []struct {
		error_         string
		expectedStatus int
		expectedReturn *workflows.WorkflowsGraph
	}{
		{
			error_:         "workflow is not found",
			expectedStatus: http.StatusNotFound,
			expectedReturn: nil,
		},
		{
			error_:         "internal server error",
			expectedStatus: http.StatusInternalServerError,
			expectedReturn: &workflows.WorkflowsGraph{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.error_, func(t *testing.T) {
			controller, mockServices, c, recorder := setupController(t)
			uuid := uuid.New()

			req := httptest.NewRequest(
				http.MethodGet,
				"/workflows/v1/"+uuid.String(),
				nil,
			)

			c.Request = req
			c.Params = []gin.Param{
				{
					Key:   "workflow_id",
					Value: uuid.String(),
				},
			}

			mockServices.WorkflowService.
				EXPECT().
				GetWorkflowGraphById(uuid.String()).
				Return(tt.expectedReturn, fmt.Errorf("%s", tt.error_))

			controller.GetWorkflowGraphById(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}

}

func TestControllerGetWorkflowHistorySuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows/v1/history?offset=0&limit=10",
		nil,
	)
	c.Request = req

	expected := types.Entries[workflows.WorkflowHistoryResponse]{
		Entries: []workflows.WorkflowHistoryResponse{
			{
				ID: uuid.New(),
			},
			{
				ID: uuid.New(),
			},
		},
		Total: 2,
	}

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowsHistoryData(0, 10, workflows.WorkflowHistoryFilter{}).
		Return(expected, nil)

	controller.GetWorkflowHistory(c)

	assert.Equal(t, http.StatusOK, recorder.Code)

}

func TestControllerGetWorkflowsHistoryError(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows/v1/history?offset=0&limit=10",
		nil,
	)

	c.Request = req

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowsHistoryData(0, 10, workflows.WorkflowHistoryFilter{}).
		Return(types.Entries[workflows.WorkflowHistoryResponse]{}, fmt.Errorf("service error"))

	controller.GetWorkflowHistory(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestControllerGetWorkflowsHistoryInvalidQuery(t *testing.T) {

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "invalid offset",
			query:          "/workflows/v1/history?offset=invalid&limit=10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid workflow_id",
			query:          "/workflows/v1/history?offset=0&limit=10&workflow_id=invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, _, c, recorder := setupController(t)

			req := httptest.NewRequest(
				http.MethodGet,
				tt.query,
				nil,
			)

			c.Request = req

			controller.GetWorkflowHistory(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerGetWorkflowTriggerTypesSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(http.MethodGet, "/workflows/v1/triggers", nil)
	c.Request = req

	expected := []workflows.WorkflowTriggers{}

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowTriggers().
		Return(expected, nil)

	controller.GetWorkflowTriggerTypes(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetWorkflowTriggerTypesError(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(http.MethodGet, "/workflows/v1/triggers", nil)
	c.Request = req

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowTriggers().
		Return(nil, fmt.Errorf("error"))

	controller.GetWorkflowTriggerTypes(c)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestControllerGetTaskHistoryByWorkflowHistoryIdSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	historyID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/history/"+historyID.String(), nil)

	c.Request = req
	c.Params = []gin.Param{
		{Key: "workflow_history_id", Value: historyID.String()},
	}

	mockServices.WorkflowService.
		EXPECT().
		GetWorkflowHistoryById(historyID).
		Return(&workflows.WorkflowHistoryResponse{}, nil)

	mockServices.TaskService.
		EXPECT().
		GetTaskHistoryByWorkflowHistoryId(historyID.String(), tasks.TaskHistoryFilter{}).
		Return([]tasks.TaskHistory{}, nil)

	controller.GetTaskHistoryByWorkflowHistoryId(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerCreateWorkflowSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	body := `{"name":"test workflow"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/workflows/v1",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	mockService.
		WorkflowService.
		EXPECT().
		CreateWorkflow(gomock.Any()).
		Return(&workflows.Workflows{}, nil)

	controller.CreateWorkflow(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerUpdateWorkflowSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	id := uuid.New().String()

	body := `{"name":"updated workflow"}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/workflows/v1/"+id,
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	c.Request = req
	c.Params = []gin.Param{
		{Key: "workflow_id", Value: id},
	}

	mockService.
		WorkflowService.
		EXPECT().
		UpdateWorkflow(id, gomock.Any()).
		Return(&workflows.Workflows{}, nil)

	controller.UpdateWorkflow(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetTasksByWorkflowIdSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	id := uuid.New().String()

	req := httptest.NewRequest(http.MethodGet, "/workflows/v1/"+id+"/tasks", nil)

	c.Request = req
	c.Params = []gin.Param{
		{Key: "workflow_id", Value: id},
	}

	mockService.
		WorkflowService.
		EXPECT().
		GetWorkflowById(id).
		Return(&workflows.Workflows{}, nil)

	mockService.
		TaskService.
		EXPECT().
		GetTasksByWorkflowId(id).
		Return([]tasks.Tasks{}, nil)

	controller.GetTasksByWorkflowId(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerUpdateTaskStatus(t *testing.T) {
	tests := []struct {
		name           string
		status         string
		expectedStatus int
	}{
		{
			name:           "update task status to success",
			status:         "success",
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "update task status to failed",
			status:         "failed",
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "update task status to invalid status",
			status:         "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService, c, recorder := setupController(t)
			body := `{"status":"` + tt.status + `"}`

			req := httptest.NewRequest(http.MethodPatch, "/status", strings.NewReader(body))

			c.Request = req
			c.Params = []gin.Param{
				{Key: "workflow_history_id", Value: uuid.New().String()},
				{Key: "task_id", Value: uuid.New().String()},
			}

			if tt.expectedStatus == http.StatusAccepted {
				mockService.
					TaskService.
					EXPECT().
					UpdateTaskStatus(gomock.Any(), gomock.Any(), tt.status).
					Return(&tasks.TaskHistory{}, nil)
			}

			controller.UpdateTaskStatus(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerUpdateWorkflowStatus(t *testing.T) {

	tests := []struct {
		name           string
		status         string
		expectedStatus int
	}{
		{
			name:           "update task status to success",
			status:         "success",
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "update task status to failed",
			status:         "failed",
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "update task status to invalid status",
			status:         "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller, mockService, c, recorder := setupController(t)
			body := `{"status":"` + tt.status + `"}`

			req := httptest.NewRequest(http.MethodPatch, "/workflow/status", strings.NewReader(body))

			c.Request = req
			c.Params = []gin.Param{
				{Key: "workflow_history_id", Value: uuid.New().String()},
			}

			if tt.expectedStatus == http.StatusAccepted {
				mockService.
					WorkflowService.
					EXPECT().
					UpdateWorkflowHistoryStatus(gomock.Any(), tt.status).
					Return(&workflows.WorkflowHistory{}, nil)
			}

			controller.UpdateWorkflowStatus(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}

}
