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

export type Tasks = {
  id: string;
  workflow_id: string;
  name: string;
  description?: string | null;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  parameters?: Record<string, any> | null;
  config?: string | null;
  x: number;
  y: number;
  connector_name: string | null;
  connector_id: string | null
  operation: string;
  created_at: string;
  updated_at: string;
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

export type WorkflowDataToUpdate = Partial<Pick<Workflow, "name" | "description" | "trigger_type">>

export type CreateWorkflowPayload = Partial<
  Pick<Workflow, "name" | "description">
>;

export type UpdateHandlesPayload =  Record<string, Record<string, Partial<Pick<Edges, "source_handle" | "destination_handle">>>>

export type UpdateWorkflowPayload = {
  task: Pick<Workflow, "name" | "trigger_type" | "description">
  nodes: Tasks[]
  edges: Record<string, string[]>
  handles: UpdateHandlesPayload

}

export type WorkflowFilterPayload = Partial<
  Pick<Workflow, "name" | "trigger_type">
>;


