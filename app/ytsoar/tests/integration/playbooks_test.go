package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPlaybook(t *testing.T, c *client) string {
	t.Helper()
	return c.expect(t, http.StatusOK, http.MethodPost, "/api/v1/playbooks", J{"name": unique("playbook"), "trigger_type": "manual"}).(J)["id"].(string)
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
		c.expectAll(t, []request{
			{"returns 400 when the webhook auth type is unknown", http.MethodPut, base + "/" + id, J{"trigger_parameters": J{"auth": "basic"}}, http.StatusBadRequest},
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

		t.Run("saves each MVP playbook of tools/playbooks", func(t *testing.T) {
			files, err := filepath.Glob("../../../../tools/playbooks/*.json")
			require.NoError(t, err)
			require.Len(t, files, 5)
			for _, file := range files {
				var pb J
				require.NoError(t, json.Unmarshal(must(os.ReadFile(file)), &pb))
				// manual, so the alerts of other tests do not start it
				id := c.expect(t, http.StatusOK, http.MethodPost, base, J{"name": unique(pb["name"].(string)), "description": pb["description"]}).(J)["id"].(string)
				c.expect(t, http.StatusOK, http.MethodPut, base+"/tasks/"+id, J{"nodes": pb["nodes"], "edges": pb["edges"], "handles": pb["handles"]})
				assert.Len(t, items(c.get(t, base+"/"+id+"/tasks")), len(pb["nodes"].([]any)), file)
			}
		})

		start := J{"name": "start", "operation": "start", "parameters": J{}}
		c.expectAll(t, []request{
			{"saves a start step with no edge", http.MethodPut, base + "/tasks/" + newPlaybook(t, c), J{"nodes": []J{start}, "edges": J{}}, http.StatusOK},
			{"returns 400 when the start step is missing", http.MethodPut, base + "/tasks/" + id, J{"nodes": []J{}, "edges": J{"start": []string{}}}, http.StatusBadRequest},
		})
	})

	t.Run("POST /playbooks/trigger/:id", func(t *testing.T) {
		c.expectAll(t, []request{
			{"returns 400 when the id is not a uuid", http.MethodPost, base + "/trigger/not-a-uuid", nil, http.StatusBadRequest},
			{"returns 404 when the playbook does not exist", http.MethodPost, base + "/trigger/" + uuid.NewString(), nil, http.StatusNotFound},
		})

		t.Run("returns 400 when the playbook has no start step or no trigger type", func(t *testing.T) {
			noStart := newPlaybook(t, c)
			noTrigger := c.expect(t, http.StatusOK, http.MethodPost, base, J{"name": unique("p")}).(J)["id"].(string)
			c.expect(t, http.StatusOK, http.MethodPut, base+"/tasks/"+noTrigger, J{
				"nodes": []J{{"name": "start", "operation": "start", "parameters": J{}}, {"name": "next", "operation": "transform", "parameters": J{}}},
				"edges": J{"start": []string{"next"}},
			})
			c.expect(t, http.StatusBadRequest, http.MethodPost, base+"/trigger/"+noStart, nil)
			c.expect(t, http.StatusBadRequest, http.MethodPost, base+"/trigger/"+noTrigger, nil)
		})

		t.Run("queues a run with a snapshot of each task", func(t *testing.T) {
			run := c.expect(t, http.StatusAccepted, http.MethodPost, base+"/trigger/"+id, nil).(J)
			assert.Equal(t, []any{run["playbook_history_id"]}, ids(c.get(t, base+"/history?playbook_id="+id)))
			assert.Len(t, items(c.get(t, base+"/history/"+run["playbook_history_id"].(string)+"/tasks")), 2)
		})
	})

	t.Run("POST /webhooks/:playbook_id", func(t *testing.T) {
		webhook := func(t *testing.T, auth string) string {
			id := c.expect(t, http.StatusOK, http.MethodPost, base, J{
				"name": unique("webhook"), "trigger_type": "webhook", "trigger_parameters": J{"auth": auth},
			}).(J)["id"].(string)
			c.expect(t, http.StatusOK, http.MethodPut, base+"/tasks/"+id, J{
				"nodes": []J{{"name": "start", "operation": "start", "parameters": J{}}, {"name": "next", "operation": "transform", "parameters": J{}}},
				"edges": J{"start": []string{"next"}},
			})
			return "/api/v1/webhooks/" + id
		}
		keyFor := func(t *testing.T, permissions J) (string, *client) {
			roleID := c.create(t, "/api/v1/roles", J{"name": unique("role"), "permissions": permissions})["id"].(string)
			userID, _ := newUser(t, roleID)
			key := c.create(t, "/api/v1/api-keys", J{"user_id": userID, "name": unique("siem")})["key"].(string)
			return userID, &client{http: http.DefaultClient, token: key}
		}
		post := func(t *testing.T, caller *client, path, body, contentType string) int {
			res := caller.send(t, http.MethodPost, path, bytes.NewBufferString(body), contentType)
			defer res.Body.Close()
			return res.StatusCode
		}

		keyed := webhook(t, "api_key")
		userID, key := keyFor(t, J{"playbooks": []string{"execute"}})
		_, noGrant := keyFor(t, J{"playbooks": []string{"read"}})

		t.Run("queues a webhook run with the body as parameters and the key's user", func(t *testing.T) {
			run := key.expect(t, http.StatusAccepted, http.MethodPost, keyed, J{"host": "pc-1"}).(J)
			history := items(c.get(t, base+"/history?playbook_id="+strings.TrimPrefix(keyed, "/api/v1/webhooks/")))[0].(J)
			assert.Equal(t, run["run_id"], history["id"])
			assert.Equal(t, "webhook", history["trigger_type"])
			assert.Equal(t, userID, history["triggered_by"])
			assert.Equal(t, J{"host": "pc-1"}, history["input"].(J)["parameters"])
		})

		t.Run("queues a run with no key when the auth type is none", func(t *testing.T) {
			anonymous(t).expect(t, http.StatusAccepted, http.MethodPost, webhook(t, "none"), nil)
		})

		manual := "/api/v1/webhooks/" + newPlaybook(t, c)
		for _, tc := range []struct {
			name, path, body, contentType string
			caller                        *client
			want                          int
		}{
			{"returns 401 without a key", keyed, "{}", "application/json", anonymous(t), http.StatusUnauthorized},
			{"returns 401 with an access token instead of a key", keyed, "{}", "application/json", c, http.StatusUnauthorized},
			{"returns 403 when the key's user cannot execute playbooks", keyed, "{}", "application/json", noGrant, http.StatusForbidden},
			{"returns 404 when the trigger is not webhook", manual, "{}", "application/json", key, http.StatusNotFound},
			{"returns 404 when the playbook does not exist", "/api/v1/webhooks/" + uuid.NewString(), "{}", "application/json", key, http.StatusNotFound},
			{"returns 400 when the id is not a uuid", "/api/v1/webhooks/not-a-uuid", "{}", "application/json", key, http.StatusBadRequest},
			{"returns 400 when the body is not a JSON object", keyed, "[1]", "application/json", key, http.StatusBadRequest},
			{"returns 400 when the body is not JSON", keyed, "a=1", "application/x-www-form-urlencoded", key, http.StatusBadRequest},
		} {
			t.Run(tc.name, func(t *testing.T) {
				assert.Equal(t, tc.want, post(t, tc.caller, tc.path, tc.body, tc.contentType))
			})
		}
	})

	t.Run("DELETE /playbooks/:id", func(t *testing.T) {
		name := c.get(t, base+"/"+id).(J)["name"].(string)
		c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+id, nil)

		t.Run("hides the playbook from the list", func(t *testing.T) {
			assert.Empty(t, ids(c.get(t, base+"?name="+name)))
		})
		t.Run("keeps the run history", func(t *testing.T) {
			assert.Len(t, items(c.get(t, base+"/history?playbook_id="+id)), 1)
		})
		t.Run("writes an audit log entry", func(t *testing.T) {
			entry := items(c.get(t, "/api/v1/audit?module=playbooks&action=playbook_deleted"))[0].(J)
			assert.Equal(t, id, entry["entity_id"])
		})
		c.expectAll(t, []request{
			{"returns 400 when the id is not a uuid", http.MethodDelete, base + "/not-a-uuid", nil, http.StatusBadRequest},
			{"returns 404 when the playbook is already deleted", http.MethodDelete, base + "/" + id, nil, http.StatusNotFound},
			{"returns 404 on get after the delete", http.MethodGet, base + "/" + id, nil, http.StatusNotFound},
			{"returns 404 on update after the delete", http.MethodPut, base + "/" + id, J{"description": "x"}, http.StatusNotFound},
			{"returns 404 on a graph save after the delete", http.MethodPut, base + "/tasks/" + id, J{
				"nodes": []J{{"name": "start", "operation": "start", "parameters": J{}}}, "edges": J{"start": []string{}},
			}, http.StatusNotFound},
			{"returns 404 on trigger after the delete", http.MethodPost, base + "/trigger/" + id, nil, http.StatusNotFound},
		})

		t.Run("does not start a deleted playbook on an alert event, and still starts the others", func(t *testing.T) {
			eventPlaybook := func() string {
				pid := c.expect(t, http.StatusOK, http.MethodPost, base, J{
					"name": unique("playbook"), "trigger_type": "on_create", "trigger_parameters": J{"module": "alert"},
				}).(J)["id"].(string)
				c.expect(t, http.StatusOK, http.MethodPut, base+"/tasks/"+pid, J{
					"nodes": []J{{"name": "start", "operation": "start", "parameters": J{}}, {"name": "next", "operation": "transform", "parameters": J{}}},
					"edges": J{"start": []string{"next"}},
				})
				return pid
			}
			deleted, live := eventPlaybook(), eventPlaybook()
			t.Cleanup(func() { c.expect(t, http.StatusOK, http.MethodPut, base+"/"+live, J{"trigger_type": "manual"}) })
			c.expect(t, http.StatusNoContent, http.MethodDelete, base+"/"+deleted, nil)

			// Events of earlier tests' alerts can still reach a new playbook, so match runs by this alert.
			alertID := newAlert(t, c)
			ranFor := func(pid string) bool {
				for _, run := range items(c.get(t, base+"/history?playbook_id="+pid)) {
					records, _ := run.(J)["input"].(J)["records"].([]any)
					for _, r := range records {
						if r.(J)["id"] == alertID {
							return true
						}
					}
				}
				return false
			}
			require.Eventually(t, func() bool { return ranFor(live) }, 30*time.Second, 200*time.Millisecond,
				"the alert did not start the live playbook")
			assert.False(t, ranFor(deleted))
		})
	})
}
