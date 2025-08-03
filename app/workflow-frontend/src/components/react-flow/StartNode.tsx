import {
  Handle,
  HandleType,
  Node,
  NodeProps,
  Position,
} from "@xyflow/react";
import { Play } from "lucide-react";
import { Avatar, AvatarFallback } from "../ui/avatar";
import { Tasks } from "@/services/worfklows/workflows.schema";

type NodeComponentProps = Node<Tasks>;

const HANDLES: {
  type: HandleType;
  position: Position;
  id: string;
}[] = [
  // Target Handles
  {
    type: "target",
    position: Position.Top,
    id: "target-top",
  },
  {
    type: "target",
    position: Position.Bottom,
    id: "target-bottom",
  },
  {
    type: "target",
    position: Position.Left,
    id: "target-left",
  },
  {
    type: "target",
    position: Position.Right,
    id: "target-right",
  },
  // Source Handles
  {
    type: "source",
    position: Position.Top,
    id: "source-top",
  },
  {
    type: "source",
    position: Position.Bottom,
    id: "source-bottom",
  },
  {
    type: "source",
    position: Position.Left,
    id: "source-left",
  },
  {
    type: "source",
    position: Position.Right,
    id: "source-right",
  },
];

const StartNode: React.FC<NodeProps<NodeComponentProps>> = (props) => {
  return (
    <div className="p-3 group rounded-xl">
      {/* Node Content */}
      <div className="opacity-0 group-hover:opacity-100">
        {HANDLES.map((handle) => (
                    <Handle
                      key={handle.id}
                      type={handle.type}
                      position={handle.position}
                      id={handle.id}
                      className="!w-2 !h-2"
                    />
                  ))}
      </div>
      
      <div className="flex items-center w-full gap-3">
        <Avatar className="size-10">
          <AvatarFallback>
            <Play className="size-6 text-primary" />
          </AvatarFallback>
        </Avatar>
      </div>

    </div>
  );
};

export default StartNode;
