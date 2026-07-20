package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
)

// grantProbeStatus is outside the range any real handler returns, so a route
// reaching the probe is unambiguous.
const grantProbeStatus = 599

// recordingPermission stands in for the real middleware and reports which
// grant a route demanded, without running the handler behind it.
func recordingPermission() middleware.PermissionMiddleware {
	return func(module, action string) gin.HandlerFunc {
		grant := fmt.Sprintf("%s.%s", module, action)
		return func(c *gin.Context) {
			c.Header("X-Required-Grant", grant)
			c.AbortWithStatus(grantProbeStatus)
		}
	}
}

// grantFor plays a request through the registered routes and returns the grant
// the route asked for. The handlers are never invoked, so zero-value handler
// structs are enough.
func grantFor(t *testing.T, register func(*gin.RouterGroup, middleware.PermissionMiddleware), method, path string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	register(router.Group("/api"), recordingPermission())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))

	if rec.Code == http.StatusNotFound {
		t.Fatalf("no route registered for %s %s", method, path)
	}
	if rec.Code != grantProbeStatus {
		t.Fatalf("%s %s did not pass through a permission guard (status %d) — route is ungated",
			method, path, rec.Code)
	}
	return rec.Header().Get("X-Required-Grant")
}

func TestPlaybookRouteGrants(t *testing.T) {
	h := &PlaybookHandler{}
	register := func(g *gin.RouterGroup, p middleware.PermissionMiddleware) {
		h.RegisterRoutes(g, p)
	}

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/api/playbooks/v1", "playbooks.read"},
		{http.MethodGet, "/api/playbooks/v1/history", "playbooks.read"},
		{http.MethodGet, "/api/playbooks/v1/history/abc/tasks", "playbooks.read"},
		{http.MethodGet, "/api/playbooks/v1/abc", "playbooks.read"},
		{http.MethodGet, "/api/playbooks/v1/abc/tasks", "playbooks.read"},
		{http.MethodPost, "/api/playbooks/v1", "playbooks.create"},
		{http.MethodPut, "/api/playbooks/v1/abc", "playbooks.update"},
		{http.MethodPut, "/api/playbooks/v1/tasks/abc", "playbooks.update"},
		// Executing is deliberately not covered by update.
		{http.MethodPost, "/api/playbooks/v1/trigger/abc", "playbooks.execute"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}

func TestConnectorRouteGrants(t *testing.T) {
	h := &ConnectorHandler{}
	register := func(g *gin.RouterGroup, p middleware.PermissionMiddleware) {
		h.RegisterRoutes(g, p)
	}

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/api/connectors/v1", "connectors.read"},
		{http.MethodGet, "/api/connectors/v1/abc", "connectors.read"},
		{http.MethodPost, "/api/connectors/v1", "connectors.create"},
		{http.MethodDelete, "/api/connectors/v1/abc", "connectors.delete"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}

// Every grant a route asks for must exist in the domain vocabulary. The
// columns are TEXT, so a typo like "playbook.read" would not fail at the
// database — it would silently grant nothing and 403 forever.
func TestRouteGrantsUseKnownVocabulary(t *testing.T) {
	registrars := []func(*gin.RouterGroup, middleware.PermissionMiddleware){
		(&PlaybookHandler{}).RegisterRoutes,
		(&ConnectorHandler{}).RegisterRoutes,
	}

	seen := map[string]bool{}
	recorder := func(module, action string) gin.HandlerFunc {
		assert.True(t, domain.IsValidPermissionModule(module), "unknown module %q", module)
		assert.True(t, domain.IsValidPermissionAction(action), "unknown action %q", action)
		seen[module+"."+action] = true
		return func(c *gin.Context) { c.Next() }
	}

	gin.SetMode(gin.TestMode)
	for _, register := range registrars {
		register(gin.New().Group("/api"), recorder)
	}

	assert.NotEmpty(t, seen, "no grants were declared — the registrars did not run")
}
