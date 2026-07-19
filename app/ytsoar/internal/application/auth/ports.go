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
	TouchLastLogin(ctx context.Context, id uuid.UUID) error
	CountWithRole(ctx context.Context, roleName string) (int64, error)
}

//go:generate mockgen -destination=mocks/role_repository_mock.go -package=mocks . RoleRepository

type RoleRepository interface {
	GetByName(ctx context.Context, name string) (domain.Role, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
	ListPermissionsForUser(ctx context.Context, userID uuid.UUID) (domain.PermissionSet, error)
	AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error
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
}

//go:generate mockgen -destination=mocks/password_hasher_mock.go -package=mocks . PasswordHasher

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) bool
}
