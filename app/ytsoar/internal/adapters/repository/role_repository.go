package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// RoleRepositoryImpl implements auth.RoleRepository.
type RoleRepositoryImpl struct {
	logger logger.Logger
	q      QuerierTx
	pool   *pgxpool.Pool
}

func NewRoleRepositoryImpl(log logger.Logger, q QuerierTx, pool *pgxpool.Pool) *RoleRepositoryImpl {
	return &RoleRepositoryImpl{logger: log, q: q, pool: pool}
}

func (r *RoleRepositoryImpl) queriesFromContext(ctx context.Context) db.Querier {
	if tx, ok := txFromContext(ctx); ok {
		return r.q.WithTx(tx)
	}
	return r.q
}

func (r *RoleRepositoryImpl) GetByName(ctx context.Context, name string) (domain.Role, error) {
	row, err := r.queriesFromContext(ctx).GetRoleByName(ctx, name)
	if err != nil {
		return domain.Role{}, err
	}
	return toDomainRole(row), nil
}

func (r *RoleRepositoryImpl) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	rows, err := r.queriesFromContext(ctx).ListRolesForUser(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}

	roles := make([]domain.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, toDomainRole(row))
	}
	return roles, nil
}

func (r *RoleRepositoryImpl) ListPermissionsForUser(ctx context.Context, userID uuid.UUID) (domain.PermissionSet, error) {
	rows, err := r.queriesFromContext(ctx).ListPermissionsForUser(ctx, toPgUUID(userID))
	if err != nil {
		return nil, err
	}

	perms := make(domain.PermissionSet, 0, len(rows))
	for _, row := range rows {
		perms = append(perms, domain.Permission{Module: row.Module, Action: row.Action})
	}
	return perms, nil
}

func (r *RoleRepositoryImpl) AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	return r.queriesFromContext(ctx).InsertUserRole(ctx, db.InsertUserRoleParams{
		UserID: toPgUUID(userID),
		RoleID: toPgUUID(roleID),
	})
}

func toDomainRole(row db.Role) domain.Role {
	return domain.Role{
		ID:          fromPgUUID(row.ID),
		Name:        row.Name,
		Description: fromPgText(row.Description),
		IsBuiltin:   row.IsBuiltin,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
