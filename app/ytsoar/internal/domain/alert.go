package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AlertSeverity mirrors the alert_severity pg enum. Incidents share it.
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
)

// AlertSeverities is ordered most severe first, matching the pg enum's
// declaration order.
var AlertSeverities = []AlertSeverity{
	SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow,
}

func IsValidAlertSeverity(s string) bool {
	switch AlertSeverity(s) {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	}
	return false
}

// AlertStatus mirrors the alert_status pg enum. "falsepos" is deliberate - it
// is the spelling the frontend schema uses.
type AlertStatus string

const (
	AlertStatusNew           AlertStatus = "new"
	AlertStatusInvestigating AlertStatus = "investigating"
	AlertStatusResolved      AlertStatus = "resolved"
	AlertStatusFalsePositive AlertStatus = "falsepos"
	AlertStatusClosed        AlertStatus = "closed"
)

func IsValidAlertStatus(s string) bool {
	switch AlertStatus(s) {
	case AlertStatusNew, AlertStatusInvestigating, AlertStatusResolved,
		AlertStatusFalsePositive, AlertStatusClosed:
		return true
	}
	return false
}

// OpenAlertStatuses is the open set. It must stay identical to the predicate on
// alerts_open_idx and alerts_open_fingerprint_idx, or dedup and the live counts
// will disagree with the indexes that serve them.
var OpenAlertStatuses = []AlertStatus{AlertStatusNew, AlertStatusInvestigating}

// SourceKind mirrors the source_kind pg enum.
type SourceKind string

const (
	SourceKindEDR      SourceKind = "edr"
	SourceKindIdentity SourceKind = "identity"
	SourceKindEmail    SourceKind = "email"
	SourceKindFirewall SourceKind = "firewall"
	SourceKindDLP      SourceKind = "dlp"
)

func IsValidSourceKind(s string) bool {
	switch SourceKind(s) {
	case SourceKindEDR, SourceKindIdentity, SourceKindEmail,
		SourceKindFirewall, SourceKindDLP:
		return true
	}
	return false
}

// EventType mirrors the event_type pg enum, shared by alert_events and
// incident_events. Several values have no writer until later milestones.
type EventType string

const (
	EventTypeCreated       EventType = "created"
	EventTypeStatusChanged EventType = "status_changed"
	EventTypeEscalated     EventType = "escalated"
	EventTypeTriage        EventType = "triage"
	EventTypeSLA           EventType = "sla"
	EventTypeCorrelation   EventType = "correlation"
	EventTypeAttackTag     EventType = "attack_tag"
	EventTypeLinked        EventType = "linked"
	EventTypeUnlinked      EventType = "unlinked"
	// EventTypeUpdated records a field-level edit (severity, assignee, team,
	// tags) so the timeline is a complete audit trail, not only status changes.
	EventTypeUpdated EventType = "updated"
)

// SLAState mirrors the sla_state pg enum. Nothing transitions it until the SLA
// watcher ships; every row sits at "ok".
type SLAState string

const (
	SLAStateOK       SLAState = "ok"
	SLAStateWarning  SLAState = "warning"
	SLAStateBreached SLAState = "breached"
	SLAStateMet      SLAState = "met"
)

type Alert struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	Title         string          `db:"title" json:"title"`
	Severity      AlertSeverity   `db:"severity" json:"severity"`
	Status        AlertStatus     `db:"status" json:"status"`
	SourceKind    SourceKind      `db:"source_kind" json:"source_kind"`
	Reporter      *string         `db:"reporter" json:"reporter"`
	AssigneeID    *uuid.UUID      `db:"assignee_id" json:"assignee_id"`
	TeamID        *uuid.UUID      `db:"team_id" json:"team_id"`
	Payload       json.RawMessage `db:"payload" json:"payload,omitempty"`
	Tags          []string        `db:"tags" json:"tags"`
	Triage        json.RawMessage `db:"triage" json:"triage,omitempty"`
	ClosureNote   *string         `db:"closure_note" json:"closure_note"`
	Fingerprint   string          `db:"fingerprint" json:"fingerprint"`
	DedupCount    int32           `db:"dedup_count" json:"dedup_count"`
	LastSeen      time.Time       `db:"last_seen" json:"last_seen"`
	TriagedAt     *time.Time      `db:"triaged_at" json:"triaged_at"`
	SLADeadline   *time.Time      `db:"sla_deadline" json:"sla_deadline"`
	SLABreachedAt *time.Time      `db:"sla_breached_at" json:"sla_breached_at"`
	SLAState      SLAState        `db:"sla_state" json:"sla_state"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
}

type AlertNote struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	AlertID        uuid.UUID  `db:"alert_id" json:"alert_id"`
	AuthorID       *uuid.UUID `db:"author_id" json:"author_id"`
	AuthorUsername *string    `db:"author_username" json:"author_username"`
	Body           string     `db:"body" json:"body"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

// AlertEvent is one timeline row. A nil ActorID means the system did it.
type AlertEvent struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	AlertID       uuid.UUID       `db:"alert_id" json:"alert_id"`
	Type          EventType       `db:"type" json:"type"`
	ActorID       *uuid.UUID      `db:"actor_id" json:"actor_id"`
	ActorUsername *string         `db:"actor_username" json:"actor_username"`
	Body          json.RawMessage `db:"body" json:"body"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}
