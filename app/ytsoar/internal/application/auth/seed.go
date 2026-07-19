package auth

import (
	"context"

	"github.com/yuudev14/ytsoar/internal/domain"
)

// EnsureAdminUser gives a fresh deployment someone who can log in. The check
// is "does anyone hold the admin role" rather than "is the users table empty",
// so leftover rows from earlier work can't lock everyone out.
//
// A missing ADMIN_PASSWORD is a warning, not a failure: the API still boots,
// it just has no way to hash a first password.
func (s *Service) EnsureAdminUser(ctx context.Context) error {
	count, err := s.users.CountWithRole(ctx, domain.RoleAdmin)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if s.cfg.AdminPassword == "" {
		s.logger.Warnf("no admin user exists and ADMIN_PASSWORD is unset — nobody can log in")
		return nil
	}

	adminRole, err := s.roles.GetByName(ctx, domain.RoleAdmin)
	if err != nil {
		return err
	}

	hash, err := s.hasher.Hash(s.cfg.AdminPassword)
	if err != nil {
		return err
	}

	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		user, err := s.users.Create(txCtx, CreateUserParams{
			Username:     s.cfg.AdminUsername,
			Email:        s.cfg.AdminEmail,
			PasswordHash: &hash,
			AuthProvider: domain.AuthProviderLocal,
		})
		if err != nil {
			return err
		}
		if err := s.roles.AssignToUser(txCtx, user.ID, adminRole.ID); err != nil {
			return err
		}
		s.writeAudit(txCtx, domain.AuditEntry{
			ActorID: &user.ID,
			Module:  "settings",
			Action:  "admin_seeded",
		})
		return nil
	})
	if err != nil {
		return err
	}

	s.logger.Infof("seeded admin user %q — change the password after first login", s.cfg.AdminUsername)
	return nil
}
