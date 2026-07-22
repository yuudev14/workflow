package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/token"
)

const (
	typeOIDCState = "oidc_state"
	// oidcStateTTLSeconds bounds how long a login attempt may sit at the IdP.
	oidcStateTTLSeconds = 600
)

// ListProviders returns the enabled providers for the login screen. Secrets
// never appear here — only what the browser needs to start a redirect.
func (s *Service) ListProviders(ctx context.Context) ([]ProviderSummary, error) {
	rows, err := s.providers.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProviderSummary, 0, len(rows))
	for _, p := range rows {
		out = append(out, ProviderSummary{
			ID:       p.ID.String(),
			Name:     p.Name,
			Type:     string(p.Type),
			StartURL: s.oidcStartURL(p.ID),
		})
	}
	return out, nil
}

// StartOIDC builds the redirect to the IdP and the signed state cookie value
// that the callback checks. The verifier (PKCE) and state live only in that
// cookie — there is no server-side session store.
func (s *Service) StartOIDC(ctx context.Context, providerID uuid.UUID) (redirectURL, stateCookie string, err error) {
	cfg, err := s.oidcConfigFor(ctx, providerID)
	if err != nil {
		return "", "", err
	}

	state := randomToken()
	verifier := randomToken()
	challenge := pkceChallenge(verifier)

	redirectURL, err = s.oidcClient.AuthCodeURL(ctx, cfg, s.oidcRedirectURL(providerID), state, challenge)
	if err != nil {
		return "", "", err
	}

	stateCookie, err = token.GenerateToken(jwt.MapClaims{
		"typ":      typeOIDCState,
		"pid":      providerID.String(),
		"state":    state,
		"verifier": verifier,
		"exp":      s.now().Add(oidcStateTTLSeconds * time.Second).Unix(),
		"iat":      s.now().Unix(),
	}, s.cfg.JWTSecret)
	if err != nil {
		return "", "", err
	}
	return redirectURL, stateCookie, nil
}

// CompleteOIDC handles the callback: validate state against the signed cookie,
// exchange the code, provision/sync the user, and issue a session pair. Every
// failure the browser could cause collapses to ErrOIDCState so nothing leaks.
func (s *Service) CompleteOIDC(ctx context.Context, providerID uuid.UUID, code, state, stateCookie string) (TokenPair, error) {
	verifier, err := s.verifyStateCookie(providerID, state, stateCookie)
	if err != nil {
		return TokenPair{}, err
	}

	cfg, err := s.oidcConfigFor(ctx, providerID)
	if err != nil {
		return TokenPair{}, err
	}

	identity, err := s.oidcClient.Exchange(ctx, cfg, s.oidcRedirectURL(providerID), code, verifier)
	if err != nil {
		s.logger.Warnf("oidc exchange failed for provider %s: %v", providerID, err)
		return TokenPair{}, ErrOIDCState
	}
	if identity.Subject == "" {
		return TokenPair{}, ErrOIDCState
	}

	user, isNew, err := s.resolveOIDCUser(ctx, providerID, cfg, identity)
	if err != nil {
		return TokenPair{}, err
	}

	// A failed role sync must not strand a just-provisioned user with no way in;
	// it is logged, and the next login re-syncs.
	if err := s.syncOIDCRoles(ctx, user.ID, cfg, identity.Groups); err != nil {
		s.logger.Errorf("oidc role sync failed for %s: %v", user.ID, err)
	}

	pair, err := s.issuePair(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.users.TouchLastLogin(ctx, user.ID); err != nil {
		s.logger.Warnf("could not update last_login_at for %s: %v", user.ID, err)
	}
	action := "login"
	if isNew {
		action = "jit_provisioned"
	}
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID: &user.ID,
		Module:  "auth",
		Action:  action,
		Detail:  map[string]any{"provider_id": providerID.String(), "subject": identity.Subject},
	})
	return pair, nil
}

// resolveOIDCUser looks a user up by external id, or JIT-provisions one. It
// never links by email: an IdP that lets a user set an unverified address could
// otherwise take over an existing local account.
func (s *Service) resolveOIDCUser(ctx context.Context, providerID uuid.UUID, cfg OIDCConfig, id OIDCIdentity) (domain.User, bool, error) {
	externalID := providerID.String() + "|" + id.Subject

	user, err := s.users.GetByExternalID(ctx, domain.AuthProviderOIDC, externalID)
	if err == nil {
		if !user.IsActive {
			return domain.User{}, false, ErrOIDCState
		}
		return user, false, nil
	}

	if !cfg.AllowJIT {
		return domain.User{}, false, ErrJITDisabled
	}

	username, err := s.uniqueUsername(ctx, id)
	if err != nil {
		return domain.User{}, false, err
	}
	created, err := s.users.Create(ctx, CreateUserParams{
		Username:     username,
		Email:        id.Email,
		AuthProvider: domain.AuthProviderOIDC,
		ExternalID:   &externalID,
	})
	if err != nil {
		return domain.User{}, false, err
	}
	return created, true, nil
}

