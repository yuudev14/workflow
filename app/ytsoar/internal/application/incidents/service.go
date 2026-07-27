package incidents

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

type Service struct {
	logger    logger.Logger
	repo      IncidentRepository
	alerts    AlertTimeline
	txManager contracts.TxManager
	events    contracts.ModuleEventPublisher
	users     contracts.UserDirectory
}

func NewService(
	log logger.Logger,
	repo IncidentRepository,
	alertTimeline AlertTimeline,
	txManager contracts.TxManager,
	events contracts.ModuleEventPublisher,
	users contracts.UserDirectory,
) *Service {
	return &Service{
		logger:    log,
		repo:      repo,
		alerts:    alertTimeline,
		txManager: txManager,
		events:    events,
		users:     users,
	}
}

func (s *Service) List(ctx context.Context, filter IncidentFilter) (types.CursorPage[IncidentListItem], error) {
	filter, err := filter.Normalized()
	if err != nil {
		return types.CursorPage[IncidentListItem]{}, err
	}

	items, next, err := s.repo.List(ctx, filter)
	if err != nil {
		return types.CursorPage[IncidentListItem]{}, err
	}

	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return types.CursorPage[IncidentListItem]{}, err
	}

	return types.CursorPage[IncidentListItem]{Entries: items, Total: total, NextCursor: next}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (IncidentDetail, error) {
	return s.repo.GetDetail(ctx, id)
}

func (s *Service) Summary(ctx context.Context, rng types.ResolvedRange) (IncidentsSummary, error) {
	return s.repo.Summary(ctx, rng)
}

func (s *Service) Create(ctx context.Context, payload CreateIncidentPayload, actorID *uuid.UUID) (domain.Incident, error) {
	params, err := toCreateParams(payload)
	if err != nil {
		return domain.Incident{}, err
	}

	alertIDs, err := parseUUIDs(payload.AlertIDs, "alert_ids")
	if err != nil {
		return domain.Incident{}, err
	}

	var incident domain.Incident
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		incident, err = s.repo.Create(txCtx, params)
		if err != nil {
			return err
		}

		if err = s.repo.AppendEvent(txCtx, AppendEventParams{
			IncidentID: incident.ID,
			Type:       domain.EventTypeCreated,
			ActorID:    actorID,
			Body:       mustJSON(map[string]any{"severity": incident.Severity, "status": incident.Status}),
		}); err != nil {
			return err
		}

		for _, alertID := range alertIDs {
			if err = s.link(txCtx, incident, alertID, domain.LinkSourceManual, actorID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return domain.Incident{}, err
	}

	s.publish(domain.ModuleEventIncident, domain.ModuleEventCreated, incident)
	return incident, nil
}

// CreateForEscalation runs inside the alert service's transaction, so it must
// not publish or open one of its own - the caller announces both entities once
// the whole escalation has committed.
func (s *Service) CreateForEscalation(ctx context.Context, params alerts.EscalationParams) (domain.Incident, error) {
	incident, err := s.repo.Create(ctx, CreateParams{
		Title:      params.Title,
		Severity:   params.Severity,
		Status:     domain.IncidentStatusOpen,
		AssigneeID: params.AssigneeID,
		TeamID:     params.TeamID,
		Tags:       []string{},
	})
	if err != nil {
		return domain.Incident{}, err
	}

	if err = s.repo.AppendEvent(ctx, AppendEventParams{
		IncidentID: incident.ID,
		Type:       domain.EventTypeCreated,
		ActorID:    params.ActorID,
		Body:       mustJSON(map[string]any{"escalated_from_alert_id": params.AlertID}),
	}); err != nil {
		return domain.Incident{}, err
	}

	if _, err = s.repo.LinkAlert(ctx, incident.ID, params.AlertID, domain.LinkSourceEscalate); err != nil {
		return domain.Incident{}, err
	}

	if err = s.repo.AppendEvent(ctx, AppendEventParams{
		IncidentID: incident.ID,
		Type:       domain.EventTypeLinked,
		ActorID:    params.ActorID,
		Body:       mustJSON(map[string]any{"alert_id": params.AlertID, "source": domain.LinkSourceEscalate}),
	}); err != nil {
		return domain.Incident{}, err
	}

	return incident, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, payload UpdateIncidentPayload, actorID *uuid.UUID) (domain.Incident, error) {
	params, err := toUpdateParams(payload)
	if err != nil {
		return domain.Incident{}, err
	}

	var incident domain.Incident
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		before, err := s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		incident, err = s.repo.Update(txCtx, id, params)
		if err != nil {
			return err
		}

		return s.appendUpdateEvent(txCtx, id, before, incident, actorID)
	})
	if err != nil {
		return domain.Incident{}, err
	}

	s.publish(domain.ModuleEventIncident, domain.ModuleEventUpdated, incident)
	return incident, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, payload UpdateIncidentStatusPayload, actorID *uuid.UUID) (domain.Incident, error) {
	next := domain.IncidentStatus(payload.Status)
	if !domain.IsValidIncidentStatus(payload.Status) {
		return domain.Incident{}, apperr.Wrap(apperr.Invalid, "unknown incident status", ErrValidation)
	}

	var incident domain.Incident
	err := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		current, err := s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		if !current.Status.CanTransitionTo(next) {
			return apperr.Wrap(apperr.Invalid,
				"cannot move incident from "+string(current.Status)+" to "+string(next), ErrValidation)
		}

		incident, err = s.repo.UpdateStatus(txCtx, id, next)
		if err != nil {
			return err
		}

		err = s.repo.AppendEvent(txCtx, AppendEventParams{
			IncidentID: id,
			Type:       domain.EventTypeStatusChanged,
			ActorID:    actorID,
			Body:       mustJSON(map[string]any{"from": current.Status, "to": next}),
		})
		return err
	})
	if err != nil {
		return domain.Incident{}, err
	}

	s.publish(domain.ModuleEventIncident, domain.ModuleEventUpdated, incident)
	return incident, nil
}

