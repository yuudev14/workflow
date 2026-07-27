"use client";

import React from "react";
import Link from "next/link";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import type {
  AlertFilter,
  AlertStatus,
  Severity,
  SourceKind,
} from "@/services/alerts/alerts.schema";
import { useAuth } from "@/components/provider/auth-provider";
import {
  ActiveFilters,
  CursorPagination,
  FilterMenu,
  SearchInput,
  type ActiveFilter,
} from "@/components/soar";
import { pillLabel } from "@/components/soar";
import { Skeleton } from "@/components/ui/skeleton";
import { useCursorPager } from "@/hooks/useCursorPager";
import { cn } from "@/lib/utils";
import { AlertRow } from "./_components/AlertRow";
import { SOURCE_LABEL } from "./_components/alertPresentation";

const SEVERITIES: Severity[] = ["critical", "high", "medium", "low"];
const STATUSES: AlertStatus[] = ["new", "investigating", "resolved", "falsepos", "closed"];
const SOURCE_KINDS: SourceKind[] = ["edr", "identity", "email", "firewall", "dlp"];

export default function Page() {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  const [severity, setSeverity] = React.useState<string[]>([]);
  const [status, setStatus] = React.useState<string[]>([]);
  const [sourceKind, setSourceKind] = React.useState<string[]>([]);
  const [mineOnly, setMineOnly] = React.useState(false);
  const [search, setSearch] = React.useState("");
  const [debounced, setDebounced] = React.useState("");

  // Debounced so typing does not fire a request per keystroke, each of which
  // would also reset the cursor.
  React.useEffect(() => {
    const t = setTimeout(() => setDebounced(search.trim()), 300);
    return () => clearTimeout(t);
  }, [search]);

  const filter: AlertFilter = React.useMemo(
    () => ({
      ...(severity.length ? { severity: severity as Severity[] } : {}),
      ...(status.length ? { status: status as AlertStatus[] } : {}),
      ...(sourceKind.length ? { source_kind: sourceKind as SourceKind[] } : {}),
      ...(debounced ? { q: debounced } : {}),
      ...(mineOnly && user ? { assignee_id: user.id } : {}),
    }),
    [severity, status, sourceKind, debounced, mineOnly, user]
  );

  const pager = useCursorPager(JSON.stringify(filter));

  const alertsQuery = useQuery({
    queryKey: ["alerts", filter, pager.limit, pager.cursor],
    queryFn: () =>
      AlertService.getAlerts({ ...filter, limit: pager.limit, cursor: pager.cursor }),
  });
  const summaryQuery = useQuery({
    queryKey: ["alerts-summary"],
    queryFn: () => AlertService.getAlertsSummary(),
  });

  const alerts = alertsQuery.data?.entries ?? [];
  const summary = summaryQuery.data;
  const sevCount = (s: Severity) =>
    summary?.by_severity.find((b) => b.severity === s)?.count ?? 0;
  const srcCount = (s: SourceKind) =>
    summary?.by_source.find((b) => b.source_kind === s)?.count ?? 0;

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ["alerts"] });
    queryClient.invalidateQueries({ queryKey: ["alerts-summary"] });
  };

  const clearAll = () => {
    setSeverity([]);
    setStatus([]);
    setSourceKind([]);
    setMineOnly(false);
    setSearch("");
  };

  const active: ActiveFilter[] = [
    ...severity.map((v) => ({
      group: "Severity",
      value: v,
      label: pillLabel(v as Severity),
      onRemove: () => setSeverity((s) => s.filter((x) => x !== v)),
    })),
    ...status.map((v) => ({
      group: "Status",
      value: v,
      label: pillLabel(v as AlertStatus),
      onRemove: () => setStatus((s) => s.filter((x) => x !== v)),
    })),
    ...sourceKind.map((v) => ({
      group: "Source",
      value: v,
      label: SOURCE_LABEL[v as SourceKind],
      onRemove: () => setSourceKind((s) => s.filter((x) => x !== v)),
    })),
    ...(mineOnly
      ? [
          {
            group: "Assignee",
            value: "me",
            label: "me",
            onRemove: () => setMineOnly(false),
          },
        ]
      : []),
    ...(debounced
      ? [
          {
            group: "Search",
            value: debounced,
            label: debounced,
            onRemove: () => setSearch(""),
          },
        ]
      : []),
  ];

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-4 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Alerts</h1>
            <p className="mt-1 text-[15px] text-ink-soft">
              {summary ? `${summary.total} open · ${sevCount("critical")} critical` : " "}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/alerts/dashboard"
              className="inline-flex items-center gap-2 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              Dashboard
            </Link>
            <button
              onClick={refresh}
              className="flex size-9 items-center justify-center rounded-sm border border-line-strong text-ink-soft hover:bg-paper-sunken"
            >
              <RefreshCw className={cn("size-4", alertsQuery.isFetching && "animate-spin")} />
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <FilterMenu
            label="Severity"
            selected={severity}
            onChange={setSeverity}
            options={SEVERITIES.map((s) => ({
              value: s,
              label: pillLabel(s),
              count: sevCount(s),
            }))}
          />
          <FilterMenu
            label="Status"
            selected={status}
            onChange={setStatus}
            options={STATUSES.map((s) => ({ value: s, label: pillLabel(s) }))}
          />
          <FilterMenu
            label="Source"
            selected={sourceKind}
            onChange={setSourceKind}
            options={SOURCE_KINDS.map((s) => ({
              value: s,
              label: SOURCE_LABEL[s],
              count: srcCount(s),
            }))}
          />
          <button
            onClick={() => setMineOnly((v) => !v)}
            className={cn(
              "rounded-sm border px-3 py-1.5 text-[13px] font-semibold transition-colors",
              mineOnly
                ? "border-signal-dot bg-signal-soft text-signal-text"
                : "border-line-strong text-ink-soft hover:bg-paper-sunken"
            )}
          >
            Assigned to me
          </button>
          <SearchInput
            className="ml-auto"
            placeholder="Search alerts…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <ActiveFilters filters={active} onClearAll={clearAll} />

        {alertsQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-[62px] rounded-md" />
            ))}
          </div>
        ) : alerts.length === 0 ? (
          <div className="rounded-md border border-line bg-card px-4 py-10 text-center text-[13.5px] text-ink-faint">
            No alerts match these filters.
          </div>
        ) : (
          <>
            <div className="overflow-hidden rounded-md border border-line">
              {alerts.map((a) => (
                <AlertRow key={a.id} alert={a} href={`/alerts/${a.id}`} />
              ))}
            </div>
            <CursorPagination
              page={pager.page}
              shown={alerts.length}
              total={alertsQuery.data?.total ?? alerts.length}
              limit={pager.limit}
              onLimitChange={pager.setLimit}
              canPrev={pager.canPrev}
              canNext={!!alertsQuery.data?.next_cursor}
              onPrev={pager.goPrev}
              onNext={() => pager.goNext(alertsQuery.data?.next_cursor)}
              busy={alertsQuery.isFetching}
            />
          </>
        )}
      </div>
    </div>
  );
}
