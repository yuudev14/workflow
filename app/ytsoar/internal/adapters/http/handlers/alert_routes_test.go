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
		{http.MethodGet, "/api/alerts/v1", "alerts.read"},
		{http.MethodGet, "/api/alerts/v1/summary", "alerts.read"},
		{http.MethodGet, "/api/alerts/v1/abc", "alerts.read"},
		{http.MethodGet, "/api/alerts/v1/abc/notes", "alerts.read"},

		{http.MethodPost, "/api/alerts/v1", "alerts.create"},
		{http.MethodPost, "/api/alerts/v1/batch", "alerts.create"},

		{http.MethodPatch, "/api/alerts/v1/abc", "alerts.update"},
		{http.MethodPatch, "/api/alerts/v1/abc/status", "alerts.update"},
		{http.MethodPost, "/api/alerts/v1/abc/notes", "alerts.update"},
		{http.MethodPatch, "/api/alerts/v1/abc/notes/def", "alerts.update"},
		{http.MethodDelete, "/api/alerts/v1/abc/notes/def", "alerts.update"},

		// Escalation creates an incident, so it must not be reachable with only
		// alerts:update.
		{http.MethodPost, "/api/alerts/v1/abc/escalate", "incidents.create"},
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
		{http.MethodGet, "/api/incidents/v1", "incidents.read"},
		{http.MethodGet, "/api/incidents/v1/summary", "incidents.read"},
		{http.MethodGet, "/api/incidents/v1/abc", "incidents.read"},
		{http.MethodGet, "/api/incidents/v1/abc/notes", "incidents.read"},

		{http.MethodPost, "/api/incidents/v1", "incidents.create"},

		{http.MethodPatch, "/api/incidents/v1/abc", "incidents.update"},
		{http.MethodPatch, "/api/incidents/v1/abc/status", "incidents.update"},
		{http.MethodPost, "/api/incidents/v1/abc/notes", "incidents.update"},
		{http.MethodPatch, "/api/incidents/v1/abc/notes/def", "incidents.update"},
		{http.MethodDelete, "/api/incidents/v1/abc/notes/def", "incidents.update"},
		{http.MethodPost, "/api/incidents/v1/abc/alerts", "incidents.update"},
		{http.MethodDelete, "/api/incidents/v1/abc/alerts/def", "incidents.update"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}
