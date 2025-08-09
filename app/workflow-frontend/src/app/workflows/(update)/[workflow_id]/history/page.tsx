"use client";

import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";
import WorkflowService from "@/services/worfklows/workflows";
import { useQuery } from "@tanstack/react-query";
import React from "react";

const Pages: React.FC<{ params: Promise<{ workflow_id: string }> }> = ({
  params,
}) => {
  const { workflow_id: workflowId } = React.use(params);

  const worflowHistoryQuery = useQuery({
    queryKey: [`workflow-history-${workflowId}`],
    queryFn: async () => {
      return await WorkflowService.getWorkflowsHistoryByWorkflowId(workflowId);
    },
  });

  if (worflowHistoryQuery.isLoading) {
    return <></>
  }
  return (
    <div className="flex h-[calc(100vh-4rem)]">
      <div className="w-[350px] bg-muted h-full p-4">
        <h2>Executions</h2>
        <ul className="mt-5">
          {worflowHistoryQuery.data?.entries.map((history) => (
            <li
              key={`history-${history.id}`}
              className="flex flex-col p-2 border-l-5 border-l-green-300">
              <p>date</p>
              <span className="text-sm text-muted-foreground">asdasd</span>
            </li>
          ))}
        </ul>
      </div>
      <ReactFlowPlayground
        flowProps={{
          nodes: [],
          edges: [],
        }}
      />
    </div>
  );
};

export default Pages;
