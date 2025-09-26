"use client"
import React from "react";
import ReactFlowPlayground from "@/components/react-flow/ReactFlowPlayground";

import { useQuery } from '@tanstack/react-query';
import WorkflowService from "@/services/worfklows/workflows";

const Page: React.FC<{ params: Promise<{ workflowHistoryId: string }> }> = ({params}) => {
  const { workflowHistoryId } = React.use(params)
  const taskHistoryQuery = useQuery({
    queryKey: [`worfklow-task-history-${workflowHistoryId}`],
    queryFn: async () => {
      return await WorkflowService.getTaskHistoryByWorkflowHistoryId(workflowHistoryId);
    },
  });
  return (
    <ReactFlowPlayground
      flowProps={{
        nodes: [],
        edges: [],
      }}
    />
  );
};

export default Page;
