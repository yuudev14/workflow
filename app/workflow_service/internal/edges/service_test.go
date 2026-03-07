package edges_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	mock_edges "github.com/yuudev14-workflow/workflow-service/internal/edges/mocks"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
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

func TestServiceGetEdgesByWorkflowIdSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	returnedEdges := []edges.ResponseEdges{
		{
			ID: uuid.New(),
		},
		{
			ID: uuid.New(),
		},
	}

	mockRepo.
		EXPECT().
		GetEdgesByWorkflowId(workflowID).
		Return(returnedEdges, nil)

	result, err := service.GetEdgesByWorkflowId(workflowID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestServiceGetEdgesByWorkflowIdFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	mockRepo.
		EXPECT().
		GetEdgesByWorkflowId(workflowID).
		Return(nil, fmt.Errorf("error occurred"))

	result, err := service.GetEdgesByWorkflowId(workflowID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestServiceInsertEdgesSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	edgesData := []edges.Edges{
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

	edgesData := []edges.Edges{
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

func TestServiceDeleteAllWorkflowEdgesSuccess(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	mockRepo.
		EXPECT().
		DeleteAllWorkflowEdges(nil, workflowID).
		Return(nil)

	err := service.DeleteAllWorkflowEdges(nil, workflowID)

	assert.NoError(t, err)
}

func TestServiceDeleteAllWorkflowEdgesFail(t *testing.T) {
	service, mockRepo := setupService(t)

	workflowID := uuid.New().String()

	mockRepo.
		EXPECT().
		DeleteAllWorkflowEdges(nil, workflowID).
		Return(fmt.Errorf("delete error"))

	err := service.DeleteAllWorkflowEdges(nil, workflowID)

	assert.Error(t, err)
}
