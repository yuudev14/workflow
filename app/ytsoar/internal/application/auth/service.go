// Package auth owns the session, whatever proved the identity. Local
// passwords, OIDC and LDAP all converge on issuePair: the external provider
// authenticates a user once, at login, and every request after that carries a
// token this service signed. Nothing here contacts an identity provider.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/token"
)

const (
	typeAccess  = "access"
	typeRefresh = "refresh"

	// reuseGraceWindow keeps a multi-tab race from looking like token theft:
	// two tabs refreshing at once both send the same cookie, and the loser
	// arrives just after rotation revoked it. Within this window that is a
	// plain 401 (the tab retries with the new cookie); beyond it, a replayed
	// token means the cookie leaked and every session is revoked.
	reuseGraceWindow = 10 * time.Second
)

type Service struct {
	logger    logger.Logger
	users     UserRepository
	roles     RoleRepository
	tokens    RefreshTokenRepository
	teams     TeamRepository
	audit     AuditLogRepository
	hasher    PasswordHasher
	txManager contracts.TxManager
	cfg       AuthConfig

	// now is swappable so tests can drive expiry without sleeping.
	now func() time.Time
}

func NewService(
	log logger.Logger,
	users UserRepository,
	roles RoleRepository,
	tokens RefreshTokenRepository,
	teams TeamRepository,
	audit AuditLogRepository,
	hasher PasswordHasher,
	txManager contracts.TxManager,
	cfg AuthConfig,
) *Service {
	return &Service{
		logger:    log,
		users:     users,
		roles:     roles,
		tokens:    tokens,
		teams:     teams,
		audit:     audit,
		hasher:    hasher,
		txManager: txManager,
		cfg:       cfg,
		now:       time.Now,
	}
}

// Login verifies local credentials and issues a fresh token pair.
//
// Every rejection path runs exactly one argon2 pass before returning. Skipping
// it on any branch would make that branch measurably faster and let an
// attacker sort usernames into unknown / disabled / real by reply latency.
func (s *Service) Login(ctx context.Context, username, password string) (TokenPair, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		_, _ = s.hasher.Hash(password)
		s.recordLoginFailure(ctx, nil, username, "unknown user")
		return TokenPair{}, ErrInvalidCredentials
	}

	// Checked before is_active so a disabled account costs the same as a live
	// one. Users with no local hash (OIDC/LDAP) burn the time too.
	passwordOK := false
	if user.PasswordHash != nil {
		passwordOK = s.hasher.Verify(password, *user.PasswordHash)
	} else {
		_, _ = s.hasher.Hash(password)
	}

	if !user.IsActive {
		s.recordLoginFailure(ctx, &user.ID, username, "inactive user")
		return TokenPair{}, ErrInvalidCredentials
	}
	if !passwordOK {
		s.recordLoginFailure(ctx, &user.ID, username, "bad password")
		return TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.issuePair(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.users.TouchLastLogin(ctx, user.ID); err != nil {
		s.logger.Warnf("could not update last_login_at for %s: %v", user.ID, err)
	}
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID: &user.ID,
		Module:  "auth",
		Action:  "login",
	})

	return pair, nil
}

// Refresh rotates the refresh token: the presented one is revoked and a new
// pair is issued. Presenting an already-revoked token is treated as theft.
//
// This is provider-agnostic on purpose — an OIDC or LDAP session refreshes
// here too, without the identity provider being involved.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := token.Parse(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	if typ, _ := claims["typ"].(string); typ != typeRefresh {
		return TokenPair{}, ErrInvalidCredentials
	}

	hash := hashToken(refreshToken)
	stored, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	if stored.RevokedAt != nil {
		return TokenPair{}, s.handleReuse(ctx, stored)
	}
	if s.now().After(stored.ExpiresAt) {
		return TokenPair{}, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil || !user.IsActive {
		return TokenPair{}, ErrInvalidCredentials
	}

	// Revoking the old token and storing the new one must not half-apply, or
	// the user ends up holding a cookie with no live row.
	var pair TokenPair
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.tokens.Revoke(txCtx, hash); err != nil {
			return err
		}
		issued, err := s.issuePair(txCtx, user)
		if err != nil {
			return err
		}
		pair = issued
		return nil
	})
	if err != nil {
		return TokenPair{}, err
	}

	return pair, nil
}

// handleReuse decides whether a revoked token is a benign tab race or a
// leaked cookie being replayed.
func (s *Service) handleReuse(ctx context.Context, stored domain.RefreshToken) error {
	if stored.RevokedAt != nil && s.now().Sub(*stored.RevokedAt) < reuseGraceWindow {
		return ErrInvalidCredentials
	}

	s.logger.Warnf("refresh token reuse detected for user %s — revoking all sessions", stored.UserID)
	if err := s.tokens.RevokeAllForUser(ctx, stored.UserID); err != nil {
		s.logger.Errorf("could not revoke sessions after reuse: %v", err)
	}
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID: &stored.UserID,
		Module:  "auth",
		Action:  "refresh_reuse",
	})
	return ErrInvalidCredentials
}

