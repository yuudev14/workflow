package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/yuudev14/ytsoar/internal/adapters/http/middleware"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func adminContext(t *testing.T, actor *domain.AuthUser) (*AdminHandler, *gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	if actor != nil {
		middleware.SetCurrentUser(c, *actor)
	}

	// A nil service is safe here: every case must fail before reaching it.
	return &AdminHandler{logger: logger.NewNop()}, c, recorder
}

// Deactivating yourself is easy to do by accident and awkward to undo — the
// admin who does it may be the only one who could have reversed it.
func TestDeactivateSelfIsRejected(t *testing.T) {
	actorID := uuid.New()
	handler, c, recorder := adminContext(t, &domain.AuthUser{ID: actorID, Username: "admin"})

	c.Request = httptest.NewRequest(http.MethodDelete, "/api/users/v1/"+actorID.String(), nil)
	c.Params = []gin.Param{{Key: "user_id", Value: actorID.String()}}

	handler.DeactivateUser(c)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "your own account")
}

// gin's binder cannot bind a path param to uuid.UUID, so every handler parses
// it by hand. A miss would reach postgres and come back as a 500.
func TestAdminRejectsMalformedPathIDs(t *testing.T) {
	actor := domain.AuthUser{ID: uuid.New(), Username: "admin"}

	cases := []struct {
		name    string
		param   string
		invoke  func(*AdminHandler, *gin.Context)
		method  string
		hasBody bool
	}{
		{"get user", "user_id", (*AdminHandler).GetUser, http.MethodGet, false},
		{"update user", "user_id", (*AdminHandler).UpdateUser, http.MethodPut, true},
		{"set user roles", "user_id", (*AdminHandler).SetUserRoles, http.MethodPut, true},
		{"deactivate user", "user_id", (*AdminHandler).DeactivateUser, http.MethodDelete, false},
		{"get role", "role_id", (*AdminHandler).GetRole, http.MethodGet, false},
		{"delete role", "role_id", (*AdminHandler).DeleteRole, http.MethodDelete, false},
		{"set role permissions", "role_id", (*AdminHandler).SetRolePermissions, http.MethodPut, true},
		{"get team", "team_id", (*AdminHandler).GetTeam, http.MethodGet, false},
		{"delete team", "team_id", (*AdminHandler).DeleteTeam, http.MethodDelete, false},
		{"set team members", "team_id", (*AdminHandler).SetTeamMembers, http.MethodPut, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler, c, recorder := adminContext(t, &actor)

			var body *strings.Reader
			if tc.hasBody {
				body = strings.NewReader(`{}`)
			} else {
				body = strings.NewReader("")
			}
			c.Request = httptest.NewRequest(tc.method, "/api/x/not-a-uuid", body)
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = []gin.Param{{Key: tc.param, Value: "not-a-uuid"}}

			// The handler holds a nil service, so reaching it panics. Catching
			// that here reports the real defect — validation was skipped —
			// instead of killing the binary and hiding the sibling cases.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("handler reached the service with a malformed %s: %v", tc.param, r)
					}
				}()
				tc.invoke(handler, c)
			}()

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tc.param+" must be a uuid")
		})
	}
}

// Without an authenticated actor there is nobody to attribute the audit row
// to, so the mutation must not proceed.
func TestAdminMutationsRequireAnActor(t *testing.T) {
	handler, c, recorder := adminContext(t, nil)

	c.Request = httptest.NewRequest(http.MethodPost, "/api/users/v1", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateUser(c)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}
