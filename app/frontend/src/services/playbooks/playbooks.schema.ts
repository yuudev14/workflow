export interface Playbook {
  id: string;
  name: string;
  description?: string | null;
  trigger_type?: string | null;
  trigger_parameters?: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
  tasks?: Tasks[] | null;
  edges?: Edges[] | null;
}

export interface PlaybookHistory {
  id: string;
  playbook_id: string;
  status: "success" | "failed";
  error: Record<string, any> | null;
  result: Record<string, any>;
  playbook_data: Pick<
    Playbook,
    "id" | "name" | "created_at" | "updated_at" | "description" | "trigger_type"
  >;
  triggered_at: string;
  edges: Edges[]
}

// Mirrors the API's actual history query params (PlaybookHistoryFilter in the Go
// backend): only playbook name (matched ILIKE) and playbook_id are honored.
export type PlaybookHistoryFilter = {
  name?: string;
  playbook_id?: string;
};

export type Tasks = {
  id: string;
  playbook_id: string;
  name: string;
  description: string | null;
  parameters: Record<string, unknown> | null;
  config: string | null;
  x: number;
  y: number;
  connector_name: string | null;
  connector_id: string | null;
  operation: string;
  created_at: string;
  updated_at: string;
};

export type TaskStatus = "success" | "failed" | "in_progress"

export type TaskHistory = Pick<
  Tasks,
  "id" | "name" | "description" | "parameters" | "config" |
  "x" | "y" | "connector_name" | "connector_id" | "operation"
> & {
  playbook_history_id: string;
  task_id: string;
  status: TaskStatus;
  error: Record<string, unknown> | null;
  result: Record<string, unknown> | null;
  triggered_at: string;
  destination_ids: string[];
};

export interface Edges {
  id: string;
  destination_id: string;
  source_id: string;
  playbook_id: string;
  source_handle: string | null;
  destination_handle: string | null;
}

export type PlaybookDataToUpdate = Partial<
  Pick<Playbook, "name" | "description" | "trigger_type" | "trigger_parameters">
>;

export type CreatePlaybookPayload = Partial<
  Pick<Playbook, "name" | "description">
>;

export type UpdateHandlesPayload = Record<
  string,
  Record<string, Partial<Pick<Edges, "source_handle" | "destination_handle">>>
>;

export type UpdatePlaybookPayload = {
  task: Pick<Playbook, "name" | "trigger_type" | "trigger_parameters" | "description">;
  nodes: Tasks[];
  edges: Record<string, string[]>;
  handles: UpdateHandlesPayload;
};

export type PlaybookFilterPayload = Partial<
  Pick<Playbook, "name" | "trigger_type">
>;
