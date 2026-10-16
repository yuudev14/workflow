package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTeams(t *testing.T) {
	const base = "/api/teams/v1"
	c := admin(t)
	name := unique("blue")
	id := c.create(t, base, J{"name": name})["id"].(string)

	t.Run("create", func(t *testing.T) {
		c.expectAll(t, []request{
			{"no name", http.MethodPost, base, J{}, http.StatusBadRequest},
			{"name in use", http.MethodPost, base, J{"name": name}, http.StatusConflict},
			{"bad member id", http.MethodPost, base, J{"name": unique("t"), "member_ids": []string{"x"}}, http.StatusBadRequest},
		})
	})

	t.Run("read", func(t *testing.T) {
		c.get(t, base+"/"+id)
		assert.Contains(t, ids(c.get(t, base)), id)
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
	})

	t.Run("update", func(t *testing.T) {
		c.expectAll(t, []request{
			{"description", http.MethodPut, base + "/" + id, J{"description": "the blue team"}, http.StatusOK},
			{"unknown team", http.MethodPut, base + "/" + uuid.NewString(), J{"description": "x"}, http.StatusNotFound},
		})
	})

	t.Run("a team gives its roles to its members", func(t *testing.T) {
		roleID := c.create(t, "/api/roles/v1", J{"name": unique("reader"), "permissions": J{"alerts": []string{"read"}}})["id"].(string)
		userID, user := newUser(t)
		user.expect(t, http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)

		c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/members", J{"member_ids": []string{userID}})
		c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/roles", J{"role_ids": []string{roleID}})
		user.get(t, "/api/alerts/v1")

		c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/members", J{"member_ids": []string{}})
		user.expect(t, http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)
	})

	t.Run("delete", func(t *testing.T) {
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
		c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+id, nil)
	})
}
