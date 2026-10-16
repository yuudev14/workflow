package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPlaybook(t *testing.T, c *client) string {
	t.Helper()
	return c.expect(t, http.StatusOK, http.MethodPost, "/api/playbooks/v1", J{"name": unique("playbook")}).(J)["id"].(string)
}

func TestPlaybooks(t *testing.T) {
	const base = "/api/playbooks/v1"
	c := admin(t)
	id := newPlaybook(t, c)

	t.Run("create", func(t *testing.T) {
		c.expectAll(t, []request{
			{"no name", http.MethodPost, base, J{}, http.StatusBadRequest},
			{"unknown trigger type", http.MethodPost, base, J{"name": unique("p"), "trigger_type": "cron"}, http.StatusBadRequest},
			{"webhook trigger", http.MethodPost, base, J{"name": unique("p"), "trigger_type": "webhook"}, http.StatusOK},
		})
	})

	t.Run("read", func(t *testing.T) {
		name := c.get(t, base+"/"+id).(J)["name"].(string)
		assert.Equal(t, []any{id}, ids(c.get(t, base+"?name="+name)))
		c.get(t, base+"/summary")
		c.expectAll(t, []request{
			{"unknown playbook", http.MethodGet, base + "/" + uuid.NewString(), nil, http.StatusNotFound},
			{"unknown playbook tasks", http.MethodGet, base + "/" + uuid.NewString() + "/tasks", nil, http.StatusNotFound},
		})
	})

	t.Run("update", func(t *testing.T) {
		c.expect(t, http.StatusOK, http.MethodPut, base+"/"+id, J{"description": "changed"})
		assert.Equal(t, "changed", c.get(t, base+"/"+id).(J)["description"])
	})

	t.Run("save the graph", func(t *testing.T) {
		c.expect(t, http.StatusOK, http.MethodPut, base+"/tasks/"+id, J{
			"nodes": []J{
				{"name": "start", "operation": "start", "parameters": J{}},
				{"name": "next", "operation": "transform", "parameters": J{}},
			},
			"edges": J{"start": []string{"next"}},
		})
		assert.Len(t, items(c.get(t, base+"/"+id+"/tasks")), 2)
	})

	t.Run("run", func(t *testing.T) {
		// The browser gets the run status on a WebSocket. It signs in with the refresh cookie.
		header := http.Header{"Origin": {"http://localhost:9999"}}
		for _, cookie := range c.http.Jar.Cookies(must(url.Parse(serverURL))) {
			header.Add("Cookie", cookie.String())
		}
		socket, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws/playbook", header)
		require.NoError(t, err)
		defer socket.Close()

		c.expectAll(t, []request{
			{"bad id", http.MethodPost, base + "/trigger/not-a-uuid", nil, http.StatusBadRequest},
			{"unknown playbook", http.MethodPost, base + "/trigger/" + uuid.NewString(), nil, http.StatusNotFound},
		})

		// The API only queues the run. The worker runs it later.
		run := c.expect(t, http.StatusAccepted, http.MethodPost, base+"/trigger/"+id, nil).(J)
		entries := items(c.get(t, base+"/history?playbook_id="+id))
		require.Len(t, entries, 1)
		assert.Equal(t, run["playbook_history_id"], entries[0].(J)["id"])
		assert.Equal(t, "queued", entries[0].(J)["status"])
		assert.Len(t, items(c.get(t, base+"/history/"+entries[0].(J)["id"].(string)+"/tasks")), 2)

		var event J
		socket.SetReadDeadline(time.Now().Add(5 * time.Second))
		require.NoError(t, socket.ReadJSON(&event))
		assert.Equal(t, "playbook_status", event["event"])
		assert.Equal(t, run["playbook_history_id"], event["data"].(J)["id"])
	})
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
