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
	ErrProviderNotFound  = apperr.New(apperr.NotFound, "auth provider not found")
	ErrProviderDisabled  = apperr.New(apperr.Invalid, "auth provider is disabled")
	ErrProviderNameTaken = apperr.New(apperr.Conflict, "an auth provider with that name already exists")
	// ErrOIDCState is deliberately vague - a state mismatch is the CSRF signal
	// on the one state-changing GET, and the reason must not leak to the caller.
	ErrOIDCState   = apperr.New(apperr.Unauthorized, "authentication could not be completed")
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
	// UsernameClaim / EmailClaim / FirstNameClaim / LastNameClaim name the
	// profile source claims; empty means "preferred_username" / "email" /
	// "given_name" / "family_name".
	UsernameClaim  string `json:"username_claim,omitempty"`
	EmailClaim     string `json:"email_claim,omitempty"`
	FirstNameClaim string `json:"first_name_claim,omitempty"`
	LastNameClaim  string `json:"last_name_claim,omitempty"`
	AllowJIT       bool   `json:"allow_jit"`
}

const (
	SyncModeRoles      = "roles"
	SyncModeAttributes = "attributes"
	SyncModeAll        = "all"
	SyncModeOff        = "off"
)

func (c OIDCConfig) SyncsRoles() bool {
	switch c.SyncMode {
	case SyncModeAttributes, SyncModeOff:
		return false
	default:
		return true
	}
}

// SyncsAttributes reports whether a login should refresh the user's profile
// from the IdP. Off by default: empty/"roles" keeps the historical behaviour of
// writing the profile once at JIT and leaving it alone afterwards.
func (c OIDCConfig) SyncsAttributes() bool {
	switch c.SyncMode {
	case SyncModeAttributes, SyncModeAll:
		return true
	default:
		return false
	}
}

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

// ProviderSummary is the public login-screen shape - never carries a secret.
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
	FirstName         string
	LastName          string
	Groups            []string
}

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
