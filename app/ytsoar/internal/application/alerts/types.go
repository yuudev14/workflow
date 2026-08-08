package alerts

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/types"
)

var (
	ErrAlertNotFound = apperr.New(apperr.NotFound, "alert not found")
	ErrNoteNotFound  = apperr.New(apperr.NotFound, "note not found")
	ErrNotNoteAuthor = apperr.New(apperr.Forbidden, "only the author can edit this note")
	ErrInvalidCursor = apperr.New(apperr.Invalid, "invalid cursor")
	ErrValidation    = apperr.New(apperr.Invalid, "invalid input")
)

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

type AlertFilter struct {
	Status     []string `form:"status" binding:"omitempty,dive,oneof=new investigating resolved falsepos closed"`
	Severity   []string `form:"severity" binding:"omitempty,dive,oneof=critical high medium low"`
	SourceKind []string `form:"source_kind" binding:"omitempty,dive,oneof=edr identity email firewall dlp"`
	SLAState   []string `form:"sla_state" binding:"omitempty,dive,oneof=ok warning breached met"`
	AssigneeID []string `form:"assignee_id" binding:"omitempty,dive,uuid"`
	TeamID     []string `form:"team_id" binding:"omitempty,dive,uuid"`
	// Unassigned unions with AssigneeID rather than contradicting it: a picker
	// offering "Unassigned" beside names has to mean "either".
	Unassigned *bool    `form:"unassigned" binding:"omitempty"`
	Tags       []string `form:"tags" binding:"omitempty"`
	// RFC3339 instants, not dates - a queue filter is "since 14:30 today my
	// time", and the offset removes any question of whose midnight is meant.
	// The datetime tag is gin's own validator, so a bad value 400s at bind.
	CreatedFrom *string `form:"created_from" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	CreatedTo   *string `form:"created_to" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	DedupMin    *int    `form:"dedup_min" binding:"omitempty,min=1"`
	Triaged     *bool   `form:"triaged" binding:"omitempty"`
	Search      *string `form:"q" binding:"omitempty"`
	Cursor      *string `form:"cursor" binding:"omitempty"`
	// Offset is the programmatic paging mode. Unlike the cursor it can skip or
	// repeat rows when alerts arrive mid-page; that is the trade a caller opts
	// into by using it.
	Offset *int `form:"offset" binding:"omitempty,min=0"`
	Limit  int  `form:"limit" binding:"omitempty,min=1,max=200"`
}

func (f AlertFilter) CreatedRange() (from, to *time.Time, err error) {
	if from, err = types.ParseInstant(f.CreatedFrom); err != nil {
		return nil, nil, err
	}
	if to, err = types.ParseInstant(f.CreatedTo); err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, nil, apperr.New(apperr.Invalid, "created_from must not be after created_to")
	}
	return from, to, nil
}

func (f AlertFilter) Normalized() (AlertFilter, error) {
	if f.Cursor != nil && *f.Cursor != "" && f.Offset != nil {
		return f, apperr.New(apperr.Invalid, "use either cursor or offset, not both")
	}
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	return f, nil
}

type CreateAlertPayload struct {
	Title      string          `json:"title" binding:"required"`
	Severity   string          `json:"severity" binding:"required,oneof=critical high medium low"`
	SourceKind string          `json:"source_kind" binding:"required,oneof=edr identity email firewall dlp"`
	Reporter   *string         `json:"reporter,omitempty"`
	AssigneeID *string         `json:"assignee_id,omitempty" binding:"omitempty,uuid"`
	TeamID     *string         `json:"team_id,omitempty" binding:"omitempty,uuid"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	Tags       []string        `json:"tags,omitempty"`
	CreatedAt  *time.Time      `json:"created_at,omitempty"`
}

type BatchCreateAlertPayload struct {
	Alerts []CreateAlertPayload `json:"alerts" binding:"required,min=1,max=500,dive"`
}

type UpdateAlertPayload struct {
	Severity   types.Nullable[string]   `json:"severity,omitempty"`
	AssigneeID types.Nullable[string]   `json:"assignee_id,omitempty"`
	TeamID     types.Nullable[string]   `json:"team_id,omitempty"`
	Tags       types.Nullable[[]string] `json:"tags,omitempty"`
}

type UpdateAlertStatusPayload struct {
	Status      string  `json:"status" binding:"required,oneof=new investigating resolved falsepos closed"`
	ClosureNote *string `json:"closure_note,omitempty"`
}

type EscalateAlertPayload struct {
	Title *string `json:"title,omitempty"`
}

type AddNotePayload struct {
	Body string `json:"body" binding:"required,min=1"`
}

type UpdateNotePayload struct {
	Body string `json:"body" binding:"required,min=1"`
}

type UpsertParams struct {
	Title       string
	Severity    domain.AlertSeverity
	SourceKind  domain.SourceKind
	Reporter    *string
	AssigneeID  *uuid.UUID
	TeamID      *uuid.UUID
	Payload     json.RawMessage
	Tags        []string
	Fingerprint string
	CreatedAt   *time.Time
}

type UpdateParams struct {
	Severity   types.Nullable[domain.AlertSeverity]
	AssigneeID types.Nullable[uuid.UUID]
	TeamID     types.Nullable[uuid.UUID]
	Tags       types.Nullable[[]string]
}

type AppendEventParams struct {
	AlertID uuid.UUID
	Type    domain.EventType
	ActorID *uuid.UUID
	Body    json.RawMessage
}

type AlertListItem struct {
	ID          uuid.UUID            `db:"id" json:"id"`
	Title       string               `db:"title" json:"title"`
	Severity    domain.AlertSeverity `db:"severity" json:"severity"`
	Status      domain.AlertStatus   `db:"status" json:"status"`
	SourceKind  domain.SourceKind    `db:"source_kind" json:"source_kind"`
	Reporter    *string              `db:"reporter" json:"reporter"`
	AssigneeID  *uuid.UUID           `db:"assignee_id" json:"assignee_id"`
	Assignee    *string              `db:"assignee" json:"assignee"`
	TeamID      *uuid.UUID           `db:"team_id" json:"team_id"`
	Tags        []string             `db:"tags" json:"tags"`
	DedupCount  int32                `db:"dedup_count" json:"dedup_count"`
	LastSeen    time.Time            `db:"last_seen" json:"last_seen"`
	TriagedAt   *time.Time           `db:"triaged_at" json:"triaged_at"`
	SLADeadline *time.Time           `db:"sla_deadline" json:"sla_deadline"`
	SLAState    domain.SLAState      `db:"sla_state" json:"sla_state"`
	CreatedAt   time.Time            `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time            `db:"updated_at" json:"updated_at"`
	RunCount    int                  `db:"run_count" json:"run_count"`
}

