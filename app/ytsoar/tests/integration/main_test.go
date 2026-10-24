// Package integration holds the integration tests. TestMain starts the api,
// the worker and the sandbox in this process, with a migrated Postgres. The
// tests send real HTTP requests, and the worker runs the playbooks in the
// sandbox. Each file tests one area.
//
// Use a new, empty database:
//
//	TEST_DATABASE_URL=postgres://... go test -p 1 ./...
//
// Use -p 1: the job queue tests in another package use the same queue.
package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	testPool  *pgxpool.Pool
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
	testPool = pool
	connectorsDir, err := os.MkdirTemp("", "connectors")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(connectorsDir)

	// The defaults, without a .env file. The admin seed gives the tests a user.
	cfg := config.LoadFrom(os.DevNull)
	cfg.AdminUsername, cfg.AdminPassword = adminUsername, adminPassword
	cfg.ConnectorsDir = connectorsDir
	cfg.SMTPHost, cfg.SMTPPort = fakeSMTP()
	gin.SetMode(gin.TestMode)
	router, err := app.NewAPI(ctx, cfg, logger.NewNop(), pool)
	if err != nil {
		panic(err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	serverURL, routes = server.URL, router.Routes()

	// The worker and the sandbox use the connectors of the repository.
	runCfg := cfg
	runCfg.ConnectorsDir = must(filepath.Abs("../../../connectors"))
	runCfg.SandboxAddr = freeAddr()
	runCfg.NodeTimeout = stepTimeout
	sandbox, err := app.NewSandbox(runCfg, logger.NewNop())
	if err != nil {
		panic(err)
	}
	go func() { _ = sandbox.Serve(runCfg.SandboxAddr) }()
	worker, err := app.NewWorker(runCfg, logger.NewNop(), pool)
	if err != nil {
		panic(err)
	}
	if err := worker.Start(ctx); err != nil {
		panic(err)
	}
	defer func() { _ = worker.Stop(context.Background()) }()

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
	out := c.expect(t, http.StatusOK, http.MethodPost, "/api/v1/auth/login", J{"username": username, "password": password})
	c.token = out.(J)["access_token"].(string)
	return c
}

func admin(t *testing.T) *client { return signIn(t, adminUsername, adminPassword) }

// newUser makes a user with the roles and returns its id and a signed-in client.
func newUser(t *testing.T, roleIDs ...string) (string, *client) {
	username := unique("user")
	user := admin(t).create(t, "/api/v1/users", J{
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
	_ = json.NewDecoder(res.Body).Decode(&out) // some responses have no body
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

// freeAddr returns a local address with a free port.
// mailbox gets each email that the API sends to the SMTP server of fakeSMTP
var mailbox = make(chan string, 16)

func fakeSMTP() (string, int) {
	l := must(net.Listen("tcp", "127.0.0.1:0"))
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go serveSMTP(conn)
		}
	}()
	return "127.0.0.1", l.Addr().(*net.TCPAddr).Port
}

// serveSMTP knows only what net/smtp.SendMail needs. It offers no STARTTLS and no AUTH.
func serveSMTP(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	reply := func(line string) { _, _ = fmt.Fprint(conn, line+"\r\n") }
	reply("220 test")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		switch cmd := strings.ToUpper(strings.TrimSpace(line)); cmd {
		case "DATA":
			reply("354 send the message")
			var msg strings.Builder
			for line, err = r.ReadString('\n'); err == nil && line != ".\r\n"; line, err = r.ReadString('\n') {
				msg.WriteString(line)
			}
			mailbox <- msg.String()
			reply("250 ok")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

var linkToken = regexp.MustCompile(`/set-password\?token=([A-Za-z0-9_-]+)`)

// linkFor waits for the email to the address and returns the token of its link.
func linkFor(t *testing.T, to string) string {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg := <-mailbox:
			if strings.Contains(msg, "To: "+to+"\r\n") {
				return linkToken.FindStringSubmatch(msg)[1]
			}
		case <-timeout:
			t.Fatalf("no email to %s", to)
		}
	}
}

func freeAddr() string {
	l := must(net.Listen("tcp", "127.0.0.1:0"))
	defer l.Close()
	return l.Addr().String()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func toJSON(v any) string { return string(must(json.Marshal(v))) }

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
		route == "/api/v1/auth/login" || route == "/api/v1/auth/refresh" || route == "/api/v1/auth/logout" ||
		route == "/api/v1/auth/providers" || strings.HasPrefix(route, "/api/v1/auth/oidc/") ||
		strings.HasPrefix(route, "/api/v1/auth/password/") ||
		strings.HasPrefix(route, "/api/v1/webhooks/")
}
