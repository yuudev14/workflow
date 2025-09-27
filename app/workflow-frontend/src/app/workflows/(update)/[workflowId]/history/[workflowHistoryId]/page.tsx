"use client";
import React, { useMemo } from "react";
import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";

import { useQuery } from "@tanstack/react-query";
import WorkflowService from "@/services/worfklows/workflows";
import {
  Edges,
  TaskHistory,
  Tasks,
} from "@/services/worfklows/workflows.schema";
import { PlaybookTaskNode } from "@/components/react-flow/schema";
import { Node } from "@xyflow/react";
import { FLOW_START_ID } from "@/settings/reactFlowIds";

const Page: React.FC<{ params: Promise<{ workflowHistoryId: string }> }> = ({
  params,
}) => {
  const { workflowHistoryId } = React.use(params);
  const taskHistoryQuery = useQuery({
    queryKey: [`worfklow-task-history-${workflowHistoryId}`],
    queryFn: async () => {
      return await WorkflowService.getTaskHistoryByWorkflowHistoryId(
        workflowHistoryId
      );
    },
  });

  const setMappedNodes = (task: TaskHistory) => {
    const data: Node<PlaybookTaskNode> = {
      id: task.task_id,
      data:
        task.name === FLOW_START_ID
          ? {
              label: task.task_id,
              ...task,
            }
          : task,
      position: {
        x: task.x,
        y: task.y,
      },
      // type: task.name === FLOW_START_ID ? "startNode" : "playbookNodes",
      type: task.name === FLOW_START_ID ? "startNode" : "playbookNodes",
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
    type: "removableEdge",
  });

  const nodes = useMemo(() => {
    if (taskHistoryQuery.data == undefined) {
      return [];
    }
    return taskHistoryQuery.data.tasks.map(setMappedNodes);
  }, [taskHistoryQuery.data]);

  const edges = useMemo(() => {
    if (taskHistoryQuery.data == undefined) {
      return [];
    }
    return taskHistoryQuery.data.edges.map(setMappedEdges);
  }, [taskHistoryQuery.data]);


  return (
    <ReactFlowPlayground
      flowProps={{
        nodes: nodes,
        edges: edges,
      }}
    />
  );
};

export default Page;
