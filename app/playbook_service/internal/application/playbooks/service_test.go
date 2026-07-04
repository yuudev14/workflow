package playbooks_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	mock_workflows "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
	"github.com/yuudev14/ytsoar/internal/types"
	"github.com/yuudev14/ytsoar/internal/utils"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logging.Setup("DEBUG")
	os.Exit(m.Run())
}

func setupService(t *testing.T) (playbooks.PlaybookService, *mock_workflows.MockPlaybookRepository) {
	ctrl := gomock.NewController(t)

	mockRepo := mock_workflows.NewMockPlaybookRepository(ctrl)
	service := playbooks.NewPlaybookService(mockRepo)

	t.Cleanup(ctrl.Finish)

	return service, mockRepo
}
func TestServiceGetPlaybooksDataSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedPlaybooks := []domain.Playbooks{
		{
			ID:   uuid.New(),
			Name: "Playbook 1",
		},
		{
			ID:   uuid.New(),
			Name: "Playbook 2",
		},
	}

	mockRepo.
		EXPECT().
		GetPlaybooks(gomock.Any(), 0, 10, playbooks.PlaybookFilter{}).
		Return(returnedPlaybooks, nil)

	mockRepo.
		EXPECT().
		GetPlaybooksCount(gomock.Any(), playbooks.PlaybookFilter{}).
		Return(2, nil)

	workflowsData, err := service.GetPlaybooksData(context.Background(), 0, 10, playbooks.PlaybookFilter{})

	assert.NoError(t, err)
	assert.Len(t, workflowsData.Entries, 2)
	assert.Equal(t, "Playbook 1", workflowsData.Entries[0].Name)
	assert.Equal(t, "Playbook 2", workflowsData.Entries[1].Name)
	assert.Equal(t, 2, workflowsData.Total)
}

func TestServiceGetPlaybooksDataFail(t *testing.T) {
	service, mockRepo := setupService(t)

	tests := []struct {
		name                   string
		getPlaybooksError      error
		getPlaybooksCountError error
	}{
		{name: "get workflows data error", getPlaybooksError: fmt.Errorf("error occurred"), getPlaybooksCountError: nil},
		{name: "get workflows count error", getPlaybooksError: nil, getPlaybooksCountError: fmt.Errorf("error occurred")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.
				EXPECT().
				GetPlaybooks(gomock.Any(), 0, 10, playbooks.PlaybookFilter{}).
				Return([]domain.Playbooks{}, tt.getPlaybooksError)

			if tt.getPlaybooksError == nil {
				mockRepo.
					EXPECT().
					GetPlaybooksCount(gomock.Any(), playbooks.PlaybookFilter{}).
					Return(0, tt.getPlaybooksCountError)
			}

			workflowsData, err := service.GetPlaybooksData(context.Background(), 0, 10, playbooks.PlaybookFilter{})
			assert.Error(t, err)
			assert.Empty(t, workflowsData.Entries)
			assert.Equal(t, 0, workflowsData.Total)
		})
	}
}

func TestServiceGetPlaybookByIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	uuidString := uuid.New()
	returnedPlaybook := &domain.Playbooks{
		ID:   uuidString,
		Name: "My Playbook",
	}
	// Expectation
	mockRepo.
		EXPECT().
		GetPlaybookById(gomock.Any(), uuidString.String()).
		Return(returnedPlaybook, nil)

	workflow, err := service.GetPlaybookById(context.Background(), uuidString.String())
	assert.NoError(t, err)
	assert.Equal(t, "My Playbook", workflow.Name)
}

func TestServiceGetPlaybookByIdFail(t *testing.T) {
	service, mockRepo := setupService(t)
	uuidString := uuid.New()

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				GetPlaybookById(gomock.Any(), uuidString.String()).
				Return(nil, tt.error_)

			workflow, err := service.GetPlaybookById(context.Background(), uuidString.String())
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceGetPlaybooksSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedPlaybooks := []domain.Playbooks{
		{
			ID:   uuid.New(),
			Name: "Playbook 1",
		},
		{
			ID:   uuid.New(),
			Name: "Playbook 2",
		},
	}

	mockRepo.
		EXPECT().
		GetPlaybooks(gomock.Any(), 0, 10, playbooks.PlaybookFilter{}).
		Return(returnedPlaybooks, nil)

	workflowsList, err := service.GetPlaybooks(context.Background(), 0, 10, playbooks.PlaybookFilter{})
	assert.NoError(t, err)
	assert.Len(t, workflowsList, 2)
	assert.Equal(t, "Playbook 1", workflowsList[0].Name)
	assert.Equal(t, "Playbook 2", workflowsList[1].Name)

}

func TestServiceGetPlaybookHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedHistory := []domain.PlaybookHistoryResponse{
		{
			ID:         uuid.New(),
			PlaybookID: uuid.New(),
			Status:     "completed",
		},
		{
			ID:         uuid.New(),
			PlaybookID: uuid.New(),
			Status:     "failed",
		},
	}

	mockRepo.
		EXPECT().
		GetPlaybookHistory(gomock.Any(), 0, 10, playbooks.PlaybookHistoryFilter{}).
		Return(returnedHistory, nil)

	historyList, err := service.GetPlaybookHistory(context.Background(), 0, 10, playbooks.PlaybookHistoryFilter{})
	assert.NoError(t, err)
	assert.Len(t, historyList, 2)
	assert.Equal(t, "completed", historyList[0].Status)
	assert.Equal(t, "failed", historyList[1].Status)

}

func TestServiceGetPlaybookHistoryByIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	uuidString := uuid.New()
	returnedPlaybook := &domain.PlaybookHistoryResponse{
		ID:     uuidString,
		Status: "complete",
	}
	// Expectation
	mockRepo.
		EXPECT().
		GetPlaybookHistoryById(gomock.Any(), uuidString).
		Return(returnedPlaybook, nil)

	workflow, err := service.GetPlaybookHistoryById(context.Background(), uuidString)
	assert.NoError(t, err)
	assert.Equal(t, "complete", workflow.Status)
}

func TestServiceGetPlaybookHistoryCountSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	mockRepo.
		EXPECT().
		GetPlaybookHistoryCount(gomock.Any(), playbooks.PlaybookHistoryFilter{}).
		Return(5, nil)

	count, err := service.GetPlaybookHistoryCount(context.Background(), playbooks.PlaybookHistoryFilter{})
	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestServiceGetPlaybookTriggersSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedTriggers := []domain.PlaybookTriggers{
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
		GetPlaybookTriggers(gomock.Any()).
		Return(returnedTriggers, nil)

	triggers, err := service.GetPlaybookTriggers(context.Background())
	assert.NoError(t, err)
	assert.Len(t, triggers, 2)
	assert.Equal(t, "Trigger 1", triggers[0].Name)
	assert.Equal(t, "Trigger 2", triggers[1].Name)

}

func TestServiceGetPlaybooksCountSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	mockRepo.
		EXPECT().
		GetPlaybooksCount(gomock.Any(), playbooks.PlaybookFilter{}).
		Return(10, nil)

	count, err := service.GetPlaybooksCount(context.Background(), playbooks.PlaybookFilter{})
	assert.NoError(t, err)
	assert.Equal(t, 10, count)
}

func TestServiceCreatePlaybookHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()
	returnedHistory := &domain.PlaybookHistory{
		ID:         uuid.New(),
		PlaybookID: workflowID,
		Status:     "created",
	}

	mockRepo.
		EXPECT().
		CreatePlaybookHistory(gomock.Any(), workflowID.String(), []domain.ResponseEdges{}).
		Return(returnedHistory, nil)

	history, err := service.CreatePlaybookHistory(context.Background(), workflowID.String(), []domain.ResponseEdges{})
	assert.NoError(t, err)
	assert.Equal(t, "created", history.Status)
}

func TestServiceGetPlaybookGraphByIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	uuidString := uuid.New()
	returnedPlaybook := &domain.PlaybookGraph{
		ID:   uuidString,
		Name: "My Playbook",
	}
	// Expectation
	mockRepo.
		EXPECT().
		GetPlaybookGraphById(gomock.Any(), uuidString.String()).
		Return(returnedPlaybook, nil)

	workflow, err := service.GetPlaybookGraphById(context.Background(), uuidString.String())
	assert.NoError(t, err)
	assert.Equal(t, "My Playbook", workflow.Name)
}

func TestServiceGetPlaybookGraphByIdFail(t *testing.T) {
	service, mockRepo := setupService(t)
	uuidString := uuid.New()

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				GetPlaybookGraphById(gomock.Any(), uuidString.String()).
				Return(nil, tt.error_)

			workflow, err := service.GetPlaybookGraphById(context.Background(), uuidString.String())
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceCreatePlaybookSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	newPlaybook := playbooks.PlaybookPayload{
		Name: "New Playbook",
	}

	returnedPlaybook := &domain.Playbooks{
		ID:   uuid.New(),
		Name: "New Playbook",
	}

	mockRepo.
		EXPECT().
		CreatePlaybook(gomock.Any(), newPlaybook).
		Return(returnedPlaybook, nil)

	createdPlaybook, err := service.CreatePlaybook(context.Background(), newPlaybook)
	assert.NoError(t, err)
	assert.Equal(t, "New Playbook", createdPlaybook.Name)
}

func TestServiceUpdatePlaybookSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New()
	updateData := playbooks.UpdatePlaybookData{
		Name: types.Nullable[string]{Set: true, Value: utils.StrPtr("Updated Playbook")},
	}

	returnedPlaybook := &domain.Playbooks{
		ID:   workflowID,
		Name: "Updated Playbook",
	}

	mockRepo.
		EXPECT().
		UpdatePlaybook(gomock.Any(), workflowID.String(), updateData).
		Return(returnedPlaybook, nil)

	updatedPlaybook, err := service.UpdatePlaybook(context.Background(), workflowID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Playbook", updatedPlaybook.Name)
}

