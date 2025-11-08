import { TaskHistory, Tasks } from "@/services/worfklows/workflows.schema";

export type PlaybookTaskNode = (Tasks | Partial<Partial<Tasks>>) & {
  label?: string
};

export type PlaybookTaskHistoryNode = (TaskHistory | Partial<Partial<TaskHistory>>) & {
  label?: string
};