package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type stubPermissions struct {
	set domain.PermissionSet
	err error
}

func (s *stubPermissions) PermissionsFor(_ context.Context, _ uuid.UUID) (domain.PermissionSet, error) {
	return s.set, s.err
}

// runGuarded plays a request through Auth + RequirePermission and reports
// whether the handler behind them ran.
func runGuarded(t *testing.T, src middleware.PermissionSource, module, action string, authenticated bool) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	stub := newStub()
	stub.validAccessToken = "good-access-token"

	handlerRan := false
	router := gin.New()
	router.GET("/probe",
		middleware.Auth(logger.NewNop(), stub),
		middleware.RequirePermission(logger.NewNop(), src, module, action),
		func(c *gin.Context) {
			handlerRan = true
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if authenticated {
		req.Header.Set("Authorization", "Bearer good-access-token")
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec, handlerRan
}

func TestRequirePermissionAllowsGrantedAction(t *testing.T) {
	src := &stubPermissions{set: domain.PermissionSet{
		{Module: domain.ModulePlaybooks, Action: domain.ActionRead},
		{Module: domain.ModulePlaybooks, Action: domain.ActionExecute},
	}}

	rec, ran := runGuarded(t, src, domain.ModulePlaybooks, domain.ActionExecute, true)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, ran, "handler should run when the grant is present")
}

func TestRequirePermissionDeniesMissingAction(t *testing.T) {
	src := &stubPermissions{set: domain.PermissionSet{
		{Module: domain.ModulePlaybooks, Action: domain.ActionRead},
	}}

	rec, ran := runGuarded(t, src, domain.ModulePlaybooks, domain.ActionExecute, true)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.False(t, ran, "handler must not run without the grant")
}

// Holding an action on one module must not grant it on another.
func TestRequirePermissionDeniesAcrossModules(t *testing.T) {
	src := &stubPermissions{set: domain.PermissionSet{
		{Module: domain.ModulePlaybooks, Action: domain.ActionDelete},
	}}

	rec, ran := runGuarded(t, src, domain.ModuleConnectors, domain.ActionDelete, true)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.False(t, ran)
}

func TestRequirePermissionDeniesEmptySet(t *testing.T) {
	src := &stubPermissions{set: domain.PermissionSet{}}

	rec, ran := runGuarded(t, src, domain.ModulePlaybooks, domain.ActionRead, true)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.False(t, ran)
}

// Unauthenticated is 401, not 403: the caller has no identity to judge.
func TestRequirePermissionRejectsUnauthenticated(t *testing.T) {
	src := &stubPermissions{set: domain.PermissionSet{
		{Module: domain.ModulePlaybooks, Action: domain.ActionRead},
	}}

	rec, ran := runGuarded(t, src, domain.ModulePlaybooks, domain.ActionRead, false)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, ran)
}

// A failed lookup must fail closed — this is the one that turns an outage into
// a breach if it regresses.
func TestRequirePermissionFailsClosedOnError(t *testing.T) {
	src := &stubPermissions{err: errors.New("connection refused")}

	rec, ran := runGuarded(t, src, domain.ModulePlaybooks, domain.ActionRead, true)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.False(t, ran, "handler must not run when permissions could not be loaded")
}

// RequirePermission without Auth in front of it must not fall through.
func TestRequirePermissionWithoutAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	src := &stubPermissions{set: domain.PermissionSet{
		{Module: domain.ModulePlaybooks, Action: domain.ActionRead},
	}}

	handlerRan := false
	router := gin.New()
	router.GET("/probe",
		middleware.RequirePermission(logger.NewNop(), src, domain.ModulePlaybooks, domain.ActionRead),
		func(c *gin.Context) {
			handlerRan = true
			c.Status(http.StatusOK)
		},
	)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, handlerRan)
}