package domain

import (
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
	ID           uuid.UUID    `json:"id"`
	Username     string       `json:"username"`
	Email        string       `json:"email"`
	PasswordHash *string      `json:"-"`
	FirstName    *string      `json:"first_name"`
	LastName     *string      `json:"last_name"`
	AuthProvider AuthProvider `json:"auth_provider"`
	ExternalID   *string      `json:"external_id,omitempty"`
	IsActive     bool         `json:"is_active"`
	LastLoginAt  *time.Time   `json:"last_login_at"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsBuiltin   bool      `json:"is_builtin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
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