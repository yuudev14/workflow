import { Handle, Node, NodeProps, Position, useReactFlow } from "@xyflow/react";
import { Trash2, Workflow } from "lucide-react";
import { Avatar, AvatarFallback } from "../ui/avatar";
import { Tasks } from "@/services/worfklows/workflows.schema";

type NodeComponentProps = Node<Tasks>;

const PlaybookNode: React.FC<NodeProps<NodeComponentProps>> = (props) => {
  const { deleteElements } = useReactFlow();

  const handleDelete = (e: any) => {
    e.stopPropagation(); // Prevents node selection when clicking delete
    deleteElements({ nodes: [{ id: props.id }] });
  };

  return (
    <>
      {/* Source Handles */}
      <Handle type="source" position={Position.Top} id="top-handle" />
      <Handle type="source" position={Position.Bottom} id="bottom-handle" />
      <Handle type="source" position={Position.Left} id="left-handle" />
      <Handle type="source" position={Position.Right} id="right-handle" />

      {/* Node Content */}
      <div className="flex items-center w-full gap-3">
        <Avatar className="size-7">
          <AvatarFallback>
            <Workflow className="size-4 text-primary" />
          </AvatarFallback>
        </Avatar>
        <div className="mr-9">
          <p className="font-medium">{props.data.name}</p>
        </div>
      </div>

      {/* Target Handles */}
      <Handle type="target" position={Position.Top} id="target-top" />
      <Handle type="target" position={Position.Bottom} id="target-bottom" />
      <Handle type="target" position={Position.Left} id="target-left" />
      <Handle type="target" position={Position.Right} id="target-right" />
      <div className="absolute flex items-center justify-center w-full gap-2 -bottom-8">
        <button
          className="p-1 rounded hover:bg-destructive/50"
          onClick={handleDelete}
          title="Delete Node"
        >
          <Trash2 className="w-4 h-4 text-destructive" />
        </button>
      </div>
    </>
  );
};

export default PlaybookNode;
