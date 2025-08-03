import {
  Handle,
  HandleType,
  Node,
  NodeProps,
  Position,
  useReactFlow,
} from "@xyflow/react";
import { Play, Trash2 } from "lucide-react";
import { Avatar, AvatarFallback } from "../ui/avatar";
import { Tasks } from "@/services/worfklows/workflows.schema";
import { FLOW_SELECT_TRIGGER_ID } from "@/settings/reactFlowIds";

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
  const { deleteElements } = useReactFlow();

  const handleDelete = (e: any) => {
    e.stopPropagation(); // Prevents node selection when clicking delete
    deleteElements({ nodes: [{ id: props.id }] });
  };
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
          <button
            className="absolute p-1 border rounded opacity-0 bg-accent/80 -top-2 -right-2 group-hover:opacity-100 text-destructive/50 hover:text-destructive"
            onClick={handleDelete}
            title="Delete Node">
            <Trash2 className="w-4 h-4" />
          </button>
        </>
      )}
    </div>
  );
};

export default StartNode;
