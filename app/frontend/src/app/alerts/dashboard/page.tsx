"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import MetricsService from "@/services/metrics/metrics";
import {
  BarBreakdown,
  KpiRow,
  Panel,
  PanelTitle,
  TrendChart,
  type BarRow,
  type BarTone,
} from "@/components/soar";

const SEV_TONE: Record<string, BarTone> = {
  critical: "rose",
  high: "amber",
  medium: "signal",
  low: "slate",
};

export default function Page() {
  const kpiQuery = useQuery({ queryKey: ["alert-kpis"], queryFn: () => MetricsService.getAlertKpis() });
  const summaryQuery = useQuery({
    queryKey: ["alerts-summary"],
    queryFn: () => AlertService.getAlertsSummary(),
  });

  const summary = summaryQuery.data;
  const maxSev = Math.max(...(summary?.bySeverity.map((s) => s.count) ?? [1]));
  const maxSrc = Math.max(...(summary?.bySource.map((s) => s.count) ?? [1]));

  const sevRows: BarRow[] =
    summary?.bySeverity.map((s) => ({
      label: s.severity[0].toUpperCase() + s.severity.slice(1),
      value: s.count / maxSev,
      display: s.count,
      tone: SEV_TONE[s.severity],
    })) ?? [];

  const srcRows: BarRow[] =
    summary?.bySource.map((s) => ({
      label: s.label,
      value: s.count / maxSrc,
      display: s.count,
      tone: "ink" as BarTone,
    })) ?? [];

  const pbRows: BarRow[] =
    summary?.topPlaybooks.map((p) => ({
      label: p.label,
      value: p.successRate / 100,
      display: `${p.successRate}%`,
      tone: (p.successRate >= 80 ? "moss" : "amber") as BarTone,
    })) ?? [];

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-5 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Alerts dashboard</h1>
            <p className="mt-1 text-[15px] text-ink-soft">Last 14 days · all sources</p>
          </div>
          <Link
            href="/alerts"
            className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110"
          >
            Open queue <ChevronRight className="size-3.5" />
          </Link>
        </div>

        <KpiRow metrics={kpiQuery.data} loading={kpiQuery.isLoading} />

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside={`${summary?.total ?? 0} total`}>Volume — last 14 days</PanelTitle>
            <TrendChart
              values={summary?.volume ?? []}
              startLabel="14 days ago"
              endLabel="today"
            />
          </Panel>
          <Panel>
            <PanelTitle>Alerts by severity</PanelTitle>
            <BarBreakdown rows={sevRows} />
          </Panel>
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <Panel>
            <PanelTitle>Alerts by source</PanelTitle>
            <BarBreakdown rows={srcRows} />
          </Panel>
          <Panel>
            <PanelTitle>Top playbooks triggered</PanelTitle>
            <BarBreakdown rows={pbRows} />
            <p className="mt-2 text-[12px] text-ink-faint">
              Bar = share of that playbook&apos;s runs on alerts this week · % = success rate
            </p>
          </Panel>
        </div>
      </div>
    </div>
  );
}
