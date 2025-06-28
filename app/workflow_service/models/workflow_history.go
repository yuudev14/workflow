package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowHistory struct {
	ID          uuid.UUID   `db:"id" json:"id"`
	WorkflowID  uuid.UUID   `db:"workflow_id" json:"workflow_id"`
	Status      string      `db:"status" json:"status"`
	Error       *string     `db:"error" json:"error"`
	Result      interface{} `db:"result" json:"result"`
	TriggeredAt time.Time   `db:"triggered_at" json:"triggered_at"`
}