// uniqueUsername derives a login name from the IdP claims and suffixes it until
// it is free, so a JIT create can never collide with an existing account.
func (s *Service) uniqueUsername(ctx context.Context, id OIDCIdentity) (string, error) {
	base := id.PreferredUsername
	if base == "" {
		base = id.Email
	}
	if at := strings.IndexByte(base, '@'); at >= 0 {
		base = base[:at]
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = "user"
	}

	candidate := base
	for i := 2; i < 1000; i++ {
		if _, err := s.users.GetByUsername(ctx, candidate); err != nil {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return "", ErrValidation
}

// syncOIDCRoles replaces the user's roles from the groups claim on every login.
// OIDC users only — a local user's manual roles are never touched by this path.
func (s *Service) syncOIDCRoles(ctx context.Context, userID uuid.UUID, cfg OIDCConfig, groups []string) error {
	names := desiredRoleNames(cfg, groups)

	roleIDs := make([]uuid.UUID, 0, len(names))
	for _, name := range names {
		role, err := s.roles.GetByName(ctx, name)
		if err != nil {
			s.logger.Warnf("oidc group maps to unknown role %q — skipping", name)
			continue
		}
		roleIDs = append(roleIDs, role.ID)
	}

	return s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.roles.RemoveAllFromUser(txCtx, userID); err != nil {
			return err
		}
		for _, rid := range roleIDs {
			if err := s.roles.AssignToUser(txCtx, userID, rid); err != nil {
				return err
			}
		}
		return nil
	})
}

// desiredRoleNames maps IdP groups to role names, falling back to the provider
// default when nothing matched.
func desiredRoleNames(cfg OIDCConfig, groups []string) []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		role, ok := cfg.GroupRoleMapping[g]
		if ok && role != "" && !seen[role] {
			seen[role] = true
			names = append(names, role)
		}
	}
	if len(names) == 0 && cfg.DefaultRole != "" {
		names = append(names, cfg.DefaultRole)
	}
	return names
}

// verifyStateCookie is the CSRF check on the one state-changing GET: the state
// in the query must match the one sealed in the signed cookie, and the cookie
// must be for this provider and unexpired. Returns the PKCE verifier on success.
func (s *Service) verifyStateCookie(providerID uuid.UUID, state, stateCookie string) (string, error) {
	if state == "" || stateCookie == "" {
		return "", ErrOIDCState
	}
	claims, err := token.Parse(stateCookie, s.cfg.JWTSecret)
	if err != nil {
		return "", ErrOIDCState
	}
	if typ, _ := claims["typ"].(string); typ != typeOIDCState {
		return "", ErrOIDCState
	}
	if pid, _ := claims["pid"].(string); pid != providerID.String() {
		return "", ErrOIDCState
	}
	cookieState, _ := claims["state"].(string)
	if cookieState == "" || cookieState != state {
		return "", ErrOIDCState
	}
	verifier, _ := claims["verifier"].(string)
	if verifier == "" {
		return "", ErrOIDCState
	}
	return verifier, nil
}

func (s *Service) oidcConfigFor(ctx context.Context, providerID uuid.UUID) (OIDCConfig, error) {
	p, err := s.providers.GetByID(ctx, providerID)
	if err != nil {
		return OIDCConfig{}, ErrProviderNotFound
	}
	if !p.Enabled {
		return OIDCConfig{}, ErrProviderDisabled
	}
	if p.Type != domain.AuthProviderOIDC {
		return OIDCConfig{}, ErrProviderNotFound
	}
	return decodeOIDCConfig(p.Config)
}

func (s *Service) oidcRedirectURL(providerID uuid.UUID) string {
	return fmt.Sprintf("%s/api/auth/v1/oidc/%s/callback", strings.TrimRight(s.cfg.AuthPublicURL, "/"), providerID)
}

func (s *Service) oidcStartURL(providerID uuid.UUID) string {
	return fmt.Sprintf("%s/api/auth/v1/oidc/%s/start", strings.TrimRight(s.cfg.AuthPublicURL, "/"), providerID)
}

// FrontendURL is where the callback sends the browser after a session is set.
func (s *Service) FrontendURL() string { return s.cfg.FrontendURL }

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
