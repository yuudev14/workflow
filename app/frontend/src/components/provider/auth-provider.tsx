"use client";

import React, { createContext, useCallback, useContext, useMemo } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import AuthService from "@/services/auth/auth";
import { AuthUser, Me } from "@/services/auth/auth.schema";
import { isPublicRoute } from "@/settings/routes";
import { usePathname } from "next/navigation";

export const ME_QUERY_KEY = ["auth", "me"];

type AuthContextValue = {
  user: AuthUser | null;
  roles: string[];
  permissions: Record<string, string[]>;
  /** UI-level check. The backend re-checks every request — this only decides what to show. */
  hasPermission: (module: string, action: string) => boolean;
  isLoading: boolean;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export const useAuth = (): AuthContextValue => {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return ctx;
};

// Stable identities so consumers do not re-render just because a render
// produced a fresh empty object.
const NO_PERMISSIONS: Record<string, string[]> = {};
const NO_ROLES: string[] = [];

const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const pathname = usePathname();
  const router = useRouter();
  const queryClient = useQueryClient();

  const isPublic = isPublicRoute(pathname);

  /**
   * Session recovery is just this query.
   *
   * Both tokens are httpOnly cookies that survive a reload, so there is no
   * bootstrap dance: the browser attaches the access cookie and /me either
   * answers or 401s. If the access token has expired, the axios interceptor
   * silently refreshes and retries before this query ever sees a failure.
   */
  const {
    data,
    isLoading: meLoading,
    isError,
  } = useQuery<Me>({
    queryKey: ME_QUERY_KEY,
    queryFn: AuthService.me,
    enabled: !isPublic,
    retry: false,
  });

  const logout = useCallback(async () => {
    try {
      await AuthService.logout();
    } finally {
      // Clear the cache before leaving: another user signing in on this
      // browser must never see the previous one's data.
      queryClient.clear();
      router.replace("/login");
    }
  }, [queryClient, router]);

  const permissions = data?.permissions ?? NO_PERMISSIONS;

  const hasPermission = useCallback(
    (module: string, action: string) => permissions[module]?.includes(action) ?? false,
    [permissions],
  );

  const value = useMemo<AuthContextValue>(
    () => ({
      user: data?.user ?? null,
      roles: data?.roles ?? NO_ROLES,
      permissions,
      hasPermission,
      isLoading: meLoading,
      logout,
    }),
    [data, permissions, hasPermission, meLoading, logout],
  );

  // Public pages render immediately — they must work with no session at all.
  if (isPublic) {
    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
  }

  // Hold the app back until we know who the user is, so protected pages never
  // flash before a redirect.
  if (meLoading) {
    return (
      <div className="flex h-screen items-center justify-center text-sm text-muted-foreground">
        Loading…
      </div>
    );
  }

  // /me failed even after the interceptor's refresh attempt: no usable session.
  if (isError || !data) {
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      router.replace("/login");
    }
    return null;
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export default AuthProvider;
