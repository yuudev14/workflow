package domain

// RunInput is the trigger data a playbook run is given, reachable in templates as
// `var.input`. Records are hydrated server-side from ids the caller sends: list
// queries never select `payload`, which is exactly what templates read, and a
// client-supplied record would be spoofable.
//
// Records are decoded maps, NOT json.RawMessage. The Go builtin runtimes hand
// this struct straight to the template engine, and gonja cannot index into raw
// bytes - `var.input.records[0].payload` fails with "Can't use Getitem on None".
// The python/node runtimes happened to work either way because they re-parse the
// payload as JSON, so raw bytes would have broken only the Go connectors.
type RunInput struct {
	ModuleType *string          `json:"module_type"`
	Records    []map[string]any `json:"records"`
	Parameters map[string]any   `json:"parameters"`
}

// NewEmptyRunInput is what a run with no trigger data gets. Every runtime must
// see a non-nil input: with `input` undefined, `{{ var.input.parameters.x }}`
// raises in jinja2 on an ordinary manual run, and playbook authors would have to
// write defensive templates forever.
func NewEmptyRunInput() *RunInput {
	return &RunInput{
		Records:    []map[string]any{},
		Parameters: map[string]any{},
	}
}

// TemplateVars is the shape template engines must be handed - a plain map keyed
// by the wire names.
//
// Handing them the struct does NOT work: gonja resolves against Go field names,
// so `var.input.records` looks for a field literally called `records`, misses
// `Records`, and fails with "Can't use Getitem on None". The python/node runtimes
// hid this because they JSON-marshal first, which applies the json tags.
func (r *RunInput) TemplateVars() map[string]any {
	res := r.Resolved()

	var moduleType any
	if res.ModuleType != nil {
		moduleType = *res.ModuleType
	}

	return map[string]any{
		"module_type": moduleType,
		"records":     res.Records,
		"parameters":  res.Parameters,
	}
}

// Resolved fills in the empty shape for a nil or partially-populated input.
func (r *RunInput) Resolved() *RunInput {
	if r == nil {
		return NewEmptyRunInput()
	}
	out := *r
	if out.Records == nil {
		out.Records = []map[string]any{}
	}
	if out.Parameters == nil {
		out.Parameters = map[string]any{}
	}
	return &out
}
