package mcp

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

var errForbidden = errors.New("forbidden")

const (
	defaultLimit = 20
	maxLimit     = 50
	maxRuns      = 10
	maxResultLen = 2000
)

type Authenticator interface {
	VerifyAccessToken(tokenString string) (domain.AuthUser, error)
	VerifyAPIKey(ctx context.Context, key string) (domain.AuthUser, error)
	PermissionsFor(ctx context.Context, userID uuid.UUID) (domain.PermissionSet, error)
}

type Services struct {
	Auth      Authenticator
	Alerts    *alerts.Service
	Incidents *incidents.Service
	Playbooks playbooks.PlaybookService
	Tasks     tasks.TaskService
}

type handler struct {
	log logger.Logger
	Services
}

// userKey is unexported, so only this package can set the signed-in user of a request.
type userKey struct{}

type listIn struct {
	Status   string `json:"status,omitempty" jsonschema_description:"filter by status"`
	Severity string `json:"severity,omitempty" jsonschema_description:"critical, high, medium or low"`
	Query    string `json:"q,omitempty" jsonschema_description:"search text"`
	Limit    int    `json:"limit,omitempty" jsonschema_description:"rows to return, 20 by default, 50 at most"`
}

type idIn struct {
	ID string `json:"id" jsonschema_description:"the uuid"`
}

type listPlaybooksIn struct {
	Name  string `json:"name,omitempty" jsonschema_description:"filter by name"`
	Limit int    `json:"limit,omitempty" jsonschema_description:"rows to return, 20 by default, 50 at most"`
}

type listRunsIn struct {
	PlaybookID string `json:"playbook_id,omitempty" jsonschema_description:"only runs of this playbook"`
	ModuleType string `json:"module_type,omitempty" jsonschema_description:"alert or incident, use it with record_id"`
	RecordID   string `json:"record_id,omitempty" jsonschema_description:"only runs against this alert or incident"`
	Limit      int    `json:"limit,omitempty" jsonschema_description:"rows to return, 20 by default, 50 at most"`
}

// The views embed the service detail and hide its timeline, because a timeline is long and
// the notes and the runs already say what happened.
type alertView struct {
	alerts.AlertDetail
	Timeline []string `json:"timeline,omitempty"`
}

type incidentView struct {
	incidents.IncidentDetail
	Timeline []string `json:"timeline,omitempty"`
}

// J is a JSON object. A tool builds its output with it, so the output holds only the fields an
// LLM needs.
type J = map[string]any
