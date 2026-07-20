import { AuthUser } from "../auth/auth.schema";

export interface UserWithRoles extends AuthUser {
  roles: string[];
}

export interface Role {
  id: string;
  name: string;
  description: string | null;
  is_builtin: boolean;
  created_at: string;
  updated_at: string;
  /** {module: [actions]} — replaced wholesale by the matrix editor, never patched. */
  permissions: Record<string, string[]>;
}

export interface TeamMember {
  id: string;
  username: string;
  email: string;
}

export interface Team {
  id: string;
  name: string;
  description: string | null;
  created_at: string;
  updated_at: string;
  members: TeamMember[];
}

/** actor_username is null for system rows and for actors whose account was removed. */
export interface AuditLog {
  id: string;
  actor_id: string | null;
  actor_username: string | null;
  module: string;
  action: string;
  entity_id: string | null;
  detail: unknown;
  created_at: string;
}

export interface CreateUserPayload {
  username: string;
  email: string;
  password: string;
  first_name?: string | null;
  last_name?: string | null;
  role_ids?: string[];
}

/**
 * Partial update. The backend reads absence and presence differently — an
 * omitted key leaves the column alone, an explicit `null` clears it — so never
 * send a key you did not mean to change.
 */
export interface UpdateUserPayload {
  email?: string;
  first_name?: string | null;
  last_name?: string | null;
  is_active?: boolean;
}

export interface RolePayload {
  name: string;
  description?: string | null;
  permissions?: Record<string, string[]>;
}

export interface UpdateRolePayload {
  name?: string;
  description?: string | null;
}

export interface TeamPayload {
  name: string;
  description?: string | null;
  member_ids?: string[];
}

export interface UpdateTeamPayload {
  name?: string;
  description?: string | null;
}

export interface UserFilter {
  offset?: number;
  limit?: number;
  search?: string;
  is_active?: boolean;
  role?: string;
}

export interface TeamFilter {
  offset?: number;
  limit?: number;
  search?: string;
}

export interface AuditFilter {
  offset?: number;
  limit?: number;
  actor_id?: string;
  module?: string;
  action?: string;
  from?: string;
  to?: string;
}
