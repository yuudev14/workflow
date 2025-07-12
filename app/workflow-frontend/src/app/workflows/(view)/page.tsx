"use client";

import React from "react";
import { useQuery } from "@tanstack/react-query";
import WorkflowService from "@/services/worfklows/workflows";
import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";

const WorkflowListLoader = () => {
  return [...Array(5)].map((_, _i) => (
    <li key={`workflow-list-skeleton-${_i}`}>
      <Skeleton className="h-[81.33px] rounded-sm"></Skeleton>
    </li>
  ));
};

export default function Page() {
  const workflowQuery = useQuery({
    queryKey: ["workflow-lists"],
    queryFn: async () => {
      return WorkflowService.getWorkflows();
    },
  });

  return (
    <div className="flex h-full py-3">
      <ul className="flex flex-col w-full gap-4">
        {workflowQuery.isLoading ? (
          <WorkflowListLoader />
        ) : (
          workflowQuery.data?.entries?.map((workflow) => (
            <Link
              href={"/workflows/update/" + workflow.id}
              key={`playbook-${workflow.id}`}
              className="flex flex-col items-start w-full gap-2 p-4 text-sm leading-tight border rounded-sm dark:hover:bg-muted/30 hover:bg-background/30 bg-background shadow-2xs dark:shadow-none whitespace-nowrap dark:bg-muted/50 hover:text-sidebar-accent-foreground">
              <div className="flex flex-col w-full">
                <p className="text-lg font-medium">{workflow.name}</p>
                <p className="text-xs">active</p>
              </div>
            </Link>
          ))
        )}
      </ul>
    </div>
  );
}
