package alerts

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . AlertRepository

type AlertRepository interface {
	List(ctx context.Context, filter AlertFilter) ([]AlertListItem, *string, error)
	Count(ctx context.Context, filter AlertFilter) (int, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Alert, error)
	GetDetail(ctx context.Context, id uuid.UUID) (AlertDetail, error)
	Upsert(ctx context.Context, params UpsertParams) (alert domain.Alert, inserted bool, err error)
	Update(ctx context.Context, id uuid.UUID, params UpdateParams) (domain.Alert, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AlertStatus, closureNote *string) (domain.Alert, error)
	AppendEvent(ctx context.Context, params AppendEventParams) error
	Summary(ctx context.Context) (AlertsSummary, error)

	AddNote(ctx context.Context, alertID uuid.UUID, authorID *uuid.UUID, body string) (domain.AlertNote, error)
	GetNote(ctx context.Context, noteID uuid.UUID) (domain.AlertNote, error)
	UpdateNote(ctx context.Context, noteID uuid.UUID, body string) (domain.AlertNote, error)
	DeleteNote(ctx context.Context, noteID uuid.UUID) error
	ListNotes(ctx context.Context, alertID uuid.UUID) ([]domain.AlertNote, error)
}

//go:generate mockgen -destination=mocks/incident_linker_mock.go -package=mocks . IncidentLinker

// IncidentLinker is the slice of the incidents domain that escalation needs. It
// is defined here, in the consumer, so alerts does not depend on the whole
// incidents service.
type IncidentLinker interface {
	CreateForEscalation(ctx context.Context, params EscalationParams) (domain.Incident, error)
}

type EscalationParams struct {
	Title      string
	Severity   domain.AlertSeverity
	AlertID    uuid.UUID
	AssigneeID *uuid.UUID
	TeamID     *uuid.UUID
	ActorID    *uuid.UUID
}
