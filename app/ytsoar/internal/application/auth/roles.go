package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

var ErrBuiltinRole = apperr.New(apperr.Invalid, "builtin roles cannot be modified")

func (s *Service) ListRoles(ctx context.Context) ([]domain.RoleWithPermissions, error) {
	return s.roles.List(ctx)
}

func (s *Service) GetRole(ctx context.Context, id uuid.UUID) (domain.RoleWithPermissions, error) {
	return s.roles.GetWithPermissions(ctx, id)
}

func (s *Service) CreateRole(ctx context.Context, actorID uuid.UUID, input RoleInput) (domain.RoleWithPermissions, error) {
	permissions, err := parsePermissionMatrix(input.Permissions)
	if err != nil {
		return domain.RoleWithPermissions{}, err
	}

	var created domain.Role
	txErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		role, err := s.roles.Create(ctx, input.Name, input.Description)
		if err != nil {
			return err
		}
		created = role
		return s.roles.ReplacePermissions(ctx, role.ID, permissions)
	})
	if txErr != nil {
		return domain.RoleWithPermissions{}, txErr
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "role_created",
		EntityID: entityID(created.ID),
		Detail:   map[string]any{"name": created.Name, "permissions": input.Permissions},
	})

	return s.roles.GetWithPermissions(ctx, created.ID)
}

func (s *Service) UpdateRole(ctx context.Context, actorID, id uuid.UUID, input UpdateRoleInput) (domain.RoleWithPermissions, error) {
	if err := s.rejectBuiltin(ctx, id); err != nil {
		return domain.RoleWithPermissions{}, err
	}

	if _, err := s.roles.Update(ctx, id, UpdateRoleParams{
		Name:        input.Name,
		Description: input.Description,
	}); err != nil {
		return domain.RoleWithPermissions{}, err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "role_updated",
		EntityID: entityID(id),
	})

	return s.roles.GetWithPermissions(ctx, id)
}

// SetRolePermissions replaces the whole matrix in one transaction. A partial
// apply would leave a role holding grants nobody chose.
func (s *Service) SetRolePermissions(ctx context.Context, actorID, id uuid.UUID, matrix map[string][]string) (domain.RoleWithPermissions, error) {
	if err := s.rejectBuiltin(ctx, id); err != nil {
		return domain.RoleWithPermissions{}, err
	}

	permissions, err := parsePermissionMatrix(matrix)
	if err != nil {
		return domain.RoleWithPermissions{}, err
	}

	// ReplacePermissions deletes then re-inserts, so without a transaction a
	// failure mid-rewrite leaves the role holding a partial matrix.
	if err := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		return s.roles.ReplacePermissions(ctx, id, permissions)
	}); err != nil {
		return domain.RoleWithPermissions{}, err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "role_permissions_changed",
		EntityID: entityID(id),
		Detail:   map[string]any{"permissions": matrix},
	})

	return s.roles.GetWithPermissions(ctx, id)
}

func (s *Service) DeleteRole(ctx context.Context, actorID, id uuid.UUID) error {
	if err := s.rejectBuiltin(ctx, id); err != nil {
		return err
	}

	rows, err := s.roles.Delete(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrRoleNotFound
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "role_deleted",
		EntityID: entityID(id),
	})
	return nil
}

func (s *Service) rejectBuiltin(ctx context.Context, id uuid.UUID) error {
	role, err := s.roles.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return ErrBuiltinRole
	}
	return nil
}

// parsePermissionMatrix validates every module and action against the domain
// vocabulary. The columns are TEXT, so the database accepts anything — a typo
// would be stored happily and then grant nothing, forever.
func parsePermissionMatrix(matrix map[string][]string) (domain.PermissionSet, error) {
	permissions := make(domain.PermissionSet, 0, len(matrix))
	for module, actions := range matrix {
		if !domain.IsValidPermissionModule(module) {
			return nil, apperr.Wrap(apperr.Invalid,
				fmt.Sprintf("unknown permission module %q", module), ErrValidation)
		}
		for _, action := range actions {
			if !domain.IsValidPermissionAction(action) {
				return nil, apperr.Wrap(apperr.Invalid,
					fmt.Sprintf("unknown permission action %q", action), ErrValidation)
			}
			permissions = append(permissions, domain.Permission{Module: module, Action: action})
		}
	}
	return permissions, nil
}
