package goconnectors_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/goconnectors"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/goconnectors/mocks"
	"github.com/yuudev14/ytsoar/internal/adapters/templating"
	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type echoConnector struct {
	gotConfigs map[string]any
	gotParams  map[string]any
	gotOp      string
	gotActor   *string
}

func (e *echoConnector) Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error) {
	e.gotConfigs, e.gotParams, e.gotOp, e.gotActor = configs, params, operation, domain.ActorLabel(ctx)
	return map[string]any{"echoed": params["msg"]}, nil
}

func request(t *testing.T, connectorID string, operation string, params map[string]any, steps map[string]any, config *string) execution.ExecutionRequest {
	t.Helper()
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	return execution.ExecutionRequest{
		PlaybookName: "enrich",
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        connectorID + "_node",
			ConnectorID: &connectorID,
			Config:      config,
			Operation:   operation,
			Parameters:  raw,
		},
		Steps:             steps,
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
	}
}

func TestBuiltinRuntimeTemplatesParamsAndPassesConfig(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine())
	echo := &echoConnector{}
	registry.Register("echo", echo)

	req := request(t, "echo", "run",
		map[string]any{"msg": `{{ var.steps["prev node"] }}`},
		map[string]any{"prev node": "hi"}, nil)
	req.Config = map[string]any{"key": "value"}
	raw, err := registry.Execute(context.Background(), req)

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "hi", decoded["echoed"])
	assert.Equal(t, "value", echo.gotConfigs["key"])
	assert.Equal(t, "run", echo.gotOp)
	assert.Contains(t, *echo.gotActor, "playbook: enrich (run ")
}

func TestBuiltinRuntimeUnknownConnector(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine())

	_, err := registry.Execute(context.Background(),
		request(t, "ghost", "run", nil, nil, nil))

	assert.ErrorContains(t, err, "ghost")
}

func TestBuiltinRuntimeIDs(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine())
	registry.Register("http_request", goconnectors.NewHTTPRequestConnector())

	assert.Equal(t, []string{"http_request"}, registry.IDs())
}

func TestHTTPRequestConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "yes", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [1, 2]}`))
	}))
	defer server.Close()

	conn := goconnectors.NewHTTPRequestConnector()
	result, err := conn.Execute(context.Background(),
		map[string]any{"headers": map[string]any{"X-Custom": "yes"}},
		map[string]any{"url": server.URL}, "get_request")

	require.NoError(t, err)
	out := result.(map[string]any)
	assert.Equal(t, http.StatusOK, out["status"])
	assert.Equal(t, map[string]any{"items": []any{float64(1), float64(2)}}, out["body"])
}

func TestHTTPRequestConnectorUnknownOperation(t *testing.T) {
	conn := goconnectors.NewHTTPRequestConnector()

	_, err := conn.Execute(context.Background(), nil, map[string]any{"url": "http://x"}, "post_request")

	assert.ErrorContains(t, err, "post_request")
}

func TestYTSoarConnector(t *testing.T) {
	alertID := uuid.New()
	setup := func(t *testing.T) (*goconnectors.YTSoarConnector, *mocks.MockAlertService) {
		ctrl := gomock.NewController(t)
		service := mocks.NewMockAlertService(ctrl)
		return goconnectors.NewYTSoarConnector(service, mocks.NewMockIncidentService(ctrl)), service
	}

	t.Run("update_alert skips the PATCH when only the status is set", func(t *testing.T) {
		conn, service := setup(t)
		note := "benign"
		service.EXPECT().UpdateStatus(gomock.Any(), alertID,
			alerts.UpdateAlertStatusPayload{Status: "resolved", ClosureNote: &note}, nil).
			Return(domain.Alert{ID: alertID, Status: domain.AlertStatusResolved}, nil)

		out, err := conn.Execute(context.Background(), nil, map[string]any{
			"alert_id": alertID.String(), "status": "resolved", "closure_note": note, "severity": " ", "tags": "",
		}, "update_alert")

		require.NoError(t, err)
		assert.Equal(t, domain.AlertStatusResolved, out.(domain.Alert).Status)
	})
	t.Run("update_alert splits the tags", func(t *testing.T) {
		conn, service := setup(t)
		service.EXPECT().Update(gomock.Any(), alertID, gomock.Any(), nil).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, payload alerts.UpdateAlertPayload, _ *uuid.UUID) (domain.Alert, error) {
				assert.Equal(t, []string{"phishing", "vip"}, *payload.Tags.Value)
				return domain.Alert{ID: alertID}, nil
			})

		_, err := conn.Execute(context.Background(), nil,
			map[string]any{"alert_id": alertID.String(), "tags": "phishing, vip,"}, "update_alert")

		require.NoError(t, err)
	})
	t.Run("create_alert checks the API rules", func(t *testing.T) {
		conn, _ := setup(t)

		_, err := conn.Execute(context.Background(), nil,
			map[string]any{"title": "x", "severity": "urgent", "source_kind": "edr"}, "create_alert")

		assert.ErrorContains(t, err, "Severity")
	})
	t.Run("returns an error for a bad id", func(t *testing.T) {
		conn, _ := setup(t)

		_, err := conn.Execute(context.Background(), nil, map[string]any{"alert_id": "nope"}, "delete_alert")

		assert.ErrorContains(t, err, "alert_id must be a uuid")
	})
}
