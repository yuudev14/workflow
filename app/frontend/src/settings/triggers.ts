// Static trigger types. Must stay in sync with the backend `trigger_type` enum
// (app/ytsoar/internal/domain/playbook.go) — the API rejects anything else on save.
export const TRIGGER_MANUAL = "manual"
export const TRIGGER_WEBHOOK = "webhook"
export const TRIGGER_REFERENCED = "referenced"
export const TRIGGER_ON_CREATE = "on_create"
export const TRIGGER_ON_UPDATE = "on_update"
export const TRIGGER_ON_DELETE = "on_delete"

// the module-event triggers share the same options panel (module, condition, ...)
export const MODULE_EVENT_TRIGGERS = [
  TRIGGER_ON_CREATE,
  TRIGGER_ON_UPDATE,
  TRIGGER_ON_DELETE,
] as const

export const isModuleEventTrigger = (t?: string | null) =>
  (MODULE_EVENT_TRIGGERS as readonly string[]).includes(t ?? "")
