"use client";
import React, { useMemo, useState } from "react";
import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";

import { useQuery } from "@tanstack/react-query";
import WorkflowService from "@/services/worfklows/workflows";
import { Edges, TaskHistory } from "@/services/worfklows/workflows.schema";
import { PlaybookTaskHistoryNode } from "@/components/react-flow/schema";
import { Node } from "@xyflow/react";
import { FLOW_START_ID } from "@/settings/reactFlowIds";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";

const Page: React.FC<{ params: Promise<{ workflowHistoryId: string }> }> = ({
  params,
}) => {
  const { workflowHistoryId } = React.use(params);
  const [currentNode, setCurrentNode] = useState<TaskHistory>();
  const taskHistoryQuery = useQuery({
    queryKey: [`worfklow-task-history-${workflowHistoryId}`],
    queryFn: async () => {
      return await WorkflowService.getTaskHistoryByWorkflowHistoryId(
        workflowHistoryId
      );
    },
  });

  const setMappedNodes = (task: TaskHistory) => {
    const data: Node<PlaybookTaskHistoryNode> = {
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
    <div className="flex flex-1">
      <ReactFlowPlayground
        flowProps={{
          nodes: nodes,
          edges: edges,
          onNodeDoubleClick: (e, node) => {
            console.log(node.data);

            setCurrentNode(node.data as TaskHistory);
          },
        }}
      />
      {currentNode && (
        <div className="flex flex-col gap-2 bg-background px-5 py-7 border-l h-full w-[600px] max-w-[600px] right-0">
          <h2>{currentNode.name}</h2>
          <Tabs className="flex-1" defaultValue="output">
            <TabsList>
              <TabsTrigger value="output">Output</TabsTrigger>
              <TabsTrigger value="parameters">Parameters</TabsTrigger>
            </TabsList>

            <TabsContent
              className="flex-1 px-5 py-4 overflow-auto bg-accent/10"
              key={`workflow-history-node-output`}
              value="output">
              <pre>
                <code>{JSON.stringify(currentNode.result, null, 2)}</code>
              </pre>
            </TabsContent>
            <TabsContent
              className="flex-1 px-2 overflow-auto"
              key={`workflow-history-node-parameters`}
              value="parameters">
              {currentNode.parameters && (
                <Accordion
                  type="single"
                  collapsible
                  className="w-full"
                  defaultValue="item-1">
                  {Object.entries(currentNode.parameters).map(([key, val]) => (
                    <AccordionItem value={key}>
                      <AccordionTrigger className="font-bold text-lg">{key}</AccordionTrigger>
                      <AccordionContent className="flex flex-col gap-4 text-balance">
                        <pre className="bg-accent/20 py-2 px-1">
                          <code>{val as any}</code>
                        </pre>
                      </AccordionContent>
                    </AccordionItem>
                  ))}
                </Accordion>
              )}
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  );
};

export default Page;
