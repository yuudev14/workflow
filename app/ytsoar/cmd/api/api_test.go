package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// The API tests send real HTTP requests through the production router, the
// services and a migrated Postgres, like Rails request specs. Use a new, empty
// database: TEST_DATABASE_URL=postgres://... go test ./cmd/api
const testAdmin, testPassword = "admin", "api-test-password"

type J = map[string]any

type server struct {
	url    string
	routes gin.RoutesInfo
}

type client struct {
	t     *testing.T
	srv   *server
	http  *http.Client
	token string
}

func newTestServer(t *testing.T) *server {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithCancel(context.Background())
	pool, err := db.NewPool(ctx, dbURL)
	require.NoError(t, err)

	// The defaults, without a .env file. The admin seed gives the tests a user.
	cfg := config.LoadFrom(os.DevNull)
	cfg.AdminUsername, cfg.AdminPassword = testAdmin, testPassword
	cfg.ConnectorsDir = t.TempDir()
	gin.SetMode(gin.TestMode)
	router, err := newRouter(ctx, cfg, logger.NewNop(), pool)
	require.NoError(t, err)
	ts := httptest.NewServer(router)
	t.Cleanup(func() {
		ts.Close()
		cancel()
		pool.Close()
	})
	return &server{url: ts.URL, routes: router.Routes()}
}

// anonymous returns a client with no session. It keeps cookies, like a browser.
func (s *server) anonymous(t *testing.T) *client {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	// It does not follow redirects, so a test can check them.
	noRedirect := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client{t: t, srv: s, http: &http.Client{Jar: jar, CheckRedirect: noRedirect}}
}

