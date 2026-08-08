package alerts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

type Service struct {
	logger    logger.Logger
	repo      AlertRepository
	incidents IncidentLinker
	txManager contracts.TxManager
	events    contracts.ModuleEventPublisher
	users     contracts.UserDirectory
}

func NewService(
	log logger.Logger,
	repo AlertRepository,
	incidents IncidentLinker,
	txManager contracts.TxManager,
	events contracts.ModuleEventPublisher,
	users contracts.UserDirectory,
) *Service {
	return &Service{
		logger:    log,
		repo:      repo,
		incidents: incidents,
		txManager: txManager,
		events:    events,
		users:     users,
	}
}

func (s *Service) List(ctx context.Context, filter AlertFilter) (types.CursorPage[AlertListItem], error) {
	filter, err := filter.Normalized()
	if err != nil {
		return types.CursorPage[AlertListItem]{}, err
	}

	items, next, err := s.repo.List(ctx, filter)
	if err != nil {
		return types.CursorPage[AlertListItem]{}, err
	}

	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return types.CursorPage[AlertListItem]{}, err
	}

	return types.CursorPage[AlertListItem]{Entries: items, Total: total, NextCursor: next}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (AlertDetail, error) {
	return s.repo.GetDetail(ctx, id)
}

func (s *Service) Summary(ctx context.Context, rng types.ResolvedRange) (AlertsSummary, error) {
	return s.repo.Summary(ctx, rng)
}

// A fingerprint collision against a still-open alert is the same finding
// recurring, so it publishes alert.updated rather than alert.created - otherwise
// a storm fires every on_create playbook once per occurrence.
func (s *Service) Create(ctx context.Context, payload CreateAlertPayload, actorID *uuid.UUID) (domain.Alert, error) {
	params, err := s.toUpsertParams(payload)
	if err != nil {
		return domain.Alert{}, err
	}

	var (
		alert    domain.Alert
		inserted bool
	)
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		alert, inserted, err = s.repo.Upsert(txCtx, params)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		return s.repo.AppendEvent(txCtx, AppendEventParams{
			AlertID: alert.ID,
			Type:    domain.EventTypeCreated,
			ActorID: actorID,
			Body:    mustJSON(map[string]any{"reporter": alert.Reporter, "source_kind": alert.SourceKind}),
		})
	})
	if err != nil {
		return domain.Alert{}, err
	}

	event := domain.ModuleEventUpdated
	if inserted {
		event = domain.ModuleEventCreated
	}
	s.publish(domain.ModuleEventAlert, event, alert)

	return alert, nil
}

// Each item gets its own transaction so one malformed alert cannot roll back a
// whole forwarder batch.
func (s *Service) CreateBatch(ctx context.Context, payload BatchCreateAlertPayload, actorID *uuid.UUID) ([]domain.Alert, int, error) {
	created := make([]domain.Alert, 0, len(payload.Alerts))
	failed := 0

	for i, item := range payload.Alerts {
		alert, err := s.Create(ctx, item, actorID)
		if err != nil {
			failed++
			s.logger.Warnw("batch ingest item failed", "index", i, "err", err)
			continue
		}
		created = append(created, alert)
	}

	return created, failed, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, payload UpdateAlertPayload, actorID *uuid.UUID) (domain.Alert, error) {
	params, err := toUpdateParams(payload)
	if err != nil {
		return domain.Alert{}, err
	}

	var alert domain.Alert
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		before, err := s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		alert, err = s.repo.Update(txCtx, id, params)
		if err != nil {
			return err
		}

		return s.appendUpdateEvent(txCtx, id, before, alert, actorID)
	})
	if err != nil {
		return domain.Alert{}, err
	}

	s.publish(domain.ModuleEventAlert, domain.ModuleEventUpdated, alert)
	return alert, nil
}

func (s *Service) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	payload UpdateAlertStatusPayload,
	actorID *uuid.UUID,
) (domain.Alert, error) {
	if !domain.IsValidAlertStatus(payload.Status) {
		return domain.Alert{}, apperr.Wrap(apperr.Invalid, "unknown alert status", ErrValidation)
	}
	status := domain.AlertStatus(payload.Status)

	closureNote := payload.ClosureNote
	if closureNote != nil {
		trimmed := strings.TrimSpace(*closureNote)
		if trimmed == "" {
			closureNote = nil
		} else {
			closureNote = &trimmed
		}
	}

	var alert domain.Alert
	err := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		alert, err = s.repo.UpdateStatus(txCtx, id, status, closureNote)
		if err != nil {
			return err
		}

		body := map[string]any{"from": current.Status, "to": status}
		if closureNote != nil {
			body["closure_note"] = *closureNote
		}

		return s.repo.AppendEvent(txCtx, AppendEventParams{
			AlertID: id,
			Type:    domain.EventTypeStatusChanged,
			ActorID: actorID,
			Body:    mustJSON(body),
		})
	})
	if err != nil {
		return domain.Alert{}, err
	}

	s.publish(domain.ModuleEventAlert, domain.ModuleEventUpdated, alert)
	return alert, nil
}

