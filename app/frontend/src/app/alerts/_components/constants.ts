import type { StatusOption } from "@/components/soar";

export const ALERT_STATUS_OPTIONS: StatusOption[] = [
  { value: "new" },
  { value: "investigating" },
  { value: "resolved" },
  { value: "falsepos", label: "False positive", divider: true },
  { value: "closed" },
];
