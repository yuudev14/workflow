package auth

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

const maskedSecret = "********"

type AdminProvider struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Name    string          `json:"name"`
	Enabled bool            `json:"enabled"`
	Config  json.RawMessage `json:"config"`
}

func (s *Service) ListProvidersAdmin(ctx context.Context) ([]AdminProvider, error) {
	rows, err := s.providers.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminProvider, 0, len(rows))
	for _, p := range rows {
		out = append(out, adminProviderOf(p))
	}
	return out, nil
}

func (s *Service) CreateProvider(ctx context.Context, actorID uuid.UUID, input ProviderInput) (AdminProvider, error) {
	typ, err := parseProviderType(input.Type)
	if err != nil {
		return AdminProvider{}, err
	}
	config := input.Config
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	// Validate now so a malformed config (e.g. the whole provider object pasted
	// into the config field) fails here with a clear message, not later at login.
	if typ == domain.AuthProviderOIDC {
		if _, err := decodeOIDCConfig(config); err != nil {
			return AdminProvider{}, err
		}
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	p, err := s.providers.Create(ctx, typ, input.Name, config, enabled)
	if err != nil {
		return AdminProvider{}, err
	}
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   "settings",
		Action:   "auth_provider_created",
		EntityID: new(p.ID.String()),
	})
	return adminProviderOf(p), nil
}

// UpdateProvider patches name/enabled/config. A config that arrives with the
// masked-secret sentinel keeps the stored secret - the admin didn't retype it.
func (s *Service) UpdateProvider(ctx context.Context, actorID, id uuid.UUID, input UpdateProviderInput) (AdminProvider, error) {
	existing, err := s.providers.GetByID(ctx, id)
	if err != nil {
		return AdminProvider{}, ErrProviderNotFound
	}

	config := input.Config
	if len(config) > 0 {
		config = preserveMaskedSecret(existing.Config, config)
		if existing.Type == domain.AuthProviderOIDC {
			if _, err := decodeOIDCConfig(config); err != nil {
				return AdminProvider{}, err
			}
		}
	}

	p, err := s.providers.Update(ctx, id, UpdateProviderParams{
		Name:    input.Name,
		Config:  config,
		Enabled: input.Enabled,
	})
	if err != nil {
		return AdminProvider{}, err
	}
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   "settings",
		Action:   "auth_provider_updated",
		EntityID: new(p.ID.String()),
	})
	return adminProviderOf(p), nil
}

// preserveMaskedSecret swaps a masked client_secret back to the stored value so
// a round-trip through the masked admin view can't wipe the real secret. On any
// parse trouble it returns the incoming config unchanged - validation downstream
// still catches a genuinely broken config.
func preserveMaskedSecret(existingConfig, incoming json.RawMessage) json.RawMessage {
	var next map[string]any
	if err := json.Unmarshal(incoming, &next); err != nil {
		return incoming
	}
	if secret, ok := next["client_secret"].(string); !ok || secret != maskedSecret {
		return incoming
	}

	var prev map[string]any
	if err := json.Unmarshal(existingConfig, &prev); err == nil {
		next["client_secret"] = prev["client_secret"]
	}
	merged, err := json.Marshal(next)
	if err != nil {
		return incoming
	}
	return merged
}

func adminProviderOf(p AuthProvider) AdminProvider {
	return AdminProvider{
		ID:      p.ID.String(),
		Type:    string(p.Type),
		Name:    p.Name,
		Enabled: p.Enabled,
		Config:  maskSecrets(p.Config),
	}
}

// maskSecrets replaces any client_secret / bind_password with a sentinel so the
// admin view never ships a live credential to the browser.
func maskSecrets(raw json.RawMessage) json.RawMessage {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return json.RawMessage(`{}`)
	}
	for _, key := range []string{"client_secret", "bind_password"} {
		if v, ok := m[key].(string); ok && v != "" {
			m[key] = maskedSecret
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

func parseProviderType(t string) (domain.AuthProvider, error) {
	switch domain.AuthProvider(t) {
	case domain.AuthProviderOIDC, domain.AuthProviderLDAP:
		return domain.AuthProvider(t), nil
	default:
		return "", apperr.New(apperr.Invalid, "provider type must be oidc or ldap")
	}
}
