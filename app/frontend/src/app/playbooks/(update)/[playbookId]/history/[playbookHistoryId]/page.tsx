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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { JsonTree, KVList, StatusPill, type PillVariant } from "@/components/soar";

const STATUS_PILL: Record<string, PillVariant> = {
  success: "success",
  failed: "failed",
  in_progress: "running",
};

const Page: React.FC<{ params: Promise<{ playbookHistoryId: string }> }> = ({
  params,
}) => {
  const { playbookHistoryId } = React.use(params);
  const [currentNode, setCurrentNode] = useState<TaskHistory>();
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
  const paramItems = currentNode?.parameters
    ? Object.entries(currentNode.parameters).map(([k, v]) => ({
        k,
        v: typeof v === "object" ? JSON.stringify(v) : String(v),
      }))
    : [];

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
        <div className="flex h-full w-[380px] max-w-[380px] flex-col gap-3 border-l border-line bg-card px-5 py-5">
          <div className="flex items-start justify-between">
            <div>
              <div className="text-[15px] font-semibold">{currentNode.name}</div>
              <div className="mt-1 flex items-center gap-2">
                <StatusPill variant={STATUS_PILL[currentNode.status] ?? "neutral"} />
              </div>
            </div>
            <button
              onClick={() => setCurrentNode(undefined)}
              className="flex size-7 items-center justify-center rounded-sm text-ink-soft hover:bg-paper-sunken"
            >
              <X className="size-4" />
            </button>
          </div>

          <button
            className="inline-flex w-fit items-center gap-1.5 rounded-sm border border-line-strong px-2.5 py-1.5 text-[13px] font-semibold text-ink-soft hover:bg-paper-sunken"
            title="Retry from this step (coming soon)"
          >
            <RefreshCw className="size-3.5" /> Retry from here
          </button>

          <Tabs className="flex flex-1 flex-col" defaultValue="output">
            <TabsList>
              <TabsTrigger value="output">Output</TabsTrigger>
              <TabsTrigger value="parameters">Parameters</TabsTrigger>
            </TabsList>
            <TabsContent value="output" className="flex-1 overflow-auto">
              <JsonTree data={output} />
            </TabsContent>
            <TabsContent value="parameters" className="flex-1 overflow-auto">
              {paramItems.length ? (
                <KVList items={paramItems} />
              ) : (
                <p className="text-[13px] text-ink-faint">No parameters recorded.</p>
              )}
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  );
};

export default Page;
