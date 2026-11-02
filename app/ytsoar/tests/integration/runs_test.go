package integration

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stepTimeout is the step timeout of the worker and the sandbox in these tests.
const stepTimeout = 3 * time.Second

// step returns a graph node that runs code in the sandbox.
func step(name, connector, code string) J {
	operation := map[string]string{"code_snippet_py": "python_code", "code_snippet_js": "javascript_code"}[connector]
	return J{"name": name, "operation": operation, "connector_id": connector, "connector_name": connector, "parameters": J{"code": code}}
}

// runPlaybook saves the graph, starts a run and waits until the run ends.
// It returns the run and its tasks by name.
func runPlaybook(t *testing.T, c *client, nodes []J, edges J, handles ...J) (J, map[string]J) {
	t.Helper()
	id := newPlaybook(t, c)
	nodes = append([]J{{"name": "start", "operation": "start", "parameters": J{}}}, nodes...)
	graph := J{"nodes": nodes, "edges": edges}
	if len(handles) > 0 {
		graph["handles"] = handles[0]
	}
	c.expect(t, http.StatusOK, http.MethodPut, "/api/v1/playbooks/tasks/"+id, graph)
	runID := c.expect(t, http.StatusAccepted, http.MethodPost, "/api/v1/playbooks/trigger/"+id, nil).(J)["playbook_history_id"].(string)

	var run J
	require.Eventually(t, func() bool {
		run = items(c.get(t, "/api/v1/playbooks/history?playbook_id="+id))[0].(J)
		return run["status"] == "success" || run["status"] == "failed"
	}, 30*time.Second, 200*time.Millisecond, "the run did not end")

	tasks := map[string]J{}
	for _, task := range items(c.get(t, "/api/v1/playbooks/history/"+runID+"/tasks")) {
		tasks[task.(J)["name"].(string)] = task.(J)
	}
	return run, tasks
}

