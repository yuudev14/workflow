package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/user_repository_mock.go -package=mocks . UserRepository

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	Create(ctx context.Context, params CreateUserParams) (domain.User, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateUserParams) (domain.User, error)
	SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	TouchLastLogin(ctx context.Context, id uuid.UUID) error
	CountWithRole(ctx context.Context, roleName string) (int64, error)
	List(ctx context.Context, offset, limit int, filter UserFilter) ([]domain.UserWithRoles, error)
	Count(ctx context.Context, filter UserFilter) (int, error)
	GetWithRoles(ctx context.Context, id uuid.UUID) (domain.UserWithRoles, error)
}

//go:generate mockgen -destination=mocks/role_repository_mock.go -package=mocks . RoleRepository

type RoleRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.Role, error)
	GetByName(ctx context.Context, name string) (domain.Role, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	ListPermissionsForUser(ctx context.Context, userID uuid.UUID) (domain.PermissionSet, error)
	AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveAllFromUser(ctx context.Context, userID uuid.UUID) error

	List(ctx context.Context) ([]domain.RoleWithPermissions, error)
	GetWithPermissions(ctx context.Context, id uuid.UUID) (domain.RoleWithPermissions, error)
	Create(ctx context.Context, name string, description *string) (domain.Role, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateRoleParams) (domain.Role, error)
	// Delete refuses builtin roles at the query level and reports whether a
	// row was removed.
	Delete(ctx context.Context, id uuid.UUID) (int64, error)
	ReplacePermissions(ctx context.Context, roleID uuid.UUID, permissions domain.PermissionSet) error
}

//go:generate mockgen -destination=mocks/team_repository_mock.go -package=mocks . TeamRepository

type TeamRepository interface {
	List(ctx context.Context, offset, limit int, filter TeamFilter) ([]domain.TeamWithMembers, error)
	Count(ctx context.Context, filter TeamFilter) (int, error)
	GetWithMembers(ctx context.Context, id uuid.UUID) (domain.TeamWithMembers, error)
	Create(ctx context.Context, name string, description *string) (domain.Team, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateTeamParams) (domain.Team, error)
	Delete(ctx context.Context, id uuid.UUID) (int64, error)
	ReplaceMembers(ctx context.Context, teamID uuid.UUID, userIDs []uuid.UUID) error
}

//go:generate mockgen -destination=mocks/refresh_token_repository_mock.go -package=mocks . RefreshTokenRepository

type RefreshTokenRepository interface {
	Insert(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/audit_log_repository_mock.go -package=mocks . AuditLogRepository

type AuditLogRepository interface {
	Insert(ctx context.Context, entry domain.AuditEntry) error
	List(ctx context.Context, offset, limit int, filter AuditFilter) ([]domain.AuditLog, error)
	Count(ctx context.Context, filter AuditFilter) (int, error)
}

//go:generate mockgen -destination=mocks/password_hasher_mock.go -package=mocks . PasswordHasher

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) bool
}
