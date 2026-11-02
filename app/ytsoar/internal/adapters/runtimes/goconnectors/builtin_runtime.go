// Package goconnectors runs the built-in connectors compiled into the worker. They are our
// code, not user code, so they can run in-process. User-written code still goes to the
// sandbox. Each builtin has an info.json dir in the connectors tree with "runtime": "go"
// (virtual, BuiltinRuntime is the implementation).
package goconnectors

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/goconnectors_mock.go -package=mocks . TemplateEngine

// TemplateEngine renders {{ var.steps["node"] }} templates in params. It has jinja2
// semantics (gonja adapter) and is golden-tested against the python side.
type TemplateEngine interface {
	Render(value any, variables map[string]any) (any, error)
}

// BuiltinConnector is one built-in connector (for example http_request).
// configs is the opened connector config. params are already templated.
type BuiltinConnector interface {
	Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error)
}

// BuiltinRuntime runs the built-in connectors (http_request). They are our
// code, so they run inside the WORKER, with no sandbox. It fits execution.TaskRuntime. The
// StaticResolver of the worker returns it for the connector ids you Register here.
type BuiltinRuntime struct {
	logger     logger.Logger
	template   TemplateEngine
	connectors map[string]BuiltinConnector
}

func NewBuiltinRuntime(log logger.Logger, template TemplateEngine) *BuiltinRuntime {
	return &BuiltinRuntime{
		logger:     log,
		template:   template,
		connectors: map[string]BuiltinConnector{},
	}
}

func (r *BuiltinRuntime) Register(id string, connector BuiltinConnector) {
	r.connectors[id] = connector
}

// IDs returns the registered connector ids. The composition root maps each one to this
// runtime in the resolver.
func (r *BuiltinRuntime) IDs() []string {
	ids := make([]string, 0, len(r.connectors))
	for id := range r.connectors {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *BuiltinRuntime) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	if req.Task.ConnectorID == nil {
		return nil, fmt.Errorf("connector id is none for %s", req.Task.Name)
	}
	connector, ok := r.connectors[*req.Task.ConnectorID]
	if !ok {
		return nil, fmt.Errorf("go connector %q is not registered", *req.Task.ConnectorID)
	}

	params := map[string]any{}
	if len(req.Task.Parameters) > 0 {
		if err := json.Unmarshal(req.Task.Parameters, &params); err != nil {
			return nil, fmt.Errorf("could not decode parameters for %s: %w", req.Task.Name, err)
		}
	}
	rendered, err := r.template.Render(params, map[string]any{"steps": req.Steps, "input": req.Input.TemplateVars()})
	if err != nil {
		return nil, err
	}
	renderedParams, ok := rendered.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("templated parameters for %s are not an object (got %T)", req.Task.Name, rendered)
	}

	config := req.Config
	if config == nil {
		config = map[string]any{}
	}

	r.logger.Debugw("running go connector",
		"connector", *req.Task.ConnectorID, "operation", req.Task.Operation)
	result, err := connector.Execute(ctx, config, renderedParams, req.Task.Operation)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}
