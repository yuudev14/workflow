package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUsers(t *testing.T) {
	const base = "/api/users/v1"
	c := admin(t)
	username := unique("analyst")
	id := c.create(t, base, J{"username": username, "email": username + "@ytsoar.test", "password": "password123"})["id"].(string)

	t.Run("create", func(t *testing.T) {
		name := unique("u")
		c.expectAll(t, []request{
			{"bad email", http.MethodPost, base, J{"username": name, "email": "bad", "password": "password123"}, http.StatusBadRequest},
			{"short password", http.MethodPost, base, J{"username": name, "email": name + "@ytsoar.test", "password": "short"}, http.StatusBadRequest},
			{"bad role id", http.MethodPost, base, J{"username": name, "email": name + "@ytsoar.test", "password": "password123", "role_ids": []string{"x"}}, http.StatusBadRequest},
			{"username in use", http.MethodPost, base, J{"username": username, "email": name + "@ytsoar.test", "password": "password123"}, http.StatusConflict},
			{"email in use", http.MethodPost, base, J{"username": name, "email": username + "@ytsoar.test", "password": "password123"}, http.StatusConflict},
		})
	})

	t.Run("read", func(t *testing.T) {
		assert.Equal(t, username, c.get(t, base+"/"+id).(J)["username"])
		assert.Equal(t, []any{id}, ids(c.get(t, base+"?search="+username)))
		assert.Contains(t, ids(c.get(t, base+"/assignable")), id)
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
	})

	t.Run("update", func(t *testing.T) {
		adminEmail := c.get(t, "/api/auth/v1/me").(J)["user"].(J)["email"]
		c.expectAll(t, []request{
			{"name", http.MethodPut, base + "/" + id, J{"first_name": "Ana"}, http.StatusOK},
			{"email in use", http.MethodPut, base + "/" + id, J{"email": adminEmail}, http.StatusConflict},
			{"unknown user", http.MethodPut, base + "/" + uuid.NewString(), J{"first_name": "x"}, http.StatusNotFound},
		})
	})

	t.Run("password", func(t *testing.T) {
		c.expect(t, http.StatusBadRequest, http.MethodPut, base+"/"+id+"/password", J{"password": "short"})
		c.expect(t, http.StatusNoContent, http.MethodPut, base+"/"+id+"/password", J{"password": "password456"})
		anonymous(t).expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": "password123"})
		signIn(t, username, "password456")
	})

	t.Run("deactivate", func(t *testing.T) {
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
		anonymous(t).expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": "password456"})
		assert.NotContains(t, ids(c.get(t, base+"/assignable")), id)
	})
}

func TestAuditLog(t *testing.T) {
	c := admin(t)
	newUser(t)
	assert.NotEmpty(t, items(c.get(t, "/api/audit/v1")))
	c.expect(t, http.StatusBadRequest, http.MethodGet, "/api/audit/v1?actor_id=x", nil)
}
