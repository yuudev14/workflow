package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/auth"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// AuthProviderRepositoryImpl implements auth.AuthProviderRepository.
type AuthProviderRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewAuthProviderRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *AuthProviderRepositoryImpl {
	return &AuthProviderRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *AuthProviderRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *AuthProviderRepositoryImpl) ListEnabled(ctx context.Context) ([]auth.AuthProvider, error) {
	rows, err := r.queriesFromContext(ctx).ListEnabledAuthProviders(ctx)
	if err != nil {
		return nil, err
	}
	return mapProviders(rows), nil
}

func (r *AuthProviderRepositoryImpl) List(ctx context.Context) ([]auth.AuthProvider, error) {
	rows, err := r.queriesFromContext(ctx).ListAuthProviders(ctx)
	if err != nil {
		return nil, err
	}
	return mapProviders(rows), nil
}

func (r *AuthProviderRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (auth.AuthProvider, error) {
	row, err := r.queriesFromContext(ctx).GetAuthProviderByID(ctx, toPgUUID(id))
	if err != nil {
		return auth.AuthProvider{}, mapNoRows(err, auth.ErrProviderNotFound)
	}
	return toDomainProvider(row), nil
}

func (r *AuthProviderRepositoryImpl) Create(ctx context.Context, typ domain.AuthProvider, name string, config json.RawMessage, enabled bool) (auth.AuthProvider, error) {
	row, err := r.queriesFromContext(ctx).CreateAuthProvider(ctx, db.CreateAuthProviderParams{
		Type:    db.AuthProviderType(typ),
		Name:    name,
		Config:  config,
		Enabled: enabled,
	})
	if err != nil {
		return auth.AuthProvider{}, err
	}
	return toDomainProvider(row), nil
}

func (r *AuthProviderRepositoryImpl) Update(ctx context.Context, id uuid.UUID, params auth.UpdateProviderParams) (auth.AuthProvider, error) {
	arg := db.UpdateAuthProviderParams{ID: toPgUUID(id)}
	if params.Name != nil {
		arg.NameSet = true
		arg.Name = toPgTextFromString(*params.Name)
	}
	if len(params.Config) > 0 {
		arg.ConfigSet = true
		arg.Config = params.Config
	}
	if params.Enabled != nil {
		arg.EnabledSet = true
		arg.Enabled = pgtype.Bool{Bool: *params.Enabled, Valid: true}
	}

	row, err := r.queriesFromContext(ctx).UpdateAuthProvider(ctx, arg)
	if err != nil {
		return auth.AuthProvider{}, mapNoRows(err, auth.ErrProviderNotFound)
	}
	return toDomainProvider(row), nil
}

func mapProviders(rows []db.AuthProvider) []auth.AuthProvider {
	out := make([]auth.AuthProvider, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomainProvider(row))
	}
	return out
}

func toDomainProvider(row db.AuthProvider) auth.AuthProvider {
	return auth.AuthProvider{
		ID:      fromPgUUID(row.ID),
		Type:    domain.AuthProvider(row.Type),
		Name:    row.Name,
		Enabled: row.Enabled,
		Config:  row.Config,
	}
}
