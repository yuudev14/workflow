package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthProviders(t *testing.T) {
	const base = "/api/auth-providers/v1"
	c := admin(t)
	config := J{"issuer": "http://localhost:8180/realms/test", "client_id": "ytsoar", "client_secret": "secret"}
	name := unique("keycloak")
	id := c.create(t, base, J{"type": "oidc", "name": name, "enabled": true, "config": config})["id"].(string)

	t.Run("POST /auth-providers", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the name is missing", http.MethodPost, base, J{"type": "oidc"}, http.StatusBadRequest},
			{"returns 409 when the name is in use", http.MethodPost, base, J{"type": "oidc", "name": name, "config": config}, http.StatusConflict},
		})
	})

	t.Run("GET /auth-providers", func(t *testing.T) {
		t.Run("lists the provider", func(t *testing.T) {
			assert.Contains(t, ids(c.get(t, base)), id)
		})
		t.Run("shows an enabled provider on the login page", func(t *testing.T) {
			assert.Contains(t, ids(anonymous(t).get(t, "/api/auth/v1/providers")), id)
		})
	})

	t.Run("PUT /auth-providers/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 404 when the provider does not exist", http.MethodPut, base + "/" + uuid.NewString(), J{"name": "x"}, http.StatusNotFound},
			{"disables the provider", http.MethodPut, base + "/" + id, J{"enabled": false}, http.StatusOK},
		})
		t.Run("removes a disabled provider from the login page", func(t *testing.T) {
			assert.NotContains(t, ids(anonymous(t).get(t, "/api/auth/v1/providers")), id)
		})
	})
}
