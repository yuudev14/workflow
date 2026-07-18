package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/connectors"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// ConnectorRepositoryImpl persists connector audit rows. It implements
// connectors.ConnectorRepository.
type ConnectorRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewConnectorRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *ConnectorRepositoryImpl {
	return &ConnectorRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *ConnectorRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *ConnectorRepositoryImpl) Upsert(ctx context.Context, record domain.ConnectorRecord) (domain.ConnectorRecord, error) {
	row, err := r.queriesFromContext(ctx).UpsertConnector(ctx, db.UpsertConnectorParams{
		ID:         record.ID,
		Name:       record.Name,
		Runtime:    record.Runtime,
		Version:    record.Version,
		Checksum:   record.Checksum,
		UploadedBy: record.UploadedBy,
	})
	if err != nil {
		return domain.ConnectorRecord{}, err
	}
	return toDomainConnectorRecord(row), nil
}

func (r *ConnectorRepositoryImpl) Delete(ctx context.Context, connectorID string) error {
	rows, err := r.queriesFromContext(ctx).DeleteConnectorRecord(ctx, connectorID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return connectors.ErrConnectorNotFound
	}
	return nil
}

func toDomainConnectorRecord(row db.Connector) domain.ConnectorRecord {
	return domain.ConnectorRecord{
		ID:         row.ID,
		Name:       row.Name,
		Runtime:    row.Runtime,
		Version:    row.Version,
		Checksum:   row.Checksum,
		UploadedBy: row.UploadedBy,
		Enabled:    row.Enabled,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
