// Package templating renders {{ var.steps["node"] }} params for Go built-in
// connectors with gonja (a jinja2 engine), matching what jinja2 does for
// python connectors and nunjucks for node ones: template strings render to
// strings, everything else passes through untouched.
package templating

import (
	"math"
	"strings"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
)

// GonjaEngine implements goconnectors.TemplateEngine.
type GonjaEngine struct{}

func NewGonjaEngine() *GonjaEngine {
	return &GonjaEngine{}
}

// Render walks maps/lists and renders every string containing a template
// marker against {"var": variables}.
func (e *GonjaEngine) Render(value any, variables map[string]any) (any, error) {
	normalized, _ := normalizeNumbers(variables).(map[string]any)
	return e.render(value, normalized)
}

func (e *GonjaEngine) render(value any, variables map[string]any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			rendered, err := e.render(item, variables)
			if err != nil {
				return nil, err
			}
			out[key] = rendered
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			rendered, err := e.render(item, variables)
			if err != nil {
				return nil, err
			}
			out[i] = rendered
		}
		return out, nil
	case string:
		if !strings.Contains(v, "{{") && !strings.Contains(v, "{%") {
			return v, nil
		}
		template, err := gonja.FromString(v)
		if err != nil {
			return nil, err
		}
		return template.ExecuteToString(exec.NewContext(map[string]any{"var": variables}))
	default:
		return value, nil
	}
}

// normalizeNumbers converts whole-number float64s (what encoding/json gives
// every JSON number) to int64 so templates render "200", not "200.0" — the
// python side json.loads keeps ints as ints and jinja2 prints them plain.
func normalizeNumbers(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = normalizeNumbers(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalizeNumbers(item)
		}
		return out
	case float64:
		if v == math.Trunc(v) && math.Abs(v) < float64(1<<53) {
			return int64(v)
		}
		return v
	default:
		return value
	}
}
