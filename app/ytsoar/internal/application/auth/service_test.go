package auth_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
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

	mockUsers     *mock_auth.MockUserRepository
	mockRoles     *mock_auth.MockRoleRepository
	mockTokens    *mock_auth.MockRefreshTokenRepository
	mockAPIKeys   *mock_auth.MockAPIKeyRepository
	mockLinks     *mock_auth.MockUserTokenRepository
	mockMailer    *mock_auth.MockMailer
	mockTeams     *mock_auth.MockTeamRepository
	mockProviders *mock_auth.MockAuthProviderRepository
	mockOIDC      *mock_auth.MockOIDCClient

	// txCalls counts the WithinTransaction calls, so a test can check that a multi-statement
	// rewrite ran inside one
	txCalls    *int
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
	mockAPIKeys := mock_auth.NewMockAPIKeyRepository(ctrl)
	mockLinks := mock_auth.NewMockUserTokenRepository(ctrl)
	mockMailer := mock_auth.NewMockMailer(ctrl)
	mockTeams := mock_auth.NewMockTeamRepository(ctrl)
	mockProviders := mock_auth.NewMockAuthProviderRepository(ctrl)
	mockOIDC := mock_auth.NewMockOIDCClient(ctrl)
	mockHasher := mock_auth.NewMockPasswordHasher(ctrl)

	// audit rows are a side effect of almost every path and never change the result, so they
	// are allowed but not checked unless a test cares
	mockAudit := mock_auth.NewMockAuditLogRepository(ctrl)
	mockAudit.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// transactions pass through, so the closure runs against the mocks
	txCalls := 0
	mockTx := mock_contracts.NewMockTxManager(ctrl)
	mockTx.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			txCalls++
			return fn(ctx)
		}).
		AnyTimes()

	service := auth.NewService(
		logger.NewNop(),
		mockUsers,
		mockRoles,
		mockTokens,
		mockAPIKeys,
		mockLinks,
		mockMailer,
		mockTeams,
		mockAudit,
		mockProviders,
		mockHasher,
		mockOIDC,
		mockTx,
		auth.AuthConfig{
			JWTSecret:       testSecret,
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			AdminUsername:   "admin",
			AdminEmail:      "admin@ytsoar.local",
			AdminPassword:   "admin-password",
			FrontendURL:     "http://localhost:9999",
			AuthPublicURL:   "http://localhost:9999/auth-api",
		},
	)

	return &testEnv{
		service:       service,
		mockUsers:     mockUsers,
		mockRoles:     mockRoles,
		mockTokens:    mockTokens,
		mockAPIKeys:   mockAPIKeys,
		mockLinks:     mockLinks,
		mockMailer:    mockMailer,
		mockTeams:     mockTeams,
		mockProviders: mockProviders,
		mockOIDC:      mockOIDC,
		txCalls:       &txCalls,
		mockAudit:     mockAudit,
		mockHasher:    mockHasher,
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
	// the throwaway hash keeps an unknown username as slow as a real one, or the reply latency
	// lists the accounts
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

// A disabled account must cost the same as a live one, so the password is still checked
// before is_active.
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

// An access token must not work where a refresh token is expected.
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

// issueRefreshToken logs in through the service, so the token is signed and shaped like a
// real one.
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

func TestAPIKeys(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateAPIKey", func(t *testing.T) {
		t.Run("returns the key one time and stores only its hash", func(t *testing.T) {
			env := setupTest(t)
			user := activeUser()
			var stored auth.CreateAPIKeyParams
			env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
			env.mockAPIKeys.EXPECT().Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p auth.CreateAPIKeyParams) (domain.APIKey, error) {
					stored = p
					return domain.APIKey{ID: uuid.New(), UserID: p.UserID, Name: p.Name, Prefix: p.Prefix}, nil
				})

			created, err := env.service.CreateAPIKey(ctx, uuid.New(), auth.CreateAPIKeyInput{UserID: user.ID.String(), Name: "splunk"})

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(created.Key, domain.APIKeyPrefix))
			assert.Equal(t, sha(created.Key), stored.KeyHash)
			assert.Equal(t, created.Key[:len(domain.APIKeyPrefix)+4], created.Prefix)
			assert.Equal(t, "alice", created.Username)
		})

		t.Run("returns an error when expires_at is in the past", func(t *testing.T) {
			env := setupTest(t)
			past := time.Now().Add(-time.Hour)

			_, err := env.service.CreateAPIKey(ctx, uuid.New(), auth.CreateAPIKeyInput{UserID: uuid.NewString(), Name: "old", ExpiresAt: &past})

			assert.ErrorIs(t, err, auth.ErrAPIKeyExpiry)
		})
	})

	t.Run("VerifyAPIKey", func(t *testing.T) {
		key := domain.APIKeyPrefix + "secret"

		t.Run("returns the owner of the key and saves the time of use", func(t *testing.T) {
			env := setupTest(t)
			stored := domain.APIKey{ID: uuid.New(), UserID: uuid.New(), Username: "siem"}
			env.mockAPIKeys.EXPECT().GetByHash(gomock.Any(), sha(key)).Return(stored, nil)
			env.mockAPIKeys.EXPECT().Touch(gomock.Any(), stored.ID, gomock.Any()).Return(nil)

			user, err := env.service.VerifyAPIKey(ctx, key)

			require.NoError(t, err)
			assert.Equal(t, domain.AuthUser{ID: stored.UserID, Username: "siem"}, user)
		})

		t.Run("does not save the time of use again within a minute", func(t *testing.T) {
			env := setupTest(t)
			recent := time.Now().Add(-10 * time.Second)
			env.mockAPIKeys.EXPECT().GetByHash(gomock.Any(), sha(key)).Return(domain.APIKey{LastUsedAt: &recent}, nil)

			_, err := env.service.VerifyAPIKey(ctx, key)

			assert.NoError(t, err)
		})

		t.Run("rejects an expired key", func(t *testing.T) {
			env := setupTest(t)
			past := time.Now().Add(-time.Second)
			env.mockAPIKeys.EXPECT().GetByHash(gomock.Any(), sha(key)).Return(domain.APIKey{ExpiresAt: &past}, nil)

			_, err := env.service.VerifyAPIKey(ctx, key)

			assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
		})

		t.Run("rejects an unknown or deleted key", func(t *testing.T) {
			env := setupTest(t)
			env.mockAPIKeys.EXPECT().GetByHash(gomock.Any(), sha(key)).Return(domain.APIKey{}, auth.ErrAPIKeyNotFound)

			_, err := env.service.VerifyAPIKey(ctx, key)

			assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
		})
	})

	t.Run("DeleteAPIKey", func(t *testing.T) {
		t.Run("returns not found when the key does not exist", func(t *testing.T) {
			env := setupTest(t)
			env.mockAPIKeys.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(int64(0), nil)

			assert.ErrorIs(t, env.service.DeleteAPIKey(ctx, uuid.New(), uuid.New()), auth.ErrAPIKeyNotFound)
		})
	})
}

