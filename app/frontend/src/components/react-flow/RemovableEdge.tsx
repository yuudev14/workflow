import React from "react";
import {
  BaseEdge,
  EdgeLabelRenderer,
  getBezierPath,
  useReactFlow,
  type EdgeProps,
  type Node,
} from "@xyflow/react";
import { X } from "lucide-react";
import { CONDITION_OUTPUT_HANDLE } from "@/settings/reactFlowIds";

// A condition-branch edge carries a source_handle that is a case id (not a
// directional source-*/target-* handle and not the raw output handle). We
// color those edges and tag them with the branch so routing is visible.
function isBranchHandle(handle?: string | null) {
  return (
    !!handle &&
    handle !== CONDITION_OUTPUT_HANDLE &&
    !handle.startsWith("source-") &&
    !handle.startsWith("target-")
  );
}

// The handle is a stable case id (a uuid). Resolve it to the branch's human
// label from the source node's cases so the edge reads "high severity" / "If"
// instead of the raw id. The id stays the wire handle; this is display only.
function branchLabel(node: Node | undefined, handle?: string | null): string {
  if (!handle) return "";
  if (handle === "else") return "Else";
  const params = (node?.data?.parameters ?? {}) as Record<string, unknown>;
  for (const val of Object.values(params)) {
    if (!Array.isArray(val)) continue;
    const idx = val.findIndex(
      (c) => c && typeof c === "object" && (c as { id?: string }).id === handle
    );
    if (idx >= 0) {
      const name = (val[idx] as { name?: string }).name?.trim();
      return name || (idx === 0 ? "If" : `Else if ${idx}`);
    }
  }
  return handle;
}

export default function RemovableEdge({
  id,
  source,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  sourceHandleId,
  style = {},
  markerEnd,
}: EdgeProps) {
  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const branch = isBranchHandle(sourceHandleId);
  const edgeStyle = branch ? { ...style, stroke: "var(--signal-dot)" } : style;

  const { setEdges, getNode } = useReactFlow();
  const label = branch ? branchLabel(getNode(source), sourceHandleId) : "";
  const onEdgeClick = () => {
    setEdges((edges) => edges.filter((edge) => edge.id !== id));
  };

  return (
    <>
      <BaseEdge path={edgePath} markerEnd={markerEnd} style={edgeStyle} />
      <EdgeLabelRenderer>
        <div
          className="nodrag nopan absolute h-4 w-4 rounded-full bg-accent"
          style={{
            transform: `translate(-50%, -50%) translate(${sourceX}px,${sourceY}px)`,
          }}
        />
        {branch && (
          <div
            className="nodrag nopan absolute -translate-x-1/2 -translate-y-1/2 rounded-full border border-signal-dot bg-card px-2 py-0.5 text-[11px] font-semibold text-signal-text"
            style={{
              transform: `translate(-50%, -50%) translate(${(sourceX + labelX) / 2}px,${
                (sourceY + labelY) / 2
              }px)`,
            }}
          >
            {label}
          </div>
        )}
        <div
          className="nodrag nopan pointer-events-auto absolute origin-center"
          style={{
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
          }}
        >
          <button
            className="flex h-[26px] w-[26px] cursor-pointer items-center justify-center rounded-full border-4 border-card bg-paper-sunken text-ink-soft hover:bg-rose-soft hover:text-rose-text"
            onClick={onEdgeClick}
          >
            <X size={12} />
          </button>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}
