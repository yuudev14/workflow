package templating_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/templating"
)

// Golden tests: the same expressions the python side renders with jinja2 and
// the node side with nunjucks - {{ var.steps["node name"] }} et al.
func TestGonjaRendersStepsAccess(t *testing.T) {
	engine := templating.NewGonjaEngine()
	variables := map[string]any{"steps": map[string]any{
		"A":       "hello",
		"my task": map[string]any{"status": float64(200)},
	}}

	cases := []struct {
		name     string
		input    any
		expected any
	}{
		{"simple step", `{{ var.steps["A"] }}`, "hello"},
		{"bracket key with spaces", `{{ var.steps["my task"].status }}`, "200"},
		{"embedded in text", `code={{ var.steps["my task"].status }}!`, "code=200!"},
		{"plain string untouched", "no templates here", "no templates here"},
		{"number untouched", float64(42), float64(42)},
		{"bool untouched", true, true},
		{"nil untouched", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := engine.Render(tc.input, variables)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, out)
		})
	}
}

func TestGonjaRendersNestedStructures(t *testing.T) {
	engine := templating.NewGonjaEngine()
	variables := map[string]any{"steps": map[string]any{"A": "v"}}

	out, err := engine.Render(map[string]any{
		"url":   `{{ var.steps["A"] }}/items`,
		"count": float64(3),
		"tags":  []any{`{{ var.steps["A"] }}`, "plain"},
	}, variables)

	require.NoError(t, err)
	rendered := out.(map[string]any)
	assert.Equal(t, "v/items", rendered["url"])
	assert.Equal(t, float64(3), rendered["count"])
	assert.Equal(t, []any{"v", "plain"}, rendered["tags"])
}

func TestGonjaUndefinedRendersEmpty(t *testing.T) {
	engine := templating.NewGonjaEngine()

	out, err := engine.Render(`{{ var.steps["ghost"] }}`, map[string]any{"steps": map[string]any{}})

	// jinja2's default Undefined renders as empty string - parity
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

// var.input is the trigger-data namespace. The list index matters: records is a
// list of module rows, so `records[0]` is how every playbook reaches the alert
// that triggered it.
func TestGonjaRendersInputAccess(t *testing.T) {
	engine := templating.NewGonjaEngine()
	variables := map[string]any{
		"steps": map[string]any{},
		"input": map[string]any{
			"module_type": "alert",
			"records": []any{
				map[string]any{
					"severity": "critical",
					"payload":  map[string]any{"host.hostname": "web-01"},
				},
			},
			"parameters": map[string]any{"reason": "phishing"},
		},
	}

	cases := []struct {
		name     string
		input    any
		expected any
	}{
		{"module type", `{{ var.input.module_type }}`, "alert"},
		{"record list index", `{{ var.input.records[0].severity }}`, "critical"},
		{"dotted payload key", `{{ var.input.records[0].payload["host.hostname"] }}`, "web-01"},
		{"parameter", `{{ var.input.parameters.reason }}`, "phishing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := engine.Render(tc.input, variables)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, out)
		})
	}
}

// Records must arrive as decoded maps. They were json.RawMessage once, and gonja
// cannot index into raw bytes - the Go builtins failed with "Can't use Getitem on
// None" while python/node kept working, because those re-parse the payload.
// Marshalling here mirrors exactly what the record resolver does.
func TestGonjaRendersInputRecordsDecodedFromJSON(t *testing.T) {
	engine := templating.NewGonjaEngine()

	raw := `{"id":"a1","severity":"critical","payload":{"host":{"hostname":"WIN-DC02"},"user":{"name":"svc"}}}`
	var record map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &record))

	variables := map[string]any{
		"steps": map[string]any{},
		"input": map[string]any{
			"module_type": "alert",
			"records":     []map[string]any{record},
			"parameters":  map[string]any{},
		},
	}

	out, err := engine.Render(`{{ var.input.records[0].payload["host"]["hostname"] }}`, variables)
	require.NoError(t, err)
	assert.Equal(t, "WIN-DC02", out)

	out, err = engine.Render(`{{ var.input.records[0].severity }}`, variables)
	require.NoError(t, err)
	assert.Equal(t, "critical", out)
}

// An empty input must render rather than raise: a manual run from the editor has
// no trigger data, and a playbook author should not have to write defensive
// templates for that case.
func TestGonjaEmptyInputRendersEmpty(t *testing.T) {
	engine := templating.NewGonjaEngine()
	variables := map[string]any{
		"steps": map[string]any{},
		"input": map[string]any{"records": []any{}, "parameters": map[string]any{}},
	}

	out, err := engine.Render(`{{ var.input.parameters.missing }}`, variables)
	require.NoError(t, err)
	assert.Equal(t, "", out)
}