func (s *Service) ListNotes(ctx context.Context, incidentID uuid.UUID) ([]domain.IncidentNote, error) {
	return s.repo.ListNotes(ctx, incidentID)
}

func (s *Service) AddNote(ctx context.Context, incidentID uuid.UUID, payload AddNotePayload, actorID *uuid.UUID) (domain.IncidentNote, error) {
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		return domain.IncidentNote{}, apperr.Wrap(apperr.Invalid, "note body cannot be empty", ErrValidation)
	}
	return s.repo.AddNote(ctx, incidentID, actorID, body)
}

func (s *Service) UpdateNote(ctx context.Context, noteID uuid.UUID, payload UpdateNotePayload, actorID *uuid.UUID) (domain.IncidentNote, error) {
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		return domain.IncidentNote{}, apperr.Wrap(apperr.Invalid, "note body cannot be empty", ErrValidation)
	}

	if err := s.assertNoteAuthor(ctx, noteID, actorID); err != nil {
		return domain.IncidentNote{}, err
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

func (s *Service) LinkAlert(ctx context.Context, incidentID uuid.UUID, payload LinkAlertPayload, actorID *uuid.UUID) error {
	alertID, err := uuid.Parse(payload.AlertID)
	if err != nil {
		return apperr.Wrap(apperr.Invalid, "alert_id must be a uuid", ErrValidation)
	}

	var incident domain.Incident
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		incident, err = s.repo.GetByID(txCtx, incidentID)
		if err != nil {
			return err
		}
		return s.link(txCtx, incident, alertID, domain.LinkSourceManual, actorID)
	})
	if err != nil {
		return err
	}

	s.publish(domain.ModuleEventIncident, domain.ModuleEventUpdated, incident)
	return nil
}

