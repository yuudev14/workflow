package auth_test

// Reuse detection and expiry are clock-bound, but nothing here needs to
// control the clock: the stored row's timestamps are relative to now, so
// -1h lands outside the grace window and -2s lands inside it.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// signRefreshToken mints a token shaped like the one the service issues.
func signRefreshToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	now := time.Now()
	signed, err := token.GenerateToken(jwt.MapClaims{
		"sub": userID.String(),
		"typ": "refresh",
		"iat": now.Unix(),
		"exp": now.Add(168 * time.Hour).Unix(),
		"jti": uuid.NewString(),
	}, testSecret)
	require.NoError(t, err)
	return signed
}

// hashOf is the lookup key the service is expected to use. Matching on this
// instead of gomock.Any() is what proves the raw token never reaches the
// database — the reason a leaked refresh_tokens table is worthless.
func hashOf(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// The stored token_hash must be a SHA-256 digest, never the token itself.
func TestRefreshLooksUpByHashNotRawToken(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	refreshToken := signRefreshToken(t, userID)

	var lookedUp string
	env.mockTokens.EXPECT().
		GetByHash(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, tokenHash string) (domain.RefreshToken, error) {
			lookedUp = tokenHash
			return domain.RefreshToken{}, errors.New("not found")
		})

	_, err := env.service.Refresh(context.Background(), refreshToken)

	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.NotEqual(t, refreshToken, lookedUp, "the raw token must never be the lookup key")
	assert.Equal(t, hashOf(refreshToken), lookedUp)
	assert.Len(t, lookedUp, 64, "sha256 hex digest")
}

// Rotation must store the hash of the *new* token, not the new token itself.
func TestRefreshStoresHashOfNewToken(t *testing.T) {
	env := setupTest(t)
	user := activeUser()
	refreshToken := signRefreshToken(t, user.ID)

	var stored string
	env.mockTokens.EXPECT().GetByHash(gomock.Any(), hashOf(refreshToken)).Return(domain.RefreshToken{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	env.mockUsers.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
	env.mockTokens.EXPECT().Revoke(gomock.Any(), hashOf(refreshToken)).Return(nil)
	env.mockTokens.EXPECT().
		Insert(gomock.Any(), user.ID, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, tokenHash string, _ time.Time) error {
			stored = tokenHash
			return nil
		})

	pair, err := env.service.Refresh(context.Background(), refreshToken)

	require.NoError(t, err)
	assert.NotEqual(t, pair.RefreshToken, stored, "the raw token must never be persisted")
	assert.Equal(t, hashOf(pair.RefreshToken), stored)
}

// A token revoked long ago and presented again means the cookie leaked, so
// every session for that user is killed.
func TestRefreshReuseOutsideGraceWindowRevokesEverything(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	refreshToken := signRefreshToken(t, userID)

	revokedAt := time.Now().Add(-time.Hour)
	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &revokedAt,
	}, nil)
	env.mockTokens.EXPECT().RevokeAllForUser(gomock.Any(), userID).Return(nil)

	_, err := env.service.Refresh(context.Background(), refreshToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

// Two tabs refreshing at once both send the same cookie; the loser arrives
// just after rotation revoked it. That is not theft, so the other tab's
// session must survive — RevokeAllForUser is deliberately not expected here,
// so calling it fails this test.
func TestRefreshReuseInsideGraceWindowKeepsSessions(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	refreshToken := signRefreshToken(t, userID)

	revokedAt := time.Now().Add(-2 * time.Second)
	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &revokedAt,
	}, nil)

	_, err := env.service.Refresh(context.Background(), refreshToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestRefreshRejectsExpiredStoredToken(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	refreshToken := signRefreshToken(t, userID)

	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(-time.Minute),
	}, nil)

	_, err := env.service.Refresh(context.Background(), refreshToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestVerifyRefreshTokenForWSRejectsRevoked(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	refreshToken := signRefreshToken(t, userID)

	revokedAt := time.Now().Add(-time.Minute)
	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
		RevokedAt: &revokedAt,
	}, nil)

	_, err := env.service.VerifyRefreshTokenForWS(context.Background(), refreshToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestVerifyRefreshTokenForWSAcceptsLiveToken(t *testing.T) {
	env := setupTest(t)
	userID := uuid.New()
	refreshToken := signRefreshToken(t, userID)

	env.mockTokens.EXPECT().GetByHash(gomock.Any(), gomock.Any()).Return(domain.RefreshToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil)
	env.mockUsers.EXPECT().GetByID(gomock.Any(), userID).Return(domain.User{
		ID:       userID,
		Username: "alice",
		IsActive: true,
	}, nil)

	authUser, err := env.service.VerifyRefreshTokenForWS(context.Background(), refreshToken)

	require.NoError(t, err)
	assert.Equal(t, userID, authUser.ID)
	assert.Equal(t, "alice", authUser.Username)
}

// An access token must never satisfy the websocket handshake.
func TestVerifyRefreshTokenForWSRejectsAccessToken(t *testing.T) {
	env := setupTest(t)
	user := activeUser()

	env.mockUsers.EXPECT().GetByUsername(gomock.Any(), user.Username).Return(user, nil)
	env.mockHasher.EXPECT().Verify(gomock.Any(), gomock.Any()).Return(true)
	env.mockTokens.EXPECT().Insert(gomock.Any(), user.ID, gomock.Any(), gomock.Any()).Return(nil)
	env.mockUsers.EXPECT().TouchLastLogin(gomock.Any(), user.ID).Return(nil)

	pair, err := env.service.Login(context.Background(), user.Username, "correct-password")
	require.NoError(t, err)

	_, err = env.service.VerifyRefreshTokenForWS(context.Background(), pair.AccessToken)

	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}
