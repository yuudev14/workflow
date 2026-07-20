"use client";

import { useAuth } from "@/components/provider/auth-provider";

/**
 * Whether the signed-in user holds a grant. This only decides what to render —
 * the backend re-checks every request, so hiding a button is never the boundary.
 */
export function usePermission(module: string, action: string): boolean {
  const { hasPermission } = useAuth();
  return hasPermission(module, action);
}
