"use client";
import React, { useMemo } from "react";

import { readableDate } from "@/lib/utils";
import WorkflowService from "@/services/worfklows/workflows";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname } from "next/navigation";

const Layout: React.FC<
  { params: Promise<{ workflowId: string }> } & Readonly<{
    children: React.ReactNode;
  }>
> = ({ params, children }) => {
  const { workflowId } = React.use(params);
  const pathname = usePathname();

  const historyId = useMemo(() => {
    const taskHistoryIdMatch = pathname.match(/history\/([a-fA-F0-9-]{36})/);
    return taskHistoryIdMatch ? taskHistoryIdMatch[1] : null;
  }, [pathname]);

  const worflowHistoryQuery = useQuery({
    queryKey: [`workflow-history-${workflowId}`],
    queryFn: async () => {
      return await WorkflowService.getWorkflowsHistoryByWorkflowId(workflowId);
    },
  });

  

  if (worflowHistoryQuery.isLoading) {
    return <></>;
  }
  return (
    <div className="flex h-[calc(100vh-4rem)]">
      <div className="w-[350px] bg-muted h-full p-4">
        <Link href={`/workflows/${workflowId}`}>back</Link>
        <h2>Executions</h2>
        <ul className="mt-5">
          {worflowHistoryQuery.data?.entries.map((history) => (
            <li
              key={`history-${history.id}`}
              className={`flex flex-col border-l-5 border-l-green-300 cursor-pointer ${history.id === historyId ? "bg-accent" : ""}`}>
              <Link
                href={`/workflows/${workflowId}/history/${history.id}`}
                className="p-2 pl-4 cursor-pointer">
                <p>{readableDate(history.triggered_at)}</p>
                <span className="text-sm text-muted-foreground">
                  {history.status}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      </div>
      {children}
    </div>
  );
};

export default Layout;
