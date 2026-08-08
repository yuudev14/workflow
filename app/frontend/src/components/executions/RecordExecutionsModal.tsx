"use client";

import React, { useEffect, useMemo, useState } from "react";
import { History } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState, StatusPill, type PillVariant } from "@/components/soar";
import { cn, readableDate } from "@/lib/utils";
import RunHistoryDetail from "./RunHistoryDetail";

/** One entry of the `runs[]` both alert and incident details return. */
export interface RecordRun {
  playbook_history_id: string;
  playbook_id: string;
  playbook?: string | null;
  status: string;
  trigger_type?: string | null;
  triggered_by?: string | null;
  created_at: string;
}

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

// Executions for one alert or incident, in the same split modal as the global
// ExecutionsModal: left rail lists the record's runs, right pane replays the
// selected run's flow. The rows come from the record detail's `runs[]` rather
// than a history query - that list is already scoped to this record.
const RecordExecutionsModal: React.FC<{
  open: boolean;
  onOpenChange: (open: boolean) => void;
  moduleType: "alert" | "incident";
  runs: RecordRun[];
  action?: React.ReactNode;
}> = ({ open, onOpenChange, moduleType, runs, action }) => {
  const [selected, setSelected] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const filtered = useMemo(
    () => (statusFilter ? runs.filter((r) => r.status === statusFilter) : runs),
    [runs, statusFilter]
  );

  // Default the right pane to the most recent (first) run in the current view.
  useEffect(() => {
    if (!open) return;
    if (selected && filtered.some((r) => r.playbook_history_id === selected)) return;
    setSelected(filtered[0]?.playbook_history_id ?? null);
  }, [open, filtered, selected]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[80vh] max-w-[calc(100%-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[2000px]">
        <DialogHeader className="border-b border-line px-5 py-3">
          <DialogTitle>Executions on this {moduleType}</DialogTitle>
        </DialogHeader>

        <div className="flex min-h-0 flex-1">
          {/* left rail: filters + run list */}
          <div className="flex w-[340px] shrink-0 flex-col border-r border-line bg-card">
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
              {filtered.length === 0 ? (
                <EmptyState
                  icon={History}
                  title={statusFilter ? "No matching runs" : `No runs on this ${moduleType}`}
                  description={
                    statusFilter
                      ? "Try a different status filter."
                      : `Run a playbook on this ${moduleType} to see it here.`
                  }
                  action={statusFilter ? undefined : action}
                  className="py-10"
                />
              ) : (
                <div className="flex flex-col gap-1.5">
                  {filtered.map((r) => {
                    const active = r.playbook_history_id === selected;
                    return (
                      <button
                        key={r.playbook_history_id}
                        type="button"
                        onClick={() => setSelected(r.playbook_history_id)}
                        className={cn(
                          "flex items-center gap-2.5 rounded-md border px-3 py-2.5 text-left transition-colors",
                          active
                            ? "border-signal-dot/30 bg-signal-soft"
                            : "border-line bg-card shadow-sm hover:bg-paper-sunken"
                        )}
                      >
                        <span
                          className={cn("size-2.5 shrink-0 rounded-full", stalk(r.status))}
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-[13px] font-semibold">
                            {r.playbook ?? "Deleted playbook"}
                          </div>
                          <div className="truncate text-[11.5px] text-ink-faint">
                            {readableDate(r.created_at, "MMM D, HH:mm:ss")}
                            {r.trigger_type && ` · ${r.trigger_type}`}
                            {r.triggered_by && ` · ${r.triggered_by}`}
                          </div>
                        </div>
                        <StatusPill
                          variant={STATUS_PILL[r.status] ?? "neutral"}
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

export default RecordExecutionsModal;
