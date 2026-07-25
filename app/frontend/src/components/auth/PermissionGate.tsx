"use client";

import * as React from "react";
import { usePermission } from "@/hooks/usePermission";

/** Renders children only when the user holds the grant. Cosmetic — see usePermission. */
export function PermissionGate({
  module,
  action,
  fallback = null,
  children,
}: {
  module: string;
  action: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
}) {
  return <>{usePermission(module, action) ? children : fallback}</>;
}

export default PermissionGate;
