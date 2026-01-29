import { Handle, HandleType, Position } from "@xyflow/react";
import React from "react";

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

const RenderHandles: React.FC<{shouldRenderNode?: boolean}> = ({shouldRenderNode = true}) => {
  return (
    <div className="opacity-0 group-hover:opacity-100">
      {HANDLES.map((handle) => (
        <Handle
          key={handle.id}
          type={handle.type}
          position={handle.position}
          id={handle.id}
          className={shouldRenderNode ? "!w-2 !h-2" : "opacity-0"}
        />
      ))}
    </div>
  );
};

export default RenderHandles;
