package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newPlaybook(t *testing.T, c *client) string {
	t.Helper()
	return c.expect(t, http.StatusOK, http.MethodPost, "/api/v1/playbooks", J{"name": unique("playbook")}).(J)["id"].(string)
}

func TestPlaybooks(t *testing.T) {
	const base = "/api/v1/playbooks"
	c := admin(t)
	id := newPlaybook(t, c)

	t.Run("POST /playbooks", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the name is missing", http.MethodPost, base, J{}, http.StatusBadRequest},
			{"returns 400 when the trigger type is unknown", http.MethodPost, base, J{"name": unique("p"), "trigger_type": "cron"}, http.StatusBadRequest},
			{"creates a playbook with a webhook trigger", http.MethodPost, base, J{"name": unique("p"), "trigger_type": "webhook"}, http.StatusOK},
		})
	})

	t.Run("GET /playbooks", func(t *testing.T) {
		t.Run("finds a playbook by its name", func(t *testing.T) {
			name := c.get(t, base+"/"+id).(J)["name"].(string)
			assert.Equal(t, []any{id}, ids(c.get(t, base+"?name="+name)))
		})
		c.expectAll(t, []request{
			{"returns the summary", http.MethodGet, base + "/summary", nil, http.StatusOK},
			{"returns 404 when the playbook does not exist", http.MethodGet, base + "/" + uuid.NewString(), nil, http.StatusNotFound},
			{"returns 404 for the tasks of an unknown playbook", http.MethodGet, base + "/" + uuid.NewString() + "/tasks", nil, http.StatusNotFound},
		})
	})

	t.Run("PUT /playbooks/:id", func(t *testing.T) {
		t.Run("changes the description", func(t *testing.T) {
			c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id, J{"description": "changed"})
			assert.Equal(t, "changed", c.get(t, base+"/"+id).(J)["description"])
		})
	})

	t.Run("PUT /playbooks/tasks/:id", func(t *testing.T) {
		t.Run("saves the graph", func(t *testing.T) {
			c.expect(t, http.StatusOK, http.MethodPut, base+"/tasks/"+id, J{
				"nodes": []J{
					{"name": "start", "operation": "start", "parameters": J{}},
					{"name": "next", "operation": "transform", "parameters": J{}},
				},
				"edges": J{"start": []string{"next"}},
			})
			assert.Len(t, items(c.get(t, base+"/"+id+"/tasks")), 2)
		})

		start := J{"name": "start", "operation": "start", "parameters": J{}}
		c.expectAll(t, []request{
			{"returns 400 when the start step has no edge", http.MethodPut, base + "/tasks/" + id, J{"nodes": []J{start}, "edges": J{}}, http.StatusBadRequest},
			{"returns 400 when the start step is missing", http.MethodPut, base + "/tasks/" + id, J{"nodes": []J{}, "edges": J{"start": []string{}}}, http.StatusBadRequest},
		})
	})

	t.Run("POST /playbooks/trigger/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the id is not a uuid", http.MethodPost, base + "/trigger/not-a-uuid", nil, http.StatusBadRequest},
			{"returns 404 when the playbook does not exist", http.MethodPost, base + "/trigger/" + uuid.NewString(), nil, http.StatusNotFound},
		})

		t.Run("queues a run with a snapshot of each task", func(t *testing.T) {
			run := c.expect(t, http.StatusAccepted, http.MethodPost, base+"/trigger/"+id, nil).(J)
			assert.Equal(t, []any{run["playbook_history_id"]}, ids(c.get(t, base+"/history?playbook_id="+id)))
			assert.Len(t, items(c.get(t, base+"/history/"+run["playbook_history_id"].(string)+"/tasks")), 2)
		})
	})
}
