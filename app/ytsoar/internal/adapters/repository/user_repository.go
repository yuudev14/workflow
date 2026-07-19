package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// UserRepositoryImpl implements auth.UserRepository.
type UserRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewUserRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *UserRepositoryImpl {
	return &UserRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *UserRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).GetUserById(ctx, toPgUUID(id))
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).GetUserByUsername(ctx, username)
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) Create(ctx context.Context, params auth.CreateUserParams) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).CreateUser(ctx, db.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: toPgText(params.PasswordHash),
		FirstName:    toPgText(params.FirstName),
		LastName:     toPgText(params.LastName),
		AuthProvider: db.AuthProviderType(params.AuthProvider),
		ExternalID:   toPgText(params.ExternalID),
		IsActive:     true,
	})
	if err != nil {
		return domain.User{}, err
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) TouchLastLogin(ctx context.Context, id uuid.UUID) error {
	return r.queriesFromContext(ctx).TouchUserLastLogin(ctx, toPgUUID(id))
}

func (r *UserRepositoryImpl) CountWithRole(ctx context.Context, roleName string) (int64, error) {
	return r.queriesFromContext(ctx).CountUsersWithRole(ctx, roleName)
}

func toDomainUser(row db.User) domain.User {
	return domain.User{
		ID:           fromPgUUID(row.ID),
		Username:     row.Username,
		Email:        row.Email,
		PasswordHash: fromPgText(row.PasswordHash),
		FirstName:    fromPgText(row.FirstName),
		LastName:     fromPgText(row.LastName),
		AuthProvider: domain.AuthProvider(row.AuthProvider),
		ExternalID:   fromPgText(row.ExternalID),
		IsActive:     row.IsActive,
		LastLoginAt:  fromPgTimestampPtr(row.LastLoginAt),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
