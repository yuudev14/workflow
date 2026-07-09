import { Node, NodeProps } from "@xyflow/react";
import { Zap } from "lucide-react";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import { Glyph } from "@/components/soar";
import { PlaybookTaskNode } from "../schema";
import RenderHandles from "./RenderHandles";
import DeleteNodeButton from "./DeleteNodeButton";

const StartNode: React.FC<NodeProps<Node<PlaybookTaskNode>>> = (props) => {
  // module-event triggers show "alert · created"; keep the generic label hidden
  const detail =
    props.data?.trigger_module && props.data?.trigger_event
      ? `${props.data.trigger_module} · ${props.data.trigger_event}`
      : undefined;

  return (
    <div className="group relative flex w-[112px] flex-col items-center gap-1 rounded-md border-[1.5px] border-moss-dot bg-card px-2 py-3 text-center shadow-md">
      <Glyph icon={Zap} tone="signal" />
      <div className="text-[13px] font-semibold leading-tight">Trigger</div>
      {detail && (
        <div className="truncate text-[11px] capitalize text-ink-faint">{detail}</div>
      )}

      {props.id != FLOW_SELECT_TRIGGER_ID && (
        <>
          <RenderHandles />
          <DeleteNodeButton nodeId={props.id} />
        </>
      )}
    </div>
  );
};

export default StartNode;
