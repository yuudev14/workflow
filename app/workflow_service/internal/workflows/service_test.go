package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
	"github.com/yuudev14-workflow/workflow-service/models"
	"github.com/yuudev14-workflow/workflow-service/service"
)

type MockWorkflowRepository struct {
	mock.Mock
}

func (m *MockWorkflowRepository) GetWorkflows(offset, limit int, filter dto.WorkflowFilter) ([]models.Workflows, error) {
	args := m.Called(offset, limit, filter)
	return args.Get(0).([]models.Workflows), args.Error(1)
}

func TestWorkflowServiceImpl_GetWorkflows_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockWorkflowRepository)
	svc := &workflows.WorkflowServiceImpl{WorkflowRepository: mockRepo}

	expectedWorkflows := []models.Workflows{
		{ID: 1, Name: "test-workflow"},
		{ID: 2, Name: "another-workflow"},
	}

	offset, limit := 0, 10
	filter := dto.WorkflowFilter{Status: "active"}

	// Setup mock expectation
	mockRepo.On("GetWorkflows", offset, limit, filter).Return(expectedWorkflows, nil).Once()

	// Act
	result, err := svc.GetWorkflows(offset, limit, filter)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedWorkflows, result)
	assert.Len(t, result, 2)

	// Verify mock interactions
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNumberOfCalls(t, "GetWorkflows", 1)
}

func TestWorkflowServiceImpl_GetWorkflows_Error(t *testing.T) {
	// Arrange
	mockRepo := new(MockWorkflowRepository)
	svc := &workflows.WorkflowServiceImpl{WorkflowRepository: mockRepo}

	expectedErr := assert.AnError // or any error type
	offset, limit := 0, 10
	filter := dto.WorkflowFilter{}

	mockRepo.On("GetWorkflows", offset, limit, filter).Return([]models.Workflows{}, expectedErr).Once()

	// Act
	result, err := svc.GetWorkflows(offset, limit, filter)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Empty(t, result)
	mockRepo.AssertExpectations(t)
}

func TestWorkflowServiceImpl_GetWorkflows_EmptyResult(t *testing.T) {
	// Arrange
	mockRepo := new(MockWorkflowRepository)
	svc := &service.WorkflowServiceImpl{WorkflowRepository: mockRepo}

	offset, limit := 0, 10
	filter := dto.WorkflowFilter{}

	mockRepo.On("GetWorkflows", offset, limit, filter).Return([]models.Workflows{}, nil).Once()

	// Act
	result, err := svc.GetWorkflows(offset, limit, filter)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result)
	mockRepo.AssertExpectations(t)
}
