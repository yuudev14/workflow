package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// ConnectorConfigRepository implements connectors.ConfigRepository.
type ConnectorConfigRepository struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewConnectorConfigRepository(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *ConnectorConfigRepository {
	return &ConnectorConfigRepository{logger: log, q: q, pool: pool}
}

func (r *ConnectorConfigRepository) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *ConnectorConfigRepository) ListConfigs(ctx context.Context, connectorID string) ([]domain.ConnectorConfig, error) {
	rows, err := r.queriesFromContext(ctx).ListConnectorConfigs(ctx, connectorID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ConnectorConfig, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainConnectorConfig(row))
	}
	return out, nil
}

func (r *ConnectorConfigRepository) GetConfig(ctx context.Context, connectorID string, id uuid.UUID) (domain.ConnectorConfig, error) {
	row, err := r.queriesFromContext(ctx).GetConnectorConfig(ctx, db.GetConnectorConfigParams{
		ConnectorID: connectorID,
		ID:          toPgUUID(id),
	})
	if err != nil {
		return domain.ConnectorConfig{}, mapNoRows(err, connectors.ErrConfigNotFound)
	}
	return toDomainConnectorConfig(row), nil
}

func (r *ConnectorConfigRepository) GetConfigForRun(ctx context.Context, connectorID string, name *string) (domain.ConnectorConfig, error) {
	row, err := r.queriesFromContext(ctx).GetConnectorConfigForRun(ctx, db.GetConnectorConfigForRunParams{
		ConnectorID: connectorID,
		Name:        toPgText(name),
	})
	if err != nil {
		return domain.ConnectorConfig{}, mapNoRows(err, connectors.ErrConfigNotFound)
	}
	return toDomainConnectorConfig(row), nil
}

func (r *ConnectorConfigRepository) CreateConfig(ctx context.Context, config domain.ConnectorConfig) (domain.ConnectorConfig, error) {
	row, err := r.queriesFromContext(ctx).InsertConnectorConfig(ctx, db.InsertConnectorConfigParams{
		ConnectorID: config.ConnectorID,
		Name:        config.Name,
		Config:      config.Config,
		IsDefault:   config.IsDefault,
		CreatedBy:   toPgUUIDPtr(config.CreatedBy),
	})
	if err != nil {
		return domain.ConnectorConfig{}, mapUniqueViolation(err, connectors.ErrConfigExists)
	}
	return toDomainConnectorConfig(row), nil
}

func (r *ConnectorConfigRepository) UpdateConfig(ctx context.Context, connectorID string, id uuid.UUID, update connectors.ConfigUpdate) (domain.ConnectorConfig, error) {
	isDefault := pgtype.Bool{}
	if update.IsDefault != nil {
		isDefault = pgtype.Bool{Bool: *update.IsDefault, Valid: true}
	}
	row, err := r.queriesFromContext(ctx).UpdateConnectorConfig(ctx, db.UpdateConnectorConfigParams{
		Name:        toPgText(update.Name),
		Config:      update.Config,
		IsDefault:   isDefault,
		UpdatedBy:   toPgUUIDPtr(update.UpdatedBy),
		ConnectorID: connectorID,
		ID:          toPgUUID(id),
	})
	if err != nil {
		return domain.ConnectorConfig{}, mapUniqueViolation(mapNoRows(err, connectors.ErrConfigNotFound), connectors.ErrConfigExists)
	}
	return toDomainConnectorConfig(row), nil
}

func (r *ConnectorConfigRepository) ClearDefault(ctx context.Context, connectorID string) error {
	return r.queriesFromContext(ctx).ClearDefaultConnectorConfig(ctx, connectorID)
}

func (r *ConnectorConfigRepository) DeleteConfig(ctx context.Context, connectorID string, id uuid.UUID) error {
	rows, err := r.queriesFromContext(ctx).DeleteConnectorConfig(ctx, db.DeleteConnectorConfigParams{
		ConnectorID: connectorID,
		ID:          toPgUUID(id),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return connectors.ErrConfigNotFound
	}
	return nil
}

func toDomainConnectorConfig(row db.ConnectorConfig) domain.ConnectorConfig {
	return domain.ConnectorConfig{
		ID:          fromPgUUID(row.ID),
		ConnectorID: row.ConnectorID,
		Name:        row.Name,
		Config:      row.Config,
		IsDefault:   row.IsDefault,
		CreatedBy:   fromPgUUIDPtr(row.CreatedBy),
		UpdatedBy:   fromPgUUIDPtr(row.UpdatedBy),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
