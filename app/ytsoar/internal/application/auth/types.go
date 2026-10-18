package auth

import (
	"time"

	"github.com/google/uuid"

	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/types"
)

var (
	ErrInvalidCredentials = apperr.New(apperr.Unauthorized, "invalid credentials")
	ErrUserNotFound       = apperr.New(apperr.NotFound, "user not found")
	ErrUserTaken          = apperr.New(apperr.Conflict, "a user with that username or email already exists")
	ErrRoleNotFound       = apperr.New(apperr.NotFound, "role not found")
	ErrRoleTaken          = apperr.New(apperr.Conflict, "a role with that name already exists")
	ErrTokenNotFound      = apperr.New(apperr.NotFound, "refresh token not found")
	ErrValidation         = apperr.New(apperr.Invalid, "invalid input")
)

type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash *string
	FirstName    *string
	LastName     *string
	AuthProvider domain.AuthProvider
	ExternalID   *string
	// EmailVerifiedAt is nil for a local user until a set-password link is used
	EmailVerifiedAt *time.Time
}

type UpdateUserParams struct {
	Email     types.Nullable[string]
	FirstName types.Nullable[string]
	LastName  types.Nullable[string]
	IsActive  types.Nullable[bool]
}

type UserFilter struct {
	Search   *string `form:"search"`
	IsActive *bool   `form:"is_active"`
	RoleName *string `form:"role"`
}

// CreateUserInput comes from the handler layer: a plaintext password and role ids. Neither
// reaches the repository as it is.
type CreateUserInput struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	// Password is optional. Without it, the user gets an invite email to set one.
	Password  string   `json:"password" binding:"omitempty,min=8"`
	FirstName *string  `json:"first_name"`
	LastName  *string  `json:"last_name"`
	RoleIDs   []string `json:"role_ids"`
}

type UpdateUserInput struct {
	Email     types.Nullable[string] `json:"email"`
	FirstName types.Nullable[string] `json:"first_name"`
	LastName  types.Nullable[string] `json:"last_name"`
	IsActive  types.Nullable[bool]   `json:"is_active"`
}

type SetPasswordInput struct {
	Password string `json:"password" binding:"required,min=8"`
}

type SetUserRolesInput struct {
	RoleIDs []string `json:"role_ids"`
}

type UpdateRoleParams struct {
	Name        types.Nullable[string]
	Description types.Nullable[string]
}

type UpdateTeamParams struct {
	Name        types.Nullable[string]
	Description types.Nullable[string]
}

type TeamFilter struct {
	Search *string `form:"search"`
}

// RoleInput has the whole matrix. Permissions are replaced, not patched, so the editor
// cannot leave a half-applied set of grants.
type RoleInput struct {
	Name        string              `json:"name" binding:"required"`
	Description *string             `json:"description"`
	Permissions map[string][]string `json:"permissions"`
}

type UpdateRoleInput struct {
	Name        types.Nullable[string] `json:"name"`
	Description types.Nullable[string] `json:"description"`
}

type TeamInput struct {
	Name        string   `json:"name" binding:"required"`
	Description *string  `json:"description"`
	MemberIDs   []string `json:"member_ids"`
	RoleIDs     []string `json:"role_ids"`
}

type UpdateTeamInput struct {
	Name        types.Nullable[string] `json:"name"`
	Description types.Nullable[string] `json:"description"`
}

type SetTeamMembersInput struct {
	MemberIDs []string `json:"member_ids"`
}

type SetTeamRolesInput struct {
	RoleIDs []string `json:"role_ids"`
}

type AuditFilter struct {
	ActorID *string    `form:"actor_id" binding:"omitempty,uuid"`
	Module  *string    `form:"module"`
	Action  *string    `form:"action"`
	From    *time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To      *time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
}

type TokenPair struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshToken    string
	RefreshExpires  time.Time
}

type Me struct {
	User        domain.User         `json:"user"`
	Roles       []string            `json:"roles"`
	Permissions map[string][]string `json:"permissions"`
}

type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	AdminUsername   string
	AdminEmail      string
	AdminPassword   string

	// OIDC. FrontendURL is where the callback sends the browser on success. AuthPublicURL is
	// the browser base used to build the IdP redirect_uri.
	FrontendURL   string
	AuthPublicURL string
}

const (
	TokenInvite        = "invite"
	TokenPasswordReset = "password_reset"
)

type SetLinkParams struct {
	UserID    uuid.UUID
	TokenHash string
	Purpose   string
	SentAt    time.Time
	// ResendAfter: an older link is replaced, a newer one is kept
	ResendAfter time.Time
}

type UserLink struct {
	UserID  uuid.UUID
	Purpose string
	SentAt  time.Time
}

type ResetPasswordInput struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type ForgotPasswordInput struct {
	Email string `json:"email" binding:"required,email"`
}