type IncidentRef struct {
	ID       uuid.UUID             `db:"id" json:"id"`
	Title    string                `db:"title" json:"title"`
	Status   domain.IncidentStatus `db:"status" json:"status"`
	Severity domain.AlertSeverity  `db:"severity" json:"severity"`
	Source   domain.LinkSource     `db:"link_source" json:"link_source"`
}

type AlertDetail struct {
	domain.Alert
	Assignee        *string             `json:"assignee"`
	Timeline        []domain.AlertEvent `json:"timeline"`
	Notes           []domain.AlertNote  `json:"notes"`
	LinkedIncidents []IncidentRef       `json:"linked_incidents"`
	RelatedAlerts   []RelatedAlert      `json:"related_alerts"`
	RunCount        int                 `json:"run_count"`
	Runs            []RunRef            `json:"runs"`
}

// RunRef is one playbook run that acted on this record. PlaybookID is what lets
// the UI deep-link to the run itself rather than the whole executions list; the
// name is nullable because the playbook may since have been deleted. TriggerType
// and TriggeredBy are nullable too - runs predating the run-input migration have
// neither, and a scheduled or event-driven run has no user behind it.
type RunRef struct {
	PlaybookHistoryID uuid.UUID `json:"playbook_history_id"`
	PlaybookID        uuid.UUID `json:"playbook_id"`
	Playbook          *string   `json:"playbook"`
	Status            string    `json:"status"`
	TriggerType       *string   `json:"trigger_type"`
	TriggeredBy       *string   `json:"triggered_by"`
	CreatedAt         time.Time `json:"created_at"`
}

type RelatedAlert struct {
	ID         uuid.UUID `json:"id"`
	Title      string    `json:"title"`
	Score      float64   `json:"score"`
	SharedIOCs []string  `json:"shared_iocs"`
	CreatedAt  time.Time `json:"created_at"`
}

type SeverityBucket struct {
	Severity domain.AlertSeverity `db:"severity" json:"severity"`
	Count    int                  `db:"count" json:"count"`
}

type SourceBucket struct {
	SourceKind domain.SourceKind `db:"source_kind" json:"source_kind"`
	Count      int               `db:"count" json:"count"`
}

type PlaybookSuccess struct {
	Label       string  `db:"label" json:"label"`
	SuccessRate float64 `db:"success_rate" json:"success_rate"`
}

// VolumePoint is one point on the volume series. It carries the timestamp it
// covers because the range is caller-picked: length, start and bucket width all
// vary per request, so the chart cannot derive its own x-axis labels.
type VolumePoint struct {
	BucketStart time.Time `db:"bucket_start" json:"bucket_start"`
	Count       int       `db:"count" json:"count"`
}

// Total, BySeverity and BySource stay all-time-open on purpose: the queue header
// and its filter counts read them, so range-scoping would silently turn
// "47 open" into "opened in the last 14 days". The range applies to Volume and
// the window counts only.
type AlertsSummary struct {
	Total        int                 `json:"total"`
	BySeverity   []SeverityBucket    `json:"by_severity"`
	BySource     []SourceBucket      `json:"by_source"`
	TopPlaybooks []PlaybookSuccess   `json:"top_playbooks"`
	Volume       []VolumePoint       `json:"volume"`
	Range        types.ResolvedRange `json:"range"`
	Created      types.WindowCount   `json:"created"`
	Resolved     types.WindowCount   `json:"resolved"`
	// MTTTSeconds is mean time to triage: created_at to triaged_at.
	MTTTSeconds types.WindowCount `json:"mttt_seconds"`
}
