"use client";

import React from "react";
import { useQuery } from "@tanstack/react-query";
import WorkflowService from "@/services/worfklows/workflows";
import Link from "next/link";

export default function Page() {
  const workflowQuery = useQuery({
    queryKey: ["workflow-lists"],
    queryFn: async () => {
      return WorkflowService.getWorkflows();
    },
  });

  if (workflowQuery.data === undefined) {
    return <></>;
  }

  return (
    <div className="flex h-full py-3">
      <ul className="flex flex-col w-full gap-4">
        {workflowQuery.data.entries.map((workflow) => (
          <Link
            href={"/workflows/" + workflow.id}
            key={`playbook-${workflow.id}`}
            className="flex flex-col items-start w-full gap-2 p-4 text-sm leading-tight border rounded-sm whitespace-nowrap dark:bg-muted/50 hover:bg-muted/30 hover:text-sidebar-accent-foreground">
            <div className="flex flex-col w-full">
              <p className="text-lg font-medium">{workflow.name}</p>
              <p className="text-xs">active</p>
            </div>
          </Link>
        ))}
      </ul>
    </div>
  );
}
