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

// oidcProviderWithConfig builds an enabled oidc provider from a partial config. It fills
// issuer and client_id, so tests state only what they test.
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

// oidcProviderRow builds an enabled oidc provider where soc-analysts -> analyst, and
// viewer is the default.
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

	// the query state differs from the sealed one: the CSRF signal
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

	// mockUsers.Create is not set: a match by external_id must not create the user again
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

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).
		Return(oidcProviderWithConfig(pid, map[string]any{"client_secret": "sealed:s3cr3t"}), nil)
	env.mockOIDC.EXPECT().
		AuthCodeURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, cfg auth.OIDCConfig, _, _, _ string) (string, error) {
			assert.Equal(t, "s3cr3t", cfg.ClientSecret, "the IdP gets the decrypted secret")
			return "https://idp.example/authorize?state=abc", nil
		})

	redirect, cookie, err := env.service.StartOIDC(context.Background(), pid)
	require.NoError(t, err)
	assert.Contains(t, redirect, "https://idp.example/authorize")

	// the cookie must be a valid, verifiable oidc_state token for this pid
	claims, err := token.Parse(cookie, testSecret)
	require.NoError(t, err)
	assert.Equal(t, "oidc_state", claims["typ"])
	assert.Equal(t, pid.String(), claims["pid"])
	assert.NotEmpty(t, claims["verifier"])
}

// One IdP group that maps to a list of roles assigns all of them.
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

// No mapping table: the IdP sends a value that is already a role name (Azure app roles,
// Okta roles), and it is assigned directly.
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

// A group that maps to nothing and is not a role resolves to nothing, so the user gets the
// default role and does not lose all access.
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

	// the candidate is not a real role, so it is skipped and the default resolves
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

// sync_mode "attributes" leaves roles to an admin: a login refreshes the session but never
// changes the roles, even with a matching group claim.
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
	// no mockRoles expectations on purpose: any role call fails the test
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// A lookup that fails for any reason except "no such user" must not go on to JIT.
// Provisioning an existing account again would hit the external_id unique index and report
// a database outage as a bad login.
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
	// no Create expectation: gomock fails the test if the user is provisioned

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

func TestCompleteOIDCSyncModeAllPushesAttributes(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	stale := "Bobby"
	existing := domain.User{
		ID: uuid.New(), Username: "bob", Email: "old@corp", FirstName: &stale,
		AuthProvider: domain.AuthProviderOIDC, IsActive: true,
	}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).
		Return(oidcProviderWithConfig(pid, map[string]any{"sync_mode": "all"}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Email: "new@corp", FirstName: "Bob", LastName: "Stone"}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), existing.ID).Return(nil)

	env.mockUsers.EXPECT().Update(gomock.Any(), existing.ID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, p auth.UpdateUserParams) (domain.User, error) {
			require.True(t, p.Email.Set)
			assert.Equal(t, "new@corp", *p.Email.Value)
			require.True(t, p.FirstName.Set)
			assert.Equal(t, "Bob", *p.FirstName.Value)
			require.True(t, p.LastName.Set)
			assert.Equal(t, "Stone", *p.LastName.Value)
			return existing, nil
		})
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// The default mode syncs roles only: a profile edited in YTSoar must stay.
func TestCompleteOIDCDefaultModeDoesNotPushAttributes(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	existing := domain.User{ID: uuid.New(), Username: "bob", Email: "kept@corp", AuthProvider: domain.AuthProviderOIDC, IsActive: true}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).
		Return(oidcProviderWithConfig(pid, map[string]any{}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Email: "different@corp", FirstName: "Bob"}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), existing.ID).Return(nil)
	// no Update expectation: gomock fails the test if the profile is written
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// An unchanged profile costs no write, and a claim that the IdP omits must not blank a
// value set by hand.
func TestCompleteOIDCAttributeSyncSkipsUnchangedAndEmpty(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|kc-sub"
	kept := "Bob"
	existing := domain.User{
		ID: uuid.New(), Username: "bob", Email: "same@corp", FirstName: &kept,
		AuthProvider: domain.AuthProviderOIDC, IsActive: true,
	}

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).
		Return(oidcProviderWithConfig(pid, map[string]any{"sync_mode": "attributes"}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "kc-sub", Email: "same@corp", FirstName: "Bob"}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).Return(existing, nil)
	// no Update expectation and no role expectations
	env.mockTokens.EXPECT().Insert(gomock.Any(), existing.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), existing.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// The Google preset has sync_mode "attributes" (Google sends no groups), so a JIT user
// would get no roles. default_role must still apply at the provisioning login.
func TestCompleteOIDCJITGetsDefaultRoleEvenWhenRoleSyncIsOff(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|g-sub"

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderWithConfig(pid, map[string]any{
		"sync_mode":    "attributes",
		"default_role": "viewer",
		"allow_jit":    true,
	}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "g-sub", PreferredUsername: "carol", Email: "carol@corp"}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).
		Return(domain.User{}, auth.ErrUserNotFound)
	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "carol").Return(domain.User{}, auth.ErrUserNotFound)

	created := domain.User{ID: uuid.New(), Username: "carol", Email: "carol@corp", AuthProvider: domain.AuthProviderOIDC, IsActive: true}
	env.mockUsers.EXPECT().Create(gomock.Any(), gomock.Any()).Return(created, nil)

	viewer := domain.Role{ID: uuid.New(), Name: "viewer"}
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "viewer").Return(viewer, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), created.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), created.ID, viewer.ID).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), created.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), created.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}

// ...but the IdP groups are still ignored in that mode: a group that maps to admin must
// not add a role through the provisioning path.
func TestCompleteOIDCJITRoleSyncOffIgnoresGroups(t *testing.T) {
	env := setupTest(t)
	pid := uuid.New()
	cookie := stateCookie(t, pid, "s", "v", time.Now().Add(time.Minute))
	externalID := pid.String() + "|g-sub"

	env.mockProviders.EXPECT().GetByID(gomock.Any(), pid).Return(oidcProviderWithConfig(pid, map[string]any{
		"sync_mode":          "attributes",
		"default_role":       "viewer",
		"group_role_mapping": map[string]any{"soc-admins": "admin"},
		"allow_jit":          true,
	}), nil)
	env.mockOIDC.EXPECT().Exchange(gomock.Any(), gomock.Any(), gomock.Any(), "code", "v").
		Return(auth.OIDCIdentity{Subject: "g-sub", PreferredUsername: "carol", Groups: []string{"soc-admins"}}, nil)
	env.mockUsers.EXPECT().GetByExternalID(gomock.Any(), domain.AuthProviderOIDC, externalID).
		Return(domain.User{}, auth.ErrUserNotFound)
	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "carol").Return(domain.User{}, auth.ErrUserNotFound)

	created := domain.User{ID: uuid.New(), Username: "carol", AuthProvider: domain.AuthProviderOIDC, IsActive: true}
	env.mockUsers.EXPECT().Create(gomock.Any(), gomock.Any()).Return(created, nil)

	viewer := domain.Role{ID: uuid.New(), Name: "viewer"}
	// only viewer is looked up, with no GetByName("admin") expectation, so the mapping would
	// fail this test
	env.mockRoles.EXPECT().GetByName(gomock.Any(), "viewer").Return(viewer, nil)
	env.mockRoles.EXPECT().RemoveAllFromUser(gomock.Any(), created.ID).Return(nil)
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), created.ID, viewer.ID).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), created.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), created.ID).Return(nil)

	_, err := env.service.CompleteOIDC(context.Background(), pid, "code", "s", cookie)
	require.NoError(t, err)
}