func (s *Service) Escalate(
	ctx context.Context,
	id uuid.UUID,
	payload EscalateAlertPayload,
	actorID *uuid.UUID,
) (domain.Incident, error) {
	var (
		incident domain.Incident
		alert    domain.Alert
	)

	err := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		alert, err = s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		title := alert.Title
		if payload.Title != nil && strings.TrimSpace(*payload.Title) != "" {
			title = strings.TrimSpace(*payload.Title)
		}

		incident, err = s.incidents.CreateForEscalation(txCtx, EscalationParams{
			Title:      title,
			Severity:   alert.Severity,
			AlertID:    alert.ID,
			AssigneeID: alert.AssigneeID,
			TeamID:     alert.TeamID,
			ActorID:    actorID,
		})
		if err != nil {
			return err
		}

		if alert.Status == domain.AlertStatusNew {
			alert, err = s.repo.UpdateStatus(txCtx, id, domain.AlertStatusInvestigating, nil)
			if err != nil {
				return err
			}
		}

		return s.repo.AppendEvent(txCtx, AppendEventParams{
			AlertID: id,
			Type:    domain.EventTypeEscalated,
			ActorID: actorID,
			Body:    mustJSON(map[string]any{"incident_id": incident.ID, "incident_title": incident.Title}),
		})
	})
	if err != nil {
		return domain.Incident{}, err
	}

	s.publish(domain.ModuleEventIncident, domain.ModuleEventCreated, incident)
	s.publish(domain.ModuleEventAlert, domain.ModuleEventUpdated, alert)

	return incident, nil
}

func (s *Service) ListNotes(ctx context.Context, alertID uuid.UUID) ([]domain.AlertNote, error) {
	return s.repo.ListNotes(ctx, alertID)
}

func (s *Service) AddNote(ctx context.Context, alertID uuid.UUID, payload AddNotePayload, actorID *uuid.UUID) (domain.AlertNote, error) {
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		return domain.AlertNote{}, apperr.Wrap(apperr.Invalid, "note body cannot be empty", ErrValidation)
	}
	return s.repo.AddNote(ctx, alertID, actorID, body)
}

func (s *Service) UpdateNote(ctx context.Context, noteID uuid.UUID, payload UpdateNotePayload, actorID *uuid.UUID) (domain.AlertNote, error) {
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		return domain.AlertNote{}, apperr.Wrap(apperr.Invalid, "note body cannot be empty", ErrValidation)
	}

	if err := s.assertNoteAuthor(ctx, noteID, actorID); err != nil {
		return domain.AlertNote{}, err
	}
	return s.repo.UpdateNote(ctx, noteID, body)
}

func (s *Service) DeleteNote(ctx context.Context, noteID uuid.UUID, actorID *uuid.UUID) error {
	if err := s.assertNoteAuthor(ctx, noteID, actorID); err != nil {
		return err
	}
	return s.repo.DeleteNote(ctx, noteID)
}

// A note is evidence in a post-incident review, so only its author may rewrite
// it. A note whose author was deleted becomes permanently read-only.
func (s *Service) assertNoteAuthor(ctx context.Context, noteID uuid.UUID, actorID *uuid.UUID) error {
	note, err := s.repo.GetNote(ctx, noteID)
	if err != nil {
		return err
	}
	if note.AuthorID == nil || actorID == nil || *note.AuthorID != *actorID {
		return ErrNotNoteAuthor
	}
	return nil
}

// Fire-and-log: the write already committed, so failing the request here would
// report nothing happened when something did.
func (s *Service) publish(module, event string, entity any) {
	if err := s.events.Publish(module, event, entity); err != nil {
		s.logger.Errorw("failed to publish module event",
			"module", module, "event", event, "err", err)
	}
}

func (s *Service) toUpsertParams(payload CreateAlertPayload) (UpsertParams, error) {
	assigneeID, err := parseOptionalUUID(payload.AssigneeID, "assignee_id")
	if err != nil {
		return UpsertParams{}, err
	}
	teamID, err := parseOptionalUUID(payload.TeamID, "team_id")
	if err != nil {
		return UpsertParams{}, err
	}

	body := payload.Payload
	if len(body) == 0 {
		body = json.RawMessage(`{}`)
	}

	tags := payload.Tags
	if tags == nil {
		tags = []string{}
	}

	sourceKind := domain.SourceKind(payload.SourceKind)

	return UpsertParams{
		Title:       payload.Title,
		Severity:    domain.AlertSeverity(payload.Severity),
		SourceKind:  sourceKind,
		Reporter:    payload.Reporter,
		AssigneeID:  assigneeID,
		TeamID:      teamID,
		Payload:     body,
		Tags:        tags,
		Fingerprint: Fingerprint(sourceKind, payload.Reporter, payload.Title, body),
		CreatedAt:   payload.CreatedAt,
	}, nil
}

var primaryEntityPaths = [][]string{
	{"host", "hostname"},
	{"host", "name"},
	{"hostname"},
	{"user", "name"},
	{"source", "ip"},
	{"src_ip"},
}

