package tasks

import (
	"github.com/google/uuid"
	"github.com/yuudev14-workflow/workflow-service/internal/edges"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
)

type TaskPayload struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Parameters    *map[string]interface{} `json:"parameters,omitempty"`
	ConnectorName *string                 `json:"connector_name"`
	ConnectorID   *string                 `json:"connector_id"`
	Operation     string                  `json:"operation"`
	Config        types.Nullable[string]  `json:"config,omitempty"`
	X             float32                 `form:"x,default=0"`
	Y             float32                 `form:"y,default=0"`
}

type TaskHistoryFilter struct {
	WorkflowID *uuid.UUID `form:"workflow_id"`
}

type UpdateTaskHistoryData struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Parameters    interface{}            `json:"parameters,omitempty"`
	ConnectorName types.Nullable[string] `json:"connector_name"`
	ConnectorID   types.Nullable[string] `json:"connector_id"`
	Operation     string                 `json:"operation"`
	Config        types.Nullable[string] `json:"config,omitempty"`
	X             float32                `form:"x,default=0"`
	Y             float32                `form:"y,default=0"`
	Status        types.Nullable[string] `json:"status,omitempty"`
	Error         types.Nullable[string] `json:"error,omitempty"`
	Result        interface{}            `json:"result,omitempty"`
}

type UpdateWorkflowtasks struct {
	Task    *workflows.UpdateWorkflowData           `json:"task"`
	Nodes   []TaskPayload                           `json:"nodes"`
	Edges   map[string][]string                     `json:"edges"`
	Handles *map[string]map[string]edges.EdgeHandle `json:"handles"`
}

type UpdateWorkflowTaskHistoryStatus struct {
	Status string `json:"status" binding:"required"`
}
