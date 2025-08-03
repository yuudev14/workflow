"use client";

import WorkflowService from '@/services/worfklows/workflows';
import { useQuery } from '@tanstack/react-query';
import React from 'react'
import WorkflowListLoader from '../_components/WorkflowListLoader';
import Link from 'next/link';



const Page = () => {
  const workflowHistoryQuery = useQuery({
    queryKey: ["workflow-history-lists"],
    queryFn: async () => {
      return WorkflowService.getWorkflowsHistory();
    },
  });
  return (
    <div className="flex h-full py-3">
      <ul className="flex flex-col w-full gap-4">
        {workflowHistoryQuery.isLoading ? (
          <WorkflowListLoader />
        ) : (
          workflowHistoryQuery.data?.entries?.map((workflow) => (
            <Link
              href={"/workflows/update/" + workflow.id}
              key={`playbook-${workflow.id}`}
              className="flex flex-col items-start w-full gap-2 p-4 text-sm leading-tight border rounded-sm dark:hover:bg-muted/30 hover:bg-background/30 bg-background shadow-2xs dark:shadow-none whitespace-nowrap dark:bg-muted/50 hover:text-sidebar-accent-foreground">
              <div className="flex flex-col w-full">
                <p className="text-lg font-medium">{workflow.workflow_data.name}</p>
                <p className="text-xs">{workflow.triggered_at}</p>
              </div>
            </Link>
          ))
        )}
      </ul>
    </div>
  )
}

export default Page