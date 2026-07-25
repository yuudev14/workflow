package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/types"
)

var ErrTeamNotFound = apperr.New(apperr.NotFound, "team not found")

func (s *Service) ListTeams(ctx context.Context, offset, limit int, filter TeamFilter) (types.Entries[domain.TeamWithMembers], error) {
	teams, err := s.teams.List(ctx, offset, limit, filter)
	if err != nil {
		return types.Entries[domain.TeamWithMembers]{}, err
	}

	total, err := s.teams.Count(ctx, filter)
	if err != nil {
		return types.Entries[domain.TeamWithMembers]{}, err
	}

	return types.Entries[domain.TeamWithMembers]{Entries: teams, Total: total}, nil
}

func (s *Service) GetTeam(ctx context.Context, id uuid.UUID) (domain.TeamWithMembers, error) {
	return s.teams.GetWithMembers(ctx, id)
}

func (s *Service) CreateTeam(ctx context.Context, actorID uuid.UUID, input TeamInput) (domain.TeamWithMembers, error) {
	memberIDs, err := parseUUIDs(input.MemberIDs, "member_ids")
	if err != nil {
		return domain.TeamWithMembers{}, err
	}
	roleIDs, err := parseUUIDs(input.RoleIDs, "role_ids")
	if err != nil {
		return domain.TeamWithMembers{}, err
	}

	var created domain.Team
	txErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		team, err := s.teams.Create(ctx, input.Name, input.Description)
		if err != nil {
			return err
		}
		created = team
		if err := s.teams.ReplaceMembers(ctx, team.ID, memberIDs); err != nil {
			return err
		}
		if err := s.verifyRolesExist(ctx, roleIDs); err != nil {
			return err
		}
		return s.teams.ReplaceRoles(ctx, team.ID, roleIDs)
	})
	if txErr != nil {
		return domain.TeamWithMembers{}, txErr
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "team_created",
		EntityID: entityID(created.ID),
		Detail:   map[string]any{"name": created.Name},
	})

	return s.teams.GetWithMembers(ctx, created.ID)
}

func (s *Service) UpdateTeam(ctx context.Context, actorID, id uuid.UUID, input UpdateTeamInput) (domain.TeamWithMembers, error) {
	if _, err := s.teams.Update(ctx, id, UpdateTeamParams{
		Name:        input.Name,
		Description: input.Description,
	}); err != nil {
		return domain.TeamWithMembers{}, err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "team_updated",
		EntityID: entityID(id),
	})

	return s.teams.GetWithMembers(ctx, id)
}

func (s *Service) SetTeamMembers(ctx context.Context, actorID, id uuid.UUID, memberIDStrings []string) (domain.TeamWithMembers, error) {
	memberIDs, err := parseUUIDs(memberIDStrings, "member_ids")
	if err != nil {
		return domain.TeamWithMembers{}, err
	}

	// Same reason as SetRolePermissions: delete-then-insert must not half-apply.
	if err := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		return s.teams.ReplaceMembers(ctx, id, memberIDs)
	}); err != nil {
		return domain.TeamWithMembers{}, err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "team_members_changed",
		EntityID: entityID(id),
		Detail:   map[string]any{"member_ids": memberIDStrings},
	})

	return s.teams.GetWithMembers(ctx, id)
}

// SetTeamRoles replaces the roles a team grants its members. This escalates
// privilege for everyone in the team, so it stays behind settings.update and is
// always audited. Builtin roles may be granted freely — that only *uses* a
// role; ErrBuiltinRole still guards editing a builtin's own matrix.
func (s *Service) SetTeamRoles(ctx context.Context, actorID, id uuid.UUID, roleIDStrings []string) (domain.TeamWithMembers, error) {
	roleIDs, err := parseUUIDs(roleIDStrings, "role_ids")
	if err != nil {
		return domain.TeamWithMembers{}, err
	}

	if err := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		if err := s.verifyRolesExist(ctx, roleIDs); err != nil {
			return err
		}
		return s.teams.ReplaceRoles(ctx, id, roleIDs)
	}); err != nil {
		return domain.TeamWithMembers{}, err
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "team_roles_changed",
		EntityID: entityID(id),
		Detail:   map[string]any{"role_ids": roleIDStrings},
	})

	return s.teams.GetWithMembers(ctx, id)
}

// verifyRolesExist fails before any write, so an unknown id surfaces as a clean
// not-found rather than an opaque foreign-key violation.
func (s *Service) verifyRolesExist(ctx context.Context, roleIDs []uuid.UUID) error {
	for _, roleID := range roleIDs {
		if _, err := s.roles.GetByID(ctx, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) DeleteTeam(ctx context.Context, actorID, id uuid.UUID) error {
	rows, err := s.teams.Delete(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTeamNotFound
	}

	s.writeAudit(ctx, domain.AuditEntry{
		ActorID:  &actorID,
		Module:   domain.ModuleSettings,
		Action:   "team_deleted",
		EntityID: entityID(id),
	})
	return nil
}
