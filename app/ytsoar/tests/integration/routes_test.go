package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests read the routes from the router, so a new route is tested too.
func TestRoutes(t *testing.T) {
	t.Run("returns 401 on every private route without a session", func(t *testing.T) {
		c := anonymous(t)
		for _, r := range routes {
			if public(r.Path) {
				continue
			}
			code, _ := c.do(t, r.Method, fill(r.Path), J{})
			assert.Equal(t, http.StatusUnauthorized, code, "%s %s", r.Method, r.Path)
		}
	})

	t.Run("returns 403 on every route with a grant for a user with no role", func(t *testing.T) {
		_, user := newUser(t)
		// Every user can read their own profile and the list of assignable users.
		// The WebSocket uses the refresh cookie, not the token, so this test skips it.
		open := map[string]bool{"/api/v1/auth/me": true, "/api/v1/users/assignable": true, "/ws/playbook": true}
		for _, r := range routes {
			if public(r.Path) || open[r.Path] {
				continue
			}
			code, _ := user.do(t, r.Method, fill(r.Path), J{})
			assert.Equal(t, http.StatusForbidden, code, "%s %s", r.Method, r.Path)
		}
		user.get(t, "/api/v1/auth/me")
		user.get(t, "/api/v1/users/assignable")
	})
}
