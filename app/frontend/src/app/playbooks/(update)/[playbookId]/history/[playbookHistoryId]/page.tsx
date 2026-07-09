"use client";
import React, { useMemo, useState } from "react";
import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";

import { useQuery } from "@tanstack/react-query";
import PlaybookService from "@/services/playbooks/playbooks";
import { Edges, TaskHistory } from "@/services/playbooks/playbooks.schema";
import { PlaybookTaskHistoryNode } from "@/components/react-flow/schema";
import { Node } from "@xyflow/react";
import { FLOW_START_ID } from "@/settings/reactFlowIds";
import { RefreshCw, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { JsonTree, KVList, StatusPill, type PillVariant } from "@/components/soar";

const STATUS_PILL: Record<string, PillVariant> = {
  success: "success",
  failed: "failed",
  in_progress: "running",
  skipped: "skipped",
};

const Page: React.FC<{ params: Promise<{ playbookHistoryId: string }> }> = ({
  params,
}) => {
  const { playbookHistoryId } = React.use(params);
  const [currentNode, setCurrentNode] = useState<TaskHistory>();
  const [tab, setTab] = useState<"output" | "parameters">("output");
  const taskHistoryQuery = useQuery({
    queryKey: [`worfklow-task-history-${playbookHistoryId}`],
    queryFn: async () => {
      return await PlaybookService.getTaskHistoryByPlaybookHistoryId(playbookHistoryId);
    },
  });

  const setMappedNodes = (task: TaskHistory) => {
    const data: Node<PlaybookTaskHistoryNode> = {
      id: task.task_id,
      data: task.name === FLOW_START_ID ? { label: task.task_id, ...task } : task,
      position: { x: task.x, y: task.y },
      type: task.name === FLOW_START_ID ? "startNode" : "taskHistoryNode",
      draggable: true,
    };
    return data;
  };

  const setMappedEdges = (edge: Edges) => ({
    id: edge.id,
    source: edge.source_id,
    target: edge.destination_id,
    sourceHandle: edge.source_handle || "source-top",
    targetHandle: edge.destination_handle || "target-top",
    type: "edgeHistory",
  });

  const nodes = useMemo(() => {
    if (taskHistoryQuery.data == undefined) return [];
    return taskHistoryQuery.data.tasks.map(setMappedNodes);
  }, [taskHistoryQuery.data]);

  const edges = useMemo(() => {
    if (taskHistoryQuery.data == undefined) return [];
    return taskHistoryQuery.data.edges.map(setMappedEdges);
  }, [taskHistoryQuery.data]);

  const failed = currentNode?.status === "failed";
  const output = failed ? currentNode?.error ?? {} : currentNode?.result ?? {};

  // Parameters: split code / multiline strings (rendered as a code block) from
  // short scalars (rendered as key/value rows), so a code snippet's source isn't
  // squeezed into a right-aligned text cell.
  const paramEntries = currentNode?.parameters
    ? Object.entries(currentNode.parameters)
    : [];
  const isCodey = (k: string, v: unknown) =>
    typeof v === "string" && (k === "code" || v.includes("\n") || v.length > 60);
  const codeParams = paramEntries.filter(([k, v]) => isCodey(k, v)) as [string, string][];
  const scalarParams = paramEntries
    .filter(([k, v]) => !isCodey(k, v))
    .map(([k, v]) => ({
      k,
      v: typeof v === "object" ? JSON.stringify(v) : String(v),
    }));

  return (
    <div className="flex flex-1">
      <ReactFlowPlayground
        flowProps={{
          nodes,
          edges,
          onNodeDoubleClick: (_e, node) => setCurrentNode(node.data as TaskHistory),
        }}
      />
      {currentNode && (
        <aside className="flex h-full w-[380px] shrink-0 flex-col border-l border-line bg-card">
          {/* header: name + status/context subtitle */}
          <div className="flex items-start justify-between gap-3 border-b border-line px-5 py-4">
            <div className="min-w-0">
              <div className="truncate text-[15px] font-semibold">{currentNode.name}</div>
              <div className="mt-1.5 flex items-center gap-2">
                <StatusPill variant={STATUS_PILL[currentNode.status] ?? "neutral"} />
                {(currentNode.connector_name || currentNode.operation) && (
                  <span className="truncate text-[11.5px] text-ink-faint">
                    {currentNode.connector_name ?? currentNode.operation}
                  </span>
                )}
              </div>
            </div>
            <button
              onClick={() => setCurrentNode(undefined)}
              className="flex size-7 shrink-0 items-center justify-center rounded-sm text-ink-soft hover:bg-paper-sunken hover:text-foreground"
            >
              <X className="size-4" />
            </button>
          </div>

          {/* action row */}
          <div className="border-b border-line px-5 py-3">
            <button
              className="inline-flex w-fit items-center gap-1.5 rounded-sm border border-line-strong px-2.5 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
              title="Retry from this step (coming soon)"
            >
              <RefreshCw className="size-3.5" /> Retry from here
            </button>
          </div>

          {/* tabs + content */}
          <div className="flex flex-1 flex-col overflow-hidden px-5 py-4">
            <div className="mb-3 inline-flex w-fit gap-1 rounded-sm border border-line bg-paper-sunken p-1">
              {(["output", "parameters"] as const).map((t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setTab(t)}
                  className={cn(
                    "rounded-[6px] px-3 py-1.5 text-[12px] font-semibold capitalize transition-colors",
                    tab === t ? "bg-card text-ink shadow-sm" : "text-ink-soft hover:text-ink"
                  )}
                >
                  {t}
                </button>
              ))}
            </div>

            <div className="flex-1 overflow-auto">
              {tab === "output" ? (
                <JsonTree data={output} />
              ) : codeParams.length || scalarParams.length ? (
                <div className="flex flex-col gap-4">
                  {codeParams.map(([k, v]) => (
                    <div key={k} className="flex flex-col gap-1.5">
                      <div className="text-[11px] font-semibold uppercase tracking-wide text-ink-soft">
                        {k}
                      </div>
                      <pre className="overflow-x-auto rounded-md border border-line bg-paper-sunken p-3.5 font-mono text-[13px] leading-[1.75] text-ink whitespace-pre">
                        {v}
                      </pre>
                    </div>
                  ))}
                  {scalarParams.length > 0 && (
                    <div className="flex flex-col gap-1.5">
                      <div className="text-[11px] font-semibold uppercase tracking-wide text-ink-soft">
                        Parameters used
                      </div>
                      <KVList items={scalarParams} />
                    </div>
                  )}
                </div>
              ) : (
                <p className="text-[13px] text-ink-faint">No parameters recorded.</p>
              )}
            </div>
          </div>
        </aside>
      )}
    </div>
  );
};

export default Page;
