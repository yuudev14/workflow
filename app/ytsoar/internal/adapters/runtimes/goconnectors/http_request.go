package goconnectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxResponseBytes limits how much of a response body is read. The body goes into the
// task-history result column, so an unlimited read would blow up the worker memory and the
// row.
const maxResponseBytes = 10 << 20

// HTTPRequestConnector is the Go builtin http client. The result has the same shape as
// http_request_js: {"status": <code>, "body": <json-or-text>}.
type HTTPRequestConnector struct {
	client *http.Client
}

func NewHTTPRequestConnector() *HTTPRequestConnector {
	return &HTTPRequestConnector{
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// methods maps each operation to its HTTP method.
var methods = map[string]string{
	"get_request":    http.MethodGet,
	"post_request":   http.MethodPost,
	"put_request":    http.MethodPut,
	"delete_request": http.MethodDelete,
}

func (c *HTTPRequestConnector) Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error) {
	method, ok := methods[operation]
	if !ok {
		return nil, fmt.Errorf("operation (%s) does not exist in HTTPRequestConnector", operation)
	}
	return c.request(ctx, method, configs, params)
}

// request sends params["body"] as it is. params["headers"] is a JSON object (the templates
// make it a string) and wins over the headers of the config.
func (c *HTTPRequestConnector) request(ctx context.Context, method string, configs map[string]any, params map[string]any) (any, error) {
	url, _ := params["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url parameter is required")
	}
	body, _ := params["body"].(string)
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != "" && json.Valid([]byte(body)) {
		req.Header.Set("Content-Type", "application/json")
	}
	paramHeaders, err := headersOf(params["headers"])
	if err != nil {
		return nil, err
	}
	configHeaders, _ := configs["headers"].(map[string]any)
	for _, headers := range []map[string]any{configHeaders, paramHeaders} {
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

	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		parsed = string(raw)
	}
	return map[string]any{"status": resp.StatusCode, "body": parsed}, nil
}

func headersOf(value any) (map[string]any, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		return v, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		headers := map[string]any{}
		if err := json.Unmarshal([]byte(v), &headers); err != nil {
			return nil, fmt.Errorf("headers must be a JSON object: %w", err)
		}
		return headers, nil
	default:
		return nil, fmt.Errorf("headers must be a JSON object")
	}
}
