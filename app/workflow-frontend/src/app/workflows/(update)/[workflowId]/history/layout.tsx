"use client";
import React, { useMemo } from "react";

import { readableDate } from "@/lib/utils";
import WorkflowService from "@/services/worfklows/workflows";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { MoveLeft } from "lucide-react";
import { TaskStatus } from "@/services/worfklows/workflows.schema";

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
    staleTime: 0,
    gcTime: 0,
  });

  const borderStatusIndicator = (status: TaskStatus) => {
    if (status === "in_progress") return "border-l-yellow-300";
    if (status === "success") return "border-l-green-300";
    if (status === "failed") return "border-l-red-300";
  };

  if (worflowHistoryQuery.isLoading) {
    return <></>;
  }
  return (
    <div className="flex h-[calc(100vh-4rem)]">
      <div className="w-[350px] bg-muted h-full p-4 overflow-auto">
        <Link
          href={`/workflows/${workflowId}`}
          className="flex items-center gap-1 text-xs underline">
          <MoveLeft size={12} /> back
        </Link>
        <h2 className="mt-2">Executions</h2>
        <ul className="mt-5">
          {worflowHistoryQuery.data?.entries.map((history) => (
            <li
              key={`history-${history.id}`}
              className={`flex flex-col border-l-5 ${borderStatusIndicator(
                history.status
              )} cursor-pointer ${
                history.id === historyId ? "bg-accent" : ""
              }`}>
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
      <>

      {children}
      </>
    </div>
  );
};

export default Layout;
