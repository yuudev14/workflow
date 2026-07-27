"use client";

import React from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import type {
  IncidentFilter,
  IncidentStatus,
  Severity,
} from "@/services/incidents/incidents.schema";
import { usePermission } from "@/hooks/usePermission";
import { useAuth } from "@/components/provider/auth-provider";
import {
  ActiveFilters,
  CursorPagination,
  FilterMenu,
  SearchInput,
  pillLabel,
  type ActiveFilter,
} from "@/components/soar";
import { Skeleton } from "@/components/ui/skeleton";
import { useCursorPager } from "@/hooks/useCursorPager";
import { cn } from "@/lib/utils";
import { IncidentRow } from "./_components/IncidentRow";
import { NewIncidentDialog } from "./_components/NewIncidentDialog";

const TABS = ["Open", "Resolved", "All"] as const;
const SEVERITIES: Severity[] = ["critical", "high", "medium", "low"];
const STATUSES: IncidentStatus[] = [
  "open",
  "investigating",
  "contained",
  "resolved",
  "closed",
];

export default function Page() {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  const [tab, setTab] = React.useState<(typeof TABS)[number]>("Open");
  const [severity, setSeverity] = React.useState<string[]>([]);
  const [status, setStatus] = React.useState<string[]>([]);
  const [mineOnly, setMineOnly] = React.useState(false);
  const [search, setSearch] = React.useState("");
  const [debounced, setDebounced] = React.useState("");
  const [dialogOpen, setDialogOpen] = React.useState(false);

  const canCreate = usePermission("incidents", "create");

  React.useEffect(() => {
    const t = setTimeout(() => setDebounced(search.trim()), 300);
    return () => clearTimeout(t);
  }, [search]);

  // An explicit status filter wins over the tab - otherwise picking "resolved"
  // while on the Open tab sends contradictory predicates and returns nothing.
  // `open: true` is the predicate incidents_open_idx is built on.
  const tabFilter: IncidentFilter = React.useMemo(() => {
    if (status.length) return {};
    if (tab === "Open") return { open: true };
    if (tab === "Resolved") return { status: ["resolved", "closed"] };
    return {};
  }, [tab, status.length]);

  const filter: IncidentFilter = React.useMemo(
    () => ({
      ...tabFilter,
      ...(severity.length ? { severity: severity as Severity[] } : {}),
      ...(status.length ? { status: status as IncidentStatus[] } : {}),
      ...(debounced ? { q: debounced } : {}),
      ...(mineOnly && user ? { assignee_id: user.id } : {}),
    }),
    [tabFilter, severity, status, debounced, mineOnly, user]
  );

  const pager = useCursorPager(JSON.stringify(filter));

  const incidentsQuery = useQuery({
    queryKey: ["incidents", filter, pager.limit, pager.cursor],
    queryFn: () =>
      IncidentService.getIncidents({ ...filter, limit: pager.limit, cursor: pager.cursor }),
  });
  const summaryQuery = useQuery({
    queryKey: ["incidents-summary"],
    queryFn: () => IncidentService.getIncidentsSummary(),
  });

  const incidents = incidentsQuery.data?.entries ?? [];
  const summary = summaryQuery.data;
  const sevCount = (s: Severity) =>
    summary?.severity_mix.find((b) => b.severity === s)?.count ?? 0;
  const statusCount = (s: IncidentStatus) =>
    summary?.status_mix.find((b) => b.status === s)?.count ?? 0;

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ["incidents"] });
    queryClient.invalidateQueries({ queryKey: ["incidents-summary"] });
  };

  const createMutation = useMutation({
    mutationFn: (payload: { title: string; severity: Severity }) =>
      IncidentService.createIncident(payload),
    onSuccess: () => {
      refresh();
      setDialogOpen(false);
    },
  });

  const clearAll = () => {
    setSeverity([]);
    setStatus([]);
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
      label: pillLabel(v as IncidentStatus),
      onRemove: () => setStatus((s) => s.filter((x) => x !== v)),
    })),
    ...(mineOnly
      ? [{ group: "Owner", value: "me", label: "me", onRemove: () => setMineOnly(false) }]
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

  const breached = summary?.sla_at_risk.filter((s) => s.breached).length ?? 0;

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-4 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Incidents</h1>
            <p className="mt-1 text-[15px] text-ink-soft">
              {summary ? `${summary.open_total} open · ${breached} SLA breach` : " "}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/incidents/dashboard"
              className="inline-flex items-center gap-2 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              Dashboard
            </Link>
            <button
              onClick={refresh}
              className="flex size-9 items-center justify-center rounded-sm border border-line-strong text-ink-soft hover:bg-paper-sunken"
            >
              <RefreshCw className={cn("size-4", incidentsQuery.isFetching && "animate-spin")} />
            </button>
            <button
              onClick={() => setDialogOpen(true)}
              disabled={!canCreate}
              className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              <Plus className="size-4" /> New incident
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="flex w-fit gap-1 rounded-sm border border-line bg-paper-sunken p-[3px]">
            {TABS.map((t) => (
              <button
                key={t}
                onClick={() => setTab(t)}
                disabled={status.length > 0}
                title={status.length > 0 ? "Clear the Status filter to use these tabs" : undefined}
                className={cn(
                  "rounded-[6px] px-3 py-1.5 text-[13px] font-semibold transition-colors disabled:opacity-40",
                  tab === t && status.length === 0
                    ? "bg-card text-foreground shadow-sm"
                    : "text-ink-soft hover:text-foreground"
                )}
              >
                {t}
              </button>
            ))}
          </div>
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
            options={STATUSES.map((s) => ({
              value: s,
              label: pillLabel(s),
              count: statusCount(s),
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
            placeholder="Search incidents…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <ActiveFilters filters={active} onClearAll={clearAll} />

        {incidentsQuery.isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-[62px] rounded-md" />
            ))}
          </div>
        ) : incidents.length === 0 ? (
          <div className="rounded-md border border-line bg-card px-4 py-10 text-center text-[13.5px] text-ink-faint">
            No incidents match these filters.
          </div>
        ) : (
          <>
            <div className="overflow-hidden rounded-md border border-line">
              {incidents.map((i) => (
                <IncidentRow key={i.id} incident={i} href={`/incidents/${i.id}`} />
              ))}
            </div>
            <CursorPagination
              page={pager.page}
              shown={incidents.length}
              total={incidentsQuery.data?.total ?? incidents.length}
              limit={pager.limit}
              onLimitChange={pager.setLimit}
              canPrev={pager.canPrev}
              canNext={!!incidentsQuery.data?.next_cursor}
              onPrev={pager.goPrev}
              onNext={() => pager.goNext(incidentsQuery.data?.next_cursor)}
              busy={incidentsQuery.isFetching}
            />
          </>
        )}
      </div>

      <NewIncidentDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        pending={createMutation.isPending}
        onSubmit={(payload) => createMutation.mutate(payload)}
      />
    </div>
  );
}
