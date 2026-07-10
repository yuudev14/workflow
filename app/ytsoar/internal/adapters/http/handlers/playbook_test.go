package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	mock_edges "github.com/yuudev14/ytsoar/internal/application/edges/mocks"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	mock_workflow_application "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	mock_workflows "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	mock_tasks "github.com/yuudev14/ytsoar/internal/application/tasks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
	"go.uber.org/mock/gomock"
)

type setupMockServices struct {
	PlaybookService            *mock_workflows.MockPlaybookService
	TaskService                *mock_tasks.MockTaskService
	EdgeService                *mock_edges.MockEdgeService
	PlaybookApplicationService *mock_workflow_application.MockPlaybookApplicationService
}

func setupController(t *testing.T) (
	*PlaybookHandler,
	*setupMockServices,
	*gin.Context,
	*httptest.ResponseRecorder,
) {

	ctrl := gomock.NewController(t)

	mockPlaybookService := mock_workflows.NewMockPlaybookService(ctrl)
	mockTaskService := mock_tasks.NewMockTaskService(ctrl)
	mockEdgeService := mock_edges.NewMockEdgeService(ctrl)
	mockPlaybookAppService := mock_workflow_application.NewMockPlaybookApplicationService(ctrl)

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	controller := &PlaybookHandler{
		logger:                     logger.NewNop(),
		PlaybookService:            mockPlaybookService,
		TaskService:                mockTaskService,
		EdgeService:                mockEdgeService,
		PlaybookApplicationService: mockPlaybookAppService,
	}

	return controller,
		&setupMockServices{
			PlaybookService:            mockPlaybookService,
			TaskService:                mockTaskService,
			EdgeService:                mockEdgeService,
			PlaybookApplicationService: mockPlaybookAppService,
		},
		c,
		recorder
}

func TestControllerGetPlaybooksSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/playbooks/v1?offset=0&limit=10",
		nil,
	)

	c.Request = req

	expected := types.Entries[domain.Playbooks]{
		Entries: []domain.Playbooks{
			{
				Name: "Playbook 1",
			},
			{
				Name: "Playbook 2",
			},
		},
		Total: 2,
	}

	mockServices.
		PlaybookService.
		EXPECT().
		GetPlaybooksData(gomock.Any(), 0, 10, playbooks.PlaybookFilter{}).
		Return(expected, nil)

	controller.GetPlaybooks(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetPlaybooksError(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/playbooks/v1?offset=0&limit=10",
		nil,
	)

	c.Request = req

	mockServices.PlaybookService.
		EXPECT().
		GetPlaybooksData(gomock.Any(), 0, 10, playbooks.PlaybookFilter{}).
		Return(types.Entries[domain.Playbooks]{}, fmt.Errorf("service error"))

	controller.GetPlaybooks(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestControllerGetPlaybooksInvalidQuery(t *testing.T) {

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "invalid offset",
			query:          "/playbooks/v1?offset=invalid&limit=10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid playbook_id",
			query:          "/playbooks/v1?offset=0&limit=10&playbook_id=invalid-uuid",
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

			controller.GetPlaybooks(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerGetPlaybookGraphByIdSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)
	uuid := uuid.New()

	req := httptest.NewRequest(
		http.MethodGet,
		"/playbooks/v1/"+uuid.String(),
		nil,
	)

	c.Request = req
	c.Params = []gin.Param{
		{
			Key:   "playbook_id",
			Value: uuid.String(),
		},
	}

	expected := &domain.PlaybookGraph{
		ID:   uuid,
		Name: "Playbook 1",
	}

	mockServices.PlaybookService.
		EXPECT().
		GetPlaybookGraphById(gomock.Any(), uuid.String()).
		Return(expected, nil)

	controller.GetPlaybookGraphById(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetPlaybookGraphByIdError(t *testing.T) {

	tests := []struct {
		error_         string
		expectedStatus int
		expectedReturn *domain.PlaybookGraph
	}{
		{
			error_:         "workflow is not found",
			expectedStatus: http.StatusNotFound,
			expectedReturn: nil,
		},
		{
			error_:         "internal server error",
			expectedStatus: http.StatusInternalServerError,
			expectedReturn: &domain.PlaybookGraph{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.error_, func(t *testing.T) {
			controller, mockServices, c, recorder := setupController(t)
			uuid := uuid.New()

			req := httptest.NewRequest(
				http.MethodGet,
				"/playbooks/v1/"+uuid.String(),
				nil,
			)

			c.Request = req
			c.Params = []gin.Param{
				{
					Key:   "playbook_id",
					Value: uuid.String(),
				},
			}

			mockServices.PlaybookService.
				EXPECT().
				GetPlaybookGraphById(gomock.Any(), uuid.String()).
				Return(tt.expectedReturn, fmt.Errorf("%s", tt.error_))

			controller.GetPlaybookGraphById(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}

}

func TestControllerGetPlaybookHistorySuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/playbooks/v1/history?offset=0&limit=10",
		nil,
	)
	c.Request = req

	expected := types.Entries[domain.PlaybookHistoryResponse]{
		Entries: []domain.PlaybookHistoryResponse{
			{
				ID: uuid.New(),
			},
			{
				ID: uuid.New(),
			},
		},
		Total: 2,
	}

	mockServices.PlaybookService.
		EXPECT().
		GetPlaybooksHistoryData(gomock.Any(), 0, 10, playbooks.PlaybookHistoryFilter{}).
		Return(expected, nil)

	controller.GetPlaybookHistory(c)

	assert.Equal(t, http.StatusOK, recorder.Code)

}

func TestControllerGetPlaybooksHistoryError(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/playbooks/v1/history?offset=0&limit=10",
		nil,
	)

	c.Request = req

	mockServices.PlaybookService.
		EXPECT().
		GetPlaybooksHistoryData(gomock.Any(), 0, 10, playbooks.PlaybookHistoryFilter{}).
		Return(types.Entries[domain.PlaybookHistoryResponse]{}, fmt.Errorf("service error"))

	controller.GetPlaybookHistory(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestControllerGetPlaybooksHistoryInvalidQuery(t *testing.T) {

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "invalid offset",
			query:          "/playbooks/v1/history?offset=invalid&limit=10",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid playbook_id",
			query:          "/playbooks/v1/history?offset=0&limit=10&playbook_id=invalid-uuid",
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

			controller.GetPlaybookHistory(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerGetTaskHistoryByPlaybookHistoryIdSuccess(t *testing.T) {
	controller, mockServices, c, recorder := setupController(t)

	historyID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/history/"+historyID.String(), nil)

	c.Request = req
	c.Params = []gin.Param{
		{Key: "playbook_history_id", Value: historyID.String()},
	}

	mockServices.PlaybookService.
		EXPECT().
		GetPlaybookHistoryById(gomock.Any(), historyID).
		Return(&domain.PlaybookHistoryResponse{}, nil)

	mockServices.TaskService.
		EXPECT().
		GetTaskHistoryByPlaybookHistoryId(gomock.Any(), historyID.String(), tasks.TaskHistoryFilter{}).
		Return([]domain.TaskHistory{}, nil)

	controller.GetTaskHistoryByPlaybookHistoryId(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerCreatePlaybookSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	body := `{"name":"test workflow"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/playbooks/v1",
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	mockService.
		PlaybookService.
		EXPECT().
		CreatePlaybook(gomock.Any(), gomock.Any()).
		Return(&domain.Playbooks{}, nil)

	controller.CreatePlaybook(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerUpdatePlaybookSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	id := uuid.New().String()

	body := `{"name":"updated workflow"}`

	req := httptest.NewRequest(
		http.MethodPut,
		"/playbooks/v1/"+id,
		strings.NewReader(body),
	)

	req.Header.Set("Content-Type", "application/json")

	c.Request = req
	c.Params = []gin.Param{
		{Key: "playbook_id", Value: id},
	}

	mockService.
		PlaybookService.
		EXPECT().
		UpdatePlaybook(gomock.Any(), id, gomock.Any()).
		Return(&domain.Playbooks{}, nil)

	controller.UpdatePlaybook(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetTasksByPlaybookIdSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	id := uuid.New().String()

	req := httptest.NewRequest(http.MethodGet, "/playbooks/v1/"+id+"/tasks", nil)

	c.Request = req
	c.Params = []gin.Param{
		{Key: "playbook_id", Value: id},
	}

	mockService.
		PlaybookService.
		EXPECT().
		GetPlaybookById(gomock.Any(), id).
		Return(&domain.Playbooks{}, nil)

	mockService.
		TaskService.
		EXPECT().
		GetTasksByPlaybookId(gomock.Any(), id).
		Return([]domain.Tasks{}, nil)

	controller.GetTasksByPlaybookId(c)

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
				{Key: "playbook_history_id", Value: uuid.New().String()},
				{Key: "task_id", Value: uuid.New().String()},
			}

			if tt.expectedStatus == http.StatusAccepted {
				mockService.
					TaskService.
					EXPECT().
					UpdateTaskStatus(gomock.Any(), gomock.Any(), gomock.Any(), tt.status).
					Return(&domain.TaskHistory{}, nil)
			}

			controller.UpdateTaskStatus(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestControllerUpdatePlaybookStatus(t *testing.T) {

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
				{Key: "playbook_history_id", Value: uuid.New().String()},
			}

			if tt.expectedStatus == http.StatusAccepted {
				mockService.
					PlaybookService.
					EXPECT().
					UpdatePlaybookHistoryStatus(gomock.Any(), gomock.Any(), tt.status).
					Return(&domain.PlaybookHistory{}, nil)
			}

			controller.UpdatePlaybookStatus(c)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}

}
