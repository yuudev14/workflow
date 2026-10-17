package domain

// RunInput is the trigger data of a playbook run, available in templates as `var.input`.
// Records are loaded on the server from the ids the caller sends. List queries never
// select `payload` (which templates read), and a record from the client could be faked.
//
// Records are decoded maps, not json.RawMessage. The Go builtin runtimes give this struct
// directly to the template engine, and gonja cannot index raw bytes:
// `var.input.records[0].payload` fails with "Can't use Getitem on None". The python/node
// runtimes work either way because they parse the payload as JSON again.
type RunInput struct {
	ModuleType *string          `json:"module_type"`
	Records    []map[string]any `json:"records"`
	Parameters map[string]any   `json:"parameters"`
}

// NewEmptyRunInput is for a run with no trigger data. Every runtime must see a non-nil
// input. If `input` is undefined, `{{ var.input.parameters.x }}` raises in jinja2 on a
// normal manual run.
func NewEmptyRunInput() *RunInput {
	return &RunInput{
		Records:    []map[string]any{},
		Parameters: map[string]any{},
	}
}

// TemplateVars is the shape for the template engines: a plain map with the wire names.
//
// Do not pass the struct. gonja looks up Go field names, so `var.input.records` looks for
// a field called `records`, misses `Records` and fails with "Can't use Getitem on None".
// The python/node runtimes hid this, because they marshal to JSON first (json tags).
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

// Resolved fills the empty shape for a nil or partly filled input.
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
