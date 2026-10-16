package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRoles(t *testing.T) {
	const base = "/api/roles/v1"
	c := admin(t)
	name := unique("reader")
	id := c.create(t, base, J{"name": name, "permissions": J{"alerts": []string{"read"}}})["id"].(string)

	t.Run("POST /roles", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the name is missing", http.MethodPost, base, J{}, http.StatusBadRequest},
			{"returns 409 when the name is in use", http.MethodPost, base, J{"name": name}, http.StatusConflict},
		})
	})

	t.Run("GET /roles", func(t *testing.T) {
		t.Run("lists the role", func(t *testing.T) {
			assert.Contains(t, ids(c.get(t, base)), id)
		})
		t.Run("returns the role", func(t *testing.T) {
			c.get(t, base+"/"+id)
		})
		t.Run("returns 404 when the role does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
		})
	})

	t.Run("PUT /roles/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"changes the description", http.MethodPut, base + "/" + id, J{"description": "reads alerts"}, http.StatusOK},
			{"returns 404 when the role does not exist", http.MethodPut, base + "/" + uuid.NewString(), J{"description": "x"}, http.StatusNotFound},
		})
	})

	t.Run("the builtin admin role", func(t *testing.T) {
		var adminRole string
		for _, role := range items(c.get(t, base)) {
			if role.(J)["name"] == "admin" {
				adminRole = role.(J)["id"].(string)
			}
		}
		c.expectAll(t, []request{
			{"returns 400 on a rename", http.MethodPut, base + "/" + adminRole, J{"name": "root"}, http.StatusBadRequest},
			{"returns 400 on a delete", http.MethodDelete, base + "/" + adminRole, nil, http.StatusBadRequest},
		})
	})

	t.Run("PUT /roles/:id/permissions", func(t *testing.T) {
		userID, user := newUser(t)
		c.expect(t, http.StatusOK, http.MethodPut, "/api/users/v1/"+userID+"/roles", J{"role_ids": []string{id}})

		t.Run("gives only the granted actions to the users of the role", func(t *testing.T) {
			user.get(t, "/api/alerts/v1")
			user.expect(t, http.StatusForbidden, http.MethodGet, "/api/incidents/v1", nil)
			user.expect(t, http.StatusForbidden, http.MethodPost, "/api/alerts/v1", J{"title": "x", "severity": "low", "source_kind": "edr"})
		})

		t.Run("changes the grants at once, not at the next login", func(t *testing.T) {
			c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/permissions", J{"permissions": J{"incidents": []string{"read"}}})
			user.get(t, "/api/incidents/v1")
			user.expect(t, http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)
		})
	})

	t.Run("DELETE /roles/:id", func(t *testing.T) {
		t.Run("deletes the role", func(t *testing.T) {
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+id, nil)
		})
	})
}
