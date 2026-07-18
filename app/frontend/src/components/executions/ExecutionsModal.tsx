"use client";

import React, { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { History, Search } from "lucide-react";

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

// Global executions across every playbook, in a split modal: left rail lists all
// runs (with filters), right pane replays the selected run's flow — same detail
// view as the /history route, but selection stays in the modal. The app-wide
// status socket invalidates ["playbooks-history-all"] on playbook_status, so an
// open list flips to finished live.
const ExecutionsModal: React.FC = () => {
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string | undefined>();
  const [nameSearch, setNameSearch] = useState("");
  const [debouncedName, setDebouncedName] = useState("");

  // Debounce the name box so we hit the API's `name` (ILIKE) filter at rest,
  // not on every keystroke.
  useEffect(() => {
    const t = setTimeout(() => setDebouncedName(nameSearch.trim()), 300);
    return () => clearTimeout(t);
  }, [nameSearch]);

  const query = useQuery({
    queryKey: ["playbooks-history-all", debouncedName],
    queryFn: () =>
      PlaybookService.getPlaybooksHistory(
        0,
        50,
        debouncedName ? { name: debouncedName } : {}
      ),
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
        <button className="inline-flex items-center gap-1.5 rounded-sm border border-line px-2.5 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken">
          <History className="size-3.5" /> Executions
        </button>
      </DialogTrigger>
      <DialogContent className="flex h-[80vh] max-w-[calc(100%-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[2000px]">
        <DialogHeader className="border-b border-line px-5 py-3">
          <DialogTitle>Executions</DialogTitle>
        </DialogHeader>

        <div className="flex min-h-0 flex-1">
          {/* left rail: filters + run list */}
          <div className="flex w-[340px] shrink-0 flex-col border-r border-line bg-card">
            <div className="border-b border-line p-3">
              <div className="relative">
                <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-ink-faint" />
                <input
                  value={nameSearch}
                  onChange={(e) => setNameSearch(e.target.value)}
                  placeholder="Search by playbook name…"
                  className="w-full rounded-sm border border-line bg-paper-sunken py-1.5 pl-8 pr-2.5 text-[12.5px] text-ink placeholder:text-ink-faint focus:border-signal-dot/40 focus:outline-none"
                />
              </div>
            </div>
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
                  title={
                    debouncedName || statusFilter
                      ? "No matching runs"
                      : "No executions yet"
                  }
                  description={
                    debouncedName || statusFilter
                      ? "Try a different name or status filter."
                      : "Trigger a playbook to see its runs here."
                  }
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

export default ExecutionsModal;
