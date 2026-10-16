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

	t.Run("create", func(t *testing.T) {
		c.expectAll(t, []request{
			{"no name", http.MethodPost, base, J{}, http.StatusBadRequest},
			{"name in use", http.MethodPost, base, J{"name": name}, http.StatusConflict},
		})
	})

	t.Run("read", func(t *testing.T) {
		c.get(t, base+"/"+id)
		assert.Contains(t, ids(c.get(t, base)), id)
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
	})

	t.Run("update", func(t *testing.T) {
		c.expectAll(t, []request{
			{"description", http.MethodPut, base + "/" + id, J{"description": "reads alerts"}, http.StatusOK},
			{"unknown role", http.MethodPut, base + "/" + uuid.NewString(), J{"description": "x"}, http.StatusNotFound},
		})
	})

	t.Run("builtin roles cannot change", func(t *testing.T) {
		var adminRole string
		for _, role := range items(c.get(t, base)) {
			if role.(J)["name"] == "admin" {
				adminRole = role.(J)["id"].(string)
			}
		}
		c.expectAll(t, []request{
			{"rename", http.MethodPut, base + "/" + adminRole, J{"name": "root"}, http.StatusBadRequest},
			{"delete", http.MethodDelete, base + "/" + adminRole, nil, http.StatusBadRequest},
		})
	})

	t.Run("a role gives its grants to its users", func(t *testing.T) {
		userID, user := newUser(t)
		user.expect(t, http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)

		c.expect(t, http.StatusOK, http.MethodPut, "/api/users/v1/"+userID+"/roles", J{"role_ids": []string{id}})
		user.get(t, "/api/alerts/v1")
		user.expect(t, http.StatusForbidden, http.MethodGet, "/api/incidents/v1", nil)
		user.expect(t, http.StatusForbidden, http.MethodPost, "/api/alerts/v1", J{"title": "x", "severity": "low", "source_kind": "edr"})

		// The grants change at once, not at the next login.
		c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/permissions", J{"permissions": J{"incidents": []string{"read"}}})
		user.get(t, "/api/incidents/v1")
		user.expect(t, http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)
	})

	t.Run("delete", func(t *testing.T) {
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+id, nil)
	})
}
