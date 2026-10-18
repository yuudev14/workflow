package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, mapNoRows(err, auth.ErrUserNotFound)
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) GetByExternalID(ctx context.Context, provider domain.AuthProvider, externalID string) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).GetUserByExternalId(ctx, db.GetUserByExternalIdParams{
		AuthProvider: db.AuthProviderType(provider),
		ExternalID:   toPgTextFromString(externalID),
	})
	if err != nil {
		return domain.User{}, mapNoRows(err, auth.ErrUserNotFound)
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) Create(ctx context.Context, params auth.CreateUserParams) (domain.User, error) {
	row, err := r.queriesFromContext(ctx).CreateUser(ctx, db.CreateUserParams{
		Username:        params.Username,
		Email:           params.Email,
		PasswordHash:    toPgText(params.PasswordHash),
		FirstName:       toPgText(params.FirstName),
		LastName:        toPgText(params.LastName),
		AuthProvider:    db.AuthProviderType(params.AuthProvider),
		ExternalID:      toPgText(params.ExternalID),
		IsActive:        true,
		EmailVerifiedAt: toPgTimestampPtr(params.EmailVerifiedAt),
	})
	if err != nil {
		return domain.User{}, mapUniqueViolation(err, auth.ErrUserTaken)
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
		// UPDATE ... RETURNING gives no row for an unknown id. That is a 404, not a database
		// failure.
		return domain.User{}, mapUniqueViolation(mapNoRows(err, auth.ErrUserNotFound), auth.ErrUserTaken)
	}
	return toDomainUser(row), nil
}

func (r *UserRepositoryImpl) SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return r.queriesFromContext(ctx).SetUserPassword(ctx, db.SetUserPasswordParams{
		ID:           toPgUUID(id),
		PasswordHash: toPgTextFromString(passwordHash),
	})
}

// MarkEmailVerified keeps the first time. A second link does not move it.
func (r *UserRepositoryImpl) MarkEmailVerified(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.queriesFromContext(ctx).MarkUserEmailVerified(ctx, db.MarkUserEmailVerifiedParams{
		ID:              toPgUUID(id),
		EmailVerifiedAt: pgtype.Timestamp{Time: at, Valid: true},
	})
}

func (r *UserRepositoryImpl) SetLink(ctx context.Context, params auth.SetLinkParams) (bool, error) {
	q := r.queriesFromContext(ctx)
	id := toPgUUID(params.UserID)
	token := pgtype.Text{String: params.TokenHash, Valid: true}
	sentAt := pgtype.Timestamp{Time: params.SentAt, Valid: true}
	resendAfter := pgtype.Timestamp{Time: params.ResendAfter, Valid: true}

	var rows int64
	var err error
	if params.Purpose == auth.TokenInvite {
		rows, err = q.SetUserInvitationToken(ctx, db.SetUserInvitationTokenParams{
			ID: id, InvitationToken: token, InvitationSentAt: sentAt, ResendAfter: resendAfter,
		})
	} else {
		rows, err = q.SetUserResetPasswordToken(ctx, db.SetUserResetPasswordTokenParams{
			ID: id, ResetPasswordToken: token, ResetPasswordSentAt: sentAt, ResendAfter: resendAfter,
		})
	}
	return rows > 0, err
}

// TakeLink looks for an invitation first, then a reset, so one route takes both links.
func (r *UserRepositoryImpl) TakeLink(ctx context.Context, tokenHash string) (auth.UserLink, error) {
	q := r.queriesFromContext(ctx)
	token := pgtype.Text{String: tokenHash, Valid: true}

	invite, err := q.TakeUserInvitationToken(ctx, token)
	if err == nil {
		return auth.UserLink{UserID: fromPgUUID(invite.ID), Purpose: auth.TokenInvite, SentAt: invite.InvitationSentAt.Time}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return auth.UserLink{}, err
	}
	reset, err := q.TakeUserResetPasswordToken(ctx, token)
	if err != nil {
		return auth.UserLink{}, mapNoRows(err, auth.ErrInvalidLink)
	}
	return auth.UserLink{UserID: fromPgUUID(reset.ID), Purpose: auth.TokenPasswordReset, SentAt: reset.ResetPasswordSentAt.Time}, nil
}

