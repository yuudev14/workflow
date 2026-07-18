package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
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
}
