package models

import (
	"time"

	"github.com/google/uuid"
)

type TaskHistory struct {
	ID                uuid.UUID   `db:"id" json:"id"`
	WorkflowHistoryID uuid.UUID   `db:"workflow_history_id" json:"workflow_history_id"`
	TaskID            uuid.UUID   `db:"task_id" json:"task_id"`
	Status            string      `db:"status" json:"status"`
	Error             *string     `db:"error" json:"error"`
	Result            interface{} `db:"result" json:"result"`
	TriggeredAt       time.Time   `db:"triggered_at" json:"triggered_at"`
}
