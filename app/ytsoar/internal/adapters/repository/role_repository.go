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

func (r *RoleRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (domain.Role, error) {
	row, err := r.queriesFromContext(ctx).GetRoleById(ctx, toPgUUID(id))
	if err != nil {
		return domain.Role{}, mapNoRows(err, auth.ErrRoleNotFound)
	}
	return toDomainRole(row), nil
}

func (r *RoleRepositoryImpl) GetByName(ctx context.Context, name string) (domain.Role, error) {
	row, err := r.queriesFromContext(ctx).GetRoleByName(ctx, name)
	if err != nil {
		return domain.Role{}, mapNoRows(err, auth.ErrRoleNotFound)
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

func (r *RoleRepositoryImpl) RemoveAllFromUser(ctx context.Context, userID uuid.UUID) error {
	return r.queriesFromContext(ctx).DeleteUserRoles(ctx, toPgUUID(userID))
}

func (r *RoleRepositoryImpl) List(ctx context.Context) ([]domain.RoleWithPermissions, error) {
	rows, err := r.queriesFromContext(ctx).ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	roles := make([]domain.RoleWithPermissions, 0, len(rows))
	for _, row := range rows {
		permissions, err := r.listPermissions(ctx, fromPgUUID(row.ID))
		if err != nil {
			return nil, err
		}
		roles = append(roles, domain.RoleWithPermissions{
			Role:        toDomainRole(row),
			Permissions: permissions.ToMap(),
		})
	}
	return roles, nil
}

func (r *RoleRepositoryImpl) GetWithPermissions(ctx context.Context, id uuid.UUID) (domain.RoleWithPermissions, error) {
	role, err := r.GetByID(ctx, id)
	if err != nil {
		return domain.RoleWithPermissions{}, err
	}

	permissions, err := r.listPermissions(ctx, id)
	if err != nil {
		return domain.RoleWithPermissions{}, err
	}
	return domain.RoleWithPermissions{Role: role, Permissions: permissions.ToMap()}, nil
}

func (r *RoleRepositoryImpl) Create(ctx context.Context, name string, description *string) (domain.Role, error) {
	row, err := r.queriesFromContext(ctx).CreateRole(ctx, db.CreateRoleParams{
		Name:        name,
		Description: toPgText(description),
	})
	if err != nil {
		return domain.Role{}, err
	}
	return toDomainRole(row), nil
}

func (r *RoleRepositoryImpl) Update(ctx context.Context, id uuid.UUID, params auth.UpdateRoleParams) (domain.Role, error) {
	row, err := r.queriesFromContext(ctx).UpdateRole(ctx, db.UpdateRoleParams{
		ID:             toPgUUID(id),
		NameSet:        params.Name.Set,
		Name:           toPgTextFromNullable(params.Name),
		DescriptionSet: params.Description.Set,
		Description:    toPgTextFromNullable(params.Description),
	})
	if err != nil {
		return domain.Role{}, mapNoRows(err, auth.ErrRoleNotFound)
	}
	return toDomainRole(row), nil
}

func (r *RoleRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queriesFromContext(ctx).DeleteRole(ctx, toPgUUID(id))
}

// ReplacePermissions clears the matrix and rewrites it. The caller runs it in
// a transaction, so a failure mid-rewrite cannot leave the role partly granted.
func (r *RoleRepositoryImpl) ReplacePermissions(ctx context.Context, roleID uuid.UUID, permissions domain.PermissionSet) error {
	q := r.queriesFromContext(ctx)
	if err := q.DeleteRolePermissions(ctx, toPgUUID(roleID)); err != nil {
		return err
	}

	for _, perm := range permissions {
		if err := q.InsertRolePermission(ctx, db.InsertRolePermissionParams{
			RoleID: toPgUUID(roleID),
			Module: perm.Module,
			Action: perm.Action,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepositoryImpl) listPermissions(ctx context.Context, roleID uuid.UUID) (domain.PermissionSet, error) {
	rows, err := r.queriesFromContext(ctx).ListRolePermissions(ctx, toPgUUID(roleID))
	if err != nil {
		return nil, err
	}

	perms := make(domain.PermissionSet, 0, len(rows))
	for _, row := range rows {
		perms = append(perms, domain.Permission{Module: row.Module, Action: row.Action})
	}
	return perms, nil
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
