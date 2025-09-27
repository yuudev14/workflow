import {
  Node,
  NodeProps,
} from "@xyflow/react";
import { Workflow } from "lucide-react";
import { Avatar, AvatarFallback } from "../../ui/avatar";
import { TaskHistory } from "@/services/worfklows/workflows.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";
import RenderHandles from "./RenderHandles";
import DeleteNodeButton from "./DeleteNodeButton";
import { useMemo } from "react";
import { borderStatusIndicator } from "@/lib/utils";


const TaskHistoryNode: React.FC<NodeProps<Node<TaskHistory>>> = (props) => {

  const statusIndicator = useMemo(() => {
    return borderStatusIndicator(props.data.status)
  }, [props.data.status])

  return (
    <div className={`p-3 border-2 ${statusIndicator} group rounded-[7px]`}>
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
        </>
      )}
    </div>
  );
};

export default TaskHistoryNode;
