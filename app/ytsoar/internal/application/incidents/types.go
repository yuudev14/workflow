package incidents

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/types"
)

var (
	ErrIncidentNotFound = apperr.New(apperr.NotFound, "incident not found")
	ErrAlertNotLinked   = apperr.New(apperr.NotFound, "alert is not linked to this incident")
	ErrNoteNotFound     = apperr.New(apperr.NotFound, "note not found")
	ErrNotNoteAuthor    = apperr.New(apperr.Forbidden, "only the author can edit this note")
	ErrValidation       = apperr.New(apperr.Invalid, "invalid input")
)

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

type IncidentFilter struct {
	Status     []string `form:"status" binding:"omitempty,dive,oneof=open investigating contained resolved closed"`
	Severity   []string `form:"severity" binding:"omitempty,dive,oneof=critical high medium low"`
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
	Search      *string `form:"q" binding:"omitempty"`
	Cursor      *string `form:"cursor" binding:"omitempty"`
	// Offset is the programmatic paging mode. Unlike the cursor it can skip or
	// repeat rows when incidents arrive mid-page; that is the trade a caller
	// opts into by using it.
	Offset *int `form:"offset" binding:"omitempty,min=0"`
	Limit  int  `form:"limit" binding:"omitempty,min=1,max=200"`
	// Open filters to the not-yet-closed set, matching incidents_open_idx.
	Open *bool `form:"open" binding:"omitempty"`
}

