package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/yuudev14/ytsoar/internal/token"

	"github.com/yuudev14/ytsoar/internal/application/auth"
	mock_auth "github.com/yuudev14/ytsoar/internal/application/auth/mocks"
	mock_contracts "github.com/yuudev14/ytsoar/internal/application/contracts/mocks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type testEnv struct {
	service *auth.Service

	mockUsers  *mock_auth.MockUserRepository
	mockRoles  *mock_auth.MockRoleRepository
	mockTokens *mock_auth.MockRefreshTokenRepository
	mockAudit  *mock_auth.MockAuditLogRepository
	mockHasher *mock_auth.MockPasswordHasher
}

const testSecret = "test-secret"

func setupTest(t *testing.T) *testEnv {
	t.Helper()
	ctrl := gomock.NewController(t)

	mockUsers := mock_auth.NewMockUserRepository(ctrl)
	mockRoles := mock_auth.NewMockRoleRepository(ctrl)
	mockTokens := mock_auth.NewMockRefreshTokenRepository(ctrl)
	mockHasher := mock_auth.NewMockPasswordHasher(ctrl)

	// Audit rows are a side effect of nearly every path and never change the
	// outcome, so they are allowed but not asserted unless a test cares.
	mockAudit := mock_auth.NewMockAuditLogRepository(ctrl)
	mockAudit.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// transactions pass straight through so the closure runs against the mocks
	mockTx := mock_contracts.NewMockTxManager(ctrl)
	mockTx.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		AnyTimes()

	service := auth.NewService(
		logger.NewNop(),
		mockUsers,
		mockRoles,
		mockTokens,
		mockAudit,
		mockHasher,
		mockTx,
		auth.AuthConfig{
			JWTSecret:       testSecret,
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			AdminUsername:   "admin",
			AdminEmail:      "admin@ytsoar.local",
			AdminPassword:   "admin-password",
		},
	)

	return &testEnv{
		service:    service,
		mockUsers:  mockUsers,
		mockRoles:  mockRoles,
		mockTokens: mockTokens,
		mockAudit:  mockAudit,
		mockHasher: mockHasher,
	}
}

func activeUser() domain.User {
	hash := "stored-hash"
	return domain.User{
		ID:           uuid.New(),
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: &hash,
		AuthProvider: domain.AuthProviderLocal,
		IsActive:     true,
	}
}

func TestLoginSuccess(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(user, nil)
	env.mockHasher.EXPECT().Verify("correct-password", "stored-hash").Return(true)
	env.mockTokens.EXPECT().Insert(gomock.Any(), user.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), user.ID).Return(nil)

	pair, err := env.service.Login(context.Background(), "alice", "correct-password")

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, pair.RefreshToken)
	assert.True(t, pair.AccessExpiresAt.Before(pair.RefreshExpires))
}

