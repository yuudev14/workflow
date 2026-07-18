"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { History } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { EmptyState, StatusPill, type PillVariant } from "@/components/soar";
import PlaybookService from "@/services/playbooks/playbooks";
import { cn, readableDate } from "@/lib/utils";
import RunHistoryDetail from "./RunHistoryDetail";

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

// Left-rail status filters. `value` undefined means "all".
const FILTERS: { label: string; value?: string }[] = [
  { label: "All" },
  { label: "Success", value: "success" },
  { label: "Failed", value: "failed" },
  { label: "Running", value: "in_progress" },
];

// Run history for a single playbook, in a modal. Mirrors the /history route's
// split view (left rail list + right flow replay), but selection stays in the
// modal — no navigation. Shares the route's query key so an open Runs view and
// this modal read the same cache.
const PlaybookHistoryModal: React.FC<{
  playbookId: string;
  playbookName?: string;
}> = ({ playbookId, playbookName }) => {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const query = useQuery({
    queryKey: [`workflow-history-${playbookId}`],
    queryFn: () => PlaybookService.getPlaybooksHistoryByPlaybookId(playbookId),
    enabled: open,
    staleTime: 0,
    gcTime: 0,
  });

  const entries = query.data?.entries ?? [];
  const filtered = useMemo(
    () =>
      statusFilter ? entries.filter((e) => e.status === statusFilter) : entries,
    [entries, statusFilter]
  );

  // Default the right pane to the most recent (first) run in the current view.
  useEffect(() => {
    if (!open) return;
    if (selected && filtered.some((e) => e.id === selected)) return;
    setSelected(filtered[0]?.id ?? null);
  }, [open, filtered, selected]);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost">
          <History /> History
        </Button>
      </DialogTrigger>
      <DialogContent className="flex h-[80vh] max-w-[calc(100%-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-6xl">
        <DialogHeader className="border-b border-line px-5 py-3">
          <DialogTitle className="truncate">
            {playbookName ? `${playbookName} · Run history` : "Run history"}
          </DialogTitle>
        </DialogHeader>

        <div className="flex min-h-0 flex-1">
          {/* left rail: filters + run list */}
          <div className="flex w-[320px] shrink-0 flex-col border-r border-line bg-card">
            <div className="flex flex-wrap gap-1 border-b border-line p-3">
              {FILTERS.map((f) => {
                const active = statusFilter === f.value;
                return (
                  <button
                    key={f.label}
                    type="button"
                    onClick={() => setStatusFilter(f.value)}
                    className={cn(
                      "rounded-sm px-2.5 py-1 text-[12px] font-semibold transition-colors",
                      active
                        ? "bg-signal-soft text-signal-text"
                        : "text-ink-soft hover:bg-paper-sunken hover:text-ink"
                    )}
                  >
                    {f.label}
                  </button>
                );
              })}
            </div>

            <div className="flex-1 overflow-auto p-3">
              <p className="mb-2 px-1 text-[11px] font-semibold uppercase tracking-wide text-ink-faint">
                {filtered.length} run{filtered.length === 1 ? "" : "s"}
              </p>
              {query.isLoading ? (
                <div className="flex flex-col gap-1.5">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <div
                      key={i}
                      className="h-14 animate-pulse rounded-md bg-paper-sunken"
                    />
                  ))}
                </div>
              ) : filtered.length === 0 ? (
                <EmptyState
                  icon={History}
                  title="No runs"
                  description="No runs match this filter."
                  className="py-10"
                />
              ) : (
                <div className="flex flex-col gap-1.5">
                  {filtered.map((h) => {
                    const active = h.id === selected;
                    return (
                      <button
                        key={h.id}
                        type="button"
                        onClick={() => setSelected(h.id)}
                        className={cn(
                          "flex items-center gap-2.5 rounded-md border px-3 py-2.5 text-left transition-colors",
                          active
                            ? "border-signal-dot/30 bg-signal-soft"
                            : "border-line bg-card shadow-sm hover:bg-paper-sunken"
                        )}
                      >
                        <span
                          className={cn(
                            "size-2.5 shrink-0 rounded-full",
                            stalk(h.status)
                          )}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-[13px] font-semibold">
                            {readableDate(h.triggered_at, "MMM D, HH:mm:ss")}
                          </div>
                        </div>
                        <StatusPill
                          variant={STATUS_PILL[h.status] ?? "neutral"}
                          noDot
                        />
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          {/* right pane: flow replay of the selected run */}
          <div className="flex min-w-0 flex-1">
            {selected ? (
              <RunHistoryDetail key={selected} playbookHistoryId={selected} />
            ) : (
              <div className="flex flex-1 items-center justify-center">
                <EmptyState
                  icon={History}
                  title="Select a run"
                  description="Pick a run from the list to replay its flow."
                />
              </div>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default PlaybookHistoryModal;
