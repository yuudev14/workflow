package playbooks

import (
	"encoding/json"

	"github.com/google/uuid"

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
	// ModuleType and RecordID answer "what has run against this alert?", which
	// is the direction both detail pages ask in.
	ModuleType *string `form:"module_type" binding:"omitempty,oneof=alert incident"`
	RecordID   *string `form:"record_id" binding:"omitempty,uuid"`
}

// RunStamp is the provenance written onto a playbook_history row: who started
// the run, how, and against what. All fields are optional - an editor trigger
// stamps only the trigger type.
type RunStamp struct {
	TriggerType *string
	TriggeredBy *uuid.UUID
	Input       *domain.RunInput
}

// RunPlaybookPayload is the body of the module-side run endpoints.
//
// Records are capped at 100 because the hydrated payload rides in `input_json`
// on *every* node's gRPC call: 100 records at ~2KB is ~200KB per node against
// gRPC's 4MB default.
type RunPlaybookPayload struct {
	PlaybookID string         `json:"playbook_id" binding:"required,uuid"`
	RecordIDs  []string       `json:"record_ids" binding:"required,min=1,max=100,dive,uuid"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// PlaybooksSummary backs the automation KPI row. Every figure is paired with the
// preceding window of equal length, which is what makes a rendered delta mean
// something - the deleted metrics mock had a bare "+2" with no baseline at all.
//
// Playbooks.Previous is the count that existed before the window opened, so the
// delta reads as "playbooks added in this window".
type PlaybooksSummary struct {
	Range       types.ResolvedRange `json:"range"`
	Playbooks   types.WindowCount   `json:"playbooks"`
	Runs        types.WindowCount   `json:"runs"`
	Failed      types.WindowCount   `json:"failed"`
	SuccessRate types.WindowRate    `json:"success_rate"`
}

// TriggerType is validated with domain.IsValidTriggerType in the service -
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
	Task    *UpdatePlaybookData                     `json:"task"`
	Nodes   []tasks.TaskPayload                     `json:"nodes"`
	Edges   map[string][]string                     `json:"edges"`
	Handles map[string]map[string]domain.EdgeHandle `json:"handles"`
}
