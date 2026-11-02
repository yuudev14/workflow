package goconnectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/ytsoar_mock.go -package=mocks . AlertService,IncidentService

type AlertService interface {
	Create(ctx context.Context, payload alerts.CreateAlertPayload, actorID *uuid.UUID) (domain.Alert, error)
	Update(ctx context.Context, id uuid.UUID, payload alerts.UpdateAlertPayload, actorID *uuid.UUID) (domain.Alert, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, payload alerts.UpdateAlertStatusPayload, actorID *uuid.UUID) (domain.Alert, error)
	Delete(ctx context.Context, id uuid.UUID, actorID *uuid.UUID) error
	AddNote(ctx context.Context, alertID uuid.UUID, payload alerts.AddNotePayload, actorID *uuid.UUID) (domain.AlertNote, error)
	Escalate(ctx context.Context, id uuid.UUID, payload alerts.EscalateAlertPayload, actorID *uuid.UUID) (domain.Incident, error)
}

type IncidentService interface {
	Create(ctx context.Context, payload incidents.CreateIncidentPayload, actorID *uuid.UUID) (domain.Incident, error)
	Update(ctx context.Context, id uuid.UUID, payload incidents.UpdateIncidentPayload, actorID *uuid.UUID) (domain.Incident, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, payload incidents.UpdateIncidentStatusPayload, actorID *uuid.UUID) (domain.Incident, error)
	Delete(ctx context.Context, id uuid.UUID, actorID *uuid.UUID) error
	AddNote(ctx context.Context, incidentID uuid.UUID, payload incidents.AddNotePayload, actorID *uuid.UUID) (domain.IncidentNote, error)
	LinkAlert(ctx context.Context, incidentID uuid.UUID, payload incidents.LinkAlertPayload, actorID *uuid.UUID) error
}

// YTSoarConnector changes alerts and incidents from a playbook step. It calls the services
// in the worker, so it needs no API key. The actor is nil, and the runtime labels it.
type YTSoarConnector struct {
	alerts    AlertService
	incidents IncidentService
}

func NewYTSoarConnector(alertService AlertService, incidentService IncidentService) *YTSoarConnector {
	return &YTSoarConnector{alerts: alertService, incidents: incidentService}
}

// the fields that a PATCH changes; a step that sets none of them skips the PATCH
var patchFields = []string{"severity", "assignee_id", "team_id", "tags"}

func (c *YTSoarConnector) Execute(ctx context.Context, _ map[string]any, params map[string]any, operation string) (any, error) {
	params = cleanParams(params)
	switch operation {
	case "create_alert":
		var payload alerts.CreateAlertPayload
		if err := decode(params, &payload); err != nil {
			return nil, err
		}
		return c.alerts.Create(ctx, payload, nil)
	case "update_alert":
		id, err := idParam(params, "alert_id")
		if err != nil {
			return nil, err
		}
		var alert domain.Alert
		if hasAny(params, patchFields) {
			var payload alerts.UpdateAlertPayload
			if err := decode(params, &payload); err != nil {
				return nil, err
			}
			if alert, err = c.alerts.Update(ctx, id, payload, nil); err != nil {
				return nil, err
			}
		}
		if hasAny(params, []string{"status"}) {
			var payload alerts.UpdateAlertStatusPayload
			if err := decode(params, &payload); err != nil {
				return nil, err
			}
			return c.alerts.UpdateStatus(ctx, id, payload, nil)
		}
		return alert, nil
	case "delete_alert":
		id, err := idParam(params, "alert_id")
		if err != nil {
			return nil, err
		}
		return map[string]any{"deleted": true}, c.alerts.Delete(ctx, id, nil)
	case "add_alert_note":
		id, err := idParam(params, "alert_id")
		if err != nil {
			return nil, err
		}
		var payload alerts.AddNotePayload
		if err := decode(params, &payload); err != nil {
			return nil, err
		}
		return c.alerts.AddNote(ctx, id, payload, nil)
	case "escalate_alert":
		id, err := idParam(params, "alert_id")
		if err != nil {
			return nil, err
		}
		var payload alerts.EscalateAlertPayload
		if err := decode(params, &payload); err != nil {
			return nil, err
		}
		return c.alerts.Escalate(ctx, id, payload, nil)
	case "create_incident":
		var payload incidents.CreateIncidentPayload
		if err := decode(params, &payload); err != nil {
			return nil, err
		}
		return c.incidents.Create(ctx, payload, nil)
	case "update_incident":
		id, err := idParam(params, "incident_id")
		if err != nil {
			return nil, err
		}
		var incident domain.Incident
		if hasAny(params, patchFields) {
			var payload incidents.UpdateIncidentPayload
			if err := decode(params, &payload); err != nil {
				return nil, err
			}
			if incident, err = c.incidents.Update(ctx, id, payload, nil); err != nil {
				return nil, err
			}
		}
		if hasAny(params, []string{"status"}) {
			var payload incidents.UpdateIncidentStatusPayload
			if err := decode(params, &payload); err != nil {
				return nil, err
			}
			return c.incidents.UpdateStatus(ctx, id, payload, nil)
		}
		return incident, nil
	case "delete_incident":
		id, err := idParam(params, "incident_id")
		if err != nil {
			return nil, err
		}
		return map[string]any{"deleted": true}, c.incidents.Delete(ctx, id, nil)
	case "add_incident_note":
		id, err := idParam(params, "incident_id")
		if err != nil {
			return nil, err
		}
		var payload incidents.AddNotePayload
		if err := decode(params, &payload); err != nil {
			return nil, err
		}
		return c.incidents.AddNote(ctx, id, payload, nil)
	case "link_alert":
		id, err := idParam(params, "incident_id")
		if err != nil {
			return nil, err
		}
		var payload incidents.LinkAlertPayload
		if err := decode(params, &payload); err != nil {
			return nil, err
		}
		return map[string]any{"linked": true}, c.incidents.LinkAlert(ctx, id, payload, nil)
	default:
		return nil, fmt.Errorf("ytsoar does not support operation %q", operation)
	}
}

// cleanParams turns the text fields of the editor into API values: an empty field is not
// set, a list field is split on commas, and the payload field is a JSON object.
func cleanParams(params map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range params {
		text, isText := value.(string)
		if isText {
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			value = text
		}
		switch {
		case isText && (key == "tags" || key == "alert_ids"):
			list := []string{}
			for _, item := range strings.Split(text, ",") {
				if item = strings.TrimSpace(item); item != "" {
					list = append(list, item)
				}
			}
			value = list
		case isText && key == "payload" && json.Valid([]byte(text)):
			value = json.RawMessage(text)
		}
		out[key] = value
	}
	return out
}

// decode applies the same binding rules as the API routes.
func decode(params map[string]any, out any) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}
	if err := binding.Validator.ValidateStruct(out); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}
	return nil
}

func idParam(params map[string]any, name string) (uuid.UUID, error) {
	text, _ := params[name].(string)
	id, err := uuid.Parse(text)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s must be a uuid, got %q", name, text)
	}
	return id, nil
}

func hasAny(params map[string]any, keys []string) bool {
	for _, key := range keys {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}
