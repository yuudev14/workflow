import { TaskHistory, Tasks } from "@/services/playbooks/playbooks.schema";

export type PlaybookTaskNode = (Tasks | Partial<Partial<Tasks>>) & {
  label?: string
  // module-event trigger config, held on the start node (Phase 1: client-side
  // only — the backend persists trigger_type today, module/event is Phase 2/3).
  trigger_module?: "alert" | "incident"
  trigger_event?: "created" | "updated"
};

export type PlaybookTaskHistoryNode = (TaskHistory | Partial<Partial<TaskHistory>>) & {
  label?: string
};