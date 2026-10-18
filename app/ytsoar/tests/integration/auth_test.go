package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
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

	t.Run("API keys", func(t *testing.T) {
		c := admin(t)
		base := "/api/api-keys/v1"
		userID, _ := newUser(t)
		withKey := func(t *testing.T) (string, *client) {
			created := c.create(t, base, J{"user_id": userID, "name": unique("siem")})
			return created["id"].(string), &client{http: http.DefaultClient, token: created["key"].(string)}
		}

		t.Run("POST /api-keys", func(t *testing.T) {
			c.expectAll(t, []request{
				{"returns 400 when the name is missing", http.MethodPost, base, J{"user_id": userID}, http.StatusBadRequest},
				{"returns 400 when the expiry is in the past", http.MethodPost, base, J{"user_id": userID, "name": "old", "expires_at": "2020-01-01T00:00:00Z"}, http.StatusBadRequest},
				{"returns 404 when the user does not exist", http.MethodPost, base, J{"user_id": uuid.NewString(), "name": "x"}, http.StatusNotFound},
			})
		})

		t.Run("signs in as the owner of the key", func(t *testing.T) {
			_, key := withKey(t)
			me := key.get(t, "/api/auth/v1/me").(J)
			assert.Equal(t, userID, me["user"].(J)["id"])
		})

		t.Run("lists the key without the key itself", func(t *testing.T) {
			id, _ := withKey(t)
			list := c.get(t, base+"?user_id="+userID)
			assert.Contains(t, ids(list), id)
			for _, item := range items(list) {
				assert.NotContains(t, item.(J), "key")
			}
		})

		t.Run("returns 401 after the key is deleted", func(t *testing.T) {
			id, key := withKey(t)
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
			key.expect(t, http.StatusUnauthorized, http.MethodGet, "/api/auth/v1/me", nil)
		})

		t.Run("returns 401 after the owner is deactivated", func(t *testing.T) {
			_, key := withKey(t)
			c.expect(t, http.StatusNoContent, http.MethodDelete, "/api/users/v1/"+userID, nil)
			key.expect(t, http.StatusUnauthorized, http.MethodGet, "/api/auth/v1/me", nil)
		})
	})

	t.Run("invite and password reset", func(t *testing.T) {
		c := admin(t)
		reset, forgot := "/api/auth/v1/password/reset", "/api/auth/v1/password/forgot"
		username := unique("invited")
		email := username + "@ytsoar.test"
		user := c.create(t, "/api/users/v1", J{"username": username, "email": email})
		id := user["id"].(string)
		assert.Nil(t, user["email_verified_at"])
		token := linkFor(t, email)

		t.Run("POST /users/:id/invite", func(t *testing.T) {
			t.Run("returns 409 when the last email is less than a minute old", func(t *testing.T) {
				c.expect(t, http.StatusConflict, http.MethodPost, "/api/users/v1/"+id+"/invite", nil)
			})
		})

		t.Run("POST /login before the invite is used", func(t *testing.T) {
			login := "/api/auth/v1/login"
			anonymous(t).expectAll(t, []request{
				{"returns 400 when the password is empty", http.MethodPost, login, J{"username": username, "password": ""}, http.StatusBadRequest},
				{"returns 401 because the user has no password yet", http.MethodPost, login, J{"username": username, "password": "password123"}, http.StatusUnauthorized},
			})
		})

		t.Run("POST /password/reset", func(t *testing.T) {
			anonymous(t).expectAll(t, []request{
				{"returns 400 when the password is too short", http.MethodPost, reset, J{"token": token, "password": "short"}, http.StatusBadRequest},
				{"sets the password with the invite link", http.MethodPost, reset, J{"token": token, "password": "password123"}, http.StatusNoContent},
				{"returns 400 when the link is used again", http.MethodPost, reset, J{"token": token, "password": "password123"}, http.StatusBadRequest},
			})
			signIn(t, username, "password123")
			assert.NotNil(t, c.get(t, "/api/users/v1/"+id).(J)["email_verified_at"])
		})

		t.Run("POST /password/forgot", func(t *testing.T) {
			t.Run("returns 204 when the email does not exist", func(t *testing.T) {
				anonymous(t).expect(t, http.StatusNoContent, http.MethodPost, forgot, J{"email": unique("nobody") + "@ytsoar.test"})
			})

			t.Run("sends a link that sets a new password and ends the old sessions", func(t *testing.T) {
				old := signIn(t, username, "password123")
				anonymous(t).expect(t, http.StatusNoContent, http.MethodPost, forgot, J{"email": email})
				anonymous(t).expect(t, http.StatusNoContent, http.MethodPost, reset, J{"token": linkFor(t, email), "password": "password456"})
				old.expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)
				signIn(t, username, "password456")
			})
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