func (f IncidentFilter) CreatedRange() (from, to *time.Time, err error) {
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

func (f IncidentFilter) Normalized() (IncidentFilter, error) {
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

type CreateIncidentPayload struct {
	Title      string   `json:"title" binding:"required"`
	Severity   string   `json:"severity" binding:"required,oneof=critical high medium low"`
	Status     *string  `json:"status,omitempty" binding:"omitempty,oneof=open investigating contained resolved closed"`
	AssigneeID *string  `json:"assignee_id,omitempty" binding:"omitempty,uuid"`
	TeamID     *string  `json:"team_id,omitempty" binding:"omitempty,uuid"`
	Tags       []string `json:"tags,omitempty"`
	AlertIDs   []string `json:"alert_ids,omitempty" binding:"omitempty,dive,uuid"`
}

type UpdateIncidentPayload struct {
	Severity   types.Nullable[string]   `json:"severity,omitempty"`
	AssigneeID types.Nullable[string]   `json:"assignee_id,omitempty"`
	TeamID     types.Nullable[string]   `json:"team_id,omitempty"`
	Tags       types.Nullable[[]string] `json:"tags,omitempty"`
}

type UpdateIncidentStatusPayload struct {
	Status string `json:"status" binding:"required,oneof=open investigating contained resolved closed"`
}

type AddNotePayload struct {
	Body string `json:"body" binding:"required,min=1"`
}

type UpdateNotePayload struct {
	Body string `json:"body" binding:"required,min=1"`
}

type LinkAlertPayload struct {
	AlertID string `json:"alert_id" binding:"required,uuid"`
}

type CreateParams struct {
	Title      string
	Severity   domain.AlertSeverity
	Status     domain.IncidentStatus
	AssigneeID *uuid.UUID
	TeamID     *uuid.UUID
	Tags       []string
}

type UpdateParams struct {
	Severity   types.Nullable[domain.AlertSeverity]
	AssigneeID types.Nullable[uuid.UUID]
	TeamID     types.Nullable[uuid.UUID]
	Tags       types.Nullable[[]string]
}

type AppendEventParams struct {
	IncidentID uuid.UUID
	Type       domain.EventType
	ActorID    *uuid.UUID
	Body       json.RawMessage
}

type IncidentListItem struct {
	ID          uuid.UUID             `db:"id" json:"id"`
	Title       string                `db:"title" json:"title"`
	Severity    domain.AlertSeverity  `db:"severity" json:"severity"`
	Status      domain.IncidentStatus `db:"status" json:"status"`
	AssigneeID  *uuid.UUID            `db:"assignee_id" json:"assignee_id"`
	Assignee    *string               `db:"assignee" json:"assignee"`
	TeamID      *uuid.UUID            `db:"team_id" json:"team_id"`
	Tags        []string              `db:"tags" json:"tags"`
	AlertCount  int                   `db:"alert_count" json:"alert_count"`
	RunCount    int                   `db:"run_count" json:"run_count"`
	ResolvedAt  *time.Time            `db:"resolved_at" json:"resolved_at"`
	SLADeadline *time.Time            `db:"sla_deadline" json:"sla_deadline"`
	SLAState    domain.SLAState       `db:"sla_state" json:"sla_state"`
	CreatedAt   time.Time             `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time             `db:"updated_at" json:"updated_at"`
}

// Runs and IOCs stay empty until module-event triggers link a playbook run to an
// incident and IOC extraction ships. The fields exist so the response shape does
// not change when they do.
type IncidentDetail struct {
	domain.Incident
	Assignee     *string                      `json:"assignee"`
	AlertCount   int                          `json:"alert_count"`
	RunCount     int                          `json:"run_count"`
	LinkedAlerts []domain.IncidentLinkedAlert `json:"linked_alerts"`
	Timeline     []domain.IncidentEvent       `json:"timeline"`
	Notes        []domain.IncidentNote        `json:"notes"`
	Runs         []IncidentRun                `json:"runs"`
	IOCs         []IOC                        `json:"iocs"`
}

// PlaybookID is what lets the UI deep-link to the run itself rather than the
// whole executions list. TriggerType and TriggeredBy are nullable: runs predating
// the run-input migration have neither, and a scheduled or event-driven run has
// no user behind it.
type IncidentRun struct {
	PlaybookHistoryID uuid.UUID `json:"playbook_history_id"`
	PlaybookID        uuid.UUID `json:"playbook_id"`
	Playbook          string    `json:"playbook"`
	Status            string    `json:"status"`
	TriggerType       *string   `json:"trigger_type"`
	TriggeredBy       *string   `json:"triggered_by"`
	CreatedAt         time.Time `json:"created_at"`
}

type IOC struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type StatusBucket struct {
	Status domain.IncidentStatus `db:"status" json:"status"`
	Count  int                   `db:"count" json:"count"`
}

type SeverityBucket struct {
	Severity domain.AlertSeverity `db:"severity" json:"severity"`
	Count    int                  `db:"count" json:"count"`
}

type SLARisk struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Title       string     `db:"title" json:"title"`
	SLADeadline *time.Time `db:"sla_deadline" json:"sla_deadline"`
	Breached    bool       `db:"breached" json:"breached"`
}

// MTTRPoint is one point on the resolution-time series. It carries the
// timestamp it covers because the range is caller-picked: length, start and
// bucket width all vary per request, so the chart cannot derive its own x-axis.
type MTTRPoint struct {
	BucketStart time.Time `db:"bucket_start" json:"bucket_start"`
	AvgSeconds  int       `db:"avg_seconds" json:"avg_seconds"`
}

// OpenTotal, StatusMix and SeverityMix stay all-time-open on purpose: the queue
// header and its filter counts read them, so range-scoping would silently turn
// "6 open" into "opened in the last 14 days".
type IncidentsSummary struct {
	OpenTotal   int                 `json:"open_total"`
	StatusMix   []StatusBucket      `json:"status_mix"`
	SeverityMix []SeverityBucket    `json:"severity_mix"`
	MTTRTrend   []MTTRPoint         `json:"mttr_trend"`
	SLAAtRisk   []SLARisk           `json:"sla_at_risk"`
	Range       types.ResolvedRange `json:"range"`
	Created     types.WindowCount   `json:"created"`
	Resolved    types.WindowCount   `json:"resolved"`
	// MTTRSeconds is mean time to resolve across the whole window, beside the
	// same figure for the preceding window.
	MTTRSeconds types.WindowCount `json:"mttr_seconds"`
}
