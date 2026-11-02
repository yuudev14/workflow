package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIncidents(t *testing.T) {
	const base = "/api/v1/incidents"
	c := admin(t)
	id := c.create(t, base, J{"title": unique("incident"), "severity": "medium"})["id"].(string)

	t.Run("POST /incidents", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the severity is missing", http.MethodPost, base, J{"title": "x"}, http.StatusBadRequest},
			{"returns 400 when the status is unknown", http.MethodPost, base, J{"title": "x", "severity": "low", "status": "done"}, http.StatusBadRequest},
			{"returns 400 when an alert id is not a uuid", http.MethodPost, base, J{"title": "x", "severity": "low", "alert_ids": []string{"x"}}, http.StatusBadRequest},
			{"creates an incident with an alert", http.MethodPost, base, J{"title": unique("i"), "severity": "low", "alert_ids": []string{newAlert(t, c)}}, http.StatusCreated},
		})
	})

	t.Run("GET /incidents", func(t *testing.T) {
		t.Run("finds an incident by its title", func(t *testing.T) {
			title := c.get(t, base+"/"+id).(J)["title"].(string)
			assert.Equal(t, []any{id}, ids(c.get(t, base+"?q="+title)))
		})
		c.expectAll(t, []request{
			{"lists only the open incidents", http.MethodGet, base + "?open=true", nil, http.StatusOK},
			{"returns 400 when the status filter is unknown", http.MethodGet, base + "?status=done", nil, http.StatusBadRequest},
			{"returns the summary", http.MethodGet, base + "/summary", nil, http.StatusOK},
		})
	})

	t.Run("GET /incidents/:id", func(t *testing.T) {
		t.Run("returns the incident", func(t *testing.T) {
			assert.Equal(t, "medium", c.get(t, base+"/"+id).(J)["severity"])
		})
		t.Run("returns 404 when the incident does not exist", func(t *testing.T) {
			c.expect(t, http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
		})
	})

	t.Run("PATCH /incidents/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"changes the severity", http.MethodPatch, base + "/" + id, J{"severity": "high"}, http.StatusOK},
			{"returns 400 when the status is unknown", http.MethodPatch, base + "/" + id + "/status", J{"status": "done"}, http.StatusBadRequest},
			{"changes the status", http.MethodPatch, base + "/" + id + "/status", J{"status": "contained"}, http.StatusOK},
		})
		incident := c.get(t, base+"/"+id).(J)
		assert.Equal(t, "high", incident["severity"])
		assert.Equal(t, "contained", incident["status"])
	})

	t.Run("notes", func(t *testing.T) {
		t.Run("adds, changes and deletes a note", func(t *testing.T) {
			noteID := c.create(t, base+"/"+id+"/notes", J{"body": "first"})["id"].(string)
			c.expect(t, http.StatusOK, http.MethodPatch, base+"/"+id+"/notes/"+noteID, J{"body": "changed"})
			assert.Len(t, items(c.get(t, base+"/"+id+"/notes")), 1)
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
		})
	})

	t.Run("linked alerts", func(t *testing.T) {
		t.Run("returns 400 when the alert id is not a uuid", func(t *testing.T) {
			c.expect(t, http.StatusBadRequest, http.MethodPost, base+"/"+id+"/alerts", J{"alert_id": "x"})
		})

		t.Run("links and unlinks an alert", func(t *testing.T) {
			alertID := newAlert(t, c)
			c.expect(t, http.StatusNoContent, http.MethodPost, base+"/"+id+"/alerts", J{"alert_id": alertID})
			assert.Contains(t, ids(c.get(t, base+"/"+id).(J)["linked_alerts"]), alertID)
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id+"/alerts/"+alertID, nil)
			assert.NotContains(t, ids(c.get(t, base+"/"+id).(J)["linked_alerts"]), alertID)
		})
	})

	t.Run("POST /incidents/run", func(t *testing.T) {
		t.Run("queues a run for the incident", func(t *testing.T) {
			c.expect(t, http.StatusAccepted, http.MethodPost, base+"/run", J{"playbook_id": newPlaybook(t, c), "record_ids": []string{id}})
		})
	})

	t.Run("DELETE /incidents/:id", func(t *testing.T) {
		deleted := c.create(t, base, J{"title": unique("deleted"), "severity": "low"})["id"].(string)
		c.expectAll(t, []request{
			{"deletes the incident", http.MethodDelete, base + "/" + deleted, nil, http.StatusNoContent},
			{"hides the incident", http.MethodGet, base + "/" + deleted, nil, http.StatusNotFound},
			{"returns 404 when the incident is already deleted", http.MethodDelete, base + "/" + deleted, nil, http.StatusNotFound},
		})
	})
}
