package auth

import (
	"encoding/json"
	"fmt"
	"slices"

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
	InternalIssuer string   `json:"internal_issuer,omitempty"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	Scopes         []string `json:"scopes,omitempty"`
	// GroupsClaim names the token claim carrying the role source. It may be a
	// dotted path (e.g. "realm_access.roles" for Keycloak realm roles); empty
	// means "groups".
	GroupsClaim string `json:"groups_claim,omitempty"`
	// GroupRoleMapping maps an IdP group/role value to one or more YTSoar role
	// names. A config value may be a single string or a list; both decode to a
	// list. An unmapped value passes through as a candidate role name.
	GroupRoleMapping RoleMapping `json:"group_role_mapping,omitempty"`
	DefaultRole      string      `json:"default_role,omitempty"`
	// SyncMode controls what a login re-syncs. Empty/"roles"/"all" re-sync roles
	// from the IdP each login; "attributes"/"off" leave roles to an admin. See
	// SyncsRoles.
	SyncMode string `json:"sync_mode,omitempty"`
	// UsernameClaim / EmailClaim name the JIT profile source claims; empty means
	// "preferred_username" / "email".
	UsernameClaim string `json:"username_claim,omitempty"`
	EmailClaim    string `json:"email_claim,omitempty"`
	AllowJIT      bool   `json:"allow_jit"`
}

// Sync modes for OIDCConfig.SyncMode — mirrors FortiSOAR's IdP attribute-sync
// toggle. Roles are re-synced from the IdP unless the admin opts out.
const (
	SyncModeRoles      = "roles"
	SyncModeAttributes = "attributes"
	SyncModeAll        = "all"
	SyncModeOff        = "off"
)

// SyncsRoles reports whether a login should replace the user's roles from the
// IdP. False when the admin owns roles (attributes-only or off).
func (c OIDCConfig) SyncsRoles() bool {
	switch c.SyncMode {
	case SyncModeAttributes, SyncModeOff:
		return false
	default:
		return true
	}
}

// RoleMapping maps an IdP group/role value to one or more YTSoar role names. It
// decodes a config value written as either a bare string or a list of strings,
// so "admin" and ["admin","analyst"] are both valid on the wire.
type RoleMapping map[string][]string

func (m *RoleMapping) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(RoleMapping, len(raw))
	for key, val := range raw {
		var list []string
		if err := json.Unmarshal(val, &list); err == nil {
			out[key] = list
			continue
		}
		var single string
		if err := json.Unmarshal(val, &single); err == nil {
			out[key] = []string{single}
			continue
		}
		return fmt.Errorf("group_role_mapping[%q] must be a string or a list of strings", key)
	}
	*m = out
	return nil
}

func decodeOIDCConfig(raw json.RawMessage) (OIDCConfig, error) {
	var cfg OIDCConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return OIDCConfig{}, apperr.Wrap(apperr.Invalid, "provider config is malformed", ErrProviderDisabled)
	}
	if cfg.Issuer == "" || cfg.ClientID == "" {
		return OIDCConfig{}, apperr.New(apperr.Invalid, "provider config missing issuer or client_id")
	}
	// Without the openid scope the IdP returns no id_token, which would surface
	// as a generic login failure rather than the config error it is.
	if len(cfg.Scopes) > 0 && !slices.Contains(cfg.Scopes, "openid") {
		return OIDCConfig{}, apperr.New(apperr.Invalid, `provider config scopes must include "openid"`)
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
