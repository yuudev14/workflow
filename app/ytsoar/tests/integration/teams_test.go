package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTeams(t *testing.T) {
	const base = "/api/v1/teams"
	c := admin(t)
	name := unique("blue")
	id := c.create(t, base, J{"name": name})["id"].(string)

	t.Run("POST /teams", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the name is missing", http.MethodPost, base, J{}, http.StatusBadRequest},
			{"returns 409 when the name is in use", http.MethodPost, base, J{"name": name}, http.StatusConflict},
			{"returns 400 when a member id is not a uuid", http.MethodPost, base, J{"name": unique("t"), "member_ids": []string{"x"}}, http.StatusBadRequest},
		})
	})

	t.Run("GET /teams", func(t *testing.T) {
		t.Run("lists the team", func(t *testing.T) {
			assert.Contains(t, ids(c.get(t, base)), id)
		})
		t.Run("returns the team", func(t *testing.T) {
			c.get(t, base+"/"+id)
		})
		t.Run("returns 404 when the team does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
		})
	})

	t.Run("PUT /teams/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"changes the description", http.MethodPut, base + "/" + id, J{"description": "the blue team"}, http.StatusOK},
			{"returns 404 when the team does not exist", http.MethodPut, base + "/" + uuid.NewString(), J{"description": "x"}, http.StatusNotFound},
		})
	})

	t.Run("PUT /teams/:id/members and /roles", func(t *testing.T) {
		roleID := c.create(t, "/api/v1/roles", J{"name": unique("reader"), "permissions": J{"alerts": []string{"read"}}})["id"].(string)
		userID, user := newUser(t)
		user.expect(t, http.StatusForbidden, http.MethodGet, "/api/v1/alerts", nil)
		c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/roles", J{"role_ids": []string{roleID}})

		t.Run("gives the team roles to a new member", func(t *testing.T) {
			c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/members", J{"member_ids": []string{userID}})
			user.get(t, "/api/v1/alerts")
		})

		t.Run("takes the team roles from a removed member", func(t *testing.T) {
			c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id+"/members", J{"member_ids": []string{}})
			user.expect(t, http.StatusForbidden, http.MethodGet, "/api/v1/alerts", nil)
		})
	})

	t.Run("DELETE /teams/:id", func(t *testing.T) {
		t.Run("deletes the team", func(t *testing.T) {
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+id, nil)
		})
	})
}
