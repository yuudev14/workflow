"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, Check, ChevronRight, Plus } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import type { IncidentStatus } from "@/services/incidents/incidents.schema";
import {
  LinkChip,
  StatusMenu,
  Timeline,
  SearchInput,
} from "@/components/soar";
import { Skeleton } from "@/components/ui/skeleton";
import { IncidentRow } from "./_components/IncidentRow";
import { INCIDENT_STATUS_OPTIONS } from "./_components/constants";
import { cn } from "@/lib/utils";

const TABS = ["Open", "Resolved", "All"] as const;

export default function Page() {
  const [tab, setTab] = React.useState<(typeof TABS)[number]>("Open");
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [statusOverride, setStatusOverride] = React.useState<Record<string, IncidentStatus>>({});

  const incidentsQuery = useQuery({
    queryKey: ["incidents"],
    queryFn: () => IncidentService.getIncidents(),
  });
  const incidents = incidentsQuery.data ?? [];

  const filtered = incidents.filter((i) => {
    const status = statusOverride[i.id] ?? i.status;
    if (tab === "Open") return status !== "resolved" && status !== "closed";
    if (tab === "Resolved") return status === "resolved" || status === "closed";
    return true;
  });
  const selected = incidents.find((i) => i.id === selectedId) ?? filtered[0] ?? incidents[0];

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-5 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Incidents</h1>
            <p className="mt-1 text-[15px] text-ink-soft">6 open · 1 SLA breach</p>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/incidents/dashboard"
              className="inline-flex items-center gap-2 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              Dashboard
            </Link>
            <button className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110">
              <Plus className="size-4" /> New incident
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex w-fit gap-1 rounded-sm border border-line bg-paper-sunken p-[3px]">
            {TABS.map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                className={cn(
                  "rounded-[6px] px-3 py-1.5 text-[13.5px] font-semibold transition-colors",
                  tab === t ? "bg-card text-foreground shadow-sm" : "text-ink-soft hover:text-foreground"
                )}
              >
                {t}
              </button>
            ))}
          </div>
          <SearchInput placeholder="Search incidents…" />
        </div>

        <div className="flex gap-3.5">
          <div className="flex-[1.35]">
            {incidentsQuery.isLoading ? (
              <div className="flex flex-col gap-2">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Skeleton key={i} className="h-[62px] rounded-md" />
                ))}
              </div>
            ) : (
              <div className="overflow-hidden rounded-md border border-line">
                {filtered.map((i) => (
                  <IncidentRow
                    key={i.id}
                    incident={{ ...i, status: statusOverride[i.id] ?? i.status }}
                    selected={selected?.id === i.id}
                    href={`/incidents/${i.id}`}
                    onHover={() => setSelectedId(i.id)}
                  />
                ))}
              </div>
            )}
          </div>

          {selected && (
            <div className="flex w-[340px] shrink-0 flex-col gap-3.5 rounded-md border border-line bg-card p-3.5">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="text-[15px] font-semibold">{selected.title}</div>
                  <div className="mt-1 text-[12.5px] text-ink-faint">
                    {selected.severity[0].toUpperCase() + selected.severity.slice(1)} · opened{" "}
                    {selected.openedAgo} · owner {selected.owner ?? "unassigned"}
                  </div>
                </div>
                <div className="flex flex-col items-end gap-1.5">
                  <StatusMenu
                    align="end"
                    value={statusOverride[selected.id] ?? selected.status}
                    options={INCIDENT_STATUS_OPTIONS}
                    onChange={(v) =>
                      setStatusOverride((p) => ({ ...p, [selected.id]: v as IncidentStatus }))
                    }
                  />
                  <Link
                    href={`/incidents/${selected.id}`}
                    className="inline-flex items-center gap-1 text-[13px] font-semibold text-ink-soft hover:text-foreground"
                  >
                    Open record <ChevronRight className="size-3.5" />
                  </Link>
                </div>
              </div>

              {selected.linkedAlerts && (
                <div className="flex flex-col gap-1.5">
                  <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                    Linked alerts
                  </label>
                  <div className="flex flex-wrap gap-1.5">
                    {selected.linkedAlerts.map((a) => (
                      <LinkChip key={a.id}>
                        <AlertTriangle />
                        {a.title.length > 22 ? a.title.slice(0, 20) + "…" : a.title}
                      </LinkChip>
                    ))}
                  </div>
                </div>
              )}

              {selected.timeline && (
                <div className="flex flex-col gap-1.5">
                  <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                    Timeline
                  </label>
                  <Timeline
                    entries={selected.timeline.map((t) => ({
                      title: t.title,
                      detail: t.detail && <span className="mono">{t.detail}</span>,
                      tone: t.tone,
                    }))}
                  />
                </div>
              )}

              <div className="mt-auto flex gap-2 pt-1">
                <button className="flex flex-1 items-center justify-center rounded-sm border border-line-strong px-3 py-2 text-[13px] font-semibold text-ink-soft hover:bg-paper-sunken">
                  Add note
                </button>
                <button className="flex flex-1 items-center justify-center gap-1.5 rounded-sm bg-primary px-3 py-2 text-[13px] font-semibold text-primary-foreground hover:brightness-110">
                  <Check className="size-3.5" /> Mark contained
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
