package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/token"
	"github.com/yuudev14/ytsoar/internal/types"
)

const (
	typeOIDCState = "oidc_state"
	// oidcStateTTLSeconds: how long a login can stay at the IdP
	oidcStateTTLSeconds = 600
)

// ListProviders returns the enabled providers for the login screen. No secrets, only what
// the browser needs.
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

// StartOIDC builds the IdP redirect and the signed state cookie value. State and PKCE
// verifier live only in that cookie, so there is no server session store.
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

// CompleteOIDC handles the callback: check the state against the signed cookie, exchange
// the code, create or sync the user and issue the session. Any failure caused by the
// browser returns ErrOIDCState, so nothing leaks.
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

	// a failed role sync must not lock out a new user. Log it, the next login syncs again.
	if err := s.syncOIDCRoles(ctx, user.ID, cfg, identity.Groups, isNew); err != nil {
		s.logger.Errorf("oidc role sync failed for %s: %v", user.ID, err)
	}
	if err := s.syncOIDCAttributes(ctx, user, cfg, identity); err != nil {
		s.logger.Errorf("oidc attribute sync failed for %s: %v", user.ID, err)
	}
	if identity.EmailVerified && user.EmailVerifiedAt == nil {
		if err := s.users.MarkEmailVerified(ctx, user.ID, s.now()); err != nil {
			s.logger.Warnf("could not mark the email of %s as verified: %v", user.ID, err)
		}
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

// resolveOIDCUser finds the user by external id or creates one. It never links by email:
// an IdP with unverified emails could take over a local account.
func (s *Service) resolveOIDCUser(ctx context.Context, providerID uuid.UUID, cfg OIDCConfig, id OIDCIdentity) (domain.User, bool, error) {
	externalID := providerID.String() + "|" + id.Subject

	user, err := s.users.GetByExternalID(ctx, domain.AuthProviderOIDC, externalID)
	switch {
	case err == nil:
		if !user.IsActive {
			return domain.User{}, false, ErrOIDCState
		}
		return user, false, nil
	case !errors.Is(err, ErrUserNotFound):
		return domain.User{}, false, err
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
		FirstName:    optionalString(id.FirstName),
		LastName:     optionalString(id.LastName),
		AuthProvider: domain.AuthProviderOIDC,
		ExternalID:   &externalID,
	})
	if err != nil {
		return domain.User{}, false, err
	}
	return created, true, nil
}

// uniqueUsername builds a login name from the claims and adds a suffix until it is free
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
		_, err := s.users.GetByUsername(ctx, candidate)
		if errors.Is(err, ErrUserNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return "", ErrValidation
}

// syncOIDCRoles replaces the roles from the groups claim on each login, unless sync_mode
// leaves roles to an admin. OIDC users only.
func (s *Service) syncOIDCRoles(ctx context.Context, userID uuid.UUID, cfg OIDCConfig, groups []string, isNew bool) error {
	if !cfg.SyncsRoles() {
		if !isNew {
			return nil
		}
		// a new account needs a starting role, or it signs in with no permissions. Only
		// default_role applies here, the IdP groups are not used in this mode.
		groups = nil
	}

	roleIDs := make([]uuid.UUID, 0)
	for _, name := range desiredRoleNames(cfg, groups) {
		role, err := s.roles.GetByName(ctx, name)
		if err != nil {
			// a candidate that is not a real role (most IdP groups) is skipped, never fail the login
			continue
		}
		roleIDs = append(roleIDs, role.ID)
	}
	// nothing resolved: use the provider default. Check resolved roles, not candidate names,
	// so a group with no real role still gets the default.
	if len(roleIDs) == 0 && cfg.DefaultRole != "" {
		if role, err := s.roles.GetByName(ctx, cfg.DefaultRole); err != nil {
			s.logger.Warnf("oidc default_role %q is not a known role", cfg.DefaultRole)
		} else {
			roleIDs = append(roleIDs, role.ID)
		}
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

// syncOIDCAttributes updates the stored profile when sync_mode asks for it. Only non-empty
// claims are written, so a missing given_name cannot blank a name set by hand. The username
// is never synced: it is the login id, and renaming it would confuse the audit trail.
func (s *Service) syncOIDCAttributes(ctx context.Context, user domain.User, cfg OIDCConfig, id OIDCIdentity) error {
	if !cfg.SyncsAttributes() {
		return nil
	}

	var params UpdateUserParams
	changed := false
	set := func(field *types.Nullable[string], value string) {
		*field = types.Nullable[string]{Value: new(value), Set: true}
		changed = true
	}
	if id.Email != "" && id.Email != user.Email {
		set(&params.Email, id.Email)
	}
	if id.FirstName != "" && (user.FirstName == nil || *user.FirstName != id.FirstName) {
		set(&params.FirstName, id.FirstName)
	}
	if id.LastName != "" && (user.LastName == nil || *user.LastName != id.LastName) {
		set(&params.LastName, id.LastName)
	}
	if !changed {
		return nil
	}

	_, err := s.users.Update(ctx, user.ID, params)
	return err
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return new(s)
}

// desiredRoleNames turns IdP groups into candidate role names. A group_role_mapping entry
// (one to many) wins. Otherwise the raw value is the candidate, so IdPs that send role names
// (Azure, Okta) need no mapping. The caller drops unknown names and applies the default.
func desiredRoleNames(cfg OIDCConfig, groups []string) []string {
	seen := map[string]bool{}
	names := make([]string, 0, len(groups))
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for _, g := range groups {
		if mapped, ok := cfg.GroupRoleMapping[g]; ok {
			for _, r := range mapped {
				add(r)
			}
			continue
		}
		add(g) // passthrough
	}
	return names
}

// verifyStateCookie is the CSRF check for the callback GET. The state in the query must
// match the signed cookie, for this provider and not expired. Returns the PKCE verifier.
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
	config, err := mapSecrets(p.Config, s.secrets.Open)
	if err != nil {
		return OIDCConfig{}, err
	}
	return decodeOIDCConfig(config)
}

func (s *Service) oidcRedirectURL(providerID uuid.UUID) string {
	return fmt.Sprintf("%s/api/v1/auth/oidc/%s/callback", strings.TrimRight(s.cfg.AuthPublicURL, "/"), providerID)
}

func (s *Service) oidcStartURL(providerID uuid.UUID) string {
	return fmt.Sprintf("%s/api/v1/auth/oidc/%s/start", strings.TrimRight(s.cfg.AuthPublicURL, "/"), providerID)
}

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
