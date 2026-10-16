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

	t.Run("create", func(t *testing.T) {
		c.expectAll(t, []request{
			{"no title", http.MethodPost, base, J{"severity": "low", "source_kind": "edr"}, http.StatusBadRequest},
			{"unknown severity", http.MethodPost, base, J{"title": "x", "severity": "huge", "source_kind": "edr"}, http.StatusBadRequest},
			{"unknown source", http.MethodPost, base, J{"title": "x", "severity": "low", "source_kind": "fax"}, http.StatusBadRequest},
			{"bad assignee", http.MethodPost, base, J{"title": "x", "severity": "low", "source_kind": "edr", "assignee_id": "x"}, http.StatusBadRequest},
			{"with tags", http.MethodPost, base, J{"title": unique("a"), "severity": "low", "source_kind": "email", "tags": []string{"phish"}}, http.StatusCreated},
			{"empty batch", http.MethodPost, base + "/batch", J{"alerts": []J{}}, http.StatusBadRequest},
			{"batch", http.MethodPost, base + "/batch", J{"alerts": []J{{"title": unique("b"), "severity": "low", "source_kind": "dlp"}}}, http.StatusCreated},
		})
	})

	t.Run("read", func(t *testing.T) {
		alert := c.get(t, base+"/"+id).(J)
		assert.Equal(t, "high", alert["severity"])
		assert.Equal(t, []any{id}, ids(c.get(t, base+"?q="+alert["title"].(string))))
		c.get(t, base+"/summary")
		c.expectAll(t, []request{
			{"unknown alert", http.MethodGet, base + "/" + uuid.NewString(), nil, http.StatusNotFound},
			{"filter by severity", http.MethodGet, base + "?severity=high&status=new", nil, http.StatusOK},
			{"unknown severity filter", http.MethodGet, base + "?severity=huge", nil, http.StatusBadRequest},
			{"limit too big", http.MethodGet, base + "?limit=500", nil, http.StatusBadRequest},
		})
	})

	t.Run("update", func(t *testing.T) {
		c.expectAll(t, []request{
			{"severity", http.MethodPatch, base + "/" + id, J{"severity": "critical"}, http.StatusOK},
			{"unknown status", http.MethodPatch, base + "/" + id + "/status", J{"status": "done"}, http.StatusBadRequest},
			{"status", http.MethodPatch, base + "/" + id + "/status", J{"status": "investigating"}, http.StatusOK},
			{"unknown alert", http.MethodPatch, base + "/" + uuid.NewString(), J{"severity": "low"}, http.StatusNotFound},
		})
		alert := c.get(t, base+"/"+id).(J)
		assert.Equal(t, "critical", alert["severity"])
		assert.Equal(t, "investigating", alert["status"])
	})

	t.Run("notes", func(t *testing.T) {
		c.expect(t, http.StatusBadRequest, http.MethodPost, base+"/"+id+"/notes", J{"body": ""})
		noteID := c.create(t, base+"/"+id+"/notes", J{"body": "first"})["id"].(string)
		c.expect(t, http.StatusOK, http.MethodPatch, base+"/"+id+"/notes/"+noteID, J{"body": "changed"})
		notes := items(c.get(t, base+"/"+id+"/notes"))
		require.Len(t, notes, 1)
		assert.Equal(t, "changed", notes[0].(J)["body"])
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
		c.expect(t, http.StatusNotFound, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
	})

	t.Run("run a playbook", func(t *testing.T) {
		playbookID := newPlaybook(t, c)
		c.expectAll(t, []request{
			{"no records", http.MethodPost, base + "/run", J{"playbook_id": playbookID, "record_ids": []string{}}, http.StatusBadRequest},
			{"bad playbook id", http.MethodPost, base + "/run", J{"playbook_id": "x", "record_ids": []string{id}}, http.StatusBadRequest},
			{"one alert", http.MethodPost, base + "/run", J{"playbook_id": playbookID, "record_ids": []string{id}}, http.StatusAccepted},
		})
	})

	t.Run("escalate", func(t *testing.T) {
		incident := c.create(t, base+"/"+id+"/escalate", J{"title": unique("escalated")})
		linked := c.get(t, "/api/incidents/v1/"+incident["id"].(string)).(J)
		assert.Contains(t, ids(linked["linked_alerts"]), id)
	})
}
