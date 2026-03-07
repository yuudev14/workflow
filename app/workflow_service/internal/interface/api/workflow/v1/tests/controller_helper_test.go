package workflow_http_v1_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	workflow_http_v1 "github.com/yuudev14-workflow/workflow-service/internal/interface/api/workflow/v1"
	mock_workflows "github.com/yuudev14-workflow/workflow-service/internal/workflows/mocks"
	"go.uber.org/mock/gomock"
)

func setupController(t *testing.T) (*workflow_http_v1.WorkflowController, *mock_workflows.MockWorkflowService, *gin.Context, *httptest.ResponseRecorder) {
	logging.Setup("Debug")
	ctrl := gomock.NewController(t)

	mockService := mock_workflows.NewMockWorkflowService(ctrl)

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	controller := &workflow_http_v1.WorkflowController{
		WorkflowService: mockService,
	}

	return controller, mockService, c, recorder
}
