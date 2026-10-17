package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
)

func TestAlertRouteGrants(t *testing.T) {
	h := &AlertHandler{}
	register := func(g *gin.RouterGroup, p middleware.PermissionMiddleware) {
		h.RegisterRoutes(g, p)
	}

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/api/v1/alerts", "alerts.read"},
		{http.MethodGet, "/api/v1/alerts/summary", "alerts.read"},
		{http.MethodGet, "/api/v1/alerts/abc", "alerts.read"},
		{http.MethodGet, "/api/v1/alerts/abc/notes", "alerts.read"},

		{http.MethodPost, "/api/v1/alerts", "alerts.create"},
		{http.MethodPost, "/api/v1/alerts/batch", "alerts.create"},

		// Running automation is its own grant: alerts:update is not enough to run a playbook, and
		// neither is playbooks:update.
		{http.MethodPost, "/api/v1/alerts/run", "alerts.execute"},

		{http.MethodPatch, "/api/v1/alerts/abc", "alerts.update"},
		{http.MethodPatch, "/api/v1/alerts/abc/status", "alerts.update"},
		{http.MethodPost, "/api/v1/alerts/abc/notes", "alerts.update"},
		{http.MethodPatch, "/api/v1/alerts/abc/notes/def", "alerts.update"},
		{http.MethodDelete, "/api/v1/alerts/abc/notes/def", "alerts.update"},

		// Escalation creates an incident, so alerts:update alone must not reach it.
		{http.MethodPost, "/api/v1/alerts/abc/escalate", "incidents.create"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}

func TestIncidentRouteGrants(t *testing.T) {
	h := &IncidentHandler{}
	register := func(g *gin.RouterGroup, p middleware.PermissionMiddleware) {
		h.RegisterRoutes(g, p)
	}

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/api/v1/incidents", "incidents.read"},
		{http.MethodGet, "/api/v1/incidents/summary", "incidents.read"},
		{http.MethodGet, "/api/v1/incidents/abc", "incidents.read"},
		{http.MethodGet, "/api/v1/incidents/abc/notes", "incidents.read"},

		{http.MethodPost, "/api/v1/incidents", "incidents.create"},
		{http.MethodPost, "/api/v1/incidents/run", "incidents.execute"},

		{http.MethodPatch, "/api/v1/incidents/abc", "incidents.update"},
		{http.MethodPatch, "/api/v1/incidents/abc/status", "incidents.update"},
		{http.MethodPost, "/api/v1/incidents/abc/notes", "incidents.update"},
		{http.MethodPatch, "/api/v1/incidents/abc/notes/def", "incidents.update"},
		{http.MethodDelete, "/api/v1/incidents/abc/notes/def", "incidents.update"},
		{http.MethodPost, "/api/v1/incidents/abc/alerts", "incidents.update"},
		{http.MethodDelete, "/api/v1/incidents/abc/alerts/def", "incidents.update"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}
