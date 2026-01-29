import {
  Node,
  NodeProps,
} from "@xyflow/react";
import { Workflow } from "lucide-react";
import { Avatar, AvatarFallback } from "../../ui/avatar";
import { Tasks } from "@/services/worfklows/workflows.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import RenderHandles from "./RenderHandles";
import DeleteNodeButton from "./DeleteNodeButton";


const PlaybookNode: React.FC<NodeProps<Node<Tasks>>> = (props) => {

  return (
    <div className="p-3 group rounded-xl">
      {/* Node Content */}
      <div className="flex items-center w-full gap-3">
        <Avatar className="size-10">
          <AvatarFallback>
            <Workflow className="size-6 text-primary" />
          </AvatarFallback>
        </Avatar>
        <div className="flex flex-col items-start mr-11">
          <p className="text-lg font-medium">{props.data.connector_name}</p>
          <p className="text-sm text-muted-foreground">{props.data.name}</p>
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
