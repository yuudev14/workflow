package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type UserTokenRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewUserTokenRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *UserTokenRepositoryImpl {
	return &UserTokenRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *UserTokenRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *UserTokenRepositoryImpl) Issue(ctx context.Context, params auth.IssueTokenParams) (bool, error) {
	rows, err := r.queriesFromContext(ctx).UpsertUserToken(ctx, db.UpsertUserTokenParams{
		TokenHash:   params.TokenHash,
		UserID:      toPgUUID(params.UserID),
		Purpose:     params.Purpose,
		ExpiresAt:   pgtype.Timestamp{Time: params.ExpiresAt, Valid: true},
		CreatedAt:   pgtype.Timestamp{Time: params.CreatedAt, Valid: true},
		ResendAfter: pgtype.Timestamp{Time: params.ResendAfter, Valid: true},
	})
	return rows > 0, err
}

func (r *UserTokenRepositoryImpl) Take(ctx context.Context, tokenHash string) (auth.UserToken, error) {
	row, err := r.queriesFromContext(ctx).TakeUserToken(ctx, tokenHash)
	if err != nil {
		return auth.UserToken{}, mapNoRows(err, auth.ErrInvalidLink)
	}
	return auth.UserToken{
		UserID:    fromPgUUID(row.UserID),
		Purpose:   row.Purpose,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

func (r *UserTokenRepositoryImpl) DeleteExpired(ctx context.Context, before time.Time) error {
	return r.queriesFromContext(ctx).DeleteExpiredUserTokens(ctx, pgtype.Timestamp{Time: before, Valid: true})
}
