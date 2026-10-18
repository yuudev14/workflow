package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/types"
)

func (s *Service) ListUsers(ctx context.Context, offset, limit int, filter UserFilter) (types.Entries[domain.UserWithRoles], error) {
	users, err := s.users.List(ctx, offset, limit, filter)
	if err != nil {
		return types.Entries[domain.UserWithRoles]{}, err
	}

	total, err := s.users.Count(ctx, filter)
	if err != nil {
		return types.Entries[domain.UserWithRoles]{}, err
	}

	return types.Entries[domain.UserWithRoles]{Entries: users, Total: total}, nil
}

// ListAssignableUsers feeds the assignee pickers. It is not gated on settings:read: an
// analyst who can update an alert must be able to name the assignee, and this projection
// has only id and username.
func (s *Service) ListAssignableUsers(ctx context.Context) ([]domain.AssignableUser, error) {
	return s.users.ListAssignable(ctx)
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (domain.UserWithRoles, error) {
	return s.users.GetWithRoles(ctx, id)
}

// CreateUser hashes the password and assigns the roles in one transaction. A user with no
// roles would be a live account that nobody granted. Without a password, the invite email
// goes out in the same transaction: if it fails, no user is left without a way to sign in.
func (s *Service) CreateUser(ctx context.Context, actorID uuid.UUID, input CreateUserInput) (domain.UserWithRoles, error) {
	roleIDs, err := parseUUIDs(input.RoleIDs, "role_ids")
	if err != nil {
		return domain.UserWithRoles{}, err
	}

	var hash *string
	if input.Password != "" {
		h, err := s.hasher.Hash(input.Password)
		if err != nil {
			return domain.UserWithRoles{}, err
		}
		hash = &h
	}

	var created domain.User
	txErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		user, err := s.users.Create(ctx, CreateUserParams{
			Username:     input.Username,
			Email:        input.Email,
			PasswordHash: hash,
			FirstName:    input.FirstName,
			LastName:     input.LastName,
			AuthProvider: domain.AuthProviderLocal,
		})
		if err != nil {
			return err
		}
		created = user

		if err := s.assignRoles(ctx, user.ID, roleIDs); err != nil {
			return err
		}
		if hash == nil {
			return s.sendLink(ctx, user, TokenInvite)
		}
		return nil
	})
	if txErr != nil {
		return domain.UserWithRoles{}, txErr
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "user_created",
		EntityID: entityID(created.ID),
		Detail:   map[string]any{"username": created.Username, "role_ids": input.RoleIDs},
	})

	return s.users.GetWithRoles(ctx, created.ID)
}

// UpdateUser does a partial update. Deactivating also kills every live session, or the
// user keeps working until the refresh token expires (up to a week).
func (s *Service) UpdateUser(ctx context.Context, actorID, id uuid.UUID, input UpdateUserInput) (domain.UserWithRoles, error) {
	deactivating := input.IsActive.Set && input.IsActive.Value != nil && !*input.IsActive.Value

	txErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		if _, err := s.users.Update(ctx, id, UpdateUserParams(input)); err != nil {
			return err
		}

		if deactivating {
			return s.tokens.RevokeAllForUser(ctx, id)
		}
		return nil
	})
	if txErr != nil {
		return domain.UserWithRoles{}, txErr
	}

	action := "user_updated"
	if deactivating {
		action = "user_deactivated"
	}
	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   action,
		EntityID: entityID(id),
	})

	return s.users.GetWithRoles(ctx, id)
}

// DeactivateUser is what DELETE maps to. Users are never hard-deleted, because audit rows
// and uploaded connectors point to them.
func (s *Service) DeactivateUser(ctx context.Context, actorID, id uuid.UUID) error {
	inactive := false
	_, err := s.UpdateUser(ctx, actorID, id, UpdateUserInput{
		IsActive: types.Nullable[bool]{Value: &inactive, Set: true},
	})
	return err
}

func (s *Service) SetUserRoles(ctx context.Context, actorID, id uuid.UUID, roleIDStrings []string) (domain.UserWithRoles, error) {
	roleIDs, err := parseUUIDs(roleIDStrings, "role_ids")
	if err != nil {
		return domain.UserWithRoles{}, err
	}

	txErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.roles.RemoveAllFromUser(ctx, id); err != nil {
			return err
		}
		return s.assignRoles(ctx, id, roleIDs)
	})
	if txErr != nil {
		return domain.UserWithRoles{}, txErr
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "user_roles_changed",
		EntityID: entityID(id),
		Detail:   map[string]any{"role_ids": roleIDStrings},
	})

	return s.users.GetWithRoles(ctx, id)
}

// SetUserPassword also revokes every session. An admin resets a password usually after a
// compromise, so old sessions must not stay alive.
func (s *Service) SetUserPassword(ctx context.Context, actorID, id uuid.UUID, password string) error {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}

	txErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.users.SetPassword(ctx, id, hash); err != nil {
			return err
		}
		return s.tokens.RevokeAllForUser(ctx, id)
	})
	if txErr != nil {
		return txErr
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "user_password_reset",
		EntityID: entityID(id),
	})
	return nil
}

// assignRoles checks that each role exists. An unknown id would fail on the FK with an
// unclear database error.
func (s *Service) assignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	for _, roleID := range roleIDs {
		if _, err := s.roles.GetByID(ctx, roleID); err != nil {
			return err
		}
		if err := s.roles.AssignToUser(ctx, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func parseUUIDs(ids []string, field string) ([]uuid.UUID, error) {
	parsed := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		parsedID, err := uuid.Parse(id)
		if err != nil {
			return nil, apperr.Wrap(apperr.Invalid, field+" must be uuids", ErrValidation)
		}
		parsed = append(parsed, parsedID)
	}
	return parsed, nil
}

func entityID(id uuid.UUID) *string {
	s := id.String()
	return &s
}
