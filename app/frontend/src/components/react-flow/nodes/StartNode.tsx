import { Node, NodeProps } from "@xyflow/react";
import { Zap } from "lucide-react";
import { Tasks } from "@/services/playbooks/playbooks.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import { Glyph } from "@/components/soar";
import RenderHandles from "./RenderHandles";
import DeleteNodeButton from "./DeleteNodeButton";

const StartNode: React.FC<NodeProps<Node<Tasks>>> = (props) => {
  return (
    <div className="group relative flex w-[104px] flex-col items-center gap-1.5 rounded-md border-[1.5px] border-moss-dot bg-card px-2 py-3 text-center shadow-md">
      <Glyph icon={Zap} tone="signal" />
      <div className="text-[13px] font-semibold">Trigger</div>

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