// signIn returns a client with the session of the user.
func (s *server) signIn(t *testing.T, username, password string) *client {
	c := s.anonymous(t)
	out := c.expect(http.StatusOK, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": password})
	c.token = out.(J)["access_token"].(string)
	return c
}

func (s *server) admin(t *testing.T) *client { return s.signIn(t, testAdmin, testPassword) }

// do sends one JSON request and returns the status and the decoded JSON body.
func (c *client) do(method, path string, body any) (int, any) {
	c.t.Helper()
	var reader bytes.Buffer
	if body != nil {
		require.NoError(c.t, json.NewEncoder(&reader).Encode(body))
	}
	req, err := http.NewRequest(method, c.srv.url+path, &reader)
	require.NoError(c.t, err)
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer res.Body.Close()
	var out any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// expect sends one request, checks the status and returns the body.
func (c *client) expect(code int, method, path string, body any) any {
	c.t.Helper()
	got, out := c.do(method, path, body)
	require.Equal(c.t, code, got, "%s %s: %v", method, path, out)
	return out
}

func (c *client) create(path string, body any) J {
	c.t.Helper()
	return c.expect(http.StatusCreated, http.MethodPost, path, body).(J)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
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

// path gives each route parameter a random uuid, so the route matches and
// the record does not exist.
var routeParam = regexp.MustCompile(`:[a-z_]+|\*[a-z_]+`)

func path(route string) string {
	return routeParam.ReplaceAllStringFunc(route, func(string) string { return uuid.NewString() })
}

// public routes need no session.
func public(route string) bool {
	return strings.HasPrefix(route, "/swagger/") ||
		route == "/api/auth/v1/login" || route == "/api/auth/v1/refresh" || route == "/api/auth/v1/logout" ||
		route == "/api/auth/v1/providers" || strings.HasPrefix(route, "/api/auth/v1/oidc/")
}

func TestEveryRouteNeedsASession(t *testing.T) {
	srv := newTestServer(t)
	anonymous := srv.anonymous(t)
	for _, r := range srv.routes {
		if public(r.Path) {
			continue
		}
		code, _ := anonymous.do(r.Method, path(r.Path), J{})
		assert.Equal(t, http.StatusUnauthorized, code, "%s %s", r.Method, r.Path)
	}
}

func TestEveryRouteNeedsAGrant(t *testing.T) {
	srv := newTestServer(t)
	username := unique("no-roles")
	srv.admin(t).create("/api/users/v1", J{"username": username, "email": username + "@ytsoar.test", "password": "password123"})
	user := srv.signIn(t, username, "password123")

	// Every user can read their own profile and the list of assignable users.
	// The WebSocket uses the refresh cookie, not the token, so this test skips it.
	open := map[string]bool{"/api/auth/v1/me": true, "/api/users/v1/assignable": true, "/ws/playbook": true}
	for _, r := range srv.routes {
		if public(r.Path) || open[r.Path] {
			continue
		}
		code, _ := user.do(r.Method, path(r.Path), J{})
		assert.Equal(t, http.StatusForbidden, code, "%s %s", r.Method, r.Path)
	}
	user.expect(http.StatusOK, http.MethodGet, "/api/auth/v1/me", nil)
	user.expect(http.StatusOK, http.MethodGet, "/api/users/v1/assignable", nil)
}

func TestAuth(t *testing.T) {
	srv := newTestServer(t)
	anonymous := srv.anonymous(t)

	anonymous.expect(http.StatusBadRequest, http.MethodPost, "/api/auth/v1/login", J{"username": testAdmin})
	anonymous.expect(http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": testAdmin, "password": "wrong"})
	anonymous.expect(http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)
	anonymous.expect(http.StatusOK, http.MethodGet, "/api/auth/v1/providers", nil)
	anonymous.expect(http.StatusBadRequest, http.MethodGet, "/api/auth/v1/oidc/not-a-uuid/start", nil)

	admin := srv.admin(t)
	me := admin.expect(http.StatusOK, http.MethodGet, "/api/auth/v1/me", nil).(J)
	assert.Equal(t, testAdmin, me["user"].(J)["username"])
	assert.Contains(t, me["roles"], "admin")

	// The refresh cookie gives a new access token, until the user signs out.
	refreshed := admin.expect(http.StatusOK, http.MethodPost, "/api/auth/v1/refresh", nil).(J)
	assert.NotEmpty(t, refreshed["access_token"])
	admin.expect(http.StatusOK, http.MethodPost, "/api/auth/v1/logout", nil)
	admin.expect(http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/refresh", nil)
}

func TestPlaybooks(t *testing.T) {
	admin := newTestServer(t).admin(t)
	const base = "/api/playbooks/v1"

	admin.expect(http.StatusBadRequest, http.MethodPost, base, J{})
	name := unique("playbook")
	id := admin.expect(http.StatusOK, http.MethodPost, base, J{"name": name}).(J)["id"].(string)

	admin.expect(http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
	graph := admin.expect(http.StatusOK, http.MethodGet, base+"/"+id, nil).(J)
	assert.Equal(t, name, graph["name"])

	list := admin.expect(http.StatusOK, http.MethodGet, base+"?name="+name, nil)
	assert.Equal(t, []any{id}, ids(list))
	admin.expect(http.StatusOK, http.MethodGet, base+"/summary", nil)

	admin.expect(http.StatusOK, http.MethodPut, base+"/"+id, J{"description": "changed"})
	assert.Equal(t, "changed", admin.expect(http.StatusOK, http.MethodGet, base+"/"+id, nil).(J)["description"])

	// The graph: two tasks and one edge.
	admin.expect(http.StatusOK, http.MethodPut, base+"/tasks/"+id, J{
		"nodes": []J{
			{"name": "start", "operation": "start", "parameters": J{}},
			{"name": "next", "operation": "transform", "parameters": J{}},
		},
		"edges": J{"start": []string{"next"}},
	})
	tasks := items(admin.expect(http.StatusOK, http.MethodGet, base+"/"+id+"/tasks", nil))
	assert.Len(t, tasks, 2)

	// The browser gets the run status on a WebSocket. It signs in with the refresh cookie.
	wsURL := "ws" + strings.TrimPrefix(admin.srv.url, "http") + "/ws/playbook"
	cookies := admin.http.Jar.Cookies(must(url.Parse(admin.srv.url)))
	header := http.Header{"Origin": {"http://localhost:9999"}}
	for _, cookie := range cookies {
		header.Add("Cookie", cookie.String())
	}
	socket, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer socket.Close()

	// The API only queues the run. The worker runs it later.
	admin.expect(http.StatusBadRequest, http.MethodPost, base+"/trigger/not-a-uuid", nil)
	admin.expect(http.StatusNotFound, http.MethodPost, base+"/trigger/"+uuid.NewString(), nil)
	run := admin.expect(http.StatusAccepted, http.MethodPost, base+"/trigger/"+id, nil).(J)

	var event J
	socket.SetReadDeadline(time.Now().Add(5 * time.Second))
	require.NoError(t, socket.ReadJSON(&event))
	assert.Equal(t, "playbook_status", event["event"])
	assert.Equal(t, run["playbook_history_id"], event["data"].(J)["id"])
	entries := items(admin.expect(http.StatusOK, http.MethodGet, base+"/history?playbook_id="+id, nil))
	require.Len(t, entries, 1)
	assert.Equal(t, run["playbook_history_id"], entries[0].(J)["id"])
	assert.Equal(t, "queued", entries[0].(J)["status"])

	runTasks := items(admin.expect(http.StatusOK, http.MethodGet, base+"/history/"+entries[0].(J)["id"].(string)+"/tasks", nil))
	assert.Len(t, runTasks, 2)
}

func TestAlerts(t *testing.T) {
	admin := newTestServer(t).admin(t)
	const base = "/api/alerts/v1"

	admin.expect(http.StatusBadRequest, http.MethodPost, base, J{"title": "x", "severity": "huge", "source_kind": "edr"})
	title := unique("alert")
	id := admin.create(base, J{"title": title, "severity": "high", "source_kind": "edr", "tags": []string{"api-test"}})["id"].(string)
	admin.create(base+"/batch", J{"alerts": []J{{"title": unique("batch"), "severity": "low", "source_kind": "email"}}})

	admin.expect(http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
	alert := admin.expect(http.StatusOK, http.MethodGet, base+"/"+id, nil).(J)
	assert.Equal(t, title, alert["title"])

	list := admin.expect(http.StatusOK, http.MethodGet, base+"?q="+title, nil)
	assert.Equal(t, []any{id}, ids(list))
	admin.expect(http.StatusBadRequest, http.MethodGet, base+"?severity=huge", nil)
	admin.expect(http.StatusOK, http.MethodGet, base+"/summary", nil)

	admin.expect(http.StatusOK, http.MethodPatch, base+"/"+id, J{"severity": "critical"})
	admin.expect(http.StatusBadRequest, http.MethodPatch, base+"/"+id+"/status", J{"status": "done"})
	admin.expect(http.StatusOK, http.MethodPatch, base+"/"+id+"/status", J{"status": "investigating"})
	alert = admin.expect(http.StatusOK, http.MethodGet, base+"/"+id, nil).(J)
	assert.Equal(t, "critical", alert["severity"])
	assert.Equal(t, "investigating", alert["status"])

	admin.expect(http.StatusBadRequest, http.MethodPost, base+"/"+id+"/notes", J{"body": ""})
	noteID := admin.create(base+"/"+id+"/notes", J{"body": "first"})["id"].(string)
	admin.expect(http.StatusOK, http.MethodPatch, base+"/"+id+"/notes/"+noteID, J{"body": "changed"})
	notes := items(admin.expect(http.StatusOK, http.MethodGet, base+"/"+id+"/notes", nil))
	require.Len(t, notes, 1)
	assert.Equal(t, "changed", notes[0].(J)["body"])
	admin.expect(http.StatusNoContent, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)
	admin.expect(http.StatusNotFound, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)

	playbookID := admin.expect(http.StatusOK, http.MethodPost, "/api/playbooks/v1", J{"name": unique("on-alert")}).(J)["id"].(string)
	admin.expect(http.StatusBadRequest, http.MethodPost, base+"/run", J{"playbook_id": playbookID, "record_ids": []string{}})
	admin.expect(http.StatusAccepted, http.MethodPost, base+"/run", J{"playbook_id": playbookID, "record_ids": []string{id}})

	incident := admin.create(base+"/"+id+"/escalate", J{"title": unique("escalated")})
	assert.NotEmpty(t, incident["id"])
}

func TestIncidents(t *testing.T) {
	admin := newTestServer(t).admin(t)
	const base = "/api/incidents/v1"

	admin.expect(http.StatusBadRequest, http.MethodPost, base, J{"title": "x"})
	title := unique("incident")
	id := admin.create(base, J{"title": title, "severity": "medium"})["id"].(string)

	admin.expect(http.StatusNotFound, http.MethodGet, base+"/"+uuid.NewString(), nil)
	incident := admin.expect(http.StatusOK, http.MethodGet, base+"/"+id, nil).(J)
	assert.Equal(t, title, incident["title"])

	list := admin.expect(http.StatusOK, http.MethodGet, base+"?q="+title, nil)
	assert.Equal(t, []any{id}, ids(list))
	admin.expect(http.StatusOK, http.MethodGet, base+"/summary", nil)

	admin.expect(http.StatusOK, http.MethodPatch, base+"/"+id, J{"severity": "high"})
	admin.expect(http.StatusBadRequest, http.MethodPatch, base+"/"+id+"/status", J{"status": "done"})
	admin.expect(http.StatusOK, http.MethodPatch, base+"/"+id+"/status", J{"status": "investigating"})
	incident = admin.expect(http.StatusOK, http.MethodGet, base+"/"+id, nil).(J)
	assert.Equal(t, "high", incident["severity"])
	assert.Equal(t, "investigating", incident["status"])

	noteID := admin.create(base+"/"+id+"/notes", J{"body": "first"})["id"].(string)
	admin.expect(http.StatusOK, http.MethodPatch, base+"/"+id+"/notes/"+noteID, J{"body": "changed"})
	assert.Len(t, items(admin.expect(http.StatusOK, http.MethodGet, base+"/"+id+"/notes", nil)), 1)
	admin.expect(http.StatusNoContent, http.MethodDelete, base+"/"+id+"/notes/"+noteID, nil)

	alertID := admin.create("/api/alerts/v1", J{"title": unique("linked"), "severity": "low", "source_kind": "dlp"})["id"].(string)
	admin.expect(http.StatusBadRequest, http.MethodPost, base+"/"+id+"/alerts", J{"alert_id": "not-a-uuid"})
	admin.expect(http.StatusNoContent, http.MethodPost, base+"/"+id+"/alerts", J{"alert_id": alertID})
	admin.expect(http.StatusNoContent, http.MethodDelete, base+"/"+id+"/alerts/"+alertID, nil)

	playbookID := admin.expect(http.StatusOK, http.MethodPost, "/api/playbooks/v1", J{"name": unique("on-incident")}).(J)["id"].(string)
	admin.expect(http.StatusAccepted, http.MethodPost, base+"/run", J{"playbook_id": playbookID, "record_ids": []string{id}})
}

func TestUsersRolesAndTeams(t *testing.T) {
	srv := newTestServer(t)
	admin := srv.admin(t)

	// Roles
	admin.expect(http.StatusBadRequest, http.MethodPost, "/api/roles/v1", J{})
	roleName := unique("reader")
	roleID := admin.create("/api/roles/v1", J{"name": roleName, "permissions": J{"alerts": []string{"read"}}})["id"].(string)
	admin.expect(http.StatusConflict, http.MethodPost, "/api/roles/v1", J{"name": roleName})
	admin.expect(http.StatusNotFound, http.MethodGet, "/api/roles/v1/"+uuid.NewString(), nil)
	admin.expect(http.StatusOK, http.MethodGet, "/api/roles/v1/"+roleID, nil)
	assert.Contains(t, ids(admin.expect(http.StatusOK, http.MethodGet, "/api/roles/v1", nil)), roleID)
	admin.expect(http.StatusOK, http.MethodPut, "/api/roles/v1/"+roleID, J{"description": "reads alerts"})
	admin.expect(http.StatusOK, http.MethodPut, "/api/roles/v1/"+roleID+"/permissions", J{"permissions": J{"alerts": []string{"read"}, "incidents": []string{"read"}}})

	// Users. A role gives its grants to the user.
	username := unique("analyst")
	admin.expect(http.StatusBadRequest, http.MethodPost, "/api/users/v1", J{"username": username, "email": "bad", "password": "password123"})
	userID := admin.create("/api/users/v1", J{"username": username, "email": username + "@ytsoar.test", "password": "password123"})["id"].(string)
	admin.expect(http.StatusConflict, http.MethodPost, "/api/users/v1", J{"username": username, "email": username + "@ytsoar.test", "password": "password123"})
	admin.expect(http.StatusNotFound, http.MethodGet, "/api/users/v1/"+uuid.NewString(), nil)
	admin.expect(http.StatusOK, http.MethodGet, "/api/users/v1/"+userID, nil)
	assert.Contains(t, ids(admin.expect(http.StatusOK, http.MethodGet, "/api/users/v1?search="+username, nil)), userID)
	admin.expect(http.StatusOK, http.MethodPut, "/api/users/v1/"+userID, J{"first_name": "Ana"})
	adminEmail := admin.expect(http.StatusOK, http.MethodGet, "/api/auth/v1/me", nil).(J)["user"].(J)["email"]
	admin.expect(http.StatusConflict, http.MethodPut, "/api/users/v1/"+userID, J{"email": adminEmail})

	analyst := srv.signIn(t, username, "password123")
	analyst.expect(http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)
	admin.expect(http.StatusOK, http.MethodPut, "/api/users/v1/"+userID+"/roles", J{"role_ids": []string{roleID}})
	analyst.expect(http.StatusOK, http.MethodGet, "/api/alerts/v1", nil)
	analyst.expect(http.StatusForbidden, http.MethodPost, "/api/alerts/v1", J{"title": "x", "severity": "low", "source_kind": "edr"})

	admin.expect(http.StatusBadRequest, http.MethodPut, "/api/users/v1/"+userID+"/password", J{"password": "short"})
	admin.expect(http.StatusNoContent, http.MethodPut, "/api/users/v1/"+userID+"/password", J{"password": "password456"})
	srv.anonymous(t).expect(http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": "password123"})
	srv.signIn(t, username, "password456")

	// Teams. A team gives its roles to its members.
	admin.expect(http.StatusBadRequest, http.MethodPost, "/api/teams/v1", J{})
	teamName := unique("blue")
	teamID := admin.create("/api/teams/v1", J{"name": teamName})["id"].(string)
	admin.expect(http.StatusConflict, http.MethodPost, "/api/teams/v1", J{"name": teamName})
	admin.expect(http.StatusNotFound, http.MethodGet, "/api/teams/v1/"+uuid.NewString(), nil)
	admin.expect(http.StatusOK, http.MethodGet, "/api/teams/v1/"+teamID, nil)
	assert.Contains(t, ids(admin.expect(http.StatusOK, http.MethodGet, "/api/teams/v1", nil)), teamID)
	admin.expect(http.StatusOK, http.MethodPut, "/api/teams/v1/"+teamID, J{"description": "the blue team"})
	admin.expect(http.StatusOK, http.MethodPut, "/api/teams/v1/"+teamID+"/members", J{"member_ids": []string{userID}})
	admin.expect(http.StatusOK, http.MethodPut, "/api/teams/v1/"+teamID+"/roles", J{"role_ids": []string{roleID}})
	admin.expect(http.StatusOK, http.MethodPut, "/api/users/v1/"+userID+"/roles", J{"role_ids": []string{}})
	analyst.expect(http.StatusOK, http.MethodGet, "/api/alerts/v1", nil)
	admin.expect(http.StatusNoContent, http.MethodDelete, "/api/teams/v1/"+teamID, nil)
	analyst.expect(http.StatusForbidden, http.MethodGet, "/api/alerts/v1", nil)

	// A deactivated user cannot sign in.
	admin.expect(http.StatusNoContent, http.MethodDelete, "/api/users/v1/"+userID, nil)
	srv.anonymous(t).expect(http.StatusUnauthorized, http.MethodPost, "/api/auth/v1/login", J{"username": username, "password": "password456"})
	admin.expect(http.StatusNoContent, http.MethodDelete, "/api/roles/v1/"+roleID, nil)

	// Every change above is in the audit log.
	assert.NotEmpty(t, items(admin.expect(http.StatusOK, http.MethodGet, "/api/audit/v1", nil)))
}

func TestAuthProviders(t *testing.T) {
	srv := newTestServer(t)
	admin := srv.admin(t)
	const base = "/api/auth-providers/v1"

	admin.expect(http.StatusBadRequest, http.MethodPost, base, J{"type": "oidc"})
	provider := admin.create(base, J{"type": "oidc", "name": unique("keycloak"), "enabled": false, "config": J{
		"issuer": "http://localhost:8180/realms/test", "client_id": "ytsoar", "client_secret": "secret",
	}})
	id := provider["id"].(string)
	assert.Contains(t, ids(admin.expect(http.StatusOK, http.MethodGet, base, nil)), id)
	admin.expect(http.StatusOK, http.MethodPut, base+"/"+id, J{"name": unique("renamed")})
	admin.expect(http.StatusNotFound, http.MethodPut, base+"/"+uuid.NewString(), J{"name": "x"})

	// A disabled provider is not on the login page and cannot start a login.
	assert.NotContains(t, ids(srv.anonymous(t).expect(http.StatusOK, http.MethodGet, "/api/auth/v1/providers", nil)), id)
	// A callback without the signed state cookie does not sign in.
	for _, step := range []string{"/start", "/callback?code=x&state=y"} {
		res, err := srv.anonymous(t).http.Get(srv.url + "/api/auth/v1/oidc/" + id + step)
		require.NoError(t, err)
		res.Body.Close()
		assert.Contains(t, res.Header.Get("Location"), "/login?error=", step)
	}
}

// upload sends a file in the multipart field "file".
func (c *client) upload(path string, file []byte) (int, any) {
	c.t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "connector.zip")
	require.NoError(c.t, err)
	part.Write(file)
	require.NoError(c.t, form.Close())
	req, err := http.NewRequest(http.MethodPost, c.srv.url+path, &body)
	require.NoError(c.t, err)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	res, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer res.Body.Close()
	var out any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// zipDir zips the files of a connector folder.
func zipDir(t *testing.T, dir string) []byte {
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	require.NoError(t, filepath.WalkDir(dir, func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name, _ := filepath.Rel(dir, file)
		w, err := archive.Create(filepath.ToSlash(name))
		if err != nil {
			return err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}))
	require.NoError(t, archive.Close())
	return buf.Bytes()
}

func TestConnectors(t *testing.T) {
	admin := newTestServer(t).admin(t)
	const base = "/api/connectors/v1"

	admin.expect(http.StatusBadRequest, http.MethodPost, base, nil)
	code, _ := admin.upload(base, []byte("not a zip"))
	assert.Equal(t, http.StatusBadRequest, code)

	code, out := admin.upload(base, zipDir(t, "../../../connectors/sample_js"))
	require.Equal(t, http.StatusCreated, code, out)
	assert.Contains(t, ids(admin.expect(http.StatusOK, http.MethodGet, base, nil)), "sample_js")
	admin.expect(http.StatusOK, http.MethodGet, base+"/sample_js", nil)

	admin.expect(http.StatusNoContent, http.MethodDelete, base+"/sample_js", nil)
	admin.expect(http.StatusNotFound, http.MethodGet, base+"/sample_js", nil)
	admin.expect(http.StatusNotFound, http.MethodDelete, base+"/sample_js", nil)
}
