import { LucideProps } from "lucide-react";
import { GlyphTone } from "@/components/soar";
import { TaskOperationType } from "../../../_providers/PlaybookOperationProvider";

export type OperationOption = {
  readonly label: string;
  readonly description: string;
  readonly tone: GlyphTone;
  readonly badge?: string;
  readonly badgeTone?: GlyphTone;
  readonly Icon: React.ForwardRefExoticComponent<Omit<LucideProps, "ref"> & React.RefAttributes<SVGSVGElement>>;
  readonly operation: Exclude<TaskOperationType, null>;
};

export type TaskOperationGroup = {
  readonly label: string;
  readonly options: OperationOption[];
};