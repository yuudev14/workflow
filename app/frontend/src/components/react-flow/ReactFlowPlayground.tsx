import {
  ReactFlow,
  Controls,
  Background,
  ReactFlowProps,
  Node,
  Edge,
  MiniMap,
  MarkerType,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useTheme } from "next-themes";
import { useMemo } from "react";
import { ConditionNode, PlaybookNode, StartNode, TaskHistoryNode } from "./nodes";
import RemovableEdge from "./RemovableEdge";
import EdgeHistory from "./EdgeHistory";

const nodeTypes = {
  playbookNodes: PlaybookNode,
  startNode: StartNode,
  taskHistoryNode: TaskHistoryNode,
  conditionNode: ConditionNode
};

const edgeTypes = {
  removableEdge: RemovableEdge,
  edgeHistory: EdgeHistory
};

export const defaultEdgeOptions = {
  type: "removableEdge",
  markerEnd: { type: MarkerType.ArrowClosed, width: 10, height: 10},
  style: {
      strokeWidth: 3,
  }
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const ReactFlowPlayground = <T extends Record<string, any>>({
  flowProps,
}: {
  flowProps: ReactFlowProps<Node<T>, Edge>;
}) => {
  const { theme } = useTheme();

  const reactFlowTheme = useMemo(() => {
    return ["dark", "light", "system", undefined].includes(theme)
      ? (theme as "dark" | "light" | "system" | undefined)
      : "system";
  }, [theme]);

  return (
    <div className="w-full h-full">
      <ReactFlow
        colorMode={reactFlowTheme}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        defaultEdgeOptions={defaultEdgeOptions}
        zoomOnDoubleClick={false}
        {...flowProps}>
        <MiniMap />
        <Background bgColor="var(--paper-sunken)" color="var(--line-strong)" gap={22} />
        <Controls />
      </ReactFlow>
    </div>
  );
};

export default ReactFlowPlayground;
