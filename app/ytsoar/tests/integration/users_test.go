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

	t.Run("POST /users", func(t *testing.T) {
		name := unique("u")
		c.expectAll(t, []request{
			{"returns 400 when the email is not valid", http.MethodPost, base, J{"username": name, "email": "bad", "password": "password123"}, http.StatusBadRequest},
			{"returns 400 when the password is too short", http.MethodPost, base, J{"username": name, "email": name + "@ytsoar.test", "password": "short"}, http.StatusBadRequest},
			{"returns 400 when a role id is not a uuid", http.MethodPost, base, J{"username": name, "email": name + "@ytsoar.test", "password": "password123", "role_ids": []string{"x"}}, http.StatusBadRequest},
			{"returns 409 when the username is in use", http.MethodPost, base, J{"username": username, "email": name + "@ytsoar.test", "password": "password123"}, http.StatusConflict},
			{"returns 409 when the email is in use", http.MethodPost, base, J{"username": name, "email": username + "@ytsoar.test", "password": "password123"}, http.StatusConflict},
		})
	})

	t.Run("GET /users", func(t *testing.T) {
		t.Run("finds a user by the username", func(t *testing.T) {
			assert.Equal(t, []any{id}, ids(c.get(t, base+"?search="+username)))
		})
		t.Run("lists the active users as assignable", func(t *testing.T) {
			assert.Contains(t, ids(c.get(t, base+"/assignable")), id)
		})
	})

	t.Run("GET /users/:id", func(t *testing.T) {
		t.Run("returns the user", func(t *testing.T) {
			assert.Equal(t, username, c.get(t, base+"/"+id).(J)["username"])
		})
		t.Run("returns 404 when the user does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
		})
	})

	t.Run("PUT /users/:id", func(t *testing.T) {
		adminEmail := c.get(t, "/api/auth/v1/me").(J)["user"].(J)["email"]
		c.expectAll(t, []request{
			{"changes the name", http.MethodPut, base + "/" + id, J{"first_name": "Ana"}, http.StatusOK},
			{"returns 409 when the email is in use", http.MethodPut, base + "/" + id, J{"email": adminEmail}, http.StatusConflict},
			{"returns 404 when the user does not exist", http.MethodPut, base + "/" + uuid.NewString(), J{"first_name": "x"}, http.StatusNotFound},
		})
	})

	t.Run("PUT /users/:id/password", func(t *testing.T) {
		t.Run("returns 400 when the password is too short", func(t *testing.T) {
			c.expect(t, http.StatusBadRequest, http.MethodPut, base+"/"+id+"/password", J{"password": "short"})
		})
		t.Run("replaces the old password", func(t *testing.T) {
			c.expect(t, http.StatusNoContent, http.MethodPut, base+"/"+id+"/password", J{"password": "password456"})
			anonymous(t).expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": "password123"})
			signIn(t, username, "password456")
		})
	})

	t.Run("DELETE /users/:id", func(t *testing.T) {
		t.Run("deactivates the user", func(t *testing.T) {
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
			anonymous(t).expect(t, http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": "password456"})
			assert.NotContains(t, ids(c.get(t, base+"/assignable")), id)
		})
	})
}

func TestAuditLog(t *testing.T) {
	c := admin(t)

	t.Run("GET /audit", func(t *testing.T) {
		t.Run("lists the changes", func(t *testing.T) {
			newUser(t)
			assert.NotEmpty(t, items(c.get(t, "/api/audit/v1")))
		})
		t.Run("returns 400 when the actor id is not a uuid", func(t *testing.T) {
			c.expect(t, http.StatusBadRequest, http.MethodGet, "/api/audit/v1?actor_id=x", nil)
		})
	})
}
