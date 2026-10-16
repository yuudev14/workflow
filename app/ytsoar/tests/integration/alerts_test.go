package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAlert(t *testing.T, c *client) string {
	t.Helper()
	return c.create(t, "/api/alerts/v1", J{"title": unique("alert"), "severity": "high", "source_kind": "edr"})["id"].(string)
}

func TestAlerts(t *testing.T) {
	const base = "/api/alerts/v1"
	c := admin(t)
	id := newAlert(t, c)

	t.Run("POST /alerts", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the title is missing", http.MethodPost, base, J{"severity": "low", "source_kind": "edr"}, http.StatusBadRequest},
			{"returns 400 when the severity is unknown", http.MethodPost, base, J{"title": "x", "severity": "huge", "source_kind": "edr"}, http.StatusBadRequest},
			{"returns 400 when the source is unknown", http.MethodPost, base, J{"title": "x", "severity": "low", "source_kind": "fax"}, http.StatusBadRequest},
			{"returns 400 when the assignee is not a uuid", http.MethodPost, base, J{"title": "x", "severity": "low", "source_kind": "edr", "assignee_id": "x"}, http.StatusBadRequest},
			{"creates an alert with tags", http.MethodPost, base, J{"title": unique("a"), "severity": "low", "source_kind": "email", "tags": []string{"phish"}}, http.StatusCreated},
		})
	})

	t.Run("POST /alerts/batch", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the batch is empty", http.MethodPost, base + "/batch", J{"alerts": []J{}}, http.StatusBadRequest},
			{"creates the alerts", http.MethodPost, base + "/batch", J{"alerts": []J{{"title": unique("b"), "severity": "low", "source_kind": "dlp"}}}, http.StatusCreated},
		})
	})

	t.Run("GET /alerts", func(t *testing.T) {
		t.Run("finds an alert by its title", func(t *testing.T) {
			title := c.get(t, base+"/"+id).(J)["title"].(string)
			assert.Equal(t, []any{id}, ids(c.get(t, base+"?q="+title)))
		})
		c.expectAll(t, []request{
			{"filters by severity and status", http.MethodGet, base + "?severity=high&status=new", nil, http.StatusOK},
			{"returns 400 when the severity filter is unknown", http.MethodGet, base + "?severity=huge", nil, http.StatusBadRequest},
			{"returns 400 when the limit is too big", http.MethodGet, base + "?limit=500", nil, http.StatusBadRequest},
			{"returns the summary", http.MethodGet, base + "/summary", nil, http.StatusOK},
		})
	})

	t.Run("GET /alerts/:id", func(t *testing.T) {
		t.Run("returns the alert", func(t *testing.T) {
			assert.Equal(t, "high", c.get(t, base+"/"+id).(J)["severity"])
		})
		t.Run("returns 404 when the alert does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
		})
	})

	t.Run("PATCH /alerts/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 404 when the alert does not exist", http.MethodPatch, base + "/" + uuid.NewString(), J{"severity": "low"}, http.StatusNotFound},
			{"changes the severity", http.MethodPatch, base + "/" + id, J{"severity": "critical"}, http.StatusOK},
			{"returns 400 when the status is unknown", http.MethodPatch, base + "/" + id + "/status", J{"status": "done"}, http.StatusBadRequest},
			{"changes the status", http.MethodPatch, base + "/" + id + "/status", J{"status": "investigating"}, http.StatusOK},
		})
		alert := c.get(t, base+"/"+id).(J)
		assert.Equal(t, "critical", alert["severity"])
		assert.Equal(t, "investigating", alert["status"])
	})

	t.Run("notes", func(t *testing.T) {
		t.Run("returns 400 when the body is empty", func(t *testing.T) {
			c.expect(t, http.StatusBadRequest, http.MethodPost, base+"/"+id+"/notes", J{"body": ""})
		})

		t.Run("adds, changes and deletes a note", func(t *testing.T) {
			noteID := c.create(t, base+"/"+id+"/notes", J{"body": "first"})["id"].(string)
			c.expect(t, http.StatusOK, http.MethodPatch, base+"/"+id+"/notes/"+noteID, J{"body": "changed"})
			notes := items(c.get(t, base+"/"+id+"/notes"))
			require.Len(t, notes, 1)
			assert.Equal(t, "changed", notes[0].(J)["body"])
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
			c.expect(t, http.StatusNotFound, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
		})
	})

	t.Run("POST /alerts/run", func(t *testing.T) {
		playbookID := newPlaybook(t, c)
		c.expectAll(t, []request{
			{"returns 400 when no alert is given", http.MethodPost, base + "/run", J{"playbook_id": playbookID, "record_ids": []string{}}, http.StatusBadRequest},
			{"returns 400 when the playbook id is not a uuid", http.MethodPost, base + "/run", J{"playbook_id": "x", "record_ids": []string{id}}, http.StatusBadRequest},
			{"queues a run for the alert", http.MethodPost, base + "/run", J{"playbook_id": playbookID, "record_ids": []string{id}}, http.StatusAccepted},
		})
	})

	t.Run("POST /alerts/:id/escalate", func(t *testing.T) {
		t.Run("creates an incident that links the alert", func(t *testing.T) {
			incident := c.create(t, base+"/"+id+"/escalate", J{"title": unique("escalated")})
			linked := c.get(t, "/api/incidents/v1/"+incident["id"].(string)).(J)
			assert.Contains(t, ids(linked["linked_alerts"]), id)
		})
	})
}
