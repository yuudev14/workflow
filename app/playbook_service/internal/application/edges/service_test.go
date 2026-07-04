package edges_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	mock_edges "github.com/yuudev14/ytsoar/internal/application/edges/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logging"
	"go.uber.org/mock/gomock"
)

func TestMain(m *testing.M) {
	logging.Setup("DEBUG")
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

	workflowID := uuid.New().String()

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
		GetEdgesByPlaybookId(workflowID).
		Return(returnedEdges, nil)

	result, err := service.GetEdgesByPlaybookId(workflowID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceGetEdgesByPlaybookIdFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	mockRepo.
		EXPECT().
		GetEdgesByPlaybookId(workflowID).
		Return(nil, fmt.Errorf("error occurred"))

	result, err := service.GetEdgesByPlaybookId(workflowID)

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
		InsertEdges(nil, edgesData).
		Return(edgesData, nil)

	result, err := service.InsertEdges(nil, edgesData)

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
		InsertEdges(nil, edgesData).
		Return(nil, fmt.Errorf("insert error"))

	result, err := service.InsertEdges(nil, edgesData)

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
		DeleteEdges(nil, edgeIds).
		Return(nil)

	err := service.DeleteEdges(nil, edgeIds)

	assert.NoError(t, err)
}

func TestServiceDeleteEdgesFail(t *testing.T) {
	service, mockRepo := setupService(t)

	edgeIds := []uuid.UUID{
		uuid.New(),
	}

	mockRepo.
		EXPECT().
		DeleteEdges(nil, edgeIds).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteEdges(nil, edgeIds)

	assert.Error(t, err)
}

func TestServiceDeleteAllPlaybookEdgesSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	mockRepo.
		EXPECT().
		DeleteAllPlaybookEdges(nil, workflowID).
		Return(nil)

	err := service.DeleteAllPlaybookEdges(nil, workflowID)

	assert.NoError(t, err)
}

func TestServiceDeleteAllPlaybookEdgesFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	mockRepo.
		EXPECT().
		DeleteAllPlaybookEdges(nil, workflowID).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteAllPlaybookEdges(nil, workflowID)

	assert.Error(t, err)
}