// applyUserFilter is shared by List and Count, so a filter never applies to the page but
// not the total. It assumes the users table has the alias `u`.
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

// rolesAggregate keeps the roles on the same row as the user, so a page of N users is one
// query, not N+1.
const rolesAggregate = `COALESCE((
    SELECT array_agg(r.name ORDER BY r.name)
    FROM user_roles ur JOIN roles r ON r.id = ur.role_id
    WHERE ur.user_id = u.id
), '{}') AS roles`

// userColumns are named, because the link columns are not part of domain.User.
const userColumns = `u.id, u.username, u.email, u.password_hash, u.first_name, u.last_name, u.auth_provider,
    u.external_id, u.is_active, u.last_login_at, u.created_at, u.updated_at, u.email_verified_at, `

func selectUsers(columns string) sq.SelectBuilder {
	return sq.Select(columns).From("users u").PlaceholderFormat(sq.Dollar)
}

func (r *UserRepositoryImpl) List(ctx context.Context, offset, limit int, filter auth.UserFilter) ([]domain.UserWithRoles, error) {
	stmt := applyUserFilter(selectUsers(userColumns+rolesAggregate), filter).
		OrderBy("u.username").
		Offset(uint64(offset)).
		Limit(uint64(limit))

	return CollectRowsFromSqlizer[domain.UserWithRoles](ctx, stmt, r.pool, r.logger)
}

func (r *UserRepositoryImpl) Count(ctx context.Context, filter auth.UserFilter) (int, error) {
	stmt := applyUserFilter(selectUsers("count(*)"), filter)
	return CollectOneScalarFromSqlizer[int](ctx, stmt, r.pool, r.logger)
}

// ListAssignable returns only id and username. All other user fields (email, roles,
// auth_provider, external_id, last_login_at) stay behind settings:read on the full list
// route. A colleague's username is normal SOC knowledge, and every analyst needs it for the
// assignee picker.
func (r *UserRepositoryImpl) ListAssignable(ctx context.Context) ([]domain.AssignableUser, error) {
	stmt := selectUsers("u.id, u.username").
		Where(sq.Eq{"u.is_active": true}).
		OrderBy("u.username").
		Limit(500)

	return CollectRowsFromSqlizer[domain.AssignableUser](ctx, stmt, r.pool, r.logger)
}

func (r *UserRepositoryImpl) UsernamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]string{}, nil
	}

	rows, err := CollectRowsFromSqlizer[domain.AssignableUser](ctx,
		selectUsers("u.id, u.username").Where(sq.Eq{"u.id": ids}), r.pool, r.logger)
	if err != nil {
		return nil, err
	}

	names := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Username
	}
	return names, nil
}

func (r *UserRepositoryImpl) GetWithRoles(ctx context.Context, id uuid.UUID) (domain.UserWithRoles, error) {
	stmt := selectUsers(userColumns + rolesAggregate).Where(sq.Eq{"u.id": id})

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
		ID:              fromPgUUID(row.ID),
		Username:        row.Username,
		Email:           row.Email,
		PasswordHash:    fromPgText(row.PasswordHash),
		FirstName:       fromPgText(row.FirstName),
		LastName:        fromPgText(row.LastName),
		AuthProvider:    domain.AuthProvider(row.AuthProvider),
		ExternalID:      fromPgText(row.ExternalID),
		IsActive:        row.IsActive,
		LastLoginAt:     fromPgTimestampPtr(row.LastLoginAt),
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		EmailVerifiedAt: fromPgTimestampPtr(row.EmailVerifiedAt),
	}
}
