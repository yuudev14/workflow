package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

// TriggerType says how a playbook starts. The set is fixed: each value needs real backend
// behavior, so add new triggers here (and in the trigger_type enum), never as database
// rows. The frontend keeps its own copy for the trigger picker.
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

// WebhookAuth says how a caller of a webhook playbook proves it may run it. It lives in
// trigger_parameters.auth.
type WebhookAuth string

const (
	WebhookAuthAPIKey WebhookAuth = "api_key"
	WebhookAuthNone   WebhookAuth = "none"
)

var ErrUnknownWebhookAuth = apperr.New(apperr.Invalid, "trigger_parameters.auth must be api_key or none")

// WebhookAuthOf reads the auth type from trigger parameters. A missing value is api_key, so
// a webhook playbook is never public by accident.
func WebhookAuthOf(params json.RawMessage) (WebhookAuth, error) {
	var p struct {
		Auth *WebhookAuth `json:"auth"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return "", apperr.New(apperr.Invalid, "trigger_parameters must be a JSON object")
		}
	}
	if p.Auth == nil {
		return WebhookAuthAPIKey, nil
	}
	switch *p.Auth {
	case WebhookAuthAPIKey, WebhookAuthNone:
		return *p.Auth, nil
	}
	return "", ErrUnknownWebhookAuth
}

// Run and task statuses. A run goes from queued to in_progress to success or failed.
// A task goes from pending to in_progress to success or failed. A task can also go
// from pending to skipped.
const (
	StatusQueued     = "queued"
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusSkipped    = "skipped"
)

// ErrInvalidTransition: the status change was refused. The row is not in a status that
// allows the new one, or the row does not exist.
var ErrInvalidTransition = apperr.New(apperr.Conflict, "status change is not allowed")

// Transitions maps a new status to the statuses that allow it. The repositories pass
// From(to) to the UPDATE, so the check is atomic and two workers cannot both move the same
// run from queued to in_progress.
type Transitions map[string][]string

// From returns the statuses that allow a change to `to`. An unknown status has none.
func (t Transitions) From(to string) []string { return t[to] }

var RunTransitions = Transitions{
	StatusInProgress: {StatusQueued},
	StatusSuccess:    {StatusInProgress},
	StatusFailed:     {StatusQueued, StatusInProgress},
}

// A task can succeed or fail directly from pending: the start node never reports
// in_progress, and a node can fail before it starts.
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
	DeletedAt         *time.Time      `db:"deleted_at" json:"-"`
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
	FinishedAt   *time.Time       `db:"finished_at" json:"finished_at"`
	ScheduleID   *uuid.UUID       `db:"schedule_id" json:"schedule_id"`
	ScheduleName *string          `db:"schedule_name" json:"schedule_name"`
}
