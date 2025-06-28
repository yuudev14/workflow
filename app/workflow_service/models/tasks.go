package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14-workflow/workflow-service/pkg/types"
)

type Position struct {
}

type Tasks struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	WorkflowID    string         `db:"workflow_id" json:"workflow_id"`
	Status        string         `db:"status" json:"status"`
	Name          string         `db:"name" json:"name"`
	Config        *string        `db:"config" json:"config"`
	ConnectorName string         `db:"connector_name" json:"connector_name"`
	Operation     string         `db:"operation" json:"operation"`
	Description   string         `db:"description" json:"description"`
	Parameters    types.JsonType `db:"parameters" json:"parameters"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
	Position      Position       `db:"position" json:"position"`
	X             float32        `db:"x" json:"x"`
	Y             float32        `db:"y" json:"y"`
}
