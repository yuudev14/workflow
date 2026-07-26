package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type IncidentStatus string

const (
	IncidentStatusOpen          IncidentStatus = "open"
	IncidentStatusInvestigating IncidentStatus = "investigating"
	IncidentStatusContained     IncidentStatus = "contained"
	IncidentStatusResolved      IncidentStatus = "resolved"
	IncidentStatusClosed        IncidentStatus = "closed"
)

func IsValidIncidentStatus(s string) bool {
	switch IncidentStatus(s) {
	case IncidentStatusOpen, IncidentStatusInvestigating, IncidentStatusContained,
		IncidentStatusResolved, IncidentStatusClosed:
		return true
	}
	return false
}

var ClosedIncidentStatuses = []IncidentStatus{IncidentStatusResolved, IncidentStatusClosed}

// incidentTransitions is the stepper, plus the reverse edges a real SOC needs:
// an incident can be reopened, and containment can fail back to investigating.
// Skipping forward is allowed (open -> contained) because analysts often
// contain before recording that they were investigating.
var incidentTransitions = map[IncidentStatus][]IncidentStatus{
	IncidentStatusOpen:          {IncidentStatusInvestigating, IncidentStatusContained, IncidentStatusResolved, IncidentStatusClosed},
	IncidentStatusInvestigating: {IncidentStatusContained, IncidentStatusResolved, IncidentStatusClosed, IncidentStatusOpen},
	IncidentStatusContained:     {IncidentStatusResolved, IncidentStatusClosed, IncidentStatusInvestigating},
	IncidentStatusResolved:      {IncidentStatusClosed, IncidentStatusInvestigating},
	IncidentStatusClosed:        {IncidentStatusInvestigating},
}

// CanTransitionTo reports whether the stepper permits next. Staying put is
// allowed so an idempotent retry is not an error.
func (s IncidentStatus) CanTransitionTo(next IncidentStatus) bool {
	if s == next {
		return true
	}
	for _, allowed := range incidentTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

func (s IncidentStatus) IsClosed() bool {
	return s == IncidentStatusResolved || s == IncidentStatusClosed
}

type LinkSource string

const (
	LinkSourceManual      LinkSource = "manual"
	LinkSourceEscalate    LinkSource = "escalate"
	LinkSourceCorrelation LinkSource = "correlation"
)

type Incident struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	Title         string         `db:"title" json:"title"`
	Severity      AlertSeverity  `db:"severity" json:"severity"`
	Status        IncidentStatus `db:"status" json:"status"`
	AssigneeID    *uuid.UUID     `db:"assignee_id" json:"assignee_id"`
	TeamID        *uuid.UUID     `db:"team_id" json:"team_id"`
	Tags          []string       `db:"tags" json:"tags"`
	ResolvedAt    *time.Time     `db:"resolved_at" json:"resolved_at"`
	SLADeadline   *time.Time     `db:"sla_deadline" json:"sla_deadline"`
	SLABreachedAt *time.Time     `db:"sla_breached_at" json:"sla_breached_at"`
	SLAState      SLAState       `db:"sla_state" json:"sla_state"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
}

type IncidentEvent struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	IncidentID    uuid.UUID       `db:"incident_id" json:"incident_id"`
	Type          EventType       `db:"type" json:"type"`
	ActorID       *uuid.UUID      `db:"actor_id" json:"actor_id"`
	ActorUsername *string         `db:"actor_username" json:"actor_username"`
	Body          json.RawMessage `db:"body" json:"body"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

type IncidentNote struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	IncidentID     uuid.UUID  `db:"incident_id" json:"incident_id"`
	AuthorID       *uuid.UUID `db:"author_id" json:"author_id"`
	AuthorUsername *string    `db:"author_username" json:"author_username"`
	Body           string     `db:"body" json:"body"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

type IncidentLinkedAlert struct {
	ID         uuid.UUID     `db:"id" json:"id"`
	Title      string        `db:"title" json:"title"`
	Severity   AlertSeverity `db:"severity" json:"severity"`
	SourceKind SourceKind    `db:"source_kind" json:"source_kind"`
	Reporter   *string       `db:"reporter" json:"reporter"`
	LinkSource LinkSource    `db:"link_source" json:"link_source"`
}
