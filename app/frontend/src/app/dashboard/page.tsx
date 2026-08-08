"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import IncidentService from "@/services/incidents/incidents";
import PlaybookService from "@/services/playbooks/playbooks";
import type { DateRangeParams } from "@/services/common/range";
import {
  BarBreakdown,
  DateRangePicker,
  Donut,
  KpiCard,
  PageShell,
  Panel,
  PanelTitle,
  TrendChart,
  defaultRange,
  type BarRow,
  type BarTone,
  type DonutSlice,
} from "@/components/soar";
import { countDelta, percentDelta } from "@/lib/delta";
import { humanDuration } from "@/lib/utils";

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
  // One range drives all three sections, so the deltas are comparable.
  const [range, setRange] = React.useState<DateRangeParams>(defaultRange);

  const playbooksSummary = useQuery({
    queryKey: ["playbooks-summary", range],
    queryFn: () => PlaybookService.getPlaybooksSummary(range),
  });
  const alertsSummary = useQuery({
    queryKey: ["alerts-summary", range],
    queryFn: () => AlertService.getAlertsSummary(range),
  });
  const incidentsSummary = useQuery({
    queryKey: ["incidents-summary", range],
    queryFn: () => IncidentService.getIncidentsSummary(range),
  });

  const pSum = playbooksSummary.data;
  const aSum = alertsSummary.data;
  const iSum = incidentsSummary.data;

  const maxSev = Math.max(...(aSum?.by_severity.map((s) => s.count) ?? [1]));
  const sevRows: BarRow[] =
    aSum?.by_severity.map((s) => ({
      label: s.severity[0].toUpperCase() + s.severity.slice(1),
      value: s.count / maxSev,
      display: s.count,
      tone: SEV_TONE[s.severity],
    })) ?? [];

  const slices: DonutSlice[] =
    iSum?.status_mix.map((s) => ({
      label: s.status[0].toUpperCase() + s.status.slice(1),
      value: s.count,
      color: STATUS_COLOR[s.status],
    })) ?? [];

  const volume = aSum?.volume.map((v) => ({ bucket_start: v.bucket_start, value: v.count })) ?? [];
  const mttrSeries =
    iSum?.mttr_trend.map((p) => ({ bucket_start: p.bucket_start, value: p.avg_seconds })) ?? [];

  const runs = countDelta(pSum?.runs);
  const failed = countDelta(pSum?.failed);
  const successRate = percentDelta(pSum?.success_rate);
  const playbookCount = countDelta(pSum?.playbooks);
  const alertsCreated = countDelta(aSum?.created);
  const alertsResolved = countDelta(aSum?.resolved);
  const mttt = percentDelta(aSum?.mttt_seconds);
  const incidentsOpened = countDelta(iSum?.created);
  const incidentsResolved = countDelta(iSum?.resolved);
  const mttr = percentDelta(iSum?.mttr_seconds);

  return (
    <PageShell
      title="Dashboard"
      subtitle="Operational overview across playbooks, alerts, and incidents."
      className="gap-8"
    >
      <DateRangePicker value={range} onChange={setRange} />

      <section className="flex flex-col gap-3">
        <SectionHeader title="Playbooks" href="/playbooks" cta="Open playbooks" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <KpiCard
            label="Playbooks"
            value={pSum?.playbooks.current ?? "-"}
            delta={playbookCount?.label}
            deltaDirection={playbookCount?.direction}
          />
          <KpiCard
            label="Runs"
            value={pSum?.runs.current ?? "-"}
            delta={runs?.label}
            deltaDirection={runs?.direction}
          />
          <KpiCard
            label="Failed runs"
            value={pSum?.failed.current ?? "-"}
            delta={failed?.label}
            deltaDirection={failed?.direction}
            deltaNegative
          />
          <KpiCard
            label="Success rate"
            value={pSum ? `${Math.round(pSum.success_rate.current * 100)}%` : "-"}
            delta={successRate?.label}
            deltaDirection={successRate?.direction}
          />
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHeader title="Alerts" href="/alerts" cta="Open queue" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <KpiCard label="Open alerts" value={aSum?.total ?? "-"} />
          <KpiCard
            label="Created"
            value={aSum?.created.current ?? "-"}
            delta={alertsCreated?.label}
            deltaDirection={alertsCreated?.direction}
            deltaNegative
          />
          <KpiCard
            label="Resolved"
            value={aSum?.resolved.current ?? "-"}
            delta={alertsResolved?.label}
            deltaDirection={alertsResolved?.direction}
          />
          <KpiCard
            label="Mean time to triage"
            value={aSum ? humanDuration(aSum.mttt_seconds.current) : "-"}
            delta={mttt?.label}
            deltaDirection={mttt?.direction}
            deltaNegative
          />
        </div>
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside={`${aSum?.created.current ?? 0} in range`}>Alert volume</PanelTitle>
            <TrendChart data={volume} bucket={aSum?.range.bucket ?? "day"} />
          </Panel>
          <Panel>
            <PanelTitle>Alerts by severity</PanelTitle>
            <BarBreakdown rows={sevRows} />
          </Panel>
        </div>
      </section>

      <section className="flex flex-col gap-3">
        <SectionHeader title="Incidents" href="/incidents" cta="Open cases" />
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <KpiCard label="Open incidents" value={iSum?.open_total ?? "-"} />
          <KpiCard
            label="Opened"
            value={iSum?.created.current ?? "-"}
            delta={incidentsOpened?.label}
            deltaDirection={incidentsOpened?.direction}
            deltaNegative
          />
          <KpiCard
            label="Resolved"
            value={iSum?.resolved.current ?? "-"}
            delta={incidentsResolved?.label}
            deltaDirection={incidentsResolved?.direction}
          />
          <KpiCard
            label="Mean time to resolve"
            value={iSum ? humanDuration(iSum.mttr_seconds.current) : "-"}
            delta={mttr?.label}
            deltaDirection={mttr?.direction}
            deltaNegative
          />
        </div>
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside={iSum ? humanDuration(iSum.mttr_seconds.current) : undefined}>
              Mean time to resolve
            </PanelTitle>
            <TrendChart
              data={mttrSeries}
              bucket={iSum?.range.bucket ?? "day"}
              tone="moss"
              formatValue={(v) => humanDuration(v)}
            />
          </Panel>
          <Panel>
            <PanelTitle>Incidents by status</PanelTitle>
            <Donut slices={slices} centerValue={iSum?.open_total ?? 0} centerLabel="open" />
          </Panel>
        </div>
      </section>
    </PageShell>
  );
}
