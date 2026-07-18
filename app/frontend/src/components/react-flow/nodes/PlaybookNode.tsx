import { Node, NodeProps } from "@xyflow/react";
import { Tasks } from "@/services/playbooks/playbooks.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import { Glyph, connectorGlyph } from "@/components/soar";
import { cn } from "@/lib/utils";
import RenderHandles from "./RenderHandles";
import DeleteNodeButton from "./DeleteNodeButton";

const PlaybookNode: React.FC<NodeProps<Node<Tasks>>> = (props) => {
  const g = connectorGlyph(props.data.connector_name ?? props.data.name);
  const configured = Boolean(props.data.connector_id && props.data.operation);

  return (
    <div
      className={cn(
        "group relative w-[202px] rounded-md border-[1.5px] bg-card px-3 py-2.5 shadow-md",
        configured ? "border-moss-dot" : "border-line-strong",
        props.selected && "outline outline-2 outline-offset-2 outline-signal-dot"
      )}
    >
      {configured && (
        <span className="absolute -right-1.5 -top-1.5 size-4 rounded-full border-2 border-card bg-moss-dot" />
      )}
      <div className="flex items-center gap-2.5">
        <Glyph icon={g.icon} tone={g.tone} />
        <div className="min-w-0">
          <div className="truncate text-[13.5px] font-semibold leading-tight">
            {props.data.connector_name ?? "Step"}
          </div>
          <div className="truncate text-[12px] text-ink-faint">{props.data.name}</div>
        </div>
      </div>

      {props.id != FLOW_SELECT_TRIGGER_ID && (
        <>
          <RenderHandles />
          <DeleteNodeButton nodeId={props.id} />
        </>
      )}
    </div>
  );
};

export default PlaybookNode;
