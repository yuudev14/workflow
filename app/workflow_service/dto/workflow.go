package dto

import (
	"github.com/google/uuid"
	"github.com/yuudev14-workflow/workflow-service/pkg/types"
)

type WorkflowFilter struct {
	Name *string `form:"name"`
}

type UpdateWorkflowData struct {
	Name        types.Nullable[string]    `json:"name,omitempty"`
	Description types.Nullable[string]    `json:"description,omitempty"`
	TriggerType types.Nullable[uuid.UUID] `json:"trigger_type,omitempty"`
}

type UpdateWorkflowHistoryData struct {
	Status types.Nullable[string] `json:"status,omitempty"`
	Error  types.Nullable[string] `json:"error,omitempty"`
	Result interface{}            `json:"result,omitempty"`
}

type UpdateTaskHistoryData struct {
	Status types.Nullable[string] `json:"status,omitempty"`
	Error  types.Nullable[string] `json:"error,omitempty"`
	Result interface{}            `json:"result,omitempty"`
}

type Task struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	Parameters    *map[string]interface{} `json:"parameters,omitempty"`
	ConnectorName string                  `json:"connector_name"`
	Operation     string                  `json:"operation"`
	Config        types.Nullable[string]  `json:"config,omitempty"`
	X             float32                 `form:"x,default=0"`
	Y             float32                 `form:"y,default=0"`
}

type UpdateWorkflowtasks struct {
	Task  *UpdateWorkflowData `json:"task"`
	Nodes []Task              `json:"nodes"`
	Edges map[string][]string `json:"edges"`
}

type UpdateWorkflowTaskHistoryStatus struct {
	Status string `json:"status" binding:"required"`
}

type WorkflowPayload struct {
	Name        string     `json:"name" binding:"required"`
	Description *string    `json:"description,omitempty"`
	TriggerType *uuid.UUID `json:"trigger_type,omitempty"`
}
