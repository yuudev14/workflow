package playbooks

import (
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/types"
)

type PlaybookFilter struct {
	Name       *string `form:"name" binding:"omitempty"`
	PlaybookID *string `form:"playbook_id" binding:"omitempty,uuid"`
}

type PlaybookHistoryFilter struct {
	Name       *string `form:"name" binding:"omitempty"`
	PlaybookID *string `form:"playbook_id" binding:"omitempty,uuid"`
}

type UpdatePlaybookData struct {
	Name        types.Nullable[string]    `json:"name,omitempty"`
	Description types.Nullable[string]    `json:"description,omitempty"`
	TriggerType types.Nullable[uuid.UUID] `json:"trigger_type,omitempty"`
}

type UpdatePlaybookHistoryData struct {
	Status types.Nullable[string] `json:"status,omitempty"`
	Error  types.Nullable[string] `json:"error,omitempty"`
	Result interface{}            `json:"result,omitempty"`
}

type PlaybookPayload struct {
	Name        string     `json:"name" binding:"required"`
	Description *string    `json:"description,omitempty"`
	TriggerType *uuid.UUID `json:"trigger_type,omitempty"`
}

type UpdatePlaybookTasksPayload struct {
	Task    *UpdatePlaybookData                      `json:"task"`
	Nodes   []tasks.TaskPayload                      `json:"nodes"`
	Edges   map[string][]string                      `json:"edges"`
	Handles *map[string]map[string]domain.EdgeHandle `json:"handles"`
}