func TestServiceUpdatePlaybookHistoryStatusSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New()
	newStatus := "completed"

	returnedHistory := &domain.PlaybookHistory{
		ID:     workflowHistoryID,
		Status: newStatus,
	}

	mockRepo.
		EXPECT().
		UpdatePlaybookHistoryStatus(gomock.Any(), workflowHistoryID.String(), newStatus).
		Return(returnedHistory, nil)

	updatedHistory, err := service.UpdatePlaybookHistoryStatus(context.Background(), workflowHistoryID.String(), newStatus)
	assert.NoError(t, err)
	assert.Equal(t, newStatus, updatedHistory.Status)
}

func TestServiceUpdatePlaybookHistoryStatusFail(t *testing.T) {
	service, mockRepo := setupService(t)
	workflowHistoryID := uuid.New()
	newStatus := "completed"

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				UpdatePlaybookHistoryStatus(gomock.Any(), workflowHistoryID.String(), newStatus).
				Return(nil, tt.error_)

			workflow, err := service.UpdatePlaybookHistoryStatus(context.Background(), workflowHistoryID.String(), newStatus)
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceUpdatePlaybookHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowHistoryID := uuid.New()
	updateData := playbooks.UpdatePlaybookHistoryData{
		Status: types.Nullable[string]{Set: true, Value: utils.StrPtr("completed")},
	}

	returnedHistory := &domain.PlaybookHistory{
		ID:     workflowHistoryID,
		Status: "completed",
	}

	mockRepo.
		EXPECT().
		UpdatePlaybookHistory(gomock.Any(), workflowHistoryID.String(), updateData).
		Return(returnedHistory, nil)

	updatedHistory, err := service.UpdatePlaybookHistory(context.Background(), workflowHistoryID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "completed", updatedHistory.Status)
}

func TestServiceUpdatePlaybookHistoryFail(t *testing.T) {
	service, mockRepo := setupService(t)
	workflowHistoryID := uuid.New()
	updateData := playbooks.UpdatePlaybookHistoryData{
		Status: types.Nullable[string]{Set: true, Value: utils.StrPtr("completed")},
	}

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching workflow", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				UpdatePlaybookHistory(gomock.Any(), workflowHistoryID.String(), updateData).
				Return(nil, tt.error_)

			workflow, err := service.UpdatePlaybookHistory(context.Background(), workflowHistoryID.String(), updateData)
			assert.Error(t, err)
			assert.Nil(t, workflow)
		})
	}
}

func TestServiceGetPlaybooksHistoryDataSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	returnedHistories := []domain.PlaybookHistoryResponse{
		{
			ID:     uuid.New(),
			Status: "completed",
		},
		{
			ID:     uuid.New(),
			Status: "in progress",
		},
	}

	mockRepo.
		EXPECT().
		GetPlaybookHistory(gomock.Any(), 0, 10, playbooks.PlaybookHistoryFilter{}).
		Return(returnedHistories, nil)

	mockRepo.
		EXPECT().
		GetPlaybookHistoryCount(gomock.Any(), playbooks.PlaybookHistoryFilter{}).
		Return(2, nil)

	historiesData, err := service.GetPlaybooksHistoryData(context.Background(), 0, 10, playbooks.PlaybookHistoryFilter{})

	assert.NoError(t, err)
	assert.Len(t, historiesData.Entries, 2)
	assert.Equal(t, "completed", historiesData.Entries[0].Status)
	assert.Equal(t, "in progress", historiesData.Entries[1].Status)
	assert.Equal(t, 2, historiesData.Total)
}

func TestServiceGetPlaybooksHistoryDataFail(t *testing.T) {
	service, mockRepo := setupService(t)

	tests := []struct {
		name                   string
		getPlaybooksError      error
		getPlaybooksCountError error
	}{
		{name: "get workflows data error", getPlaybooksError: fmt.Errorf("error occurred"), getPlaybooksCountError: nil},
		{name: "get workflows count error", getPlaybooksError: nil, getPlaybooksCountError: fmt.Errorf("error occurred")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.
				EXPECT().
				GetPlaybookHistory(gomock.Any(), 0, 10, playbooks.PlaybookHistoryFilter{}).
				Return([]domain.PlaybookHistoryResponse{}, tt.getPlaybooksError)

			if tt.getPlaybooksError == nil {
				mockRepo.
					EXPECT().
					GetPlaybookHistoryCount(gomock.Any(), playbooks.PlaybookHistoryFilter{}).
					Return(0, tt.getPlaybooksCountError)
			}

			workflowsData, err := service.GetPlaybooksHistoryData(context.Background(), 0, 10, playbooks.PlaybookHistoryFilter{})
			assert.Error(t, err)
			assert.Empty(t, workflowsData.Entries)
			assert.Equal(t, 0, workflowsData.Total)
		})
	}
}
