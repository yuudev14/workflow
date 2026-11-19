package integration

import (
	"context"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// callTool calls one tool, like the agent service does, and returns its text and if it is an error.
func callTool(t *testing.T, token, name string, args J) (string, bool) {
	t.Helper()
	ctx := context.Background()
	c, err := mcpclient.NewStreamableHttpClient(serverURL+"/api/v1/mcp",
		transport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer " + token}))
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, c.Start(ctx))

	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "1"}
	_, err = c.Initialize(ctx, init)
	require.NoError(t, err)

	call := mcp.CallToolRequest{}
	call.Params.Name = name
	call.Params.Arguments = args
	result, err := c.CallTool(ctx, call)
	require.NoError(t, err)
	require.NotEmpty(t, result.Content)
	return result.Content[0].(mcp.TextContent).Text, result.IsError
}

func TestMCP(t *testing.T) {
	c := admin(t)
	alertID := newAlert(t, c)
	alertTitle := c.get(t, "/api/v1/alerts/"+alertID).(J)["title"].(string)
	incidentID := c.create(t, "/api/v1/incidents", J{"title": unique("incident"), "severity": "medium"})["id"].(string)
	playbookID := newPlaybook(t, c)
	run, _ := runPlaybook(t, c, []J{step("boom", "code_snippet_py", "raise Exception('boom')")}, J{"start": []string{"boom"}})
	runID := run["id"].(string)

	t.Run("/api/v1/mcp", func(t *testing.T) {
		t.Run("returns the data of each read tool to an admin", func(t *testing.T) {
			tests := []struct {
				tool string
				args J
				want string
			}{
				{"list_alerts", J{"severity": "high"}, alertTitle},
				{"get_alert", J{"id": alertID}, alertTitle},
				{"list_incidents", J{}, incidentID},
				{"get_incident", J{"id": incidentID}, incidentID},
				{"list_playbooks", J{}, playbookID},
				{"get_playbook", J{"id": run["playbook_id"]}, "boom"},
				{"list_runs", J{"playbook_id": run["playbook_id"]}, runID},
				{"get_run", J{"id": runID}, "boom"},
			}
			for _, tt := range tests {
				t.Run(tt.tool, func(t *testing.T) {
					text, isError := callTool(t, c.token, tt.tool, tt.args)
					assert.False(t, isError, text)
					assert.Contains(t, text, tt.want)
				})
			}
		})

		t.Run("returns an error for a bad id", func(t *testing.T) {
			text, isError := callTool(t, c.token, "get_alert", J{"id": "x"})
			assert.True(t, isError)
			assert.Contains(t, text, "id must be a uuid")
		})

		t.Run("accepts an API key and returns forbidden for a module without the grant", func(t *testing.T) {
			roleID := c.create(t, "/api/v1/roles", J{"name": unique("role"), "permissions": J{"alerts": []string{"read"}}})["id"].(string)
			userID, _ := newUser(t, roleID)
			key := c.create(t, "/api/v1/api-keys", J{"user_id": userID, "name": unique("agent")})["key"].(string)

			text, isError := callTool(t, key, "get_alert", J{"id": alertID})
			assert.False(t, isError, text)

			text, isError = callTool(t, key, "get_incident", J{"id": incidentID})
			assert.True(t, isError)
			assert.Equal(t, "forbidden", text)
		})

		t.Run("returns forbidden to a user with no role", func(t *testing.T) {
			_, user := newUser(t)
			text, isError := callTool(t, user.token, "list_alerts", J{})
			assert.True(t, isError)
			assert.Equal(t, "forbidden", text)
		})
	})
}
