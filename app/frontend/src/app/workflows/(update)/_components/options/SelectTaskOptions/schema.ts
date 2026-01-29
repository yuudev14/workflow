import { LucideProps } from "lucide-react";
import { TaskOperationType } from "../../../_providers/WorkflowOperationProvider";

export type OperationOption = {
  readonly label: string;
  readonly tooltip: React.ReactNode;
  readonly Icon: React.ForwardRefExoticComponent<Omit<LucideProps, "ref"> & React.RefAttributes<SVGSVGElement>>;
  readonly operation: Exclude<TaskOperationType, null>;
};

export type TaskOperationGroup = {
  readonly label: string;
  readonly iconClass?: string;
  readonly options: OperationOption[];
};