package edges_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	mock_edges "github.com/yuudev14/ytsoar/internal/application/edges/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {

	os.Exit(m.Run())
}

func setupService(t *testing.T) (edges.EdgeService, *mock_edges.MockEdgeRepository) {
	ctrl := gomock.NewController(t)

	mockRepo := mock_edges.NewMockEdgeRepository(ctrl)
	service := edges.NewEdgeServiceImpl(mockRepo)

	t.Cleanup(ctrl.Finish)

	return service, mockRepo
}

func TestServiceGetEdgesByPlaybookIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New().String()

	returnedEdges := []domain.ResponseEdges{
		{
			ID: uuid.New(),
		},
		{
			ID: uuid.New(),
		},
	}

	mockRepo.
		EXPECT().
		GetEdgesByPlaybookId(gomock.Any(), playbookID).
		Return(returnedEdges, nil)

	result, err := service.GetEdgesByPlaybookId(context.Background(), playbookID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceGetEdgesByPlaybookIdFail(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New().String()

	mockRepo.
		EXPECT().
		GetEdgesByPlaybookId(gomock.Any(), playbookID).
		Return(nil, fmt.Errorf("error occurred"))

	result, err := service.GetEdgesByPlaybookId(context.Background(), playbookID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestServiceInsertEdgesSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	edgesData := []domain.Edges{
		{
			ID: uuid.New(),
		},
		{
			ID: uuid.New(),
		},
	}

	mockRepo.
		EXPECT().
		InsertEdges(gomock.Any(), edgesData).
		Return(edgesData, nil)

	result, err := service.InsertEdges(context.Background(), edgesData)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceInsertEdgesFail(t *testing.T) {
	service, mockRepo := setupService(t)

	edgesData := []domain.Edges{
		{
			ID: uuid.New(),
		},
	}

	mockRepo.
		EXPECT().
		InsertEdges(gomock.Any(), edgesData).
		Return(nil, fmt.Errorf("insert error"))

	result, err := service.InsertEdges(context.Background(), edgesData)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestServiceDeleteEdgesSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	edgeIds := []uuid.UUID{
		uuid.New(),
		uuid.New(),
	}

	mockRepo.
		EXPECT().
		DeleteEdges(gomock.Any(), edgeIds).
		Return(nil)

	err := service.DeleteEdges(context.Background(), edgeIds)

	assert.NoError(t, err)
}

func TestServiceDeleteEdgesFail(t *testing.T) {
	service, mockRepo := setupService(t)

	edgeIds := []uuid.UUID{
		uuid.New(),
	}

	mockRepo.
		EXPECT().
		DeleteEdges(gomock.Any(), edgeIds).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteEdges(context.Background(), edgeIds)

	assert.Error(t, err)
}

func TestServiceDeleteAllPlaybookEdgesSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New().String()

	mockRepo.
		EXPECT().
		DeleteAllPlaybookEdges(gomock.Any(), playbookID).
		Return(nil)

	err := service.DeleteAllPlaybookEdges(context.Background(), playbookID)

	assert.NoError(t, err)
}

func TestServiceDeleteAllPlaybookEdgesFail(t *testing.T) {
	service, mockRepo := setupService(t)

	playbookID := uuid.New().String()

	mockRepo.
		EXPECT().
		DeleteAllPlaybookEdges(gomock.Any(), playbookID).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteAllPlaybookEdges(context.Background(), playbookID)

	assert.Error(t, err)
}
