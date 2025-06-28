import { Tasks } from "@/services/worfklows/workflows.schema";

export type PlaybookTaskNode = (Tasks | Partial<Partial<Tasks>>) & {
  label?: string
};