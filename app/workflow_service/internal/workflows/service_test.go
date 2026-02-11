package workflows_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
	mock_workflows "github.com/yuudev14-workflow/workflow-service/internal/workflows/mocks"
	"go.uber.org/mock/gomock"
)

func TestService_GetWorkflowName_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_workflows.NewMockWorkflowService(ctrl)

	service := workflows.NewWorkflowService(mockRepo)

	uuidString := uuid.New()
	returnedWorkflow := &workflows.Workflows{
		ID:   uuidString,
		Name: "My Workflow",
	}
	// Expectation
	mockRepo.
		EXPECT().
		GetWorkflowById(uuidString.String()).
		Return(returnedWorkflow, nil)

	workflow, err := service.GetWorkflowById(uuidString.String())
	assert.NoError(t, err)
	assert.Equal(t, "My Workflow", workflow.Name)
}
