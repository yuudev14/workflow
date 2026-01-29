import { useReactFlow } from "@xyflow/react";
import { Trash2 } from "lucide-react";
import React from "react";

const DeleteNodeButton = ({ nodeId }: { nodeId: string }) => {
  const { deleteElements } = useReactFlow();
  const handleDelete = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.stopPropagation(); // Prevents node selection when clicking delete
    deleteElements({ nodes: [{ id: nodeId }] });
  };
  return (
    <button
      className="absolute p-1 border rounded opacity-0 bg-accent/80 -top-2 -right-2 group-hover:opacity-100 text-destructive/50 hover:text-destructive"
      onClick={handleDelete}
      title="Delete Node">
      <Trash2 className="w-4 h-4" />
    </button>
  );
};

export default DeleteNodeButton;
