// Package integration holds the API integration tests, like Rails request
// specs. They send real HTTP requests through the production router, the
// services and a migrated Postgres. Each file tests one area of the API.
//
// Use a new, empty database:
//
//	TEST_DATABASE_URL=postgres://... go test ./tests/integration
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/app"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

const adminUsername, adminPassword = "admin", "api-test-password"

// J is a JSON object.
type J = map[string]any

// The tests share one server.
var (
	serverURL string
	routes    gin.RoutesInfo
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		fmt.Println("skip: TEST_DATABASE_URL is not set")
		return
	}
	os.Exit(run(m, dbURL))
}

func run(m *testing.M, dbURL string) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := db.NewPool(ctx, dbURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	connectorsDir, err := os.MkdirTemp("", "connectors")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(connectorsDir)

	// The defaults, without a .env file. The admin seed gives the tests a user.
	cfg := config.LoadFrom(os.DevNull)
	cfg.AdminUsername, cfg.AdminPassword = adminUsername, adminPassword
	cfg.ConnectorsDir = connectorsDir
	gin.SetMode(gin.TestMode)
	router, err := app.NewAPI(ctx, cfg, logger.NewNop(), pool)
	if err != nil {
		panic(err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	serverURL, routes = server.URL, router.Routes()
	return m.Run()
}

// client is one browser: it keeps its cookies and its access token.
type client struct {
	http  *http.Client
	token string
}

func anonymous(t *testing.T) *client {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	// It does not follow redirects, so a test can check them.
	noRedirect := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client{http: &http.Client{Jar: jar, CheckRedirect: noRedirect}}
}

func signIn(t *testing.T, username, password string) *client {
	c := anonymous(t)
	out := c.expect(t, http.StatusOK, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": password})
	c.token = out.(J)["access_token"].(string)
	return c
}

func admin(t *testing.T) *client { return signIn(t, adminUsername, adminPassword) }

// newUser makes a user with the roles and returns its id and a signed-in client.
func newUser(t *testing.T, roleIDs ...string) (string, *client) {
	username := unique("user")
	user := admin(t).create(t, "/api/users/v1", J{
		"username": username, "email": username + "@ytsoar.test", "password": "password123", "role_ids": roleIDs,
	})
	return user["id"].(string), signIn(t, username, "password123")
}

// send sends one request and returns the response.
func (c *client) send(t *testing.T, method, path string, body *bytes.Buffer, contentType string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, serverURL+path, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	require.NoError(t, err)
	return res
}

// do sends one JSON request and returns the status and the decoded JSON body.
func (c *client) do(t *testing.T, method, path string, body any) (int, any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	res := c.send(t, method, path, &buf, "application/json")
	defer res.Body.Close()
	var out any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// expect sends one request, checks the status and returns the body.
func (c *client) expect(t *testing.T, code int, method, path string, body any) any {
	t.Helper()
	got, out := c.do(t, method, path, body)
	require.Equal(t, code, got, "%s %s: %v", method, path, out)
	return out
}

func (c *client) get(t *testing.T, path string) any {
	t.Helper()
	return c.expect(t, http.StatusOK, http.MethodGet, path, nil)
}

func (c *client) create(t *testing.T, path string, body any) J {
	t.Helper()
	return c.expect(t, http.StatusCreated, http.MethodPost, path, body).(J)
}

// request is one row of a table test.
type request struct {
	name   string
	method string
	path   string
	body   any
	want   int
}

// expectAll sends each request in its own subtest.
func (c *client) expectAll(t *testing.T, requests []request) {
	t.Helper()
	for _, r := range requests {
		t.Run(r.name, func(t *testing.T) { c.expect(t, r.want, r.method, r.path, r.body) })
	}
}

func unique(name string) string { return name + "-" + uuid.NewString()[:8] }

// items returns the objects of a list response. Some lists are in an object:
// {"entries": [...]} or {"tasks": [...]}.
func items(list any) []any {
	if page, ok := list.(J); ok {
		list = page["entries"]
		if list == nil {
			list = page["tasks"]
		}
	}
	out, _ := list.([]any)
	return out
}

// ids returns the "id" of each object in a list.
func ids(list any) []any {
	var out []any
	for _, item := range items(list) {
		out = append(out, item.(J)["id"])
	}
	return out
}

// fill gives each route parameter a random uuid, so the route matches and the
// record does not exist.
var routeParam = regexp.MustCompile(`:[a-z_]+|\*[a-z_]+`)

func fill(route string) string {
	return routeParam.ReplaceAllStringFunc(route, func(string) string { return uuid.NewString() })
}

// public routes need no session.
func public(route string) bool {
	return strings.HasPrefix(route, "/swagger/") ||
		route == "/api/auth/v1/login" || route == "/api/auth/v1/refresh" || route == "/api/auth/v1/logout" ||
		route == "/api/auth/v1/providers" || strings.HasPrefix(route, "/api/auth/v1/oidc/")
}
