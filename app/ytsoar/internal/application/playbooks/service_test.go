package playbooks_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	mock_playbooks "github.com/yuudev14/ytsoar/internal/application/playbooks/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
	"github.com/yuudev14/ytsoar/internal/utils"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {

	os.Exit(m.Run())
}

func setupService(t *testing.T) (playbooks.PlaybookService, *mock_playbooks.MockPlaybookRepository) {
	ctrl := gomock.NewController(t)

	mockRepo := mock_playbooks.NewMockPlaybookRepository(ctrl)
	service := playbooks.NewPlaybookService(logger.NewNop(), mockRepo)

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

	playbooksData, err := service.GetPlaybooksData(context.Background(), 0, 10, playbooks.PlaybookFilter{})

	assert.NoError(t, err)
	assert.Len(t, playbooksData.Entries, 2)
	assert.Equal(t, "Playbook 1", playbooksData.Entries[0].Name)
	assert.Equal(t, "Playbook 2", playbooksData.Entries[1].Name)
	assert.Equal(t, 2, playbooksData.Total)
}

func TestServiceGetPlaybooksDataFail(t *testing.T) {
	service, mockRepo := setupService(t)

	tests := []struct {
		name                   string
		getPlaybooksError      error
		getPlaybooksCountError error
	}{
		{name: "get playbooks data error", getPlaybooksError: fmt.Errorf("error occurred"), getPlaybooksCountError: nil},
		{name: "get playbooks count error", getPlaybooksError: nil, getPlaybooksCountError: fmt.Errorf("error occurred")},
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

			playbooksData, err := service.GetPlaybooksData(context.Background(), 0, 10, playbooks.PlaybookFilter{})
			assert.Error(t, err)
			assert.Empty(t, playbooksData.Entries)
			assert.Equal(t, 0, playbooksData.Total)
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

	playbook, err := service.GetPlaybookById(context.Background(), uuidString.String())
	assert.NoError(t, err)
	assert.Equal(t, "My Playbook", playbook.Name)
}

func TestServiceGetPlaybookByIdFail(t *testing.T) {
	service, mockRepo := setupService(t)
	uuidString := uuid.New()

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching playbook", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				GetPlaybookById(gomock.Any(), uuidString.String()).
				Return(nil, tt.error_)

			playbook, err := service.GetPlaybookById(context.Background(), uuidString.String())
			assert.Error(t, err)
			assert.Nil(t, playbook)
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

	playbooksList, err := service.GetPlaybooks(context.Background(), 0, 10, playbooks.PlaybookFilter{})
	assert.NoError(t, err)
	assert.Len(t, playbooksList, 2)
	assert.Equal(t, "Playbook 1", playbooksList[0].Name)
	assert.Equal(t, "Playbook 2", playbooksList[1].Name)

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

	playbook, err := service.GetPlaybookHistoryById(context.Background(), uuidString)
	assert.NoError(t, err)
	assert.Equal(t, "complete", playbook.Status)
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

func TestServiceUpdatePlaybookValidatesTriggerType(t *testing.T) {
	service, mockRepo := setupService(t)
	id := uuid.NewString()

	// unknown trigger type never reaches the repository
	bad := "not_a_trigger"
	_, err := service.UpdatePlaybook(context.Background(), id, playbooks.UpdatePlaybookData{
		TriggerType: types.Nullable[string]{Value: &bad, Set: true},
	})
	assert.ErrorContains(t, err, "unknown trigger type")

	// valid trigger type passes through
	good := string(domain.TriggerTypeWebhook)
	data := playbooks.UpdatePlaybookData{
		TriggerType: types.Nullable[string]{Value: &good, Set: true},
	}
	mockRepo.
		EXPECT().
		UpdatePlaybook(gomock.Any(), id, data).
		Return(&domain.Playbooks{TriggerType: &good}, nil)

	updated, err := service.UpdatePlaybook(context.Background(), id, data)
	assert.NoError(t, err)
	assert.Equal(t, &good, updated.TriggerType)
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

	playbookID := uuid.New()
	returnedHistory := &domain.PlaybookHistory{
		ID:         uuid.New(),
		PlaybookID: playbookID,
		Status:     "created",
	}

	mockRepo.
		EXPECT().
		CreatePlaybookHistory(gomock.Any(), playbookID.String(), []domain.ResponseEdges{}).
		Return(returnedHistory, nil)

	history, err := service.CreatePlaybookHistory(context.Background(), playbookID.String(), []domain.ResponseEdges{})
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

	playbook, err := service.GetPlaybookGraphById(context.Background(), uuidString.String())
	assert.NoError(t, err)
	assert.Equal(t, "My Playbook", playbook.Name)
}

func TestServiceGetPlaybookGraphByIdFail(t *testing.T) {
	service, mockRepo := setupService(t)
	uuidString := uuid.New()

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching playbook", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				GetPlaybookGraphById(gomock.Any(), uuidString.String()).
				Return(nil, tt.error_)

			playbook, err := service.GetPlaybookGraphById(context.Background(), uuidString.String())
			assert.Error(t, err)
			assert.Nil(t, playbook)
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

	playbookID := uuid.New()
	updateData := playbooks.UpdatePlaybookData{
		Name: types.Nullable[string]{Set: true, Value: utils.StrPtr("Updated Playbook")},
	}

	returnedPlaybook := &domain.Playbooks{
		ID:   playbookID,
		Name: "Updated Playbook",
	}

	mockRepo.
		EXPECT().
		UpdatePlaybook(gomock.Any(), playbookID.String(), updateData).
		Return(returnedPlaybook, nil)

	updatedPlaybook, err := service.UpdatePlaybook(context.Background(), playbookID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Playbook", updatedPlaybook.Name)
}

func TestServiceUpdatePlaybookHistoryStatusSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New()
	newStatus := "completed"

	returnedHistory := &domain.PlaybookHistory{
		ID:     playbookHistoryID,
		Status: newStatus,
	}

	mockRepo.
		EXPECT().
		UpdatePlaybookHistoryStatus(gomock.Any(), playbookHistoryID.String(), newStatus).
		Return(returnedHistory, nil)

	updatedHistory, err := service.UpdatePlaybookHistoryStatus(context.Background(), playbookHistoryID.String(), newStatus)
	assert.NoError(t, err)
	assert.Equal(t, newStatus, updatedHistory.Status)
}

func TestServiceUpdatePlaybookHistoryStatusFail(t *testing.T) {
	service, mockRepo := setupService(t)
	playbookHistoryID := uuid.New()
	newStatus := "completed"

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching playbook", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				UpdatePlaybookHistoryStatus(gomock.Any(), playbookHistoryID.String(), newStatus).
				Return(nil, tt.error_)

			playbook, err := service.UpdatePlaybookHistoryStatus(context.Background(), playbookHistoryID.String(), newStatus)
			assert.Error(t, err)
			assert.Nil(t, playbook)
		})
	}
}

func TestServiceUpdatePlaybookHistorySuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookHistoryID := uuid.New()
	updateData := playbooks.UpdatePlaybookHistoryData{
		Status: types.Nullable[string]{Set: true, Value: utils.StrPtr("completed")},
	}

	returnedHistory := &domain.PlaybookHistory{
		ID:     playbookHistoryID,
		Status: "completed",
	}

	mockRepo.
		EXPECT().
		UpdatePlaybookHistory(gomock.Any(), playbookHistoryID.String(), updateData).
		Return(returnedHistory, nil)

	updatedHistory, err := service.UpdatePlaybookHistory(context.Background(), playbookHistoryID.String(), updateData)
	assert.NoError(t, err)
	assert.Equal(t, "completed", updatedHistory.Status)
}

func TestServiceUpdatePlaybookHistoryFail(t *testing.T) {
	service, mockRepo := setupService(t)
	playbookHistoryID := uuid.New()
	updateData := playbooks.UpdatePlaybookHistoryData{
		Status: types.Nullable[string]{Set: true, Value: utils.StrPtr("completed")},
	}

	tests := []struct {
		name   string
		error_ error
	}{
		{name: "Playbook not found", error_: nil},
		{name: "error occured when fetching playbook", error_: fmt.Errorf("error occurd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Expectation
			mockRepo.
				EXPECT().
				UpdatePlaybookHistory(gomock.Any(), playbookHistoryID.String(), updateData).
				Return(nil, tt.error_)

			playbook, err := service.UpdatePlaybookHistory(context.Background(), playbookHistoryID.String(), updateData)
			assert.Error(t, err)
			assert.Nil(t, playbook)
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
		{name: "get playbooks data error", getPlaybooksError: fmt.Errorf("error occurred"), getPlaybooksCountError: nil},
		{name: "get playbooks count error", getPlaybooksError: nil, getPlaybooksCountError: fmt.Errorf("error occurred")},
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

			playbooksData, err := service.GetPlaybooksHistoryData(context.Background(), 0, 10, playbooks.PlaybookHistoryFilter{})
			assert.Error(t, err)
			assert.Empty(t, playbooksData.Entries)
			assert.Equal(t, 0, playbooksData.Total)
		})
	}
}