// Logout revokes the presented refresh token. It is idempotent: logging out
// twice, or with an already-expired token, still succeeds.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}

	hash := hashToken(refreshToken)
	if err := s.tokens.Revoke(ctx, hash); err != nil && !errors.Is(err, ErrTokenNotFound) {
		s.logger.Warnf("logout could not revoke token: %v", err)
	}

	if claims, err := token.Parse(refreshToken, s.cfg.JWTSecret); err == nil {
		if sub, ok := claims["sub"].(string); ok {
			if id, err := uuid.Parse(sub); err == nil {
				s.writeAudit(ctx, domain.AuditEntry{
					ActorID: &id,
					Module:  "auth",
					Action:  "logout",
				})
			}
		}
	}
	return nil
}

// Me returns the profile, role names and permission map for a user.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (Me, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return Me{}, err
	}

	roles, err := s.roles.ListForUser(ctx, userID)
	if err != nil {
		return Me{}, err
	}
	permissions, err := s.roles.ListPermissionsForUser(ctx, userID)
	if err != nil {
		return Me{}, err
	}

	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Name)
	}

	return Me{User: user, Roles: names, Permissions: permissions.ToMap()}, nil
}

// VerifyAccessToken authenticates an API request from its bearer token.
func (s *Service) VerifyAccessToken(tokenString string) (domain.AuthUser, error) {
	claims, err := token.Parse(tokenString, s.cfg.JWTSecret)
	if err != nil {
		return domain.AuthUser{}, ErrInvalidCredentials
	}
	if typ, _ := claims["typ"].(string); typ != typeAccess {
		return domain.AuthUser{}, ErrInvalidCredentials
	}
	return authUserFromClaims(claims)
}

// VerifyRefreshTokenForWS authenticates a websocket handshake, which carries
// the refresh cookie because a browser cannot set headers on a WebSocket.
// The token is only read — never rotated — but it is checked against the
// database so logout and deactivation stop reconnects immediately.
func (s *Service) VerifyRefreshTokenForWS(ctx context.Context, refreshToken string) (domain.AuthUser, error) {
	claims, err := token.Parse(refreshToken, s.cfg.JWTSecret)
	if err != nil {
		return domain.AuthUser{}, ErrInvalidCredentials
	}
	if typ, _ := claims["typ"].(string); typ != typeRefresh {
		return domain.AuthUser{}, ErrInvalidCredentials
	}

	stored, err := s.tokens.GetByHash(ctx, hashToken(refreshToken))
	if err != nil || stored.RevokedAt != nil || s.now().After(stored.ExpiresAt) {
		return domain.AuthUser{}, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil || !user.IsActive {
		return domain.AuthUser{}, ErrInvalidCredentials
	}
	return domain.AuthUser{ID: user.ID, Username: user.Username}, nil
}

// PermissionsFor backs the RequirePermission middleware.
func (s *Service) PermissionsFor(ctx context.Context, userID uuid.UUID) (domain.PermissionSet, error) {
	return s.roles.ListPermissionsForUser(ctx, userID)
}

// issuePair signs an access/refresh pair and stores the refresh hash. Every
// login path ends here, so a session looks identical whether the user typed a
// password, came back from Keycloak, or bound against LDAP.
func (s *Service) issuePair(ctx context.Context, user domain.User) (TokenPair, error) {
	now := s.now()
	accessExp := now.Add(s.cfg.AccessTokenTTL)
	refreshExp := now.Add(s.cfg.RefreshTokenTTL)

	accessToken, err := token.GenerateToken(jwt.MapClaims{
		"sub":      user.ID.String(),
		"username": user.Username,
		"typ":      typeAccess,
		"iat":      now.Unix(),
		"exp":      accessExp.Unix(),
		"jti":      uuid.NewString(),
	}, s.cfg.JWTSecret)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := token.GenerateToken(jwt.MapClaims{
		"sub": user.ID.String(),
		"typ": typeRefresh,
		"iat": now.Unix(),
		"exp": refreshExp.Unix(),
		"jti": uuid.NewString(),
	}, s.cfg.JWTSecret)
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.tokens.Insert(ctx, user.ID, hashToken(refreshToken), refreshExp); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:     accessToken,
		AccessExpiresAt: accessExp,
		RefreshToken:    refreshToken,
		RefreshExpires:  refreshExp,
	}, nil
}

func (s *Service) recordLoginFailure(ctx context.Context, actorID *uuid.UUID, username, reason string) {
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID: actorID,
		Module:  "auth",
		Action:  "login_failed",
		Detail:  map[string]any{"username": username, "reason": reason},
	})
}

// writeAudit never fails the caller — losing an audit row must not turn a
// successful login into an error.
func (s *Service) writeAudit(ctx context.Context, entry domain.AuditEntry) {
	if err := s.audit.Insert(ctx, entry); err != nil {
		s.logger.Errorf("could not write audit entry %s.%s: %v", entry.Module, entry.Action, err)
	}
}

func authUserFromClaims(claims jwt.MapClaims) (domain.AuthUser, error) {
	sub, ok := claims["sub"].(string)
	if !ok {
		return domain.AuthUser{}, ErrInvalidCredentials
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return domain.AuthUser{}, ErrInvalidCredentials
	}
	username, _ := claims["username"].(string)
	return domain.AuthUser{ID: id, Username: username}, nil
}

// hashToken is what lands in refresh_tokens: a database leak then yields no
// usable tokens.
func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}
