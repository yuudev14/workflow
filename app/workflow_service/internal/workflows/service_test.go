package workflows_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/internal/utils"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
	mock_workflows "github.com/yuudev14-workflow/workflow-service/internal/workflows/mocks"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logging.Setup("DEBUG")
	os.Exit(m.Run())
}

func setupService(t *testing.T) (workflows.WorkflowService, *mock_workflows.MockWorkflowRepository) {
	ctrl := gomock.NewController(t)

	mockRepo := mock_workflows.NewMockWorkflowRepository(ctrl)
	service := workflows.NewWorkflowService(mockRepo)

	t.Cleanup(ctrl.Finish)

	return service, mockRepo
}
func TestServiceGetWorkflowsDataSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedWorkflows := []workflows.Workflows{
		{
			ID:   uuid.New(),
			Name: "Workflow 1",
		},
		{
			ID:   uuid.New(),
			Name: "Workflow 2",
		},
	}

	mockRepo.
		EXPECT().
		GetWorkflows(0, 10, workflows.WorkflowFilter{}).
		Return(returnedWorkflows, nil)

	mockRepo.
		EXPECT().
		GetWorkflowsCount(workflows.WorkflowFilter{}).
		Return(2, nil)

	workflowsData, err := service.GetWorkflowsData(0, 10, workflows.WorkflowFilter{})

	assert.NoError(t, err)
	assert.Len(t, workflowsData.Entries, 2)
	assert.Equal(t, "Workflow 1", workflowsData.Entries[0].Name)
	assert.Equal(t, "Workflow 2", workflowsData.Entries[1].Name)
	assert.Equal(t, 2, workflowsData.Total)
}

func TestServiceGetWorkflowsDataFail(t *testing.T) {
	service, mockRepo := setupService(t)

	tests := []struct {
		name                   string
		getWorkflowsError      error
		getWorkflowsCountError error
	}{
		{name: "get workflows data error", getWorkflowsError: fmt.Errorf("error occurred"), getWorkflowsCountError: nil},
		{name: "get workflows count error", getWorkflowsError: nil, getWorkflowsCountError: fmt.Errorf("error occurred")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.
				EXPECT().
				GetWorkflows(0, 10, workflows.WorkflowFilter{}).
				Return([]workflows.Workflows{}, tt.getWorkflowsError)

			if tt.getWorkflowsError == nil {
				mockRepo.
					EXPECT().
					GetWorkflowsCount(workflows.WorkflowFilter{}).
					Return(0, tt.getWorkflowsCountError)
			}

			workflowsData, err := service.GetWorkflowsData(0, 10, workflows.WorkflowFilter{})
			assert.Error(t, err)
			assert.Empty(t, workflowsData.Entries)
			assert.Equal(t, 0, workflowsData.Total)
		})
	}
}

func TestServiceGetWorkflowByIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

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

func TestServiceGetWorkflowByIdFail(t *testing.T) {
	service, mockRepo := setupService(t)
	uuidString := uuid.New()

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Workflow not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				GetWorkflowById(uuidString.String()).
				Return(nil, tt.error_)

			workflow, err := service.GetWorkflowById(uuidString.String())
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceGetWorkflowsSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedWorkflows := []workflows.Workflows{
		{
			ID:   uuid.New(),
			Name: "Workflow 1",
		},
		{
			ID:   uuid.New(),
			Name: "Workflow 2",
		},
	}

	mockRepo.
		EXPECT().
		GetWorkflows(0, 10, workflows.WorkflowFilter{}).
		Return(returnedWorkflows, nil)

	workflowsList, err := service.GetWorkflows(0, 10, workflows.WorkflowFilter{})
	assert.NoError(t, err)
	assert.Len(t, workflowsList, 2)
	assert.Equal(t, "Workflow 1", workflowsList[0].Name)
	assert.Equal(t, "Workflow 2", workflowsList[1].Name)

}

func TestServiceGetWorkflowHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedHistory := []workflows.WorkflowHistoryResponse{
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     "completed",
		},
		{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     "failed",
		},
	}

	mockRepo.
		EXPECT().
		GetWorkflowHistory(0, 10, workflows.WorkflowHistoryFilter{}).
		Return(returnedHistory, nil)

	historyList, err := service.GetWorkflowHistory(0, 10, workflows.WorkflowHistoryFilter{})
	assert.NoError(t, err)
	assert.Len(t, historyList, 2)
	assert.Equal(t, "completed", historyList[0].Status)
	assert.Equal(t, "failed", historyList[1].Status)

}

func TestServiceGetWorkflowHistoryByIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	uuidString := uuid.New()
	returnedWorkflow := &workflows.WorkflowHistoryResponse{
		ID:     uuidString,
		Status: "complete",
	}
	// Expectation
	mockRepo.
		EXPECT().
		GetWorkflowHistoryById(uuidString).
		Return(returnedWorkflow, nil)

	workflow, err := service.GetWorkflowHistoryById(uuidString)
	assert.NoError(t, err)
	assert.Equal(t, "complete", workflow.Status)
}

func TestServiceGetWorkflowHistoryCountSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	mockRepo.
		EXPECT().
		GetWorkflowHistoryCount(workflows.WorkflowHistoryFilter{}).
		Return(5, nil)

	count, err := service.GetWorkflowHistoryCount(workflows.WorkflowHistoryFilter{})
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestServiceGetWorkflowTriggersSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedTriggers := []workflows.WorkflowTriggers{
		{
			ID:   uuid.New(),
			Name: "Trigger 1",
		},
		{
			ID:   uuid.New(),
			Name: "Trigger 2",
		},
	}

	mockRepo.
		EXPECT().
		GetWorkflowTriggers().
		Return(returnedTriggers, nil)

	triggers, err := service.GetWorkflowTriggers()
	assert.NoError(t, err)
	assert.Len(t, triggers, 2)
	assert.Equal(t, "Trigger 1", triggers[0].Name)
	assert.Equal(t, "Trigger 2", triggers[1].Name)

}

func TestServiceGetWorkflowsCountSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	mockRepo.
		EXPECT().
		GetWorkflowsCount(workflows.WorkflowFilter{}).
		Return(10, nil)

	count, err := service.GetWorkflowsCount(workflows.WorkflowFilter{})
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestServiceCreateWorkflowHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()
	returnedHistory := &workflows.WorkflowHistory{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     "created",
	}

	mockRepo.
		EXPECT().
		CreateWorkflowHistory(nil, workflowID.String(), []edges.ResponseEdges{}).
		Return(returnedHistory, nil)

	history, err := service.CreateWorkflowHistory(nil, workflowID.String(), []edges.ResponseEdges{})
	assert.NoError(t, err)
	assert.Equal(t, "created", history.Status)
}

func TestServiceGetWorkflowGraphByIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	uuidString := uuid.New()
	returnedWorkflow := &workflows.WorkflowsGraph{
		ID:   uuidString,
		Name: "My Workflow",
	}
	// Expectation
	mockRepo.
		EXPECT().
		GetWorkflowGraphById(uuidString.String()).
		Return(returnedWorkflow, nil)

	workflow, err := service.GetWorkflowGraphById(uuidString.String())
	assert.NoError(t, err)
	assert.Equal(t, "My Workflow", workflow.Name)
}

func TestServiceGetWorkflowGraphByIdFail(t *testing.T) {
	service, mockRepo := setupService(t)
	uuidString := uuid.New()

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Workflow not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				GetWorkflowGraphById(uuidString.String()).
				Return(nil, tt.error_)

			workflow, err := service.GetWorkflowGraphById(uuidString.String())
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceCreateWorkflowSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	newWorkflow := workflows.WorkflowPayload{
		Name: "New Workflow",
	}

	returnedWorkflow := &workflows.Workflows{
		ID:   uuid.New(),
		Name: "New Workflow",
	}

	mockRepo.
		EXPECT().
		CreateWorkflow(newWorkflow).
		Return(returnedWorkflow, nil)

	createdWorkflow, err := service.CreateWorkflow(newWorkflow)
	assert.NoError(t, err)
	assert.Equal(t, "New Workflow", createdWorkflow.Name)
}

func TestServiceUpdateWorkflowSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()
	updateData := workflows.UpdateWorkflowData{
		Name: types.Nullable[string]{Set: true, Value: utils.StrPtr("Updated Workflow")},
	}

	returnedWorkflow := &workflows.Workflows{
		ID:   workflowID,
		Name: "Updated Workflow",
	}

	mockRepo.
		EXPECT().
		UpdateWorkflow(workflowID.String(), updateData).
		Return(returnedWorkflow, nil)

	updatedWorkflow, err := service.UpdateWorkflow(workflowID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Workflow", updatedWorkflow.Name)
}

func TestServiceUpdateWorkflowTxSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()
	updateData := workflows.UpdateWorkflowData{
		Name: types.Nullable[string]{Set: true, Value: utils.StrPtr("Updated Workflow")},
	}

	returnedWorkflow := &workflows.Workflows{
		ID:   workflowID,
		Name: "Updated Workflow",
	}

	mockRepo.
		EXPECT().
		UpdateWorkflowTx(nil, workflowID.String(), updateData).
		Return(returnedWorkflow, nil)

	updatedWorkflow, err := service.UpdateWorkflowTx(nil, workflowID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Workflow", updatedWorkflow.Name)
}

func TestServiceUpdateWorkflowHistoryStatusSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New()
	newStatus := "completed"

	returnedHistory := &workflows.WorkflowHistory{
		ID:     workflowHistoryID,
		Status: newStatus,
	}

	mockRepo.
		EXPECT().
		UpdateWorkflowHistoryStatus(workflowHistoryID.String(), newStatus).
		Return(returnedHistory, nil)

	updatedHistory, err := service.UpdateWorkflowHistoryStatus(workflowHistoryID.String(), newStatus)
	assert.NoError(t, err)
	assert.Equal(t, newStatus, updatedHistory.Status)
}

func TestServiceUpdateWorkflowHistoryStatusFail(t *testing.T) {
	service, mockRepo := setupService(t)
	workflowHistoryID := uuid.New()
	newStatus := "completed"

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Workflow not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				UpdateWorkflowHistoryStatus(workflowHistoryID.String(), newStatus).
				Return(nil, tt.error_)

			workflow, err := service.UpdateWorkflowHistoryStatus(workflowHistoryID.String(), newStatus)
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceUpdateWorkflowHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New()
	updateData := workflows.UpdateWorkflowHistoryData{
		Status: types.Nullable[string]{Set: true, Value: utils.StrPtr("completed")},
	}

	returnedHistory := &workflows.WorkflowHistory{
		ID:     workflowHistoryID,
		Status: "completed",
	}

	mockRepo.
		EXPECT().
		UpdateWorkflowHistory(workflowHistoryID.String(), updateData).
		Return(returnedHistory, nil)

	updatedHistory, err := service.UpdateWorkflowHistory(workflowHistoryID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "completed", updatedHistory.Status)
}

func TestServiceUpdateWorkflowHistoryFail(t *testing.T) {
	service, mockRepo := setupService(t)
	workflowHistoryID := uuid.New()
	updateData := workflows.UpdateWorkflowHistoryData{
		Status: types.Nullable[string]{Set: true, Value: utils.StrPtr("completed")},
	}

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Workflow not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				UpdateWorkflowHistory(workflowHistoryID.String(), updateData).
				Return(nil, tt.error_)

			workflow, err := service.UpdateWorkflowHistory(workflowHistoryID.String(), updateData)
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}
