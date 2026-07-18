import { Node, NodeProps } from "@xyflow/react";
import { TaskHistory } from "@/services/playbooks/playbooks.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import { Glyph, connectorGlyph } from "@/components/soar";
import { cn } from "@/lib/utils";
import RenderHandles from "./RenderHandles";
import { useMemo } from "react";

const TaskHistoryNode: React.FC<NodeProps<Node<TaskHistory>>> = (props) => {
  const g = connectorGlyph(props.data.connector_name ?? props.data.name);

  const ring = useMemo(() => {
    switch (props.data.status) {
      case "success":
        return "border-moss-dot shadow-[0_0_0_3px_var(--moss-soft)]";
      case "failed":
        return "border-rose-dot shadow-[0_0_0_3px_var(--rose-soft)]";
      case "in_progress":
        return "border-amber-dot shadow-[0_0_0_3px_var(--amber-soft)]";
      default:
        return "border-line-strong";
    }
  }, [props.data.status]);

  return (
    <div
      className={cn(
        "group relative w-[202px] rounded-md border-[1.5px] bg-card px-3 py-2.5 shadow-md",
        ring,
        props.selected && "outline outline-2 outline-offset-2 outline-signal-dot"
      )}
    >
      <div className="flex items-center gap-2.5">
        <Glyph icon={g.icon} tone={g.tone} />
        <div className="min-w-0">
          <div className="truncate text-[13.5px] font-semibold leading-tight">
            {props.data.connector_name ?? "Step"}
          </div>
          <div className="truncate text-[12px] text-ink-faint">{props.data.name}</div>
        </div>
      </div>

      {props.id != FLOW_SELECT_TRIGGER_ID && <RenderHandles shouldRenderNode={false} />}
    </div>
  );
};

export default TaskHistoryNode;
