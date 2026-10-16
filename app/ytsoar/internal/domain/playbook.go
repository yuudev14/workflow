package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

// TriggerType says how a playbook starts. The set is static: each value needs
// real backend behavior, so new triggers are added here (plus the trigger_type
// enum in the schema), never as database rows. The frontend keeps its own copy
// of this list for the trigger picker.
type TriggerType string

const (
	TriggerTypeManual     TriggerType = "manual"
	TriggerTypeWebhook    TriggerType = "webhook"
	TriggerTypeReferenced TriggerType = "referenced"
	TriggerTypeOnCreate   TriggerType = "on_create"
	TriggerTypeOnUpdate   TriggerType = "on_update"
	TriggerTypeOnDelete   TriggerType = "on_delete"
)

func IsValidTriggerType(s string) bool {
	switch TriggerType(s) {
	case TriggerTypeManual, TriggerTypeWebhook, TriggerTypeReferenced,
		TriggerTypeOnCreate, TriggerTypeOnUpdate, TriggerTypeOnDelete:
		return true
	}
	return false
}

// Run and task statuses. A run is queued -> in_progress -> success|failed, a
// task is pending -> in_progress -> success|failed (or pending -> skipped).
const (
	StatusQueued     = "queued"
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusSkipped    = "skipped"
)

// ErrInvalidTransition means a status change was refused: the row is not in a
// state the target status can be reached from (or the row does not exist).
var ErrInvalidTransition = apperr.New(apperr.Conflict, "status change is not allowed")

// Transitions maps a target status to the statuses it may be reached from. The
// repositories hand From(to) to the UPDATE, so the check is atomic: two
// workers cannot both move the same run from queued to in_progress.
type Transitions map[string][]string

// From returns the statuses allowed to move to `to`; unknown targets have none.
func (t Transitions) From(to string) []string { return t[to] }

var RunTransitions = Transitions{
	StatusInProgress: {StatusQueued},
	StatusSuccess:    {StatusInProgress},
	StatusFailed:     {StatusQueued, StatusInProgress},
}

// A task can fail or succeed straight from pending: the start node never
// reports in_progress, and a node can fail before it starts.
var TaskTransitions = Transitions{
	StatusInProgress: {StatusPending},
	StatusSuccess:    {StatusPending, StatusInProgress},
	StatusFailed:     {StatusPending, StatusInProgress},
	StatusSkipped:    {StatusPending},
}

type Playbooks struct {
	ID                uuid.UUID       `db:"id" json:"id"`
	Name              string          `db:"name" json:"name"`
	Description       *string         `db:"description" json:"description"`
	TriggerType       *string         `json:"trigger_type" db:"trigger_type"`
	TriggerParameters json.RawMessage `json:"trigger_parameters" db:"trigger_parameters"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
}

type PlaybookHistory struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	PlaybookID  uuid.UUID       `db:"playbook_id" json:"playbook_id"`
	Status      string          `db:"status" json:"status"`
	Error       *string         `db:"error" json:"error"`
	Result      any             `db:"result" json:"result"`
	TriggeredAt time.Time       `db:"triggered_at" json:"triggered_at"`
	Edges       json.RawMessage `db:"edges" json:"edges"`
	TriggerType *string         `db:"trigger_type" json:"trigger_type"`
	TriggeredBy *uuid.UUID      `db:"triggered_by" json:"triggered_by"`
	Input       json.RawMessage `db:"input" json:"input"`
}

type PlaybookGraph struct {
	ID                uuid.UUID        `db:"id" json:"id"`
	Name              string           `db:"name" json:"name"`
	Description       *string          `db:"description" json:"description"`
	TriggerType       *string          `json:"trigger_type" db:"trigger_type"`
	TriggerParameters json.RawMessage  `json:"trigger_parameters" db:"trigger_parameters"`
	CreatedAt         time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time        `db:"updated_at" json:"updated_at"`
	Tasks             *json.RawMessage `db:"tasks" json:"tasks"`
	Edges             *json.RawMessage `db:"edges" json:"edges"`
}

type PlaybookHistoryResponse struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	PlaybookID   uuid.UUID        `db:"playbook_id" json:"playbook_id"`
	PlaybookData json.RawMessage  `db:"playbook_data" json:"playbook_data"`
	Status       string           `db:"status" json:"status"`
	Error        *string          `db:"error" json:"error"`
	Result       *json.RawMessage `db:"result" json:"result"`
	TriggeredAt  time.Time        `db:"triggered_at" json:"triggered_at"`
	Edges        json.RawMessage  `db:"edges" json:"edges"`
	TriggerType  *string          `db:"trigger_type" json:"trigger_type"`
	TriggeredBy  *uuid.UUID       `db:"triggered_by" json:"triggered_by"`
	Input        json.RawMessage  `db:"input" json:"input"`
}
