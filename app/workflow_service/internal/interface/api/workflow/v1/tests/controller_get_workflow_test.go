package workflow_http_v1_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
)

func TestControllerGetWorkflowsSuccess(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows?offset=0&limit=10",
		nil,
	)

	c.Request = req

	expected := types.Entries[workflows.Workflows]{
		Entries: []workflows.Workflows{
			{
				Name: "Workflow 1",
			},
			{
				Name: "Workflow 2",
			},
		},
		Total: 2,
	}

	mockService.
		EXPECT().
		GetWorkflowsData(0, 10, workflows.WorkflowFilter{}).
		Return(expected, nil)

	controller.GetWorkflows(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestControllerGetWorkflowsServiceError(t *testing.T) {
	controller, mockService, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows?offset=0&limit=10",
		nil,
	)

	c.Request = req

	mockService.
		EXPECT().
		GetWorkflowsData(0, 10, workflows.WorkflowFilter{}).
		Return(types.Entries[workflows.Workflows]{}, fmt.Errorf("service error"))

	controller.GetWorkflows(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestControllerGetWorkflowsInvalidQuery(t *testing.T) {
	controller, _, c, recorder := setupController(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/workflows?offset=invalid&limit=10",
		nil,
	)

	c.Request = req

	controller.GetWorkflows(c)

	assert.NotEqual(t, http.StatusOK, recorder.Code)
}
