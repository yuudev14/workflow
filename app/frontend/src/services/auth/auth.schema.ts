export interface AuthUser {
  id: string;
  username: string;
  email: string;
  first_name?: string | null;
  last_name?: string | null;
  auth_provider: "local" | "oidc" | "ldap";
  is_active: boolean;
  last_login_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface LoginPayload {
  username: string;
  password: string;
}

/**
 * The tokens also arrive as httpOnly cookies, which is what the browser
 * actually authenticates with — these fields exist for clients without a
 * cookie jar (curl, the CLI) and are unused by the frontend.
 */
export interface SessionResponse {
  expires_at: string;
  access_token: string;
  refresh_token: string;
}

/**
 * permissions is {module: [actions]}, e.g. {playbooks: ["read", "execute"]}.
 * It drives which UI is shown; the backend re-checks every request, so this is
 * a convenience, never the boundary.
 */
export interface Me {
  user: AuthUser;
  roles: string[];
  permissions: Record<string, string[]>;
}
