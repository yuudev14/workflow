package repository

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
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
		return domain.User{}, mapNoRows(err, auth.ErrUserNotFound)
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).GetUserByUsername(ctx, username)
	if err != nil {
		return domain.User{}, mapNoRows(err, auth.ErrUserNotFound)
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

func (r *UserRepositoryImpl) Update(ctx context.Context, id uuid.UUID, params auth.UpdateUserParams) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).UpdateUser(ctx, db.UpdateUserParams{
		ID:           toPgUUID(id),
		EmailSet:     params.Email.Set,
		Email:        toPgTextFromNullable(params.Email),
		FirstNameSet: params.FirstName.Set,
		FirstName:    toPgTextFromNullable(params.FirstName),
		LastNameSet:  params.LastName.Set,
		LastName:     toPgTextFromNullable(params.LastName),
		IsActiveSet:  params.IsActive.Set,
		IsActive:     toPgBoolFromNullable(params.IsActive),
	})
	if err != nil {
		// UPDATE … RETURNING yields no row for an unknown id, which is a 404
		// rather than a database failure.
		return domain.User{}, mapNoRows(err, auth.ErrUserNotFound)
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.queriesFromContext(ctx).SetUserPassword(ctx, db.SetUserPasswordParams{
		ID:           toPgUUID(id),
		PasswordHash: toPgTextFromString(passwordHash),
	})
}

// applyUserFilter is shared by List and Count so a filter can never apply to
// the page but not the total. It assumes the users table is aliased `u`.
func applyUserFilter(stmt sq.SelectBuilder, filter auth.UserFilter) sq.SelectBuilder {
	if filter.Search != nil {
		term := fmt.Sprint("%", *filter.Search, "%")
		stmt = stmt.Where(sq.Expr(
			"(u.username ILIKE ? OR u.email ILIKE ? OR u.first_name ILIKE ? OR u.last_name ILIKE ?)",
			term, term, term, term))
	}
	if filter.IsActive != nil {
		stmt = stmt.Where(sq.Eq{"u.is_active": *filter.IsActive})
	}
	if filter.RoleName != nil {
		stmt = stmt.Where(sq.Expr(
			"EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id"+
				" WHERE ur.user_id = u.id AND r.name = ?)", *filter.RoleName))
	}

	return stmt
}

// rolesAggregate keeps roles on the same row as the user, so a page of N users
// is one query rather than N+1.
const rolesAggregate = `COALESCE((
    SELECT array_agg(r.name ORDER BY r.name)
    FROM user_roles ur JOIN roles r ON r.id = ur.role_id
    WHERE ur.user_id = u.id
), '{}') AS roles`

func selectUsers(columns string) sq.SelectBuilder {
	return sq.Select(columns).From("users u").PlaceholderFormat(sq.Dollar)
}

func (r *UserRepositoryImpl) List(ctx context.Context, offset, limit int, filter auth.UserFilter) ([]domain.UserWithRoles, error) {
	stmt := applyUserFilter(selectUsers("u.*, "+rolesAggregate), filter).
		OrderBy("u.username").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	return CollectRowsFromSqlizer[domain.UserWithRoles](ctx, stmt, r.pool, r.logger)
}

func (r *UserRepositoryImpl) Count(ctx context.Context, filter auth.UserFilter) (int, error) {
	stmt := applyUserFilter(selectUsers("count(*)"), filter)
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

func (r *UserRepositoryImpl) GetWithRoles(ctx context.Context, id uuid.UUID) (domain.UserWithRoles, error) {
	stmt := selectUsers("u.*, " + rolesAggregate).Where(sq.Eq{"u.id": id})

	rows, err := CollectRowsFromSqlizer[domain.UserWithRoles](ctx, stmt, r.pool, r.logger)
	if err != nil {
		return domain.UserWithRoles{}, err
	}
	if len(rows) == 0 {
		return domain.UserWithRoles{}, auth.ErrUserNotFound
	}
	return rows[0], nil
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