func TestRuns(t *testing.T) {
	c := admin(t)

	t.Run("a Python step", func(t *testing.T) {
		t.Run("runs in the sandbox and saves its output", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{step("hello", "code_snippet_py", "result = 'hello from python'")}, J{"start": []string{"hello"}})
			assert.Equal(t, "success", run["status"])
			assert.Equal(t, "success", tasks["hello"]["status"])
			assert.Contains(t, toJSON(tasks["hello"]["result"]), "hello from python")
		})

		t.Run("fails the run when the code raises an error", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{step("boom", "code_snippet_py", "raise Exception('boom')")}, J{"start": []string{"boom"}})
			assert.Equal(t, "failed", run["status"])
			assert.Equal(t, "failed", tasks["boom"]["status"])
			assert.Contains(t, tasks["boom"]["error"], "boom")
		})

		t.Run("fails the run when the step runs longer than the timeout", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{step("slow", "code_snippet_py", "import time\ntime.sleep(30)")}, J{"start": []string{"slow"}})
			assert.Equal(t, "failed", run["status"])
			assert.Equal(t, "failed", tasks["slow"]["status"])
		})
	})

	t.Run("a JavaScript step", func(t *testing.T) {
		t.Run("runs in the sandbox", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{step("hello", "code_snippet_js", "result = 'hello from js'")}, J{"start": []string{"hello"}})
			assert.Equal(t, "success", run["status"], tasks["hello"]["error"])
			assert.Contains(t, toJSON(tasks["hello"]["result"]), "hello from js")
		})
	})

	t.Run("a built-in step", func(t *testing.T) {
		t.Run("runs in the worker", func(t *testing.T) {
			called := make(chan string, 1)
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called <- r.URL.Path
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer target.Close()

			run, tasks := runPlaybook(t, c, []J{{
				"name": "fetch", "operation": "get_request", "connector_id": "http_request", "connector_name": "HTTP Request",
				"parameters": J{"url": target.URL + "/ping"},
			}}, J{"start": []string{"fetch"}})
			assert.Equal(t, "success", run["status"], tasks["fetch"]["error"])
			assert.Equal(t, "/ping", <-called)
		})
	})

	t.Run("a connector step", func(t *testing.T) {
		t.Run("gets its config from the database", func(t *testing.T) {
			code, out := c.upload(t, "/api/v1/connectors?replace=true", zipDir(t, "../../../connectors/sample"))
			require.Equal(t, http.StatusCreated, code, out)
			name := unique("run")
			c.create(t, "/api/v1/connectors/sample/configs", J{"name": name, "config": J{"server_url": "https://from-db", "api_key": "k"}})

			run, tasks := runPlaybook(t, c, []J{{
				"name": "sample", "operation": "operation1", "connector_id": "sample", "connector_name": "Sample",
				"config": name, "parameters": J{},
			}}, J{"start": []string{"sample"}})
			assert.Equal(t, "success", run["status"], tasks["sample"]["error"])
			assert.Contains(t, toJSON(tasks["sample"]["result"]), "https://from-db")
		})

		t.Run("fails when the named config does not exist", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{{
				"name": "sample", "operation": "operation1", "connector_id": "sample", "connector_name": "Sample",
				"config": "missing", "parameters": J{},
			}}, J{"start": []string{"sample"}})
			assert.Equal(t, "failed", run["status"])
			assert.Contains(t, tasks["sample"]["error"], "missing")
		})
	})

	t.Run("a graph", func(t *testing.T) {
		t.Run("runs the steps in order", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{
				step("first", "code_snippet_py", "print('first')"),
				step("second", "code_snippet_py", "print('second')"),
			}, J{"start": []string{"first"}, "first": []string{"second"}})
			assert.Equal(t, "success", run["status"])
			assert.Equal(t, "success", tasks["first"]["status"])
			assert.Equal(t, "success", tasks["second"]["status"])
		})

		t.Run("gives the output of a step to the next step", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{
				step("first", "code_snippet_py", "result = 'hello'"),
				step("second", "code_snippet_py", `result = "{{ var.steps['first'].code_output }} world"`),
			}, J{"start": []string{"first"}, "first": []string{"second"}})
			assert.Equal(t, "success", run["status"], tasks["second"]["error"])
			assert.Contains(t, toJSON(tasks["second"]["result"]), "hello world")
		})

		t.Run("follows only the branch that a condition picks", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{
				step("scan", "code_snippet_py", "result = {'score': 60}"),
				{"name": "check", "operation": "switch_expression", "connector_id": "condition", "connector_name": "Condition", "parameters": J{"cases": []J{
					{"id": "high", "expression": `{{ var.steps['scan'].code_output.score > 90 }}`},
					{"id": "med", "expression": `{{ var.steps['scan'].code_output.score > 50 }}`},
				}}},
				step("high", "code_snippet_py", "print('high')"),
				step("med", "code_snippet_py", "print('med')"),
			}, J{"start": []string{"scan"}, "scan": []string{"check"}, "check": []string{"high", "med"}},
				J{"check": J{"high": J{"source_handle": "high"}, "med": J{"source_handle": "med"}}})
			assert.Equal(t, "success", run["status"], tasks["check"]["error"])
			assert.Contains(t, toJSON(tasks["check"]["result"]), `"med"`)
			assert.Equal(t, "success", tasks["med"]["status"])
			assert.Equal(t, "skipped", tasks["high"]["status"])
		})

		t.Run("does not run the steps after a failed step", func(t *testing.T) {
			run, tasks := runPlaybook(t, c, []J{
				step("boom", "code_snippet_py", "raise Exception('boom')"),
				step("after", "code_snippet_py", "print('after')"),
			}, J{"start": []string{"boom"}, "boom": []string{"after"}})
			assert.Equal(t, "failed", run["status"])
			assert.NotEqual(t, "success", tasks["after"]["status"])
		})
	})

	t.Run("an event trigger", func(t *testing.T) {
		// eventPlaybook saves a playbook that an alert event starts. Cleanup sets it back to
		// manual, so the alerts of other tests do not start it.
		eventPlaybook := func(t *testing.T, trigger string, node J) string {
			id := c.expect(t, http.StatusOK, http.MethodPost, "/api/v1/playbooks", J{
				"name": unique("playbook"), "trigger_type": trigger, "trigger_parameters": J{"module": "alert"},
			}).(J)["id"].(string)
			nodes := []J{{"name": "start", "operation": "start", "parameters": J{}}, node}
			c.expect(t, http.StatusOK, http.MethodPut, "/api/v1/playbooks/tasks/"+id, J{"nodes": nodes, "edges": J{"start": []string{node["name"].(string)}}})
			t.Cleanup(func() {
				c.expect(t, http.StatusOK, http.MethodPut, "/api/v1/playbooks/"+id, J{"trigger_type": "manual"})
			})
			return id
		}
		alertRuns := func(playbookID, alertID string) []any {
			return items(c.get(t, "/api/v1/playbooks/history?playbook_id="+playbookID+"&module_type=alert&record_id="+alertID))
		}

		t.Run("starts an on_create playbook with the new alert as input", func(t *testing.T) {
			playbookID := eventPlaybook(t, "on_create", step("title", "code_snippet_py", "result = '{{ var.input.records[0].title }}'"))
			title := unique("alert")
			alertID := c.create(t, "/api/v1/alerts", J{"title": title, "severity": "high", "source_kind": "edr"})["id"].(string)

			var run J
			require.Eventually(t, func() bool {
				runs := alertRuns(playbookID, alertID)
				if len(runs) == 0 {
					return false
				}
				run = runs[0].(J)
				return run["status"] == "success" || run["status"] == "failed"
			}, 30*time.Second, 200*time.Millisecond, "the alert did not start the playbook")
			assert.Equal(t, "success", run["status"])
			assert.Equal(t, "on_create", run["trigger_type"])

			tasks := items(c.get(t, "/api/v1/playbooks/history/"+run["id"].(string)+"/tasks"))
			assert.Contains(t, toJSON(tasks), title)
		})

		t.Run("does not start an on_update playbook again while its run on the alert is not done", func(t *testing.T) {
			playbookID := eventPlaybook(t, "on_update", step("slow", "code_snippet_py", "import time\ntime.sleep(2)"))
			alertID := newAlert(t, c)

			c.expect(t, http.StatusOK, http.MethodPatch, "/api/v1/alerts/"+alertID, J{"severity": "low"})
			require.Eventually(t, func() bool { return len(alertRuns(playbookID, alertID)) == 1 },
				30*time.Second, 200*time.Millisecond, "the update did not start the playbook")

			c.expect(t, http.StatusOK, http.MethodPatch, "/api/v1/alerts/"+alertID, J{"severity": "medium"})
			require.Eventually(t, func() bool {
				run := alertRuns(playbookID, alertID)[0].(J)
				return run["status"] == "success" || run["status"] == "failed"
			}, 30*time.Second, 200*time.Millisecond, "the run did not end")
			assert.Len(t, alertRuns(playbookID, alertID), 1)
		})
	})

	t.Run("the WebSocket", func(t *testing.T) {
		t.Run("sends the final status of a run", func(t *testing.T) {
			header := http.Header{"Origin": {"http://localhost:9999"}}
			for _, cookie := range c.http.Jar.Cookies(must(url.Parse(serverURL))) {
				header.Add("Cookie", cookie.String())
			}
			socket, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(serverURL, "http")+"/ws/playbook", header)
			require.NoError(t, err)
			defer socket.Close()

			run, _ := runPlaybook(t, c, []J{step("hello", "code_snippet_py", "print('hi')")}, J{"start": []string{"hello"}})
			require.NoError(t, socket.SetReadDeadline(time.Now().Add(10*time.Second)))
			for {
				var event J
				require.NoError(t, socket.ReadJSON(&event), "no success event")
				if data, ok := event["data"].(J); ok && event["event"] == "playbook_status" && data["id"] == run["id"] && data["status"] == "success" {
					return
				}
			}
		})
	})
}
