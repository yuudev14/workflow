"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Clock, X } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import MetricsService from "@/services/metrics/metrics";
import {
  BarBreakdown,
  Donut,
  KpiRow,
  Panel,
  PanelTitle,
  StatusPill,
  TrendChart,
  type BarRow,
  type BarTone,
  type DonutSlice,
} from "@/components/soar";

const SEV_TONE: Record<string, BarTone> = { critical: "rose", high: "amber", medium: "signal", low: "slate" };
const STATUS_COLOR: Record<string, string> = {
  investigating: "var(--amber-dot)",
  contained: "var(--signal-dot)",
  resolved: "var(--moss-dot)",
  open: "var(--rose-dot)",
  closed: "var(--slate-dot)",
};

export default function Page() {
  const kpiQuery = useQuery({ queryKey: ["incident-kpis"], queryFn: () => MetricsService.getIncidentKpis() });
  const summaryQuery = useQuery({
    queryKey: ["incidents-summary"],
    queryFn: () => IncidentService.getIncidentsSummary(),
  });
  const summary = summaryQuery.data;

  const maxSev = Math.max(...(summary?.severityMix.map((s) => s.count) ?? [1]));
  const sevRows: BarRow[] =
    summary?.severityMix.map((s) => ({
      label: s.severity[0].toUpperCase() + s.severity.slice(1),
      value: s.count / maxSev,
      display: s.count,
      tone: SEV_TONE[s.severity],
    })) ?? [];

  const slices: DonutSlice[] =
    summary?.statusMix.map((s) => ({
      label: s.status[0].toUpperCase() + s.status.slice(1),
      value: s.count,
      color: STATUS_COLOR[s.status],
    })) ?? [];

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-5 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Incidents dashboard</h1>
            <p className="mt-1 text-[15px] text-ink-soft">Last 8 weeks</p>
          </div>
          <Link
            href="/incidents"
            className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110"
          >
            Open cases <ChevronRight className="size-3.5" />
          </Link>
        </div>

        <KpiRow metrics={kpiQuery.data} loading={kpiQuery.isLoading} />

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside="2h 14m">Mean time to resolve — last 8 weeks</PanelTitle>
            <TrendChart values={summary?.mttrTrend ?? []} startLabel="8 weeks ago" endLabel="this week" />
          </Panel>
          <Panel>
            <PanelTitle>By status</PanelTitle>
            <Donut slices={slices} centerValue={summary?.openTotal ?? 0} centerLabel="open" />
          </Panel>
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <Panel>
            <PanelTitle>By severity</PanelTitle>
            <BarBreakdown rows={sevRows} />
          </Panel>
          <Panel>
            <PanelTitle>SLA at risk</PanelTitle>
            <div className="flex flex-col gap-2.5">
              {summary?.slaAtRisk.map((s) => (
                <div key={s.id} className="flex items-center justify-between text-[12.5px]">
                  <span className="flex items-center gap-1.5">
                    {s.breached ? (
                      <X className="size-3.5 text-rose-dot" />
                    ) : (
                      <Clock className="size-3.5 text-amber-dot" />
                    )}
                    {s.title}
                  </span>
                  <StatusPill variant={s.breached ? "critical" : "high"} noDot className="px-2">
                    {s.left}
                  </StatusPill>
                </div>
              ))}
            </div>
          </Panel>
        </div>
      </div>
    </div>
  );
}
