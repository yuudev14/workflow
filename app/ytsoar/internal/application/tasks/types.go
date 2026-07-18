package tasks

import (
	"github.com/yuudev14/ytsoar/internal/types"
)

type TaskPayload struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Parameters    *map[string]any        `json:"parameters,omitempty"`
	ConnectorName *string                `json:"connector_name"`
	ConnectorID   *string                `json:"connector_id"`
	Operation     string                 `json:"operation"`
	Config        types.Nullable[string] `json:"config,omitempty"`
	X             float32                `form:"x,default=0"`
	Y             float32                `form:"y,default=0"`
}

type TaskHistoryFilter struct {
	PlaybookID *string `form:"playbook_id" binding:"omitempty,uuid"`
}

type UpdateTaskHistoryData struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Parameters    any                    `json:"parameters,omitempty"`
	ConnectorName types.Nullable[string] `json:"connector_name"`
	ConnectorID   types.Nullable[string] `json:"connector_id"`
	Operation     string                 `json:"operation"`
	Config        types.Nullable[string] `json:"config,omitempty"`
	X             float32                `form:"x,default=0"`
	Y             float32                `form:"y,default=0"`
	Status        types.Nullable[string] `json:"status,omitempty"`
	Error         types.Nullable[string] `json:"error,omitempty"`
	Result        any                    `json:"result,omitempty"`
}

type UpdatePlaybookTaskHistoryStatus struct {
	Status string `json:"status" binding:"required"`
}
