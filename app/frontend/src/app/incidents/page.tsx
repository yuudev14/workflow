"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { RowSelectionState } from "@tanstack/react-table";
import { Plus, RefreshCw, Zap } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import AdminService from "@/services/admin/admin";
import type {
  Incident,
  IncidentFilter,
  IncidentStatus,
  Severity,
} from "@/services/incidents/incidents.schema";
import type { SLAState } from "@/services/alerts/alerts.schema";
import { usePermission } from "@/hooks/usePermission";
import { useAuth } from "@/components/provider/auth-provider";
import {
  ActiveFilters,
  ColumnPicker,
  CursorPagination,
  DataTable,
  FilterField,
  FilterMenu,
  MoreFilters,
  RunPlaybookDialog,
  SearchInput,
  pillLabel,
  useColumnVisibility,
  type ActiveFilter,
} from "@/components/soar";
import { Skeleton } from "@/components/ui/skeleton";
import { useCursorPager } from "@/hooks/useCursorPager";
import { useUrlFilters, type FilterSpec } from "@/hooks/useUrlFilters";
import { cn } from "@/lib/utils";
import { INCIDENT_COLUMN_DEFAULTS, incidentColumns } from "./_components/incidentColumns";
import { NewIncidentDialog } from "./_components/NewIncidentDialog";

const TABS = ["Open", "Resolved", "All"] as const;
const SEVERITIES: Severity[] = ["critical", "high", "medium", "low"];
const STATUSES: IncidentStatus[] = ["open", "investigating", "contained", "resolved", "closed"];
const SLA_STATES: SLAState[] = ["ok", "warning", "breached", "met"];

const FILTER_SPEC: FilterSpec = {
  tab: "string",
  severity: "string[]",
  status: "string[]",
  sla_state: "string[]",
  assignee_id: "string[]",
  team_id: "string[]",
  tags: "string[]",
  unassigned: "boolean",
  created_from: "string",
  created_to: "string",
  q: "string",
};

