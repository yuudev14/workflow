package models

import (
	"github.com/google/uuid"
)

type Edges struct {
	ID            uuid.UUID `db:"id" json:"id"`
	WorkflowID    uuid.UUID `db:"workflow_id" json:"workflow_id"`
	DestinationID uuid.UUID `db:"destination_id" json:"destination_id"`
	SourceID      uuid.UUID `db:"source_id" json:"source_id"`
}
