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

	t.Run("create", func(t *testing.T) {
		c.expectAll(t, []request{
			{"no name", http.MethodPost, base, J{"type": "oidc"}, http.StatusBadRequest},
			{"name in use", http.MethodPost, base, J{"type": "oidc", "name": name, "config": config}, http.StatusConflict},
		})
	})

	t.Run("read", func(t *testing.T) {
		assert.Contains(t, ids(c.get(t, base)), id)
		// An enabled provider is on the login page.
		assert.Contains(t, ids(anonymous(t).get(t, "/api/auth/v1/providers")), id)
	})

	t.Run("update", func(t *testing.T) {
		c.expectAll(t, []request{
			{"disable", http.MethodPut, base + "/" + id, J{"enabled": false}, http.StatusOK},
			{"unknown provider", http.MethodPut, base + "/" + uuid.NewString(), J{"name": "x"}, http.StatusNotFound},
		})
		assert.NotContains(t, ids(anonymous(t).get(t, "/api/auth/v1/providers")), id)
	})
}
