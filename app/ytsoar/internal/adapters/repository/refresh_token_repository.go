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

type RefreshTokenRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewRefreshTokenRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *RefreshTokenRepositoryImpl {
	return &RefreshTokenRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *RefreshTokenRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *RefreshTokenRepositoryImpl) Insert(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) error {
	_, err := r.queriesFromContext(ctx).InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		UserID:    toPgUUID(userID),
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	return err
}

func (r *RefreshTokenRepositoryImpl) GetByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	row, err := r.queriesFromContext(ctx).GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return domain.RefreshToken{}, mapNoRows(err, auth.ErrTokenNotFound)
	}
	return domain.RefreshToken{
		ID:        fromPgUUID(row.ID),
		UserID:    fromPgUUID(row.UserID),
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		RevokedAt: fromPgTimestampPtr(row.RevokedAt),
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

// Revoke returns ErrTokenNotFound when nothing was updated, so logout can tell "already
// revoked" from "revoked now" without a second query.
func (r *RefreshTokenRepositoryImpl) Revoke(ctx context.Context, tokenHash string) error {
	rows, err := r.queriesFromContext(ctx).RevokeRefreshToken(ctx, tokenHash)
	if err != nil {
		return err
	}
	if rows == 0 {
		return auth.ErrTokenNotFound
	}
	return nil
}

func (r *RefreshTokenRepositoryImpl) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	return r.queriesFromContext(ctx).RevokeAllRefreshTokensForUser(ctx, toPgUUID(userID))
}

// DeleteExpired drops the rows that expired before the given time. Expired rows are
// useless for auth (validation checks expiry), but each refresh adds one.
func (r *RefreshTokenRepositoryImpl) DeleteExpired(ctx context.Context, before time.Time) error {
	return r.queriesFromContext(ctx).DeleteExpiredRefreshTokens(ctx, pgtype.Timestamp{Time: before, Valid: true})
}
