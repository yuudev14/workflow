// Package mcp serves the read tools of YTSoar over MCP (streamable HTTP). A tool calls the same
// application service as the REST route and checks the same grant, so a client sees only what
// its user can see.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// NewHandler returns the MCP endpoint. It needs a bearer token (access token or API key). The
// server is stateless, so every request signs in again and no session outlives its token.
func NewHandler(log logger.Logger, services Services) http.Handler {
	h := &handler{log: log, Services: services}
	srv := server.NewMCPServer("ytsoar", "1.0.0")
	h.addTools(srv)

	streamable := server.NewStreamableHTTPServer(srv, server.WithStateLess(true))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := h.verify(r)
		if err != nil {
			h.log.Debugw("rejected mcp token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		streamable.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

// verify reads the Authorization header only. A cookie would open the endpoint to CSRF.
func (h *handler) verify(r *http.Request) (domain.AuthUser, error) {
	scheme, token, _ := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !strings.EqualFold(scheme, "Bearer") || token == "" {
		return domain.AuthUser{}, errors.New("missing bearer token")
	}
	if strings.HasPrefix(token, domain.APIKeyPrefix) {
		return h.Auth.VerifyAPIKey(r.Context(), token)
	}
	return h.Auth.VerifyAccessToken(token)
}

// allow gives the read grant of a module to the user of the request. It fails closed.
func (h *handler) allow(ctx context.Context, module string) error {
	user, ok := ctx.Value(userKey{}).(domain.AuthUser)
	if !ok {
		return errForbidden
	}

	permissions, err := h.Auth.PermissionsFor(ctx, user.ID)
	if err != nil {
		h.log.Errorw("could not load permissions", "user_id", user.ID, "module", module, "error", err)
		return errForbidden
	}
	if !permissions.Has(module, domain.ActionRead) {
		return errForbidden
	}
	return nil
}

// toolError shows the client only the safe message of a classified error.
func (h *handler) toolError(tool string, err error) error {
	kind, msg := apperr.KindOf(err)
	if kind == apperr.Internal {
		h.log.Errorw("mcp tool failed", "tool", tool, "error", err)
		return errors.New("internal error")
	}
	return errors.New(msg)
}

// addTool registers a read tool. The wrapper checks the grant before the tool runs. A failure
// is a tool result with IsError, not a protocol error, so the model can read it.
func addTool[In any](h *handler, srv *server.MCPServer, name, description, module string, run func(context.Context, In) (any, error)) {
	tool := mcp.NewTool(name,
		mcp.WithDescription(description),
		mcp.WithInputSchema[In](),
		mcp.WithReadOnlyHintAnnotation(true),
	)
	srv.AddTool(tool, mcp.NewTypedToolHandler(func(ctx context.Context, _ mcp.CallToolRequest, in In) (*mcp.CallToolResult, error) {
		if err := h.allow(ctx, module); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		out, err := run(ctx, in)
		if err != nil {
			return mcp.NewToolResultError(h.toolError(name, err).Error()), nil
		}
		body, err := json.Marshal(out)
		if err != nil {
			return mcp.NewToolResultError(h.toolError(name, err).Error()), nil
		}
		return mcp.NewToolResultText(string(body)), nil
	}))
}

func (h *handler) addTools(srv *server.MCPServer) {
	addTool(h, srv, "list_alerts", "List alerts, newest first.", domain.ModuleAlerts,
		func(ctx context.Context, in listIn) (any, error) {
			filter := alerts.AlertFilter{Limit: clampLimit(in.Limit)}
			filter.Status = oneOrNone(in.Status)
			filter.Severity = oneOrNone(in.Severity)
			filter.Search = textOrNil(in.Query)
			return h.Alerts.List(ctx, filter)
		})

	addTool(h, srv, "get_alert", "Get one alert with its notes, linked incidents, related alerts and runs.", domain.ModuleAlerts,
		func(ctx context.Context, in idIn) (any, error) {
			id, err := parseID("id", in.ID)
			if err != nil {
				return nil, err
			}
			detail, err := h.Alerts.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
			detail.Runs = detail.Runs[:min(len(detail.Runs), maxRuns)]
			detail.RelatedAlerts = detail.RelatedAlerts[:min(len(detail.RelatedAlerts), maxRuns)]
			return alertView{AlertDetail: detail}, nil
		})

	addTool(h, srv, "list_incidents", "List incidents, newest first.", domain.ModuleIncidents,
		func(ctx context.Context, in listIn) (any, error) {
			filter := incidents.IncidentFilter{Limit: clampLimit(in.Limit)}
			filter.Status = oneOrNone(in.Status)
			filter.Severity = oneOrNone(in.Severity)
			filter.Search = textOrNil(in.Query)
			return h.Incidents.List(ctx, filter)
		})

	addTool(h, srv, "get_incident", "Get one incident with its linked alerts, notes, IOCs and runs.", domain.ModuleIncidents,
		func(ctx context.Context, in idIn) (any, error) {
			id, err := parseID("id", in.ID)
			if err != nil {
				return nil, err
			}
			detail, err := h.Incidents.GetByID(ctx, id)
			if err != nil {
				return nil, err
			}
			detail.Runs = detail.Runs[:min(len(detail.Runs), maxRuns)]
			return incidentView{IncidentDetail: detail}, nil
		})

	addTool(h, srv, "list_playbooks", "List playbooks.", domain.ModulePlaybooks,
		func(ctx context.Context, in listPlaybooksIn) (any, error) {
			filter := playbooks.PlaybookFilter{Name: textOrNil(in.Name)}
			page, err := h.Playbooks.GetPlaybooksData(ctx, 0, clampLimit(in.Limit), filter)
			if err != nil {
				return nil, err
			}
			rows := make([]J, 0, len(page.Entries))
			for _, p := range page.Entries {
				rows = append(rows, J{"id": p.ID, "name": p.Name, "description": p.Description, "trigger_type": p.TriggerType})
			}
			return J{"entries": rows, "total": page.Total}, nil
		})

	addTool(h, srv, "get_playbook", "Get one playbook: its trigger, its steps and how they connect.", domain.ModulePlaybooks,
		func(ctx context.Context, in idIn) (any, error) {
			id, err := parseID("id", in.ID)
			if err != nil {
				return nil, err
			}
			graph, err := h.Playbooks.GetPlaybookGraphById(ctx, id.String())
			if err != nil {
				return nil, err
			}
			if graph == nil {
				return nil, apperr.New(apperr.NotFound, "playbook not found")
			}
			return J{
				"id":           graph.ID,
				"name":         graph.Name,
				"description":  graph.Description,
				"trigger_type": graph.TriggerType,
				"tasks":        pick(graph.Tasks, "id", "name", "connector_name", "operation", "description"),
				"edges":        pick(graph.Edges, "source_id", "destination_id", "source_handle"),
			}, nil
		})

	addTool(h, srv, "list_runs", "List playbook runs, newest first.", domain.ModulePlaybooks,
		func(ctx context.Context, in listRunsIn) (any, error) {
			filter, err := runFilter(in)
			if err != nil {
				return nil, err
			}
			page, err := h.Playbooks.GetPlaybooksHistoryData(ctx, 0, clampLimit(in.Limit), filter)
			if err != nil {
				return nil, err
			}
			rows := make([]J, 0, len(page.Entries))
			for _, run := range page.Entries {
				rows = append(rows, runSummary(run))
			}
			return J{"entries": rows, "total": page.Total}, nil
		})

	addTool(h, srv, "get_run", "Get one playbook run with the status, the error and the result of each task.", domain.ModulePlaybooks,
		func(ctx context.Context, in idIn) (any, error) {
			id, err := parseID("id", in.ID)
			if err != nil {
				return nil, err
			}
			run, err := h.Playbooks.GetPlaybookHistoryById(ctx, id)
			if err != nil {
				return nil, err
			}
			if run == nil {
				return nil, apperr.New(apperr.NotFound, "run not found")
			}
			taskRuns, err := h.Tasks.GetTaskHistoryByPlaybookHistoryId(ctx, id.String(), tasks.TaskHistoryFilter{})
			if err != nil {
				return nil, err
			}

			rows := make([]J, 0, len(taskRuns))
			for _, t := range taskRuns {
				rows = append(rows, J{
					"name":      t.Name,
					"connector": t.ConnectorName,
					"operation": t.Operation,
					"status":    t.Status,
					"error":     t.Error,
					"result":    clip(t.Result),
				})
			}
			summary := runSummary(*run)
			summary["input"] = clip(&run.Input)
			summary["result"] = clip(run.Result)
			summary["tasks"] = rows
			return summary, nil
		})
}
