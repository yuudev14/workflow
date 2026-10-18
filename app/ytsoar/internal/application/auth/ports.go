package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/user_repository_mock.go -package=mocks . UserRepository

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByExternalID(ctx context.Context, provider domain.AuthProvider, externalID string) (domain.User, error)
	Create(ctx context.Context, params CreateUserParams) (domain.User, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateUserParams) (domain.User, error)
	SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID, at time.Time) error
	// SetLink says false when the last link is newer than ResendAfter.
	SetLink(ctx context.Context, params SetLinkParams) (bool, error)
	// TakeLink clears the link and returns it, so a link works one time.
	TakeLink(ctx context.Context, tokenHash string) (UserLink, error)
	TouchLastLogin(ctx context.Context, id uuid.UUID) error
	CountWithRole(ctx context.Context, roleName string) (int64, error)
	List(ctx context.Context, offset, limit int, filter UserFilter) ([]domain.UserWithRoles, error)
	ListAssignable(ctx context.Context) ([]domain.AssignableUser, error)
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
	// Delete refuses builtin roles in the query and says if a row was removed.
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
	ReplaceRoles(ctx context.Context, teamID uuid.UUID, roleIDs []uuid.UUID) error
}

//go:generate mockgen -destination=mocks/refresh_token_repository_mock.go -package=mocks . RefreshTokenRepository

type RefreshTokenRepository interface {
	Insert(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context, before time.Time) error
}

//go:generate mockgen -destination=mocks/mailer_mock.go -package=mocks . Mailer

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

//go:generate mockgen -destination=mocks/api_key_repository_mock.go -package=mocks . APIKeyRepository

type APIKeyRepository interface {
	Create(ctx context.Context, params CreateAPIKeyParams) (domain.APIKey, error)
	// GetByHash finds only the keys of active users.
	GetByHash(ctx context.Context, keyHash string) (domain.APIKey, error)
	List(ctx context.Context, userID *uuid.UUID) ([]domain.APIKey, error)
	Touch(ctx context.Context, id uuid.UUID, at time.Time) error
	Delete(ctx context.Context, id uuid.UUID) (int64, error)
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

//go:generate mockgen -destination=mocks/auth_provider_repository_mock.go -package=mocks . AuthProviderRepository

type AuthProviderRepository interface {
	ListEnabled(ctx context.Context) ([]AuthProvider, error)
	List(ctx context.Context) ([]AuthProvider, error)
	GetByID(ctx context.Context, id uuid.UUID) (AuthProvider, error)
	Create(ctx context.Context, typ domain.AuthProvider, name string, config json.RawMessage, enabled bool) (AuthProvider, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateProviderParams) (AuthProvider, error)
}

// UpdateProviderParams is the partial-update shape (Set flags like the types.Nullable
// pattern that the sql CASE arms read).
type UpdateProviderParams struct {
	Name    *string
	Config  json.RawMessage
	Enabled *bool
}

//go:generate mockgen -destination=mocks/oidc_client_mock.go -package=mocks . OIDCClient

// OIDCClient is a thin layer over go-oidc/oauth2, so the OIDC flow is testable without a
// real IdP. AuthCodeURL builds the redirect to the provider. Exchange swaps the code for a
// verified identity (the adapter already checked the id_token signature and claims).
type OIDCClient interface {
	AuthCodeURL(ctx context.Context, cfg OIDCConfig, redirectURL, state, codeChallenge string) (string, error)
	Exchange(ctx context.Context, cfg OIDCConfig, redirectURL, code, codeVerifier string) (OIDCIdentity, error)
}
