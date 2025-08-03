import {
  Node,
  NodeProps,
} from "@xyflow/react";
import { Play } from "lucide-react";
import { Avatar, AvatarFallback } from "../../ui/avatar";
import { Tasks } from "@/services/worfklows/workflows.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import RenderHandles from "./RenderHandles";
import DeleteNodeButton from "./DeleteNodeButton";


const StartNode: React.FC<NodeProps<Node<Tasks>>> = (props) => {

  return (
    <div className="p-3 group rounded-xl">
      {/* Node Content */}

      <div className="flex items-center w-full gap-3">
        <Avatar className="size-10">
          <AvatarFallback>
            <Play className="size-6 text-primary" />
          </AvatarFallback>
        </Avatar>
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

export default StartNode;
