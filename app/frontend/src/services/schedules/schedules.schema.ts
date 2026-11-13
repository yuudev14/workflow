export interface Schedule {
  id: string;
  name: string;
  playbook_id: string;
  playbook_name: string;
  playbook_trigger_type: string | null;
  playbook_deleted: boolean;
  cron: string;
  timezone: string;
  enabled: boolean;
  parameters: Record<string, unknown>;
  next_run_at: string;
  last_run_at: string | null;
  created_at: string;
}

export interface CreateSchedulePayload {
  name: string;
  playbook_id: string;
  cron: string;
  timezone: string;
  enabled: boolean;
  parameters: Record<string, unknown>;
}

/** The playbook of a schedule never changes after create. */
export type UpdateSchedulePayload = Partial<Omit<CreateSchedulePayload, "playbook_id">>;
