export interface Workflow {
  id: string;
  name: string;
  description?: string | null;
  trigger_type: string;
  created_at: string;
  updated_at: string;
  tasks?: Tasks[] | null;
  edges?: Edges[] | null;
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

export type UpdateWorkflowPayload = {
  task: Pick<Workflow, "name" | "trigger_type" | "description">
  nodes: Tasks[]
  edges: Record<string, string[]>

}

export type WorkflowFilterPayload = Partial<
  Pick<Workflow, "name" | "trigger_type">
>;

export type Tasks = {
  id: string;
  workflow_id: string;
  name: string;
  description?: string | null;
  parameters?: Record<string, any> | null;
  config?: string | null;
  x: number;
  y: number;
  connector_name: string;
  operation: string;
  created_at: string;
  updated_at: string;
};

export interface Edges {
  id: string;
  destination_id: string;
  source_id: string;
  workflow_id: string;
}
