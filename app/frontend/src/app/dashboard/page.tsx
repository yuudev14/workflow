"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";

import MetricsService from "@/services/metrics/metrics";
import AlertService from "@/services/alerts/alerts";
import IncidentService from "@/services/incidents/incidents";
import {
  BarBreakdown,
  Donut,
  KpiRow,
  PageShell,
  Panel,
  PanelTitle,
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

function SectionHeader({ title, href, cta }: { title: string; href: string; cta: string }) {
  return (
    <div className="flex items-center justify-between">
      <h2 className="text-[18px]">{title}</h2>
      <Link
        href={href}
        className="inline-flex items-center gap-1 text-[13.5px] font-semibold text-signal-text hover:brightness-110"
      >
        {cta} <ChevronRight className="size-3.5" />
      </Link>
    </div>
  );
}

export default function Page() {
  const playbookKpis = useQuery({ queryKey: ["playbook-kpis"], queryFn: () => MetricsService.getPlaybookKpis() });
  const alertKpis = useQuery({ queryKey: ["alert-kpis"], queryFn: () => MetricsService.getAlertKpis() });
  const incidentKpis = useQuery({ queryKey: ["incident-kpis"], queryFn: () => MetricsService.getIncidentKpis() });
  const alertsSummary = useQuery({ queryKey: ["alerts-summary"], queryFn: () => AlertService.getAlertsSummary() });
  const incidentsSummary = useQuery({ queryKey: ["incidents-summary"], queryFn: () => IncidentService.getIncidentsSummary() });

  const aSum = alertsSummary.data;
  const iSum = incidentsSummary.data;

  const maxSev = Math.max(...(aSum?.bySeverity.map((s) => s.count) ?? [1]));
  const sevRows: BarRow[] =
    aSum?.bySeverity.map((s) => ({
      label: s.severity[0].toUpperCase() + s.severity.slice(1),
      value: s.count / maxSev,
      display: s.count,
      tone: SEV_TONE[s.severity],
    })) ?? [];

  const slices: DonutSlice[] =
    iSum?.statusMix.map((s) => ({
      label: s.status[0].toUpperCase() + s.status.slice(1),
      value: s.count,
      color: STATUS_COLOR[s.status],
    })) ?? [];

  return (
    <PageShell
      title="Dashboard"
      subtitle="Operational overview across playbooks, alerts, and incidents."
      className="gap-8"
    >
      {/* Playbooks */}
      <section className="flex flex-col gap-3">
        <SectionHeader title="Playbooks" href="/playbooks" cta="Open playbooks" />
        <KpiRow metrics={playbookKpis.data} loading={playbookKpis.isLoading} />
      </section>

      {/* Alerts */}
      <section className="flex flex-col gap-3">
        <SectionHeader title="Alerts" href="/alerts/dashboard" cta="Alerts dashboard" />
        <KpiRow metrics={alertKpis.data} loading={alertKpis.isLoading} />
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside={`${aSum?.total ?? 0} total`}>Volume — last 14 days</PanelTitle>
            <TrendChart values={aSum?.volume ?? []} startLabel="14 days ago" endLabel="today" />
          </Panel>
          <Panel>
            <PanelTitle>Alerts by severity</PanelTitle>
            <BarBreakdown rows={sevRows} />
          </Panel>
        </div>
      </section>

      {/* Incidents */}
      <section className="flex flex-col gap-3">
        <SectionHeader title="Incidents" href="/incidents/dashboard" cta="Incidents dashboard" />
        <KpiRow metrics={incidentKpis.data} loading={incidentKpis.isLoading} />
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside="2h 14m">Mean time to resolve — last 8 weeks</PanelTitle>
            <TrendChart values={iSum?.mttrTrend ?? []} startLabel="8 weeks ago" endLabel="this week" />
          </Panel>
          <Panel>
            <PanelTitle>Incidents by status</PanelTitle>
            <Donut slices={slices} centerValue={iSum?.openTotal ?? 0} centerLabel="open" />
          </Panel>
        </div>
      </section>
    </PageShell>
  );
}
