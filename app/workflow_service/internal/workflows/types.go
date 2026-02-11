package workflows

import (
	"github.com/google/uuid"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
)

type WorkflowFilter struct {
	Name       *string    `form:"name"`
	WorkflowID *uuid.UUID `form:"workflow_id"`
}

type WorkflowHistoryFilter struct {
	Name       *string    `form:"name"`
	WorkflowID *uuid.UUID `form:"workflow_id"`
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

type WorkflowPayload struct {
	Name        string     `json:"name" binding:"required"`
	Description *string    `json:"description,omitempty"`
	TriggerType *uuid.UUID `json:"trigger_type,omitempty"`
}
