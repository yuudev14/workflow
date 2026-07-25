/**
 * Mirror of the backend vocabulary in `app/ytsoar/internal/domain/auth.go`.
 * The columns are TEXT, so a string that exists here but not there is accepted
 * on write and then grants nothing — keep the two lists in sync.
 */
export const PERMISSION_MODULES = [
  "playbooks",
  "alerts",
  "incidents",
  "connectors",
  "agent",
  "schedules",
  "settings",
] as const;

export const PERMISSION_ACTIONS = [
  "read",
  "create",
  "update",
  "delete",
  "execute",
  "approve",
] as const;

export type PermissionModule = (typeof PERMISSION_MODULES)[number];
export type PermissionAction = (typeof PERMISSION_ACTIONS)[number];

export const MODULE_LABELS: Record<PermissionModule, string> = {
  playbooks: "Playbooks",
  alerts: "Alerts",
  incidents: "Incidents",
  connectors: "Connectors",
  agent: "Agent",
  schedules: "Schedules",
  settings: "Settings",
};

export const ACTION_LABELS: Record<PermissionAction, string> = {
  read: "Read",
  create: "Create",
  update: "Update",
  delete: "Delete",
  execute: "Execute",
  approve: "Approve",
};

/** Role names seeded by the auth migration; these cannot be edited or deleted. */
export const BUILTIN_ROLES = ["admin", "analyst", "viewer"];
