package auth_test

import (
	"context"
	"encoding/json"
	"errors"
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

// oidcProviderWithConfig builds an enabled oidc provider from a partial config,
// filling in the required issuer/client_id so tests only state what they exercise.
func oidcProviderWithConfig(id uuid.UUID, extra map[string]any) auth.AuthProvider {
	extra["issuer"] = "https://idp.example"
	extra["client_id"] = "cid"
	raw, _ := json.Marshal(extra)
	return auth.AuthProvider{
		ID:      id,
		Type:    domain.AuthProviderOIDC,
		Name:    "keycloak",
		Enabled: true,
		Config:  raw,
	}
}

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

// A single IdP group mapping to a list of roles assigns every one of them.
func TestCompleteOIDCOneToManyRoleMapping(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	existing := domain.User{ID: uuid.New(), Username: "bob", AuthProvider: domain.AuthProviderOIDC, IsActive: true}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderWithConfig(pid, map[string]any{
		"group_role_mapping": map[string]any{"soc-admins": []string{"admin", "analyst"}},
	}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Groups: []string{"soc-admins"}}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)

	admin := domain.Role{ID: uuid.New(), Name: "admin"}
	analyst := domain.Role{ID: uuid.New(), Name: "analyst"}
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "admin").Return(admin, nil)
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "analyst").Return(analyst, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), existing.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), existing.ID, admin.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), existing.ID, analyst.ID).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// No mapping table: the IdP emits a value that already is a role name
// (Azure app roles / Okta roles) and it is assigned directly via passthrough.
func TestCompleteOIDCPassthroughRole(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	existing := domain.User{ID: uuid.New(), Username: "bob", AuthProvider: domain.AuthProviderOIDC, IsActive: true}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderWithConfig(pid, map[string]any{
		"groups_claim": "roles",
		"default_role": "viewer",
	}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Groups: []string{"analyst"}}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)

	analyst := domain.Role{ID: uuid.New(), Name: "analyst"}
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "analyst").Return(analyst, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), existing.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), existing.ID, analyst.ID).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// A group that maps to nothing and is not itself a role resolves nothing, so
// the user still lands on the default role rather than losing all access.
func TestCompleteOIDCPassthroughMissFallsToDefault(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	existing := domain.User{ID: uuid.New(), Username: "bob", AuthProvider: domain.AuthProviderOIDC, IsActive: true}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderWithConfig(pid, map[string]any{
		"default_role": "viewer",
	}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Groups: []string{"random-ad-group"}}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)

	// passthrough candidate isn't a real role → skipped; default resolves.
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "random-ad-group").Return(domain.Role{}, errors.New("no rows"))
	viewer := domain.Role{ID: uuid.New(), Name: "viewer"}
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "viewer").Return(viewer, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), existing.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), existing.ID, viewer.ID).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// sync_mode "attributes" hands roles to an admin: a login refreshes the session
// but must never touch the user's roles, even with a matching group claim.
func TestCompleteOIDCSyncModeAttributesLeavesRoles(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	existing := domain.User{ID: uuid.New(), Username: "bob", AuthProvider: domain.AuthProviderOIDC, IsActive: true}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderWithConfig(pid, map[string]any{
		"sync_mode":          "attributes",
		"group_role_mapping": map[string]any{"soc-analysts": "analyst"},
		"default_role":       "viewer",
	}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Groups: []string{"soc-analysts"}}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)
	// deliberately no mockRoles expectations — any role call fails the test.
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// A lookup that fails for any reason other than "no such user" must not fall
// through to JIT — re-provisioning an existing account would trip the
// external_id unique index and report a database outage as a bad login.
func TestCompleteOIDCLookupFailureDoesNotProvision(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "verifier", time.Now().Add(time.Minute))
	dbDown := errors.New("connection refused")

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, true), nil)
	env.mockOIDC.EXPECT().
		Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "verifier").
		Return(auth.OIDCIdentity{Subject: "kc-sub", PreferredUsername: "alice"}, nil)
	env.mockUsers.EXPECT().
		GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, pid.String()+"|kc-sub").
		Return(domain.User{}, dbDown)
	// no Create expectation: gomock fails the test if provisioning is attempted.

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	assert.ErrorIs(t, err, dbDown)
}

func TestCompleteOIDCUsernameLookupFailureDoesNotProvision(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "verifier", time.Now().Add(time.Minute))
	dbDown := errors.New("connection refused")

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderRow(pid, true), nil)
	env.mockOIDC.EXPECT().
		Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "verifier").
		Return(auth.OIDCIdentity{Subject: "kc-sub", PreferredUsername: "alice"}, nil)
	env.mockUsers.EXPECT().
		GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, pid.String()+"|kc-sub").
		Return(domain.User{}, auth.ErrUserNotFound)
	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(domain.User{}, dbDown)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	assert.ErrorIs(t, err, dbDown)
}