func TestLoginUnknownUserStillHashes(t *testing.T) {
	env := setupTest(t)

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "ghost").Return(domain.User{}, errors.New("no rows"))
	// The throwaway hash is what keeps an unknown username as slow as a real
	// one; without it reply latency enumerates accounts.
	env.mockHasher.EXPECT().Hash("whatever").Return("", nil)

	_, err := env.service.Login(context.Background(), "ghost", "whatever")

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLoginWrongPassword(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(user, nil)
	env.mockHasher.EXPECT().Verify("wrong", "stored-hash").Return(false)

	_, err := env.service.Login(context.Background(), "alice", "wrong")

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

// A disabled account must cost the same as a live one, so the password is
// still verified before is_active is consulted.
func TestLoginInactiveUserVerifiesPasswordFirst(t *testing.T) {
	env := setupTest(t)
	user := activeUser()
	user.IsActive = false

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(user, nil)
	env.mockHasher.EXPECT().Verify("correct-password", "stored-hash").Return(true)

	_, err := env.service.Login(context.Background(), "alice", "correct-password")

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

// An OIDC/LDAP user has no local hash, but the request must not return early.
func TestLoginExternalUserWithoutHash(t *testing.T) {
	env := setupTest(t)
	user := activeUser()
	user.PasswordHash = nil
	user.AuthProvider = domain.AuthProviderOIDC

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(user, nil)
	env.mockHasher.EXPECT().Hash("anything").Return("", nil)

	_, err := env.service.Login(context.Background(), "alice", "anything")

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestRefreshRotatesToken(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	refreshToken := issueRefreshToken(t, env, user)

	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
	env.mockTokens.EXPECT().Revoke(gomock.Any(), gomock.Any()).Return(nil)
	env.mockTokens.EXPECT().Insert(gomock.Any(), user.ID, gomock.Any(), gomock.Any()).Return(nil)

	pair, err := env.service.Refresh(context.Background(), refreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEqual(t, refreshToken, pair.RefreshToken, "refresh token must rotate")
}

// An access token must not be usable where a refresh token is expected.
func TestRefreshRejectsAccessToken(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(user, nil)
	env.mockHasher.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(true)
	env.mockTokens.EXPECT().Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), gomock.Any()).Return(nil)

	pair, err := env.service.Login(context.Background(), "alice", "correct-password")
	require.NoError(t, err)

	_, err = env.service.Refresh(context.Background(), pair.AccessToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestRefreshWithGarbageToken(t *testing.T) {
	env := setupTest(t)

	_, err := env.service.Refresh(context.Background(), "not-a-jwt")

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestRefreshDeactivatedUserIsRejected(t *testing.T) {
	env := setupTest(t)
	user := activeUser()
	refreshToken := issueRefreshToken(t, env, user)

	deactivated := user
	deactivated.IsActive = false

	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(deactivated, nil)

	_, err := env.service.Refresh(context.Background(), refreshToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLogoutIsIdempotent(t *testing.T) {
	env := setupTest(t)
	user := activeUser()
	refreshToken := issueRefreshToken(t, env, user)

	// Second logout: the row is already revoked and the repository says so.
	env.mockTokens.EXPECT().Revoke(gomock.Any(), gomock.Any()).Return(auth.ErrTokenNotFound)

	assert.NoError(t, env.service.Logout(context.Background(), refreshToken))
}

func TestLogoutWithoutCookieIsNoop(t *testing.T) {
	env := setupTest(t)

	assert.NoError(t, env.service.Logout(context.Background(), ""))
}

func TestVerifyAccessTokenRejectsRefreshToken(t *testing.T) {
	env := setupTest(t)
	user := activeUser()
	refreshToken := issueRefreshToken(t, env, user)

	_, err := env.service.VerifyAccessToken(refreshToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

// A token signed by someone else must not authenticate anyone here.
func TestVerifyAccessTokenRejectsForeignSecret(t *testing.T) {
	env := setupTest(t)

	forged, err := token.GenerateToken(jwt.MapClaims{
		"sub":      uuid.NewString(),
		"username": "attacker",
		"typ":      "access",
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(time.Hour).Unix(),
	}, "a-different-secret")
	require.NoError(t, err)

	_, err = env.service.VerifyAccessToken(forged)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestVerifyAccessTokenAcceptsIssuedToken(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), "alice").Return(user, nil)
	env.mockHasher.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(true)
	env.mockTokens.EXPECT().Insert(gomock.Any(), user.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), user.ID).Return(nil)

	pair, err := env.service.Login(context.Background(), "alice", "correct-password")
	require.NoError(t, err)

	authUser, err := env.service.VerifyAccessToken(pair.AccessToken)

	require.NoError(t, err)
	assert.Equal(t, user.ID, authUser.ID)
	assert.Equal(t, "alice", authUser.Username)
}

func TestEnsureAdminUserSkipsWhenAdminExists(t *testing.T) {
	env := setupTest(t)

	env.mockUsers.EXPECT().CountWithRole(gomock.Any(), domain.RoleAdmin).Return(int64(1), nil)

	require.NoError(t, env.service.EnsureAdminUser(context.Background()))
}

func TestEnsureAdminUserCreatesAndAssignsRole(t *testing.T) {
	env := setupTest(t)
	adminRole := domain.Role{ID: uuid.New(), Name: domain.RoleAdmin, IsBuiltin: true}
	created := activeUser()

	env.mockUsers.EXPECT().CountWithRole(gomock.Any(), domain.RoleAdmin).Return(int64(0), nil)
	env.mockRoles.EXPECT().GetByName(gomock.Any(), domain.RoleAdmin).Return(adminRole, nil)
	env.mockHasher.EXPECT().Hash("admin-password").Return("hashed", nil)
	env.mockUsers.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params auth.CreateUserParams) (domain.User, error) {
			assert.Equal(t, "admin", params.Username)
			assert.Equal(t, domain.AuthProviderLocal, params.AuthProvider)
			require.NotNil(t, params.PasswordHash)
			assert.Equal(t, "hashed", *params.PasswordHash)
			return created, nil
		})
	env.mockRoles.EXPECT().AssignToUser(gomock.Any(), created.ID, adminRole.ID).Return(nil)

	require.NoError(t, env.service.EnsureAdminUser(context.Background()))
}

func TestMeReturnsRolesAndPermissions(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
	env.mockRoles.EXPECT().ListForUser(gomock.Any(), user.ID).Return([]domain.Role{
		{Name: domain.RoleAnalyst},
	}, nil)
	env.mockRoles.EXPECT().ListPermissionsForUser(gomock.Any(), user.ID).Return(domain.PermissionSet{
		{Module: domain.ModulePlaybooks, Action: domain.ActionRead},
		{Module: domain.ModulePlaybooks, Action: domain.ActionExecute},
		{Module: domain.ModuleAlerts, Action: domain.ActionRead},
	}, nil)

	me, err := env.service.Me(context.Background(), user.ID)

	require.NoError(t, err)
	assert.Equal(t, []string{domain.RoleAnalyst}, me.Roles)
	assert.ElementsMatch(t, []string{"read", "execute"}, me.Permissions[domain.ModulePlaybooks])
	assert.Equal(t, []string{"read"}, me.Permissions[domain.ModuleAlerts])
}

// issueRefreshToken logs in through the service so the returned token is
// signed and shaped exactly like a real one.
func issueRefreshToken(t *testing.T, env *testEnv, user domain.User) string {
	t.Helper()

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), user.Username).Return(user, nil)
	env.mockHasher.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(true)
	env.mockTokens.EXPECT().Insert(gomock.Any(), user.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), user.ID).Return(nil)

	pair, err := env.service.Login(context.Background(), user.Username, "correct-password")
	require.NoError(t, err)
	return pair.RefreshToken
}
