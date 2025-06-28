package models

import (
	"time"

	"github.com/google/uuid"
)

type Schedulers struct {
	ID         uuid.UUID `db:"id" json:"id"`
	WorkflowID string    `db:"workflow_id" json:"workflow_id"`
	Cron       string    `db:"cron" json:"cron"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}
