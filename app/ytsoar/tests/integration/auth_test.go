package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	t.Run("login", func(t *testing.T) {
		anonymous(t).expectAll(t, []request{
			{"no password", http.MethodPost, "/api/auth/v1/login", J{"username": adminUsername}, http.StatusBadRequest},
			{"wrong password", http.MethodPost, "/api/auth/v1/login", J{"username": adminUsername, "password": "wrong"}, http.StatusUnauthorized},
			{"unknown user", http.MethodPost, "/api/auth/v1/login", J{"username": unique("nobody"), "password": "password123"}, http.StatusUnauthorized},
			{"correct password", http.MethodPost, "/api/auth/v1/login", J{"username": adminUsername, "password": adminPassword}, http.StatusOK},
		})
	})

	t.Run("me", func(t *testing.T) {
		me := admin(t).get(t, "/api/auth/v1/me").(J)
		assert.Equal(t, adminUsername, me["user"].(J)["username"])
		assert.Contains(t, me["roles"], "admin")
	})

	t.Run("refresh and logout", func(t *testing.T) {
		anonymous(t).expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)

		// The refresh cookie gives a new access token, until the user signs out.
		c := admin(t)
		refreshed := c.expect(t, http.StatusOK, http.MethodPost, "/api/auth/v1/refresh", nil).(J)
		assert.NotEmpty(t, refreshed["access_token"])
		c.expect(t, http.StatusOK, http.MethodPost, "/api/auth/v1/logout", nil)
		c.expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)
	})

	t.Run("oidc", func(t *testing.T) {
		c := anonymous(t)
		c.get(t, "/api/auth/v1/providers")
		c.expect(t, http.StatusBadRequest, http.MethodGet, "/api/auth/v1/oidc/not-a-uuid/start", nil)

		// A disabled provider is not on the login page. It cannot start a login,
		// and a callback without the signed state cookie does not sign in.
		provider := admin(t).create(t, "/api/auth-providers/v1", J{"type": "oidc", "name": unique("sso"), "enabled": false, "config": J{
			"issuer": "http://localhost:8180/realms/test", "client_id": "ytsoar", "client_secret": "secret",
		}})
		id := provider["id"].(string)
		assert.NotContains(t, ids(c.get(t, "/api/auth/v1/providers")), id)
		for _, step := range []string{"/start", "/callback?code=x&state=y"} {
			res, err := c.http.Get(serverURL + "/api/auth/v1/oidc/" + id + step)
			require.NoError(t, err)
			res.Body.Close()
			assert.Contains(t, res.Header.Get("Location"), "/login?error=", step)
		}
	})
}
