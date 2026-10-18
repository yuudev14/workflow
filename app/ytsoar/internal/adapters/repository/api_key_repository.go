package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type APIKeyRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewAPIKeyRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *APIKeyRepositoryImpl {
	return &APIKeyRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *APIKeyRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *APIKeyRepositoryImpl) Create(ctx context.Context, params auth.CreateAPIKeyParams) (domain.APIKey, error) {
	row, err := r.queriesFromContext(ctx).InsertAPIKey(ctx, db.InsertAPIKeyParams{
		UserID:    toPgUUID(params.UserID),
		Name:      params.Name,
		KeyHash:   params.KeyHash,
		Prefix:    params.Prefix,
		CreatedBy: toPgUUID(params.CreatedBy),
		ExpiresAt: toPgTimestampPtr(params.ExpiresAt),
	})
	if err != nil {
		return domain.APIKey{}, err
	}
	return toDomainAPIKey(row, ""), nil
}

func (r *APIKeyRepositoryImpl) GetByHash(ctx context.Context, keyHash string) (domain.APIKey, error) {
	row, err := r.queriesFromContext(ctx).GetActiveAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return domain.APIKey{}, mapNoRows(err, auth.ErrAPIKeyNotFound)
	}
	return toDomainAPIKey(row.ApiKey, row.Username), nil
}

func (r *APIKeyRepositoryImpl) List(ctx context.Context, userID *uuid.UUID) ([]domain.APIKey, error) {
	rows, err := r.queriesFromContext(ctx).ListAPIKeys(ctx, toPgUUIDPtr(userID))
	if err != nil {
		return nil, err
	}
	keys := make([]domain.APIKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, toDomainAPIKey(row.ApiKey, row.Username))
	}
	return keys, nil
}

func (r *APIKeyRepositoryImpl) Touch(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.queriesFromContext(ctx).TouchAPIKey(ctx, db.TouchAPIKeyParams{
		ID:         toPgUUID(id),
		LastUsedAt: pgtype.Timestamp{Time: at, Valid: true},
	})
}

func (r *APIKeyRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queriesFromContext(ctx).DeleteAPIKey(ctx, toPgUUID(id))
}

func toDomainAPIKey(k db.ApiKey, username string) domain.APIKey {
	return domain.APIKey{
		ID:         fromPgUUID(k.ID),
		UserID:     fromPgUUID(k.UserID),
		Username:   username,
		Name:       k.Name,
		Prefix:     k.Prefix,
		CreatedBy:  fromPgUUIDPtr(k.CreatedBy),
		ExpiresAt:  fromPgTimestampPtr(k.ExpiresAt),
		LastUsedAt: fromPgTimestampPtr(k.LastUsedAt),
		CreatedAt:  k.CreatedAt.Time,
	}
}
