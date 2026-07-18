// Package goconnectors runs built-in connectors compiled into the worker.
// These are OUR code — not user code — so they may run in-process; everything
// user-authored still goes to the sandbox. Metadata parity: each builtin has
// an info.json dir in the connectors tree with "runtime": "go" (virtual — the
// registry is the implementation).
package goconnectors

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pelletier/go-toml/v2"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/goconnectors_mock.go -package=mocks . TemplateEngine

// TemplateEngine renders {{ var.steps["node"] }} templates inside params —
// jinja2 semantics (gonja adapter), golden-tested against the python side.
type TemplateEngine interface {
	Render(value any, variables map[string]any) (any, error)
}

// Connector is one built-in implementation. configs is the parsed TOML file,
// params are already templated.
type Connector interface {
	Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error)
}

// Registry implements execution.NodeRuntime for connectors registered at the
// worker's composition root.
type Registry struct {
	logger     logger.Logger
	template   TemplateEngine
	dir        string // connectors tree, for <id>/configs/<name>.toml; "" = no configs
	connectors map[string]Connector
}

func NewRegistry(log logger.Logger, template TemplateEngine, dir string) *Registry {
	return &Registry{
		logger:     log,
		template:   template,
		dir:        dir,
		connectors: map[string]Connector{},
	}
}

func (r *Registry) Register(id string, connector Connector) {
	r.connectors[id] = connector
}

// IDs returns the registered connector ids — the composition root maps each
// of them to this runtime in the resolver.
func (r *Registry) IDs() []string {
	ids := make([]string, 0, len(r.connectors))
	for id := range r.connectors {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *Registry) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
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
	rendered, err := r.template.Render(params, map[string]any{"steps": req.Steps})
	if err != nil {
		return nil, err
	}
	renderedParams, ok := rendered.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("templated parameters for %s are not an object (got %T)", req.Task.Name, rendered)
	}

	config, err := r.loadConfig(*req.Task.ConnectorID, req.Task.Config)
	if err != nil {
		return nil, err
	}

	r.logger.Debugw("running go connector",
		"connector", *req.Task.ConnectorID, "operation", req.Task.Operation)
	result, err := connector.Execute(ctx, config, renderedParams, req.Task.Operation)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (r *Registry) loadConfig(connectorID string, configName *string) (map[string]any, error) {
	config := map[string]any{}
	if r.dir == "" || configName == nil || *configName == "" {
		return config, nil
	}
	path := filepath.Join(r.dir, connectorID, "configs", *configName+".toml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read connector config %s: %w", path, err)
	}
	if err := toml.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("invalid connector config %s: %w", path, err)
	}
	return config, nil
}
