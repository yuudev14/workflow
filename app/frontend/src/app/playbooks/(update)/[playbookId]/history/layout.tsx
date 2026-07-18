"use client";
import React, { useMemo } from "react";

import { readableDate } from "@/lib/utils";
import PlaybookService from "@/services/playbooks/playbooks";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { StatusPill, type PillVariant } from "@/components/soar";
import { cn } from "@/lib/utils";

const STATUS_PILL: Record<string, PillVariant> = {
  success: "success",
  failed: "failed",
  in_progress: "running",
  skipped: "skipped",
};

const Layout: React.FC<
  { params: Promise<{ playbookId: string }> } & Readonly<{
    children: React.ReactNode;
  }>
> = ({ params, children }) => {
  const { playbookId } = React.use(params);
  const pathname = usePathname();

  const historyId = useMemo(() => {
    const match = pathname.match(/history\/([a-fA-F0-9-]{36})/);
    return match ? match[1] : null;
  }, [pathname]);

  const historyQuery = useQuery({
    queryKey: [`workflow-history-${playbookId}`],
    queryFn: () => PlaybookService.getPlaybooksHistoryByPlaybookId(playbookId),
    staleTime: 0,
    gcTime: 0,
  });

  const stalk = (status: string) =>
    status === "in_progress"
      ? "bg-amber-dot"
      : status === "success"
      ? "bg-moss-dot"
      : "bg-rose-dot";

  return (
    <div className="flex h-[calc(100vh-4rem)]">
      <div className="w-[350px] shrink-0 overflow-auto border-r border-line bg-card p-4">
        <Link
          href={`/playbooks/${playbookId}`}
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-soft hover:text-foreground"
        >
          <ArrowLeft size={13} /> Back to editor
        </Link>
        <h2 className="mt-3">Run history</h2>
        <p className="mt-1 text-[13px] text-ink-faint">
          {historyQuery.data?.entries.length ?? 0} runs
        </p>

        <div className="mt-4 flex flex-col gap-1.5">
          {historyQuery.isLoading &&
            Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-14 animate-pulse rounded-md bg-paper-sunken" />
            ))}
          {historyQuery.data?.entries.map((history) => {
            const active = history.id === historyId;
            return (
              <Link
                key={history.id}
                href={`/playbooks/${playbookId}/history/${history.id}`}
                className={cn(
                  "flex items-center gap-2.5 rounded-md border px-3 py-2.5 transition-colors",
                  active
                    ? "border-signal-dot/30 bg-signal-soft"
                    : "border-line bg-card shadow-sm hover:bg-paper-sunken"
                )}
              >
                <span className={cn("size-2.5 shrink-0 rounded-full", stalk(history.status))} />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-[13px] font-semibold">
                    {readableDate(history.triggered_at, "MMM D, HH:mm:ss")}
                  </div>
                </div>
                <StatusPill variant={STATUS_PILL[history.status] ?? "neutral"} noDot />
              </Link>
            );
          })}
        </div>
      </div>
      <div className="flex flex-1 overflow-hidden">{children}</div>
    </div>
  );
};

export default Layout;
