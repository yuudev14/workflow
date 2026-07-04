package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PlaybookTriggers struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
}

type Playbooks struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	Description *string    `db:"description" json:"description"`
	TriggerType *uuid.UUID `json:"trigger_type" db:"trigger_type"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type PlaybookHistory struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	PlaybookID  uuid.UUID       `db:"workflow_id" json:"workflow_id"`
	Status      string          `db:"status" json:"status"`
	Error       *string         `db:"error" json:"error"`
	Result      interface{}     `db:"result" json:"result"`
	TriggeredAt time.Time       `db:"triggered_at" json:"triggered_at"`
	Edges       json.RawMessage `db:"edges" json:"edges"`
}

type PlaybookGraph struct {
	ID          uuid.UUID        `db:"id" json:"id"`
	Name        string           `db:"name" json:"name"`
	Description *string          `db:"description" json:"description"`
	TriggerType *string          `json:"trigger_type" db:"trigger_type"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
	Tasks       *json.RawMessage `db:"tasks" json:"tasks"`
	Edges       *json.RawMessage `db:"edges" json:"edges"`
}

type PlaybookHistoryResponse struct {
	ID           uuid.UUID        `db:"id" json:"id"`
	PlaybookID   uuid.UUID        `db:"workflow_id" json:"workflow_id"`
	PlaybookData json.RawMessage  `db:"workflow_data" json:"workflow_data"`
	Status       string           `db:"status" json:"status"`
	Error        *string          `db:"error" json:"error"`
	Result       *json.RawMessage `db:"result" json:"result"`
	TriggeredAt  time.Time        `db:"triggered_at" json:"triggered_at"`
	Edges        json.RawMessage  `db:"edges" json:"edges"`
}
