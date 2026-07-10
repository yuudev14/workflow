package playbooks

import (
	"encoding/json"

	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/types"
)

// NOTE: uuid fields bind as *string here on purpose. gin's query binder has no
// TextUnmarshaler support and treats uuid.UUID ([16]byte) as an array, which
// fails; the `uuid` validator rule still rejects malformed values.
type PlaybookFilter struct {
	Name       *string `form:"name" binding:"omitempty"`
	PlaybookID *string `form:"playbook_id" binding:"omitempty,uuid"`
}

type PlaybookHistoryFilter struct {
	Name       *string `form:"name" binding:"omitempty"`
	PlaybookID *string `form:"playbook_id" binding:"omitempty,uuid"`
}

// TriggerType is validated with domain.IsValidTriggerType in the service —
// gin's binding rules can't look inside types.Nullable.
type UpdatePlaybookData struct {
	Name              types.Nullable[string]          `json:"name,omitempty"`
	Description       types.Nullable[string]          `json:"description,omitempty"`
	TriggerType       types.Nullable[string]          `json:"trigger_type,omitempty"`
	TriggerParameters types.Nullable[json.RawMessage] `json:"trigger_parameters,omitempty"`
}

type UpdatePlaybookHistoryData struct {
	Status types.Nullable[string] `json:"status,omitempty"`
	Error  types.Nullable[string] `json:"error,omitempty"`
	Result any                    `json:"result,omitempty"`
}

type PlaybookPayload struct {
	Name              string          `json:"name" binding:"required"`
	Description       *string         `json:"description,omitempty"`
	TriggerType       *string         `json:"trigger_type,omitempty" binding:"omitempty,oneof=manual webhook referenced on_create on_update on_delete"`
	TriggerParameters json.RawMessage `json:"trigger_parameters,omitempty"`
}

type UpdatePlaybookTasksPayload struct {
	Task    *UpdatePlaybookData                      `json:"task"`
	Nodes   []tasks.TaskPayload                      `json:"nodes"`
	Edges   map[string][]string                      `json:"edges"`
	Handles *map[string]map[string]domain.EdgeHandle `json:"handles"`
}