func TestDeleteExpiredTokens(t *testing.T) {
	t.Run("deletes the refresh tokens and the links that expired before now", func(t *testing.T) {
		env := setupTest(t)
		before := time.Now()
		near := func(_ context.Context, cutoff time.Time) error {
			assert.WithinDuration(t, before, cutoff, time.Second)
			return nil
		}
		env.mockTokens.EXPECT().DeleteExpired(gomock.Any(), gomock.Any()).DoAndReturn(near)
		env.mockLinks.EXPECT().DeleteExpired(gomock.Any(), gomock.Any()).DoAndReturn(near)

		assert.NoError(t, env.service.DeleteExpiredTokens(context.Background()))
	})
}

func TestInviteAndPasswordReset(t *testing.T) {
	ctx := context.Background()
	linkIn := func(body string) string {
		_, token, ok := strings.Cut(body, "/set-password?token=")
		require.True(t, ok, "the email has no link: %s", body)
		return strings.TrimSpace(token)
	}

	t.Run("CreateUser", func(t *testing.T) {
		t.Run("sends an invite when there is no password", func(t *testing.T) {
			env := setupTest(t)
			user := activeUser()
			user.PasswordHash = nil
			env.mockUsers.EXPECT().Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p auth.CreateUserParams) (domain.User, error) {
					assert.Nil(t, p.PasswordHash)
					return user, nil
				})
			var issued auth.IssueTokenParams
			env.mockLinks.EXPECT().Issue(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p auth.IssueTokenParams) (bool, error) {
					issued = p
					return true, nil
				})
			env.mockMailer.EXPECT().Send(gomock.Any(), user.Email, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _, body string) error {
					assert.Equal(t, sha(linkIn(body)), issued.TokenHash, "only the hash is stored")
					return nil
				})
			env.mockUsers.EXPECT().GetWithRoles(gomock.Any(), user.ID).Return(domain.UserWithRoles{User: user}, nil)

			_, err := env.service.CreateUser(ctx, uuid.New(), auth.CreateUserInput{Username: "alice", Email: user.Email})

			require.NoError(t, err)
			assert.Equal(t, auth.TokenInvite, issued.Purpose)
			assert.WithinDuration(t, time.Now().Add(72*time.Hour), issued.ExpiresAt, time.Minute)
		})
	})

	t.Run("InviteUser", func(t *testing.T) {
		t.Run("refuses an SSO user", func(t *testing.T) {
			env := setupTest(t)
			user := activeUser()
			user.AuthProvider = domain.AuthProviderOIDC
			env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)

			assert.ErrorIs(t, env.service.InviteUser(ctx, uuid.New(), user.ID), auth.ErrCannotInvite)
		})

		t.Run("sends no email when the last link is less than a minute old", func(t *testing.T) {
			env := setupTest(t)
			user := activeUser()
			env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
			env.mockLinks.EXPECT().Issue(gomock.Any(), gomock.Any()).Return(false, nil)

			assert.ErrorIs(t, env.service.InviteUser(ctx, uuid.New(), user.ID), auth.ErrEmailTooSoon)
		})
	})

	t.Run("ForgotPassword", func(t *testing.T) {
		t.Run("sends nothing for an unknown email", func(t *testing.T) {
			env := setupTest(t)
			env.mockUsers.EXPECT().GetByEmail(gomock.Any(), "nobody@example.com").Return(domain.User{}, auth.ErrUserNotFound)

			env.service.ForgotPassword(ctx, "nobody@example.com")
		})

		t.Run("sends a reset link that expires in one hour", func(t *testing.T) {
			env := setupTest(t)
			user := activeUser()
			sent := make(chan struct{})
			env.mockUsers.EXPECT().GetByEmail(gomock.Any(), user.Email).Return(user, nil)
			env.mockLinks.EXPECT().Issue(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, p auth.IssueTokenParams) (bool, error) {
					assert.Equal(t, auth.TokenPasswordReset, p.Purpose)
					assert.WithinDuration(t, time.Now().Add(time.Hour), p.ExpiresAt, time.Minute)
					return true, nil
				})
			env.mockMailer.EXPECT().Send(gomock.Any(), user.Email, gomock.Any(), gomock.Any()).
				DoAndReturn(func(context.Context, string, string, string) error {
					close(sent)
					return nil
				})

			env.service.ForgotPassword(ctx, user.Email)

			select {
			case <-sent:
			case <-time.After(5 * time.Second):
				t.Fatal("no email was sent")
			}
		})
	})

	t.Run("ResetPassword", func(t *testing.T) {
		input := auth.ResetPasswordInput{Token: "link-token", Password: "new-password"}

		t.Run("sets the password, verifies the email and ends all sessions", func(t *testing.T) {
			env := setupTest(t)
			userID := uuid.New()
			env.mockHasher.EXPECT().Hash("new-password").Return("new-hash", nil)
			env.mockLinks.EXPECT().Take(gomock.Any(), sha("link-token")).
				Return(auth.UserToken{UserID: userID, Purpose: auth.TokenInvite, ExpiresAt: time.Now().Add(time.Hour)}, nil)
			env.mockUsers.EXPECT().SetPassword(gomock.Any(), userID, "new-hash").Return(nil)
			env.mockUsers.EXPECT().MarkEmailVerified(gomock.Any(), userID, gomock.Any()).Return(nil)
			env.mockTokens.EXPECT().RevokeAllForUser(gomock.Any(), userID).Return(nil)

			require.NoError(t, env.service.ResetPassword(ctx, input))
			assert.Equal(t, 1, *env.txCalls)
		})

		t.Run("rejects an expired link", func(t *testing.T) {
			env := setupTest(t)
			env.mockHasher.EXPECT().Hash("new-password").Return("new-hash", nil)
			env.mockLinks.EXPECT().Take(gomock.Any(), sha("link-token")).
				Return(auth.UserToken{UserID: uuid.New(), ExpiresAt: time.Now().Add(-time.Second)}, nil)

			assert.ErrorIs(t, env.service.ResetPassword(ctx, input), auth.ErrInvalidLink)
		})
	})
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
