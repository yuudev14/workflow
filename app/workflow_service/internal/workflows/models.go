package workflows

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type WorkflowTriggers struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
}

type Workflows struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	Description *string    `db:"description" json:"description"`
	TriggerType *uuid.UUID `json:"trigger_type" db:"trigger_type"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

type WorkflowHistory struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	WorkflowID  uuid.UUID       `db:"workflow_id" json:"workflow_id"`
	Status      string          `db:"status" json:"status"`
	Error       *string         `db:"error" json:"error"`
	Result      interface{}     `db:"result" json:"result"`
	TriggeredAt time.Time       `db:"triggered_at" json:"triggered_at"`
	Edges       json.RawMessage `db:"edges" json:"edges"`
}