const toLocalInput = (iso?: string): string => {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

export default function Page() {
  const queryClient = useQueryClient();
  const router = useRouter();
  const { user } = useAuth();

  const canCreate = usePermission("incidents", "create");
  const canExecute = usePermission("incidents", "execute");

  const { filters, hydrated, setValue, setFilters, clearAll } = useUrlFilters(FILTER_SPEC);
  const [search, setSearch] = React.useState("");
  const [selection, setSelection] = React.useState<RowSelectionState>({});
  const [runOpen, setRunOpen] = React.useState(false);
  const [dialogOpen, setDialogOpen] = React.useState(false);

  const str = (k: string) => filters[k] as string | undefined;
  const arr = (k: string) => (filters[k] as string[] | undefined) ?? [];
  const tab = (str("tab") as (typeof TABS)[number] | undefined) ?? "Open";

  React.useEffect(() => {
    if (hydrated) setSearch((filters.q as string) ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hydrated]);

  React.useEffect(() => {
    if (!hydrated) return;
    const t = setTimeout(() => {
      const trimmed = search.trim();
      if (trimmed !== (str("q") ?? "")) setValue("q", trimmed || undefined);
    }, 300);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, hydrated]);

  const columns = React.useMemo(() => incidentColumns(), []);
  const [visibility, setVisibility] = useColumnVisibility(
    "incidents.columns",
    columns,
    INCIDENT_COLUMN_DEFAULTS,
  );

  // An explicit status filter wins over the tab - otherwise picking "resolved"
  // while on the Open tab sends contradictory predicates and returns nothing.
  // `open: true` is the predicate incidents_open_idx is built on.
  const filter: IncidentFilter = React.useMemo(() => {
    const status = arr("status");
    const tabFilter: IncidentFilter = status.length
      ? {}
      : tab === "Open"
        ? { open: true }
        : tab === "Resolved"
          ? { status: ["resolved", "closed"] }
          : {};

    return {
      ...tabFilter,
      ...(arr("severity").length ? { severity: arr("severity") as Severity[] } : {}),
      ...(status.length ? { status: status as IncidentStatus[] } : {}),
      ...(arr("sla_state").length ? { sla_state: arr("sla_state") as SLAState[] } : {}),
      ...(arr("assignee_id").length ? { assignee_id: arr("assignee_id") } : {}),
      ...(arr("team_id").length ? { team_id: arr("team_id") } : {}),
      ...(arr("tags").length ? { tags: arr("tags") } : {}),
      ...(filters.unassigned ? { unassigned: true } : {}),
      ...(str("created_from") ? { created_from: str("created_from") } : {}),
      ...(str("created_to") ? { created_to: str("created_to") } : {}),
      ...(str("q") ? { q: str("q") } : {}),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters, tab]);

  const pager = useCursorPager(JSON.stringify(filter));

  const incidentsQuery = useQuery({
    queryKey: ["incidents", filter, pager.limit, pager.cursor],
    queryFn: () =>
      IncidentService.getIncidents({ ...filter, limit: pager.limit, cursor: pager.cursor }),
  });
  const summaryQuery = useQuery({
    queryKey: ["incidents-summary", {}],
    queryFn: () => IncidentService.getIncidentsSummary(),
  });
  const assignableQuery = useQuery({
    queryKey: ["assignable-users"],
    queryFn: () => AdminService.listAssignableUsers(),
    staleTime: 5 * 60 * 1000,
  });

  const incidents = incidentsQuery.data?.entries ?? [];
  const summary = summaryQuery.data;
  const sevCount = (s: Severity) => summary?.severity_mix.find((b) => b.severity === s)?.count ?? 0;
  const statusCount = (s: IncidentStatus) =>
    summary?.status_mix.find((b) => b.status === s)?.count ?? 0;
  const usernameOf = (id: string) =>
    assignableQuery.data?.find((u) => u.id === id)?.username ?? id.slice(0, 8);

  const selectedIds = Object.keys(selection).filter((id) => selection[id]);

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

  const resetAll = () => {
    clearAll();
    setSearch("");
  };

  const overflowKeys = [
    "sla_state",
    "team_id",
    "tags",
    "unassigned",
    "created_from",
    "created_to",
  ];
  const overflowCount = overflowKeys.filter((k) => {
    const v = filters[k];
    return Array.isArray(v) ? v.length > 0 : v !== undefined && v !== "";
  }).length;

  const removeFrom = (key: string, value: string) =>
    setValue(
      key,
      arr(key).filter((x) => x !== value),
    );

  const active: ActiveFilter[] = [
    ...arr("severity").map((v) => ({
      group: "Severity",
      value: v,
      label: pillLabel(v as Severity),
      onRemove: () => removeFrom("severity", v),
    })),
    ...arr("status").map((v) => ({
      group: "Status",
      value: v,
      label: pillLabel(v as IncidentStatus),
      onRemove: () => removeFrom("status", v),
    })),
    ...arr("sla_state").map((v) => ({
      group: "SLA",
      value: v,
      label: v,
      onRemove: () => removeFrom("sla_state", v),
    })),
    ...arr("assignee_id").map((v) => ({
      group: "Owner",
      value: v,
      label: usernameOf(v),
      onRemove: () => removeFrom("assignee_id", v),
    })),
    ...arr("tags").map((v) => ({
      group: "Tag",
      value: v,
      label: v,
      onRemove: () => removeFrom("tags", v),
    })),
    ...(filters.unassigned
      ? [
          {
            group: "Owner",
            value: "unassigned",
            label: "unassigned",
            onRemove: () => setValue("unassigned", undefined),
          },
        ]
      : []),
    ...(str("created_from")
      ? [
          {
            group: "From",
            value: str("created_from")!,
            label: new Date(str("created_from")!).toLocaleString(),
            onRemove: () => setValue("created_from", undefined),
          },
        ]
      : []),
    ...(str("created_to")
      ? [
          {
            group: "To",
            value: str("created_to")!,
            label: new Date(str("created_to")!).toLocaleString(),
            onRemove: () => setValue("created_to", undefined),
          },
        ]
      : []),
    ...(str("q")
      ? [
          {
            group: "Search",
            value: str("q")!,
            label: str("q")!,
            onRemove: () => {
              setSearch("");
              setValue("q", undefined);
            },
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
                onClick={() => setValue("tab", t === "Open" ? undefined : t)}
                disabled={arr("status").length > 0}
                title={arr("status").length > 0 ? "Clear the Status filter to use these tabs" : undefined}
                className={cn(
                  "rounded-[6px] px-3 py-1.5 text-[13px] font-semibold transition-colors disabled:opacity-40",
                  tab === t && arr("status").length === 0
                    ? "bg-card text-foreground shadow-sm"
                    : "text-ink-soft hover:text-foreground",
                )}
              >
                {t}
              </button>
            ))}
          </div>
          <FilterMenu
            label="Severity"
            selected={arr("severity")}
            onChange={(v) => setValue("severity", v)}
            options={SEVERITIES.map((s) => ({
              value: s,
              label: pillLabel(s),
              count: sevCount(s),
            }))}
          />
          <FilterMenu
            label="Status"
            selected={arr("status")}
            onChange={(v) => setValue("status", v)}
            options={STATUSES.map((s) => ({
              value: s,
              label: pillLabel(s),
              count: statusCount(s),
            }))}
          />
          <FilterMenu
            label="Owner"
            selected={arr("assignee_id")}
            onChange={(v) => setValue("assignee_id", v)}
            options={(assignableQuery.data ?? []).map((u) => ({ value: u.id, label: u.username }))}
          />
          <button
            onClick={() =>
              setValue("assignee_id", user && !arr("assignee_id").includes(user.id) ? [user.id] : [])
            }
            className={cn(
              "rounded-sm border px-3 py-1.5 text-[13px] font-semibold transition-colors",
              user && arr("assignee_id").includes(user.id)
                ? "border-signal-dot bg-signal-soft text-signal-text"
                : "border-line-strong text-ink-soft hover:bg-paper-sunken",
            )}
          >
            Assigned to me
          </button>

          <MoreFilters
            activeCount={overflowCount}
            onClear={() =>
              setFilters((prev) => {
                const next = { ...prev };
                for (const k of overflowKeys) delete next[k];
                return next;
              })
            }
          >
            <FilterField label="SLA state">
              <div className="flex flex-wrap gap-1.5">
                {SLA_STATES.map((s) => {
                  const on = arr("sla_state").includes(s);
                  return (
                    <button
                      key={s}
                      onClick={() =>
                        setValue(
                          "sla_state",
                          on ? arr("sla_state").filter((x) => x !== s) : [...arr("sla_state"), s],
                        )
                      }
                      className={cn(
                        "rounded-sm border px-2 py-1 text-[12px] font-semibold",
                        on
                          ? "border-signal-dot bg-signal-soft text-signal-text"
                          : "border-line-strong text-ink-soft",
                      )}
                    >
                      {s}
                    </button>
                  );
                })}
              </div>
            </FilterField>

            <FilterField label="Assignment">
              <button
                onClick={() => setValue("unassigned", filters.unassigned ? undefined : true)}
                className={cn(
                  "w-fit rounded-sm border px-2 py-1 text-[12px] font-semibold",
                  filters.unassigned
                    ? "border-signal-dot bg-signal-soft text-signal-text"
                    : "border-line-strong text-ink-soft",
                )}
              >
                Unassigned only
              </button>
            </FilterField>

            <FilterField label="Tags (comma separated)">
              <input
                defaultValue={arr("tags").join(", ")}
                onBlur={(e) =>
                  setValue(
                    "tags",
                    e.target.value
                      .split(",")
                      .map((t) => t.trim())
                      .filter(Boolean),
                  )
                }
                placeholder="ransomware, exfil"
                className="rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12.5px] outline-none focus:border-signal-dot"
              />
            </FilterField>

            <FilterField label="Team ids (comma separated)">
              <input
                defaultValue={arr("team_id").join(", ")}
                onBlur={(e) =>
                  setValue(
                    "team_id",
                    e.target.value
                      .split(",")
                      .map((t) => t.trim())
                      .filter(Boolean),
                  )
                }
                className="rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12.5px] outline-none focus:border-signal-dot"
              />
            </FilterField>

            <FilterField label="Created between">
              <div className="flex items-center gap-1.5">
                <input
                  type="datetime-local"
                  value={toLocalInput(str("created_from"))}
                  onChange={(e) =>
                    setValue(
                      "created_from",
                      e.target.value ? new Date(e.target.value).toISOString() : undefined,
                    )
                  }
                  className="flex-1 rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12px] outline-none focus:border-signal-dot"
                />
                <input
                  type="datetime-local"
                  value={toLocalInput(str("created_to"))}
                  onChange={(e) =>
                    setValue(
                      "created_to",
                      e.target.value ? new Date(e.target.value).toISOString() : undefined,
                    )
                  }
                  className="flex-1 rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12px] outline-none focus:border-signal-dot"
                />
              </div>
            </FilterField>
          </MoreFilters>

          <ColumnPicker columns={columns} visibility={visibility} onChange={setVisibility} />

          <SearchInput
            className="ml-auto"
            placeholder="Search incidents…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <ActiveFilters filters={active} onClearAll={resetAll} />

        {selectedIds.length > 0 && (
          <div className="flex items-center gap-3 rounded-md border border-signal-dot bg-signal-soft px-3.5 py-2.5 text-[13px]">
            <span className="font-semibold text-signal-text">{selectedIds.length} selected</span>
            <button
              onClick={() => setRunOpen(true)}
              disabled={!canExecute}
              title={canExecute ? undefined : "Needs incidents:execute"}
              className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              <Zap className="size-3.5" /> Run playbook
            </button>
            <button
              onClick={() => setSelection({})}
              className="ml-auto text-[12.5px] font-semibold text-ink-soft hover:text-foreground"
            >
              Clear selection
            </button>
          </div>
        )}

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
            <DataTable<Incident>
              columns={columns}
              data={incidents}
              getRowId={(i) => i.id}
              columnVisibility={visibility}
              onColumnVisibilityChange={setVisibility}
              rowSelection={selection}
              onRowSelectionChange={setSelection}
              onRowClick={(i) => router.push(`/incidents/${i.id}`)}
            />
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

      <RunPlaybookDialog
        open={runOpen}
        onOpenChange={setRunOpen}
        moduleType="incident"
        recordIds={selectedIds}
        onLaunched={() => {
          setSelection({});
          refresh();
        }}
      />
    </div>
  );
}
