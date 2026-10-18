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

// grantProbeStatus is outside the range of any real handler, so a route that reaches the
// probe is clear.
const grantProbeStatus = 599

// recordingPermission replaces the real middleware and reports which grant a route asked
// for, without running the handler.
func recordingPermission() middleware.PermissionMiddleware {
	return func(module, action string) gin.HandlerFunc {
		grant := fmt.Sprintf("%s.%s", module, action)
		return func(c *gin.Context) {
			c.Header("X-Required-Grant", grant)
			c.AbortWithStatus(grantProbeStatus)
		}
	}
}

// grantFor sends a request through the registered routes and returns the grant that the
// route asked for. Handlers are never called, so zero-value handler structs are enough.
func grantFor(t *testing.T, register func(*gin.RouterGroup, middleware.PermissionMiddleware), method, path string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	register(router.Group("/api/v1"), recordingPermission())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(method, path, nil))

	if rec.Code == http.StatusNotFound {
		t.Fatalf("no route registered for %s %s", method, path)
	}
	if rec.Code != grantProbeStatus {
		t.Fatalf("%s %s did not pass through a permission guard (status %d) - route is ungated",
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
		{http.MethodGet, "/api/v1/playbooks", "playbooks.read"},
		{http.MethodGet, "/api/v1/playbooks/history", "playbooks.read"},
		{http.MethodGet, "/api/v1/playbooks/history/abc/tasks", "playbooks.read"},
		{http.MethodGet, "/api/v1/playbooks/abc", "playbooks.read"},
		{http.MethodGet, "/api/v1/playbooks/abc/tasks", "playbooks.read"},
		{http.MethodPost, "/api/v1/playbooks", "playbooks.create"},
		{http.MethodPut, "/api/v1/playbooks/abc", "playbooks.update"},
		{http.MethodPut, "/api/v1/playbooks/tasks/abc", "playbooks.update"},
		// executing is not covered by update on purpose
		{http.MethodPost, "/api/v1/playbooks/trigger/abc", "playbooks.execute"},
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
		{http.MethodGet, "/api/v1/connectors", "connectors.read"},
		{http.MethodGet, "/api/v1/connectors/abc", "connectors.read"},
		{http.MethodPost, "/api/v1/connectors", "connectors.create"},
		{http.MethodDelete, "/api/v1/connectors/abc", "connectors.delete"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}

// Everything under settings controls who can do what, so a gap here is a privilege
// escalation hole, not a cosmetic problem.
func TestAdminRouteGrants(t *testing.T) {
	h := &AdminHandler{}
	register := func(g *gin.RouterGroup, p middleware.PermissionMiddleware) {
		h.RegisterRoutes(g, p)
	}

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/api/v1/users", "settings.read"},
		{http.MethodGet, "/api/v1/users/abc", "settings.read"},
		{http.MethodPost, "/api/v1/users", "settings.create"},
		{http.MethodPut, "/api/v1/users/abc", "settings.update"},
		{http.MethodPut, "/api/v1/users/abc/roles", "settings.update"},
		{http.MethodPut, "/api/v1/users/abc/password", "settings.update"},
		{http.MethodPost, "/api/v1/users/abc/invite", "settings.update"},
		{http.MethodDelete, "/api/v1/users/abc", "settings.delete"},

		{http.MethodGet, "/api/v1/roles", "settings.read"},
		{http.MethodGet, "/api/v1/roles/abc", "settings.read"},
		{http.MethodPost, "/api/v1/roles", "settings.create"},
		{http.MethodPut, "/api/v1/roles/abc", "settings.update"},
		{http.MethodPut, "/api/v1/roles/abc/permissions", "settings.update"},
		{http.MethodDelete, "/api/v1/roles/abc", "settings.delete"},

		{http.MethodGet, "/api/v1/teams", "settings.read"},
		{http.MethodGet, "/api/v1/teams/abc", "settings.read"},
		{http.MethodPost, "/api/v1/teams", "settings.create"},
		{http.MethodPut, "/api/v1/teams/abc", "settings.update"},
		{http.MethodPut, "/api/v1/teams/abc/members", "settings.update"},
		{http.MethodPut, "/api/v1/teams/abc/roles", "settings.update"},
		{http.MethodDelete, "/api/v1/teams/abc", "settings.delete"},

		{http.MethodGet, "/api/v1/audit", "settings.read"},

		{http.MethodGet, "/api/v1/api-keys", "settings.read"},
		{http.MethodPost, "/api/v1/api-keys", "settings.create"},
		{http.MethodDelete, "/api/v1/api-keys/abc", "settings.delete"},

		{http.MethodGet, "/api/v1/auth-providers", "settings.read"},
		{http.MethodPost, "/api/v1/auth-providers", "settings.create"},
		{http.MethodPut, "/api/v1/auth-providers/abc", "settings.update"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, grantFor(t, register, tc.method, tc.path))
		})
	}
}

// SameSite=Lax protects only non-GET verbs, so a mutation on GET would be reachable
// cross-site. This test checks the rule, so review does not have to.
func TestNoMutationIsExposedOverGET(t *testing.T) {
	registrars := []func(*gin.RouterGroup, middleware.PermissionMiddleware){
		(&PlaybookHandler{}).RegisterRoutes,
		(&ConnectorHandler{}).RegisterRoutes,
		(&AdminHandler{}).RegisterRoutes,
	}

	gin.SetMode(gin.TestMode)
	passthrough := func(module, action string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}

	router := gin.New()
	for _, register := range registrars {
		register(router.Group("/api/v1"), passthrough)
	}

	for _, route := range router.Routes() {
		if route.Method != http.MethodGet {
			continue
		}
		for _, verb := range []string{"create", "update", "delete", "password", "trigger"} {
			assert.NotContains(t, route.Path, verb,
				"GET %s looks like a mutation; SameSite=Lax does not protect GET", route.Path)
		}
	}
}

// Every grant that a route asks for must exist in the domain list. The columns are TEXT, so
// a typo like "playbook.read" does not fail at the database, it grants nothing and returns
// 403 forever.
func TestRouteGrantsUseKnownVocabulary(t *testing.T) {
	registrars := []func(*gin.RouterGroup, middleware.PermissionMiddleware){
		(&PlaybookHandler{}).RegisterRoutes,
		(&ConnectorHandler{}).RegisterRoutes,
		(&AdminHandler{}).RegisterRoutes,
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
		register(gin.New().Group("/api/v1"), recorder)
	}

	assert.NotEmpty(t, seen, "no grants were declared - the registrars did not run")
}
