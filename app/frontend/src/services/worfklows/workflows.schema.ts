export interface Workflow {
  id: string;
  name: string;
  description?: string | null;
  trigger_type?: string | null;
  created_at: string;
  updated_at: string;
  tasks?: Tasks[] | null;
  edges?: Edges[] | null;
}

export interface WorkflowHistory {
  id: string;
  workflow_id: string;
  status: "success" | "failed";
  error: Record<string, any> | null;
  result: Record<string, any>;
  workflow_data: Pick<
    Workflow,
    "id" | "name" | "created_at" | "updated_at" | "description" | "trigger_type"
  >;
  triggered_at: string;
  edges: Edges[]
}

export type WorkflowHistoryFilter = Partial<
  Omit<WorkflowHistory, "workflow_data" | "triggered_at"> & {
    triggered_at_start: Date;
    triggered_at_end: Date;
    workflow_history_id: string;
  }
>;

export type Tasks = {
  id: string;
  workflow_id: string;
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
  workflow_history_id: string;
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
  workflow_id: string;
  source_handle: string | null;
  destination_handle: string | null;
}

export interface WorkflowTriggerType {
  id: string;
  name: string;
  description?: string | null;
}

export type WorkflowDataToUpdate = Partial<
  Pick<Workflow, "name" | "description" | "trigger_type">
>;

export type CreateWorkflowPayload = Partial<
  Pick<Workflow, "name" | "description">
>;

export type UpdateHandlesPayload = Record<
  string,
  Record<string, Partial<Pick<Edges, "source_handle" | "destination_handle">>>
>;

export type UpdateWorkflowPayload = {
  task: Pick<Workflow, "name" | "trigger_type" | "description">;
  nodes: Tasks[];
  edges: Record<string, string[]>;
  handles: UpdateHandlesPayload;
};

export type WorkflowFilterPayload = Partial<
  Pick<Workflow, "name" | "trigger_type">
>;
