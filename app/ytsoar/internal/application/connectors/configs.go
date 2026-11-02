package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:generate mockgen -destination=mocks/configs_mock.go -package=mocks . ConfigRepository,SecretBox

// ErrConfigNotFound: no config with this id (or name) for the connector.
var ErrConfigNotFound = apperr.New(apperr.NotFound, "connector config not found")

// ErrConfigExists: the connector already has a config with this name.
var ErrConfigExists = apperr.New(apperr.Conflict, "connector config name already exists")

const maskedSecret = "********"

// tasks.config is VARCHAR(50)
var configNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{0,49}$`)

// ConfigRepository saves the named configs. The values it gets and returns are sealed.
type ConfigRepository interface {
	ListConfigs(ctx context.Context, connectorID string) ([]domain.ConnectorConfig, error)
	// GetConfig and GetConfigForRun return ErrConfigNotFound when no row matches.
	GetConfig(ctx context.Context, connectorID string, id uuid.UUID) (domain.ConnectorConfig, error)
	// GetConfigForRun finds the config by name, or the default config when name is nil.
	GetConfigForRun(ctx context.Context, connectorID string, name *string) (domain.ConnectorConfig, error)
	// CreateConfig and UpdateConfig return ErrConfigExists on a duplicate name.
	CreateConfig(ctx context.Context, config domain.ConnectorConfig) (domain.ConnectorConfig, error)
	UpdateConfig(ctx context.Context, connectorID string, id uuid.UUID, update ConfigUpdate) (domain.ConnectorConfig, error)
	ClearDefault(ctx context.Context, connectorID string) error
	// DeleteConfig returns ErrConfigNotFound when no row matched.
	DeleteConfig(ctx context.Context, connectorID string, id uuid.UUID) error
}

// SecretBox encrypts config values at rest. The key lives outside the database.
type SecretBox interface {
	Seal(plain string) (string, error)
	Open(sealed string) (string, error)
}

// ConfigUpdate holds the sealed changes. A nil field keeps the stored value.
type ConfigUpdate struct {
	Name      *string
	Config    json.RawMessage
	IsDefault *bool
	UpdatedBy *uuid.UUID
}

type ConfigInput struct {
	Name      string          `json:"name" binding:"required"`
	Config    json.RawMessage `json:"config"`
	IsDefault bool            `json:"is_default"`
}

type UpdateConfigInput struct {
	Name      *string         `json:"name"`
	Config    json.RawMessage `json:"config"`
	IsDefault *bool           `json:"is_default"`
}

//go:generate mockgen -destination=mocks/config_service_mock.go -package=mocks . ConfigService

type ConfigService interface {
	ListConfigs(ctx context.Context, connectorID string) ([]domain.ConnectorConfig, error)
	CreateConfig(ctx context.Context, actorID *uuid.UUID, connectorID string, input ConfigInput) (domain.ConnectorConfig, error)
	UpdateConfig(ctx context.Context, actorID *uuid.UUID, connectorID string, id uuid.UUID, input UpdateConfigInput) (domain.ConnectorConfig, error)
	DeleteConfig(ctx context.Context, actorID *uuid.UUID, connectorID string, id uuid.UUID) error
	RunConfig(ctx context.Context, connectorID string, name *string) (map[string]any, bool, error)
}

type ConfigServiceImpl struct {
	logger  logger.Logger
	store   ConnectorStore
	repo    ConfigRepository
	secrets SecretBox
	tx      contracts.TxManager
	audit   contracts.AuditWriter
}

func NewConfigService(log logger.Logger, store ConnectorStore, repo ConfigRepository, secrets SecretBox,
	tx contracts.TxManager, audit contracts.AuditWriter) *ConfigServiceImpl {
	return &ConfigServiceImpl{logger: log, store: store, repo: repo, secrets: secrets, tx: tx, audit: audit}
}

func (s *ConfigServiceImpl) ListConfigs(ctx context.Context, connectorID string) ([]domain.ConnectorConfig, error) {
	info, err := s.store.Get(ctx, connectorID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListConfigs(ctx, connectorID)
	if err != nil {
		return nil, err
	}
	secretKeys := passwordFields(info)
	for i := range rows {
		rows[i] = s.masked(rows[i], secretKeys)
	}
	return rows, nil
}

// CreateConfig seals every string value. The first config of a connector is its default.
func (s *ConfigServiceImpl) CreateConfig(ctx context.Context, actorID *uuid.UUID, connectorID string, input ConfigInput) (domain.ConnectorConfig, error) {
	info, err := s.store.Get(ctx, connectorID)
	if err != nil {
		return domain.ConnectorConfig{}, err
	}
	if !configNamePattern.MatchString(input.Name) {
		return domain.ConnectorConfig{}, apperr.New(apperr.Invalid, "config name must be 1 to 50 letters, digits, spaces, '_', '.' or '-'")
	}
	config := input.Config
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	if config, err = mapValues(config, s.secrets.Seal); err != nil {
		return domain.ConnectorConfig{}, err
	}

	var created domain.ConnectorConfig
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.repo.ListConfigs(ctx, connectorID)
		if err != nil {
			return err
		}
		isDefault := input.IsDefault || len(existing) == 0
		if isDefault {
			if err := s.repo.ClearDefault(ctx, connectorID); err != nil {
				return err
			}
		}
		created, err = s.repo.CreateConfig(ctx, domain.ConnectorConfig{
			ConnectorID: connectorID,
			Name:        input.Name,
			Config:      config,
			IsDefault:   isDefault,
			CreatedBy:   actorID,
		})
		return err
	})
	if err != nil {
		return domain.ConnectorConfig{}, err
	}
	s.writeAudit(ctx, actorID, "connector_config_created", created)
	return s.masked(created, passwordFields(info)), nil
}

// UpdateConfig keeps a stored secret when the value is the mask, because the user did not
// type it again.
func (s *ConfigServiceImpl) UpdateConfig(ctx context.Context, actorID *uuid.UUID, connectorID string, id uuid.UUID, input UpdateConfigInput) (domain.ConnectorConfig, error) {
	info, err := s.store.Get(ctx, connectorID)
	if err != nil {
		return domain.ConnectorConfig{}, err
	}
	if input.Name != nil && !configNamePattern.MatchString(*input.Name) {
		return domain.ConnectorConfig{}, apperr.New(apperr.Invalid, "config name must be 1 to 50 letters, digits, spaces, '_', '.' or '-'")
	}
	if input.IsDefault != nil && !*input.IsDefault {
		return domain.ConnectorConfig{}, apperr.New(apperr.Invalid, "set another config as the default instead")
	}

	var updated domain.ConnectorConfig
	err = s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.repo.GetConfig(ctx, connectorID, id)
		if err != nil {
			return err
		}
		config := input.Config
		if len(config) > 0 {
			if config, err = mapValues(config, s.secrets.Seal); err != nil {
				return err
			}
			config = keepMasked(existing.Config, config)
		}
		if input.IsDefault != nil {
			if err := s.repo.ClearDefault(ctx, connectorID); err != nil {
				return err
			}
		}
		updated, err = s.repo.UpdateConfig(ctx, connectorID, id, ConfigUpdate{
			Name:      input.Name,
			Config:    config,
			IsDefault: input.IsDefault,
			UpdatedBy: actorID,
		})
		return err
	})
	if err != nil {
		return domain.ConnectorConfig{}, err
	}
	s.writeAudit(ctx, actorID, "connector_config_updated", updated)
	return s.masked(updated, passwordFields(info)), nil
}

func (s *ConfigServiceImpl) DeleteConfig(ctx context.Context, actorID *uuid.UUID, connectorID string, id uuid.UUID) error {
	if err := s.repo.DeleteConfig(ctx, connectorID, id); err != nil {
		return err
	}
	s.writeAudit(ctx, actorID, "connector_config_deleted", domain.ConnectorConfig{ID: id, ConnectorID: connectorID})
	return nil
}

// RunConfig gives the plain values for a run. found is false when the database has no such
// config, so the runtime can fall back to a TOML file.
func (s *ConfigServiceImpl) RunConfig(ctx context.Context, connectorID string, name *string) (map[string]any, bool, error) {
	if name != nil && *name == "" {
		name = nil
	}
	row, err := s.repo.GetConfigForRun(ctx, connectorID, name)
	if errors.Is(err, ErrConfigNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	plain, err := mapValues(row.Config, s.secrets.Open)
	if err != nil {
		return nil, false, err
	}
	values := map[string]any{}
	if err := json.Unmarshal(plain, &values); err != nil {
		return nil, false, err
	}
	return values, true, nil
}

// masked opens the values best effort and hides the password fields, so the browser never
// gets a live credential.
func (s *ConfigServiceImpl) masked(row domain.ConnectorConfig, secretKeys []string) domain.ConnectorConfig {
	var m map[string]any
	if err := json.Unmarshal(row.Config, &m); err != nil {
		row.Config = json.RawMessage(`{}`)
		return row
	}
	for key, val := range m {
		v, ok := val.(string)
		if !ok || v == "" {
			continue
		}
		if plain, err := s.secrets.Open(v); err == nil {
			m[key] = plain
		}
	}
	for _, key := range secretKeys {
		if v, ok := m[key].(string); ok && v != "" {
			m[key] = maskedSecret
		}
	}
	row.Config, _ = json.Marshal(m)
	return row
}

func (s *ConfigServiceImpl) writeAudit(ctx context.Context, actorID *uuid.UUID, action string, row domain.ConnectorConfig) {
	id := row.ID.String()
	if err := s.audit.Insert(ctx, domain.AuditEntry{
		ActorID:  actorID,
		Module:   domain.ModuleConnectors,
		Action:   action,
		EntityID: &id,
		Detail:   map[string]any{"connector_id": row.ConnectorID, "name": row.Name},
	}); err != nil {
		s.logger.Errorw("could not write the audit log", "action", action, "error", err)
	}
}

// passwordFields are the info.json configuration fields of type "password".
func passwordFields(info domain.ConnectorInfo) []string {
	var keys []string
	configuration, _ := info["configuration"].(map[string]any)
	fields, _ := configuration["fields"].([]any)
	for _, f := range fields {
		field, _ := f.(map[string]any)
		if name, ok := field["name"].(string); ok && field["type"] == "password" {
			keys = append(keys, name)
		}
	}
	return keys
}

// keepMasked puts the stored value back for each value that is still the mask.
func keepMasked(stored, incoming json.RawMessage) json.RawMessage {
	var next, prev map[string]any
	if err := json.Unmarshal(incoming, &next); err != nil {
		return incoming
	}
	_ = json.Unmarshal(stored, &prev)
	for key, val := range next {
		if val == maskedSecret {
			next[key] = prev[key]
		}
	}
	merged, err := json.Marshal(next)
	if err != nil {
		return incoming
	}
	return merged
}

// mapValues runs fn (seal or open) on each top-level string value. It skips the mask, so
// keepMasked can put the stored value back.
func mapValues(raw json.RawMessage, fn func(string) (string, error)) (json.RawMessage, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return nil, apperr.New(apperr.Invalid, "connector config must be a JSON object")
	}
	for key, val := range m {
		v, ok := val.(string)
		if !ok || v == "" || v == maskedSecret {
			continue
		}
		out, err := fn(v)
		if err != nil {
			return nil, apperr.Wrap(apperr.Invalid, "connector config cannot be decrypted, enter it again", err)
		}
		m[key] = out
	}
	return json.Marshal(m)
}
