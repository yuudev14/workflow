package templating_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/templating"
)

// Golden tests: the same expressions the python side renders with jinja2 and
// the node side with nunjucks — {{ var.steps["node name"] }} et al.
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

	// jinja2's default Undefined renders as empty string — parity
	require.NoError(t, err)
	assert.Equal(t, "", out)
}
