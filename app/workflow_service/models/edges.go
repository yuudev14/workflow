package models

import (
	"database/sql"

	"github.com/google/uuid"
)

type Edges struct {
	ID                uuid.UUID      `db:"id" json:"id"`
	WorkflowID        uuid.UUID      `db:"workflow_id" json:"workflow_id"`
	DestinationID     uuid.UUID      `db:"destination_id" json:"destination_id"`
	SourceID          uuid.UUID      `db:"source_id" json:"source_id"`
	SourceHandle      sql.NullString `db:"source_handle" json:"source_handle"`
	DestinationHandle sql.NullString `db:"destination_handle" json:"destination_handle"`
}
