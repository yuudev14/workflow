package auth

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

var secretKeys = []string{"client_secret", "bind_password"}

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
		out = append(out, s.adminProviderOf(p))
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
	// validate now, so a bad config (for example the whole provider object pasted into the
	// config field) fails here with a clear message, not later at login
	if typ == domain.AuthProviderOIDC {
		if _, err := decodeOIDCConfig(config); err != nil {
			return AdminProvider{}, err
		}
	}
	config, err = contracts.SealValues(s.secrets, config)
	if err != nil {
		return AdminProvider{}, err
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
	return s.adminProviderOf(p), nil
}

// UpdateProvider patches name, enabled and config. A config with the masked-secret
// sentinel keeps the stored secret, because the admin did not type it again.
func (s *Service) UpdateProvider(ctx context.Context, actorID, id uuid.UUID, input UpdateProviderInput) (AdminProvider, error) {
	existing, err := s.providers.GetByID(ctx, id)
	if err != nil {
		return AdminProvider{}, ErrProviderNotFound
	}

	config := input.Config
	if len(config) > 0 {
		if existing.Type == domain.AuthProviderOIDC {
			if _, err := decodeOIDCConfig(config); err != nil {
				return AdminProvider{}, err
			}
		}
		if config, err = contracts.SealValues(s.secrets, config); err != nil {
			return AdminProvider{}, err
		}
		config = contracts.KeepMasked(existing.Config, config)
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
	return s.adminProviderOf(p), nil
}

func (s *Service) adminProviderOf(p AuthProvider) AdminProvider {
	return AdminProvider{
		ID:      p.ID.String(),
		Type:    string(p.Type),
		Name:    p.Name,
		Enabled: p.Enabled,
		Config:  contracts.MaskedView(s.secrets, p.Config, secretKeys),
	}
}

func parseProviderType(t string) (domain.AuthProvider, error) {
	switch domain.AuthProvider(t) {
	case domain.AuthProviderOIDC, domain.AuthProviderLDAP:
		return domain.AuthProvider(t), nil
	default:
		return "", apperr.New(apperr.Invalid, "provider type must be oidc or ldap")
	}
}
