package incidents

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/repository_mock.go -package=mocks . IncidentRepository

type IncidentRepository interface {
	List(ctx context.Context, filter IncidentFilter) ([]IncidentListItem, *string, error)
	Count(ctx context.Context, filter IncidentFilter) (int, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Incident, error)
	GetDetail(ctx context.Context, id uuid.UUID) (IncidentDetail, error)
	Create(ctx context.Context, params CreateParams) (domain.Incident, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateParams) (domain.Incident, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.IncidentStatus) (domain.Incident, error)
	AppendEvent(ctx context.Context, params AppendEventParams) error
	AddNote(ctx context.Context, incidentID uuid.UUID, authorID *uuid.UUID, body string) (domain.IncidentNote, error)
	GetNote(ctx context.Context, noteID uuid.UUID) (domain.IncidentNote, error)
	UpdateNote(ctx context.Context, noteID uuid.UUID, body string) (domain.IncidentNote, error)
	DeleteNote(ctx context.Context, noteID uuid.UUID) error
	ListNotes(ctx context.Context, incidentID uuid.UUID) ([]domain.IncidentNote, error)

	// LinkAlert reports linked=false when the pair already existed, so a repeat
	// call is idempotent instead of writing a duplicate timeline entry.
	LinkAlert(ctx context.Context, incidentID, alertID uuid.UUID, source domain.LinkSource) (linked bool, err error)
	UnlinkAlert(ctx context.Context, incidentID, alertID uuid.UUID) error
	Summary(ctx context.Context) (IncidentsSummary, error)
}

//go:generate mockgen -destination=mocks/alert_timeline_mock.go -package=mocks . AlertTimeline

// AlertTimeline lets link and unlink write the other half of the story onto the
// alert, without depending on the whole alerts service.
type AlertTimeline interface {
	AppendAlertEvent(ctx context.Context, alertID uuid.UUID, eventType domain.EventType, actorID *uuid.UUID, body []byte) error
}
