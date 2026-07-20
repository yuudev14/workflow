package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuthProvider mirrors the auth_provider_type pg enum.
type AuthProvider string

const (
	AuthProviderLocal AuthProvider = "local"
	AuthProviderOIDC  AuthProvider = "oidc"
	AuthProviderLDAP  AuthProvider = "ldap"
)

// Permission modules and actions are stored as TEXT, so these constants are
// the source of truth. Adding a module is a code change, not a migration —
// keep this list in sync with the frontend's settings/permissions.ts.
const (
	ModulePlaybooks  = "playbooks"
	ModuleAlerts     = "alerts"
	ModuleIncidents  = "incidents"
	ModuleConnectors = "connectors"
	ModuleAgent      = "agent"
	ModuleSchedules  = "schedules"
	ModuleSettings   = "settings"
)

const (
	ActionRead    = "read"
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionExecute = "execute"
	ActionApprove = "approve"
)

// PermissionModules and PermissionActions bound what the roles matrix may
// contain; anything else is rejected on write and would grant nothing anyway.
var PermissionModules = []string{
	ModulePlaybooks, ModuleAlerts, ModuleIncidents, ModuleConnectors,
	ModuleAgent, ModuleSchedules, ModuleSettings,
}

var PermissionActions = []string{
	ActionRead, ActionCreate, ActionUpdate, ActionDelete,
	ActionExecute, ActionApprove,
}

func IsValidPermissionModule(module string) bool {
	for _, m := range PermissionModules {
		if m == module {
			return true
		}
	}
	return false
}

func IsValidPermissionAction(action string) bool {
	for _, a := range PermissionActions {
		if a == action {
			return true
		}
	}
	return false
}

// Builtin role names seeded by the auth migration.
const (
	RoleAdmin   = "admin"
	RoleAnalyst = "analyst"
	RoleViewer  = "viewer"
)

type User struct {
	ID           uuid.UUID    `db:"id" json:"id"`
	Username     string       `db:"username" json:"username"`
	Email        string       `db:"email" json:"email"`
	PasswordHash *string      `db:"password_hash" json:"-"`
	FirstName    *string      `db:"first_name" json:"first_name"`
	LastName     *string      `db:"last_name" json:"last_name"`
	AuthProvider AuthProvider `db:"auth_provider" json:"auth_provider"`
	ExternalID   *string      `db:"external_id" json:"external_id,omitempty"`
	IsActive     bool         `db:"is_active" json:"is_active"`
	LastLoginAt  *time.Time   `db:"last_login_at" json:"last_login_at"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updated_at"`
}

// UserWithRoles is the list/detail shape the admin UI reads. Roles come from
// an aggregate in the same query rather than a per-row lookup.
type UserWithRoles struct {
	User
	Roles []string `db:"roles" json:"roles"`
}

type Role struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	IsBuiltin   bool      `db:"is_builtin" json:"is_builtin"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// RoleWithPermissions is the matrix editor's shape: the role plus its grants
// rendered as {module: [actions]}.
type RoleWithPermissions struct {
	Role
	Permissions map[string][]string `json:"permissions"`
}

type Permission struct {
	Module string `json:"module"`
	Action string `json:"action"`
}

// PermissionSet is one user's effective grants, unioned across their roles.
type PermissionSet []Permission

func (p PermissionSet) Has(module, action string) bool {
	for _, perm := range p {
		if perm.Module == module && perm.Action == action {
			return true
		}
	}
	return false
}

// ToMap renders the set as {module: [actions]} for the /me payload.
func (p PermissionSet) ToMap() map[string][]string {
	out := make(map[string][]string)
	for _, perm := range p {
		out[perm.Module] = append(out[perm.Module], perm.Action)
	}
	return out
}

type Team struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type TeamWithMembers struct {
	Team
	Members []TeamMember `json:"members"`
}

type TeamMember struct {
	ID       uuid.UUID `db:"id" json:"id"`
	Username string    `db:"username" json:"username"`
	Email    string    `db:"email" json:"email"`
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// AuditLog is the read shape. ActorUsername is joined in and stays nil for
// system-generated rows and for actors whose account was later removed
// (actor_id is ON DELETE SET NULL — the trail outlives the user).
type AuditLog struct {
	ID            uuid.UUID       `db:"id" json:"id"`
	ActorID       *uuid.UUID      `db:"actor_id" json:"actor_id"`
	ActorUsername *string         `db:"actor_username" json:"actor_username"`
	Module        string          `db:"module" json:"module"`
	Action        string          `db:"action" json:"action"`
	EntityID      *string         `db:"entity_id" json:"entity_id"`
	Detail        json.RawMessage `db:"detail" json:"detail"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
}

type AuditEntry struct {
	ActorID  *uuid.UUID
	Module   string
	Action   string
	EntityID *string
	Detail   map[string]any
}

type AuthUser struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}
