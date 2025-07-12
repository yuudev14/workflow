import {
  ReactFlow,
  Controls,
  Background,
  ReactFlowProps,
  Node,
  Edge,
  MiniMap,
  BackgroundVariant,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useTheme } from "next-themes";
import { useMemo } from "react";
import PlaybookNode from "./PlaybookNode";

const nodeTypes = {
  playbookNodes: PlaybookNode,
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

  function resolveBgColor(mode: "dark" | "light" | "system" | undefined) {
    if (mode === "dark") return "black";
    if (mode === "light") return "white";
    if (mode === "system") {
      return window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "black"
        : "white";
    }
    return "white";
  }

  return (
    <div className="h-full w-full">
      <ReactFlow
        colorMode={reactFlowTheme}
        nodeTypes={nodeTypes}
        zoomOnDoubleClick={false}
        {...flowProps}>
        <MiniMap />
        <Background bgColor={resolveBgColor(reactFlowTheme)} />
        <Controls />
      </ReactFlow>
    </div>
  );
};

export default ReactFlowPlayground;
