"use client";

import React from "react";
import Link from "next/link";
import moment from "moment";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Play, Zap } from "lucide-react";

import PlaybookService from "@/services/playbooks/playbooks";
import { Playbook } from "@/services/playbooks/playbooks.schema";
import { Skeleton } from "@/components/ui/skeleton";
import {
  EmptyState,
  FilterChips,
  Glyph,
  SearchInput,
  StatusPill,
  connectorGlyph,
  type PillVariant,
} from "@/components/soar";

type RunInfo = { variant: PillVariant; label: string; when: string };

function TriggerBadge({ trigger }: { trigger?: string | null }) {
  const manual = (trigger ?? "").toLowerCase().includes("manual");
  return (
    <span className="inline-flex w-fit items-center gap-1.5 rounded-full border border-line bg-paper-sunken px-2.5 py-1 text-[12px] font-semibold text-ink-soft">
      {manual ? <Play className="size-3" /> : <Zap className="size-3" />}
      {trigger ? trigger.charAt(0).toUpperCase() + trigger.slice(1) : "Manual"}
    </span>
  );
}

export default function Page() {
  const [search, setSearch] = React.useState("");
  const [trigger, setTrigger] = React.useState("all");

  const playbooksQuery = useQuery({
    queryKey: ["playbook-lists"],
    queryFn: () => PlaybookService.getPlaybooks(),
  });

  // Latest run per playbook → drives the status pill (real data).
  const historyQuery = useQuery({
    queryKey: ["playbook-history-recent"],
    queryFn: () => PlaybookService.getPlaybooksHistory(0, 200),
  });

  const latestRun = React.useMemo(() => {
    const map = new Map<string, RunInfo>();
    for (const h of historyQuery.data?.entries ?? []) {
      if (map.has(h.playbook_id)) continue; // history is newest-first
      map.set(h.playbook_id, {
        variant: h.status === "success" ? "success" : "failed",
        label: h.status === "success" ? "Success" : "Failed",
        when: h.triggered_at,
      });
    }
    return map;
  }, [historyQuery.data]);

  const filtered = React.useMemo(() => {
    const list = playbooksQuery.data?.entries ?? [];
    return list.filter((p: Playbook) => {
      const matchesSearch = p.name.toLowerCase().includes(search.toLowerCase());
      const matchesTrigger =
        trigger === "all" || (p.trigger_type ?? "").toLowerCase() === trigger;
      return matchesSearch && matchesTrigger;
    });
  }, [playbooksQuery.data, search, trigger]);

  const loading = playbooksQuery.isLoading;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <FilterChips
          value={trigger}
          onChange={setTrigger}
          chips={[
            { value: "all", label: "All triggers" },
            { value: "webhook", label: "Webhook" },
            { value: "manual", label: "Manual" },
          ]}
        />
        <SearchInput
          placeholder="Search playbooks…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {loading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-[62px] rounded-md" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          title={search ? `No playbooks match “${search}”` : "No playbooks yet"}
          description="Try a different search, or create a playbook to fill the gap."
        />
      ) : (
        <div className="overflow-hidden rounded-md border border-line">
          <div className="grid grid-cols-[1.6fr_0.9fr_0.9fr_0.9fr_auto] gap-3 bg-paper-sunken px-3.5 py-2.5 text-[12px] font-semibold uppercase tracking-wide text-ink-faint">
            <div>Name</div>
            <div>Trigger</div>
            <div>Last run</div>
            <div>When</div>
            <div />
          </div>
          {filtered.map((p) => {
            const g = connectorGlyph(p.name);
            const run = latestRun.get(p.id);
            return (
              <Link
                key={p.id}
                href={`/playbooks/${p.id}`}
                className="grid grid-cols-[1.6fr_0.9fr_0.9fr_0.9fr_auto] items-center gap-3 border-t border-line bg-card px-3.5 py-3 hover:bg-paper-sunken"
              >
                <div className="flex items-center gap-2.5">
                  <Glyph icon={g.icon} tone={g.tone} />
                  <div className="min-w-0">
                    <div className="truncate text-[14px] font-semibold">{p.name}</div>
                    {p.description && (
                      <div className="truncate text-[12.5px] text-ink-faint">
                        {p.description}
                      </div>
                    )}
                  </div>
                </div>
                <TriggerBadge trigger={p.trigger_type} />
                <div>
                  {run ? (
                    <StatusPill variant={run.variant}>{run.label}</StatusPill>
                  ) : (
                    <StatusPill variant="neutral">No runs</StatusPill>
                  )}
                </div>
                <div className="font-mono text-xs text-ink-soft">
                  {run ? (
                    <>
                      {moment(run.when).fromNow()}
                      <span className="block text-[11.5px] text-ink-faint">
                        {moment(run.when).format("HH:mm:ss")}
                      </span>
                    </>
                  ) : (
                    <span className="text-ink-faint">—</span>
                  )}
                </div>
                <ChevronRight className="size-4 text-ink-faint" />
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
