"use client";

import React from "react";
import Link from "next/link";
import moment from "moment";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Zap } from "lucide-react";

import PlaybookService from "@/services/playbooks/playbooks";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, StatusPill } from "@/components/soar";
import { cn } from "@/lib/utils";

const Page = () => {
  const historyQuery = useQuery({
    queryKey: ["playbook-history-lists"],
    queryFn: () => PlaybookService.getPlaybooksHistory(0, 100),
  });

  const entries = historyQuery.data?.entries ?? [];

  if (historyQuery.isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-[58px] rounded-md" />
        ))}
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <EmptyState
        title="No executions yet"
        description="Runs will appear here once a playbook is triggered."
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-md border border-line">
      {entries.map((h) => {
        const ok = h.status === "success";
        return (
          <Link
            key={h.id}
            href={`/playbooks/${h.playbook_id}/history/${h.id}`}
            className="grid grid-cols-[26px_1fr_120px_140px_auto] items-center gap-3 border-t border-line bg-card px-3.5 py-3 first:border-t-0 hover:bg-paper-sunken"
          >
            <span
              className={cn("size-2.5 rounded-full", ok ? "bg-moss-dot" : "bg-rose-dot")}
            />
            <div className="min-w-0">
              <div className="truncate text-[13.5px] font-semibold">
                {h.playbook_data?.name ?? "Playbook run"}
              </div>
              <div className="mt-0.5 flex items-center gap-1.5 text-[12px] text-ink-faint">
                <Zap className="size-3" />
                {h.playbook_data?.trigger_type ?? "Manual"}
              </div>
            </div>
            <StatusPill variant={ok ? "success" : "failed"} />
            <span className="font-mono text-xs text-ink-soft">
              {moment(h.triggered_at).fromNow()}
            </span>
            <ChevronRight className="size-4 text-ink-faint" />
          </Link>
        );
      })}
    </div>
  );
};

export default Page;
