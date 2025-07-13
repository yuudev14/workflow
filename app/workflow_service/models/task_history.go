package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/yuudev14-workflow/workflow-service/pkg/types"
)

type TaskHistory struct {
	ID                uuid.UUID      `db:"id" json:"id"`
	WorkflowHistoryID uuid.UUID      `db:"workflow_history_id" json:"workflow_history_id"`
	TaskID            uuid.UUID      `db:"task_id" json:"task_id"`
	Status            string         `db:"status" json:"status"`
	Error             *string        `db:"error" json:"error"`
	Result            interface{}    `db:"result" json:"result"`
	TriggeredAt       time.Time      `db:"triggered_at" json:"triggered_at"`
	Name              string         `db:"name" json:"name"`
	Config            *string        `db:"config" json:"config"`
	ConnectorName     *string        `db:"connector_name" json:"connector_name"`
	ConnectorID       *string        `db:"connector_id" json:"connector_id"`
	Operation         string         `db:"operation" json:"operation"`
	Description       string         `db:"description" json:"description"`
	Parameters        types.JsonType `db:"parameters" json:"parameters"`
	X                 float32        `db:"x" json:"x"`
	Y                 float32        `db:"y" json:"y"`
	DestinationIDs    pq.StringArray `db:"destination_ids" json:"destination_ids"`
}
