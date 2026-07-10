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
import usePlaybookStatus from "@/hooks/usePlaybookStatus";

const STATUS_PILL: Record<string, PillVariant> = {
  success: "success",
  failed: "failed",
  in_progress: "running",
  skipped: "skipped",
};

// A condition step stores its branches as an array of case objects. Detect it
// so we can render readable branch rows instead of a raw JSON blob.
type CaseLike = {
  id?: string;
  expression?: string;
  left?: string;
  operator?: string;
  right?: string;
};

const isCaseArray = (v: unknown): v is CaseLike[] =>
  Array.isArray(v) &&
  v.length > 0 &&
  v.every((x) => x != null && typeof x === "object" && "id" in x);

const caseText = (c: CaseLike): string => {
  if (c.expression?.trim()) return c.expression.trim();
  const parts = [c.left, c.operator, c.right]
    .map((p) => (p ?? "").toString().trim())
    .filter(Boolean);
  return parts.length ? parts.join(" ") : "—";
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

  // live status: merges task_status rows into the query above as the run executes.
  const { connected } = usePlaybookStatus(playbookHistoryId);

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

  // Condition edges carry a branch handle (a case id, "else" or "output") that
  // doesn't exist on history nodes — React Flow would silently drop those edges.
  // This view is read-only, so anchor them to the bottom handle instead.
  const setMappedEdges = (edge: Edges) => ({
    id: edge.id,
    source: edge.source_id,
    target: edge.destination_id,
    sourceHandle: edge.source_handle?.startsWith("source-")
      ? edge.source_handle
      : edge.source_handle
        ? "source-bottom"
        : "source-top",
    targetHandle: edge.destination_handle || "target-top",
    type: "edgeHistory",
  });

  const nodes = useMemo(() => {
    if (taskHistoryQuery.data == undefined) return [];
    return taskHistoryQuery.data.tasks.map(setMappedNodes);
  }, [taskHistoryQuery.data]);

  const edges = useMemo(() => {
    if (taskHistoryQuery.data == undefined) return [];
    const { tasks, edges } = taskHistoryQuery.data;
    const statusOf = (id: string) =>
      tasks.find((t) => t.task_id === id)?.status;
    return edges.map((edge) => {
      // green when the edge was actually followed to a successful step — both
      // endpoints ran (status "success", i.e. not skipped and not failed).
      const followed =
        statusOf(edge.source_id) === "success" &&
        statusOf(edge.destination_id) === "success";
      return {
        ...setMappedEdges(edge),
        style: followed
          ? { stroke: "var(--moss-dot)", strokeWidth: 2 }
          : undefined,
      };
    });
  }, [taskHistoryQuery.data]);

  const failed = currentNode?.status === "failed";
  const output = failed ? currentNode?.error ?? {} : currentNode?.result ?? {};

  // Parameters: condition cases get their own readable branch list; code /
  // multiline strings render as a code block; short scalars as key/value rows —
  // so nothing is squeezed into a right-aligned text cell as raw JSON.
  const paramEntries = currentNode?.parameters
    ? Object.entries(currentNode.parameters)
    : [];
  const caseParams = paramEntries.filter(([, v]) => isCaseArray(v)) as [
    string,
    CaseLike[]
  ][];
  const rest = paramEntries.filter(([, v]) => !isCaseArray(v));
  const isCodey = (k: string, v: unknown) =>
    typeof v === "string" && (k === "code" || v.includes("\n") || v.length > 60);
  const codeParams = rest.filter(([k, v]) => isCodey(k, v)) as [string, string][];
  const scalarParams = rest
    .filter(([k, v]) => !isCodey(k, v))
    .map(([k, v]) => ({
      k,
      v: typeof v === "object" ? JSON.stringify(v) : String(v),
    }));

  return (
    <div className="relative flex flex-1">
      {connected && (
        <span className="absolute left-4 top-4 z-10 inline-flex items-center gap-1.5 rounded-sm border border-line bg-card px-2.5 py-1 text-[11.5px] font-semibold text-ink-soft shadow-sm">
          <span className="size-1.5 rounded-full bg-[var(--moss-dot)]" /> Live
        </span>
      )}
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
              ) : caseParams.length || codeParams.length || scalarParams.length ? (
                <div className="flex flex-col gap-4">
                  {caseParams.map(([k, cases]) => (
                    <div key={k} className="flex flex-col gap-1.5">
                      <div className="text-[11px] font-semibold uppercase tracking-wide text-ink-soft">
                        {k}
                      </div>
                      <div className="flex flex-col gap-1 rounded-md border border-line bg-paper-sunken p-2">
                        {cases.map((c, i) => (
                          <div
                            key={c.id ?? i}
                            className="flex items-center gap-2 rounded-sm px-1 py-1"
                          >
                            <span className="shrink-0 rounded-[5px] bg-card px-1.5 py-0.5 text-[10.5px] font-bold text-ink-soft">
                              {i === 0 ? "If" : `Else if ${i}`}
                            </span>
                            <code
                              className="min-w-0 flex-1 truncate font-mono text-[12px] text-ink"
                              title={caseText(c)}
                            >
                              {caseText(c)}
                            </code>
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
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
