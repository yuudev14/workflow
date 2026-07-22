package auth

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

var (
	ErrProviderNotFound = apperr.New(apperr.NotFound, "auth provider not found")
	ErrProviderDisabled = apperr.New(apperr.Invalid, "auth provider is disabled")
	// ErrOIDCState is deliberately vague — a state mismatch is the CSRF signal
	// on the one state-changing GET, and the reason must not leak to the caller.
	ErrOIDCState = apperr.New(apperr.Unauthorized, "authentication could not be completed")
	ErrJITDisabled = apperr.New(apperr.Forbidden, "no account for this identity and just-in-time provisioning is off")
)

// AuthProvider is a decoded auth_providers row. Config stays raw here; each
// provider type decodes it into its own shape (OIDCConfig, later LDAPConfig).
type AuthProvider struct {
	ID      uuid.UUID
	Type    domain.AuthProvider
	Name    string
	Enabled bool
	Config  json.RawMessage
}

// OIDCConfig is the auth_providers.config jsonb for an oidc provider.
type OIDCConfig struct {
	Issuer string `json:"issuer"`
	// InternalIssuer handles split-horizon dev: the api discovers the IdP at
	// this url while the browser is redirected to the public Issuer. Empty in
	// normal deployments.
	InternalIssuer   string            `json:"internal_issuer,omitempty"`
	ClientID         string            `json:"client_id"`
	ClientSecret     string            `json:"client_secret"`
	Scopes           []string          `json:"scopes,omitempty"`
	GroupsClaim      string            `json:"groups_claim,omitempty"`
	GroupRoleMapping map[string]string `json:"group_role_mapping,omitempty"`
	DefaultRole      string            `json:"default_role,omitempty"`
	AllowJIT         bool              `json:"allow_jit"`
}

func decodeOIDCConfig(raw json.RawMessage) (OIDCConfig, error) {
	var cfg OIDCConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return OIDCConfig{}, apperr.Wrap(apperr.Invalid, "provider config is malformed", ErrProviderDisabled)
	}
	if cfg.Issuer == "" || cfg.ClientID == "" {
		return OIDCConfig{}, apperr.New(apperr.Invalid, "provider config missing issuer or client_id")
	}
	return cfg, nil
}

// ProviderSummary is the public login-screen shape — never carries a secret.
type ProviderSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	StartURL string `json:"start_url"`
}

// OIDCIdentity is the verified subset of id_token claims the service acts on.
type OIDCIdentity struct {
	Subject           string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Groups            []string
}

// ProviderInput / UpdateProviderInput back the admin CRUD. ClientSecret is
// masked on read (see maskProviderConfig).
type ProviderInput struct {
	Type    string          `json:"type" binding:"required"`
	Name    string          `json:"name" binding:"required"`
	Config  json.RawMessage `json:"config"`
	Enabled *bool           `json:"enabled"`
}

type UpdateProviderInput struct {
	Name    *string         `json:"name"`
	Config  json.RawMessage `json:"config"`
	Enabled *bool           `json:"enabled"`
}
