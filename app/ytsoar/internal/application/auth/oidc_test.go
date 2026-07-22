package auth_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/token"
)

// oidcProviderRow builds an enabled oidc provider whose group map sends
// soc-analysts → analyst, with viewer as the default.
func oidcProviderRow(id uuid.UUID, allowJIT bool) auth.AuthProvider {
	cfg, _ := json.Marshal(map[string]any{
		"issuer":             "https://idp.example",
		"client_id":          "cid",
		"allow_jit":          allowJIT,
		"group_role_mapping": map[string]string{"soc-analysts": "analyst"},
		"default_role":       "viewer",
	})
	return auth.AuthProvider{
		ID:      id,
		Type:    domain.AuthProviderOIDC,
		Name:    "keycloak",
		Enabled: true,
		Config:  cfg,
	}
}

func stateCookie(t *testing.T, pid uuid.UUID, state, verifier string, exp time.Time) string {
	t.Helper()
	tok, err := token.GenerateToken(jwt.MapClaims{
		"typ":      "oidc_state",
		"pid":      pid.String(),
		"state":    state,
		"verifier": verifier,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	}, testSecret)
	require.NoError(t, err)
	return tok
}

func TestCompleteOIDCStateMismatchRejected(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "real-state", "verifier", time.Now().Add(time.Minute))

	// query state differs from the sealed one — the CSRF signal.
	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "forged-state", cookie)
	assert.ErrorIs(t, err, auth.ErrOIDCState)
}

func TestCompleteOIDCExpiredStateRejected(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(-time.Minute))

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	assert.ErrorIs(t, err, auth.ErrOIDCState)
}

func TestCompleteOIDCWrongProviderInCookieRejected(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	// cookie sealed for a different provider than the callback path names.
	cookie := stateCookie(t, uuid.New(), "s", "v", time.Now().Add(time.Minute))

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	assert.ErrorIs(t, err, auth.ErrOIDCState)
}

func TestCompleteOIDCJITProvisionsUser(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "verifier", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, true), nil)
	env.mockOIDC.EXPECT().
		Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "verifier").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Email: "alice@corp", PreferredUsername: "alice", Groups: []string{"soc-analysts"}}, nil)

	// no existing account for this subject → JIT.
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).
		Return(domain.User{}, auth.ErrUserNotFound)
	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(domain.User{}, auth.ErrUserNotFound)

	created := domain.User{ID: uuid.New(), Username: "alice", AuthProvider: domain.AuthProviderOIDC, IsActive: true}
	env.mockUsers.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(auth.CreateUserParams{})).
		DoAndReturn(func(_ context.Context, p auth.CreateUserParams) (domain.User, error) {
			assert.Equal(t, domain.AuthProviderOIDC, p.AuthProvider)
			require.NotNil(t, p.ExternalID)
			assert.Equal(t, externalID, *p.ExternalID)
			assert.Nil(t, p.PasswordHash, "OIDC users must have no local password")
			return created, nil
		})

	// role sync: soc-analysts → analyst, replaced wholesale in a tx.
	analyst := domain.Role{ID: uuid.New(), Name: "analyst"}
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "analyst").Return(analyst, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), created.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), created.ID, analyst.ID).Return(nil)

	env.mockTokens.EXPECT().Insert(gomock.Any(), created.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), created.ID).Return(nil)

	pair, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
}

func TestCompleteOIDCExistingUserSyncsDefaultRole(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"

	existing := domain.User{ID: uuid.New(), Username: "bob", AuthProvider: domain.AuthProviderOIDC, IsActive: true}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, false), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub"}, nil) // no groups → default role
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)

	viewer := domain.Role{ID: uuid.New(), Name: "viewer"}
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "viewer").Return(viewer, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), existing.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), existing.ID, viewer.ID).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	// mockUsers.Create is never set — a match by external_id must not JIT again.
	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

func TestCompleteOIDCJITDisabledRejected(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, false), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub"}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).
		Return(domain.User{}, auth.ErrUserNotFound)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	assert.ErrorIs(t, err, auth.ErrJITDisabled)
}

func TestCompleteOIDCInactiveUserRejected(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"

	inactive := domain.User{ID: uuid.New(), Username: "gone", AuthProvider: domain.AuthProviderOIDC, IsActive: false}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, true), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub"}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(inactive, nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	assert.ErrorIs(t, err, auth.ErrOIDCState)
}

func TestStartOIDCBuildsRedirectAndStateCookie(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, true), nil)
	env.mockOIDC.EXPECT().
		AuthCodeURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("https://idp.example/authorize?state=abc", nil)

	redirect, cookie, err := env.service.StartOIDC(context.Background(), pid)
	require.NoError(t, err)
	assert.Contains(t, redirect, "https://idp.example/authorize")

	// the cookie must be a well-formed, verifiable oidc_state token for this pid.
	claims, err := token.Parse(cookie, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "oidc_state", claims["typ"])
	assert.Equal(t, pid.String(), claims["pid"])
	assert.NotEmpty(t, claims["verifier"])
}
