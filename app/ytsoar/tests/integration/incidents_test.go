package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIncidents(t *testing.T) {
	const base = "/api/incidents/v1"
	c := admin(t)
	id := c.create(t, base, J{"title": unique("incident"), "severity": "medium"})["id"].(string)

	t.Run("create", func(t *testing.T) {
		c.expectAll(t, []request{
			{"no severity", http.MethodPost, base, J{"title": "x"}, http.StatusBadRequest},
			{"unknown status", http.MethodPost, base, J{"title": "x", "severity": "low", "status": "done"}, http.StatusBadRequest},
			{"bad alert id", http.MethodPost, base, J{"title": "x", "severity": "low", "alert_ids": []string{"x"}}, http.StatusBadRequest},
			{"with an alert", http.MethodPost, base, J{"title": unique("i"), "severity": "low", "alert_ids": []string{newAlert(t, c)}}, http.StatusCreated},
		})
	})

	t.Run("read", func(t *testing.T) {
		incident := c.get(t, base+"/"+id).(J)
		assert.Equal(t, "medium", incident["severity"])
		assert.Equal(t, []any{id}, ids(c.get(t, base+"?q="+incident["title"].(string))))
		c.get(t, base+"/summary")
		c.expectAll(t, []request{
			{"unknown incident", http.MethodGet, base + "/" + uuid.NewString(), nil, http.StatusNotFound},
			{"open only", http.MethodGet, base + "?open=true", nil, http.StatusOK},
			{"unknown status filter", http.MethodGet, base + "?status=done", nil, http.StatusBadRequest},
		})
	})

	t.Run("update", func(t *testing.T) {
		c.expectAll(t, []request{
			{"severity", http.MethodPatch, base + "/" + id, J{"severity": "high"}, http.StatusOK},
			{"unknown status", http.MethodPatch, base + "/" + id + "/status", J{"status": "done"}, http.StatusBadRequest},
			{"status", http.MethodPatch, base + "/" + id + "/status", J{"status": "contained"}, http.StatusOK},
		})
		incident := c.get(t, base+"/"+id).(J)
		assert.Equal(t, "high", incident["severity"])
		assert.Equal(t, "contained", incident["status"])
	})

	t.Run("notes", func(t *testing.T) {
		noteID := c.create(t, base+"/"+id+"/notes", J{"body": "first"})["id"].(string)
		c.expect(t, http.StatusOK, http.MethodPatch, base+"/"+id+"/notes/"+noteID, J{"body": "changed"})
		assert.Len(t, items(c.get(t, base+"/"+id+"/notes")), 1)
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
	})

	t.Run("link alerts", func(t *testing.T) {
		alertID := newAlert(t, c)
		c.expectAll(t, []request{
			{"bad alert id", http.MethodPost, base + "/" + id + "/alerts", J{"alert_id": "x"}, http.StatusBadRequest},
			{"link", http.MethodPost, base + "/" + id + "/alerts", J{"alert_id": alertID}, http.StatusNoContent},
		})
		assert.Contains(t, ids(c.get(t, base+"/"+id).(J)["linked_alerts"]), alertID)
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id+"/alerts/"+alertID, nil)
		assert.NotContains(t, ids(c.get(t, base+"/"+id).(J)["linked_alerts"]), alertID)
	})

	t.Run("run a playbook", func(t *testing.T) {
		c.expect(t, http.StatusAccepted, http.MethodPost, base+"/run", J{"playbook_id": newPlaybook(t, c), "record_ids": []string{id}})
	})
}
