"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import type { DateRangeParams } from "@/services/common/range";
import {
  BarBreakdown,
  DateRangePicker,
  KpiCard,
  Panel,
  PanelTitle,
  TrendChart,
  defaultRange,
  type BarRow,
  type BarTone,
} from "@/components/soar";
import { countDelta, percentDelta } from "@/lib/delta";
import { humanDuration } from "@/lib/utils";
import { SOURCE_LABEL } from "../_components/alertPresentation";

const SEV_TONE: Record<string, BarTone> = {
  critical: "rose",
  high: "amber",
  medium: "signal",
  low: "slate",
};

export default function Page() {
  const [range, setRange] = React.useState<DateRangeParams>(defaultRange);

  const summaryQuery = useQuery({
    queryKey: ["alerts-summary", range],
    queryFn: () => AlertService.getAlertsSummary(range),
  });

  const summary = summaryQuery.data;
  const maxSev = Math.max(...(summary?.by_severity.map((s) => s.count) ?? [1]));
  const maxSrc = Math.max(...(summary?.by_source.map((s) => s.count) ?? [1]));

  const sevRows: BarRow[] =
    summary?.by_severity.map((s) => ({
      label: s.severity[0].toUpperCase() + s.severity.slice(1),
      value: s.count / maxSev,
      display: s.count,
      tone: SEV_TONE[s.severity],
    })) ?? [];

  const srcRows: BarRow[] =
    summary?.by_source.map((s) => ({
      label: SOURCE_LABEL[s.source_kind],
      value: s.count / maxSrc,
      display: s.count,
      tone: "ink" as BarTone,
    })) ?? [];

  // success_rate is already a 0..1 fraction - the bar takes it as-is and only
  // the caption scales to a percentage.
  const pbRows: BarRow[] =
    summary?.top_playbooks.map((p) => ({
      label: p.label,
      value: p.success_rate,
      display: `${Math.round(p.success_rate * 100)}%`,
      tone: (p.success_rate >= 0.8 ? "moss" : "amber") as BarTone,
    })) ?? [];

  const volume = summary?.volume.map((v) => ({ bucket_start: v.bucket_start, value: v.count })) ?? [];
  const bucket = summary?.range.bucket ?? "day";
  const created = countDelta(summary?.created);
  const resolved = countDelta(summary?.resolved);
  const mttt = percentDelta(summary?.mttt_seconds);

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-5 px-6 py-8">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1>Alerts dashboard</h1>
            <p className="mt-1 text-[15px] text-ink-soft">
              Deltas compare against the preceding window of equal length.
            </p>
          </div>
          <Link
            href="/alerts"
            className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110"
          >
            Open queue <ChevronRight className="size-3.5" />
          </Link>
        </div>

        <DateRangePicker value={range} onChange={setRange} />

        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <KpiCard label="Open alerts" value={summary?.total ?? "-"} />
          <KpiCard
            label="Created"
            value={summary?.created.current ?? "-"}
            delta={created?.label}
            deltaDirection={created?.direction}
            deltaNegative
          />
          <KpiCard
            label="Resolved"
            value={summary?.resolved.current ?? "-"}
            delta={resolved?.label}
            deltaDirection={resolved?.direction}
          />
          <KpiCard
            label="Mean time to triage"
            value={summary ? humanDuration(summary.mttt_seconds.current) : "-"}
            delta={mttt?.label}
            deltaDirection={mttt?.direction}
            deltaNegative
          />
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside={`${summary?.created.current ?? 0} in range`}>Volume</PanelTitle>
            <TrendChart data={volume} bucket={bucket} />
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
              Playbooks with runs against alerts · % = success rate
            </p>
          </Panel>
        </div>
      </div>
    </div>
  );
}
