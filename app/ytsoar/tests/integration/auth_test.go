package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	t.Run("POST /login", func(t *testing.T) {
		anonymous(t).expectAll(t, []request{
			{"returns 400 when the password is missing", http.MethodPost, "/api/auth/v1/login", J{"username": adminUsername}, http.StatusBadRequest},
			{"returns 401 when the password is wrong", http.MethodPost, "/api/auth/v1/login", J{"username": adminUsername, "password": "wrong"}, http.StatusUnauthorized},
			{"returns 401 when the user does not exist", http.MethodPost, "/api/auth/v1/login", J{"username": unique("nobody"), "password": "password123"}, http.StatusUnauthorized},
			{"signs in with the correct password", http.MethodPost, "/api/auth/v1/login", J{"username": adminUsername, "password": adminPassword}, http.StatusOK},
		})
	})

	t.Run("GET /me", func(t *testing.T) {
		t.Run("returns the user and the roles", func(t *testing.T) {
			me := admin(t).get(t, "/api/auth/v1/me").(J)
			assert.Equal(t, adminUsername, me["user"].(J)["username"])
			assert.Contains(t, me["roles"], "admin")
		})
	})

	t.Run("POST /refresh", func(t *testing.T) {
		t.Run("returns 401 without a refresh cookie", func(t *testing.T) {
			anonymous(t).expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)
		})

		t.Run("gives a new access token until the user signs out", func(t *testing.T) {
			c := admin(t)
			refreshed := c.expect(t, http.StatusOK, http.MethodPost, "/api/auth/v1/refresh", nil).(J)
			assert.NotEmpty(t, refreshed["access_token"])
			c.expect(t, http.StatusOK, http.MethodPost, "/api/auth/v1/logout", nil)
			c.expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)
		})
	})

	t.Run("OIDC", func(t *testing.T) {
		c := anonymous(t)
		t.Run("returns 400 when the provider id is not a uuid", func(t *testing.T) {
			c.expect(t, http.StatusBadRequest, http.MethodGet, "/api/auth/v1/oidc/not-a-uuid/start", nil)
		})

		t.Run("does not sign in with a disabled provider", func(t *testing.T) {
			provider := admin(t).create(t, "/api/auth-providers/v1", J{"type": "oidc", "name": unique("sso"), "enabled": false, "config": J{
				"issuer": "http://localhost:8180/realms/test", "client_id": "ytsoar", "client_secret": "secret",
			}})
			id := provider["id"].(string)
			assert.NotContains(t, ids(c.get(t, "/api/auth/v1/providers")), id, "the login page shows it")
			// The start and a callback without the signed state cookie go back to the login page.
			for _, step := range []string{"/start", "/callback?code=x&state=y"} {
				res, err := c.http.Get(serverURL + "/api/auth/v1/oidc/" + id + step)
				require.NoError(t, err)
				res.Body.Close()
				assert.Contains(t, res.Header.Get("Location"), "/login?error=", step)
			}
		})
	})
}