// Excludes timestamps and volumes: they differ on every occurrence, which would
// defeat deduplication. Includes the primary entity, without which one rule
// firing on 200 hosts would collapse into a single row.
func Fingerprint(kind domain.SourceKind, reporter *string, title string, payload json.RawMessage) string {
	rep := ""
	if reporter != nil {
		rep = *reporter
	}

	parts := []string{string(kind), rep, title, primaryEntity(payload)}

	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func primaryEntity(payload json.RawMessage) string {
	if len(payload) == 0 {
		return ""
	}

	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return ""
	}

	for _, path := range primaryEntityPaths {
		if v, ok := lookupPath(doc, path); ok {
			return v
		}
	}
	return ""
}

func lookupPath(doc map[string]any, path []string) (string, bool) {
	var current any = doc
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = obj[key]
		if !ok {
			return "", false
		}
	}

	switch v := current.(type) {
	case string:
		if v == "" {
			return "", false
		}
		return v, true
	case float64:
		return fmt.Sprint(v), true
	default:
		return "", false
	}
}

func toUpdateParams(payload UpdateAlertPayload) (UpdateParams, error) {
	params := UpdateParams{Tags: payload.Tags}

	if payload.Severity.Set {
		if payload.Severity.Value == nil {
			return UpdateParams{}, apperr.Wrap(apperr.Invalid, "severity cannot be null", ErrValidation)
		}
		if !domain.IsValidAlertSeverity(*payload.Severity.Value) {
			return UpdateParams{}, apperr.Wrap(apperr.Invalid, "unknown severity", ErrValidation)
		}
		sev := domain.AlertSeverity(*payload.Severity.Value)
		params.Severity = types.Nullable[domain.AlertSeverity]{Value: &sev, Set: true}
	}

	assignee, err := nullableUUID(payload.AssigneeID, "assignee_id")
	if err != nil {
		return UpdateParams{}, err
	}
	params.AssigneeID = assignee

	team, err := nullableUUID(payload.TeamID, "team_id")
	if err != nil {
		return UpdateParams{}, err
	}
	params.TeamID = team

	return params, nil
}

// Unset leaves the column alone; an explicit null clears it.
func nullableUUID(in types.Nullable[string], field string) (types.Nullable[uuid.UUID], error) {
	if !in.Set {
		return types.Nullable[uuid.UUID]{}, nil
	}
	if in.Value == nil {
		return types.Nullable[uuid.UUID]{Set: true}, nil
	}

	id, err := uuid.Parse(*in.Value)
	if err != nil {
		return types.Nullable[uuid.UUID]{}, apperr.Wrap(apperr.Invalid, field+" must be a uuid", ErrValidation)
	}
	return types.Nullable[uuid.UUID]{Value: &id, Set: true}, nil
}

func parseOptionalUUID(in *string, field string) (*uuid.UUID, error) {
	if in == nil || *in == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*in)
	if err != nil {
		return nil, apperr.Wrap(apperr.Invalid, field+" must be a uuid", ErrValidation)
	}
	return &id, nil
}

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

// appendUpdateEvent records a field-level edit on the timeline. It returns
// without writing when nothing actually changed, so a no-op PATCH leaves the
// audit log alone.
func (s *Service) appendUpdateEvent(ctx context.Context, id uuid.UUID, before, after domain.Alert, actorID *uuid.UUID) error {
	changes := domain.DiffAssignable(
		before.Severity, after.Severity,
		before.AssigneeID, after.AssigneeID,
		before.TeamID, after.TeamID,
		before.Tags, after.Tags,
	)
	if len(changes) == 0 {
		return nil
	}

	s.labelAssignees(ctx, changes)

	return s.repo.AppendEvent(ctx, AppendEventParams{
		AlertID: id,
		Type:    domain.EventTypeUpdated,
		ActorID: actorID,
		Body:    mustJSON(map[string]any{"changes": changes}),
	})
}

// A missing directory is not worth failing an edit over: the event still records
// the change, just with uuids instead of names.
func (s *Service) labelAssignees(ctx context.Context, changes []domain.FieldChange) {
	if s.users == nil {
		return
	}

	ids := make([]uuid.UUID, 0, 2)
	for _, c := range changes {
		if c.Field != "assignee_id" {
			continue
		}
		if from, ok := c.From.(uuid.UUID); ok {
			ids = append(ids, from)
		}
		if to, ok := c.To.(uuid.UUID); ok {
			ids = append(ids, to)
		}
	}
	if len(ids) == 0 {
		return
	}

	names, err := s.users.UsernamesByIDs(ctx, ids)
	if err != nil {
		s.logger.Warnw("could not resolve usernames for update event", "error", err)
		return
	}
	for i := range changes {
		if changes[i].Field != "assignee_id" {
			continue
		}
		if from, ok := changes[i].From.(uuid.UUID); ok {
			if name, found := names[from]; found {
				changes[i].FromUsername = &name
			}
		}
		if to, ok := changes[i].To.(uuid.UUID); ok {
			if name, found := names[to]; found {
				changes[i].ToUsername = &name
			}
		}
	}
}
