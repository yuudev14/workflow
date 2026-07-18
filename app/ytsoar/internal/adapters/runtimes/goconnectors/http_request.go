package goconnectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxResponseBytes caps how much of a response body is read — the body lands
// in the task-history result column, so an unbounded read would balloon both
// worker memory and the row.
const maxResponseBytes = 10 << 20

// HTTPRequestConnector is the Go builtin http client — result shape matches
// http_request_js: {"status": <code>, "body": <json-or-text>}.
type HTTPRequestConnector struct {
	client *http.Client
}

func NewHTTPRequestConnector() *HTTPRequestConnector {
	return &HTTPRequestConnector{
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *HTTPRequestConnector) Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error) {
	switch operation {
	case "get_request":
		return c.getRequest(ctx, configs, params)
	default:
		return nil, fmt.Errorf("operation (%s) does not exist in HTTPRequestConnector", operation)
	}
}

func (c *HTTPRequestConnector) getRequest(ctx context.Context, configs map[string]any, params map[string]any) (any, error) {
	url, _ := params["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url parameter is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if headers, ok := configs["headers"].(map[string]any); ok {
		for key, value := range headers {
			if s, ok := value.(string); ok {
				req.Header.Set(key, s)
			}
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxResponseBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxResponseBytes)
	}

	var body any
	if err := json.Unmarshal(raw, &body); err != nil {
		body = string(raw)
	}
	return map[string]any{"status": resp.StatusCode, "body": body}, nil
}
