"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { History } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { EmptyState, StatusPill, type PillVariant } from "@/components/soar";
import PlaybookService from "@/services/playbooks/playbooks";
import { cn, readableDate } from "@/lib/utils";

const STATUS_PILL: Record<string, PillVariant> = {
  success: "success",
  failed: "failed",
  in_progress: "running",
  skipped: "skipped",
};

const stalk = (status: string) =>
  status === "in_progress"
    ? "bg-amber-dot"
    : status === "success"
    ? "bg-moss-dot"
    : status === "failed"
    ? "bg-rose-dot"
    : "bg-line-strong";

// Global executions list. Same run rows as a playbook's Run history, but across
// every playbook. The app-wide status socket invalidates ["playbooks-history-all"]
// on playbook_status, so an open list flips to finished live.
const ExecutionsModal: React.FC = () => {
  const [open, setOpen] = useState(false);

  const query = useQuery({
    queryKey: ["playbooks-history-all"],
    queryFn: () => PlaybookService.getPlaybooksHistory(0, 50),
    enabled: open,
    staleTime: 0,
    gcTime: 0,
  });

  const entries = query.data?.entries ?? [];

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <button className="inline-flex items-center gap-1.5 rounded-sm border border-line px-2.5 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken">
          <History className="size-3.5" /> Executions
        </button>
      </DialogTrigger>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Executions</DialogTitle>
        </DialogHeader>

        <div className="mt-2 max-h-[60vh] overflow-auto">
          {query.isLoading ? (
            <div className="flex flex-col gap-1.5">
              {Array.from({ length: 6 }).map((_, i) => (
                <div
                  key={i}
                  className="h-14 animate-pulse rounded-md bg-paper-sunken"
                />
              ))}
            </div>
          ) : entries.length === 0 ? (
            <EmptyState
              icon={History}
              title="No executions yet"
              description="Trigger a playbook to see its runs here."
              className="py-10"
            />
          ) : (
            <div className="flex flex-col gap-1.5">
              {entries.map((h) => (
                <Link
                  key={h.id}
                  href={`/playbooks/${h.playbook_id}/history/${h.id}`}
                  onClick={() => setOpen(false)}
                  className="flex items-center gap-2.5 rounded-md border border-line bg-card px-3 py-2.5 shadow-sm transition-colors hover:bg-paper-sunken"
                >
                  <span
                    className={cn(
                      "size-2.5 shrink-0 rounded-full",
                      stalk(h.status)
                    )}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-[13px] font-semibold">
                      {h.playbook_data?.name ?? h.playbook_id}
                    </div>
                    <div className="truncate text-[11.5px] text-ink-faint">
                      {readableDate(h.triggered_at, "MMM D, HH:mm:ss")}
                    </div>
                  </div>
                  <StatusPill
                    variant={STATUS_PILL[h.status] ?? "neutral"}
                    noDot
                  />
                </Link>
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default ExecutionsModal;