func (s *Service) UnlinkAlert(ctx context.Context, incidentID, alertID uuid.UUID, actorID *uuid.UUID) error {
	var incident domain.Incident
	err := s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		incident, err = s.repo.GetByID(txCtx, incidentID)
		if err != nil {
			return err
		}

		if err = s.repo.UnlinkAlert(txCtx, incidentID, alertID); err != nil {
			return err
		}

		if err = s.repo.AppendEvent(txCtx, AppendEventParams{
			IncidentID: incidentID,
			Type:       domain.EventTypeUnlinked,
			ActorID:    actorID,
			Body:       mustJSON(map[string]any{"alert_id": alertID}),
		}); err != nil {
			return err
		}

		return s.alerts.AppendAlertEvent(txCtx, alertID, domain.EventTypeUnlinked, actorID,
			mustJSON(map[string]any{"incident_id": incidentID, "incident_title": incident.Title}))
	})
	if err != nil {
		return err
	}

	s.publish(domain.ModuleEventIncident, domain.ModuleEventUpdated, incident)
	return nil
}

// A repeat link is a no-op rather than an error, but it must not append a
// second pair of timeline rows saying it happened twice.
func (s *Service) link(
	ctx context.Context,
	incident domain.Incident,
	alertID uuid.UUID,
	source domain.LinkSource,
	actorID *uuid.UUID,
) error {
	linked, err := s.repo.LinkAlert(ctx, incident.ID, alertID, source)
	if err != nil || !linked {
		return err
	}

	if err = s.repo.AppendEvent(ctx, AppendEventParams{
		IncidentID: incident.ID,
		Type:       domain.EventTypeLinked,
		ActorID:    actorID,
		Body:       mustJSON(map[string]any{"alert_id": alertID, "source": source}),
	}); err != nil {
		return err
	}

	return s.alerts.AppendAlertEvent(ctx, alertID, domain.EventTypeLinked, actorID,
		mustJSON(map[string]any{"incident_id": incident.ID, "incident_title": incident.Title, "source": source}))
}

// Fire-and-log: the write already committed, so failing the request here would
// report nothing happened when something did.
func (s *Service) publish(module, event string, entity any) {
	if err := s.events.Publish(module, event, entity); err != nil {
		s.logger.Errorw("failed to publish module event",
			"module", module, "event", event, "err", err)
	}
}

func toCreateParams(payload CreateIncidentPayload) (CreateParams, error) {
	assigneeID, err := parseOptionalUUID(payload.AssigneeID, "assignee_id")
	if err != nil {
		return CreateParams{}, err
	}
	teamID, err := parseOptionalUUID(payload.TeamID, "team_id")
	if err != nil {
		return CreateParams{}, err
	}

	status := domain.IncidentStatusOpen
	if payload.Status != nil {
		if !domain.IsValidIncidentStatus(*payload.Status) {
			return CreateParams{}, apperr.Wrap(apperr.Invalid, "unknown incident status", ErrValidation)
		}
		status = domain.IncidentStatus(*payload.Status)
	}

	tags := payload.Tags
	if tags == nil {
		tags = []string{}
	}

	return CreateParams{
		Title:      payload.Title,
		Severity:   domain.AlertSeverity(payload.Severity),
		Status:     status,
		AssigneeID: assigneeID,
		TeamID:     teamID,
		Tags:       tags,
	}, nil
}

func toUpdateParams(payload UpdateIncidentPayload) (UpdateParams, error) {
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

func parseUUIDs(in []string, field string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(in))
	for _, raw := range in {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, apperr.Wrap(apperr.Invalid, field+" must contain uuids", ErrValidation)
		}
		out = append(out, id)
	}
	return out, nil
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
func (s *Service) appendUpdateEvent(ctx context.Context, id uuid.UUID, before, after domain.Incident, actorID *uuid.UUID) error {
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
		IncidentID: id,
		Type:       domain.EventTypeUpdated,
		ActorID:    actorID,
		Body:       mustJSON(map[string]any{"changes": changes}),
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
