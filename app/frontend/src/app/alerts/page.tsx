"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { RowSelectionState } from "@tanstack/react-table";
import { RefreshCw, Zap } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import AdminService from "@/services/admin/admin";
import type {
  Alert,
  AlertFilter,
  AlertStatus,
  Severity,
  SLAState,
  SourceKind,
} from "@/services/alerts/alerts.schema";
import { useAuth } from "@/components/provider/auth-provider";
import { usePermission } from "@/hooks/usePermission";
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
import { ALERT_COLUMN_DEFAULTS, alertColumns } from "./_components/alertColumns";
import { SOURCE_LABEL } from "./_components/alertPresentation";

const SEVERITIES: Severity[] = ["critical", "high", "medium", "low"];
const STATUSES: AlertStatus[] = ["new", "investigating", "resolved", "falsepos", "closed"];
const SOURCE_KINDS: SourceKind[] = ["edr", "identity", "email", "firewall", "dlp"];
const SLA_STATES: SLAState[] = ["ok", "warning", "breached", "met"];

const FILTER_SPEC: FilterSpec = {
  severity: "string[]",
  status: "string[]",
  source_kind: "string[]",
  sla_state: "string[]",
  assignee_id: "string[]",
  team_id: "string[]",
  tags: "string[]",
  unassigned: "boolean",
  created_from: "string",
  created_to: "string",
  dedup_min: "number",
  triaged: "string",
  q: "string",
};

/** `datetime-local` has no offset; Date supplies the browser's own. */
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
  const canExecute = usePermission("alerts", "execute");

  const { filters, hydrated, setValue, setFilters, clearAll } = useUrlFilters(FILTER_SPEC);
  const [search, setSearch] = React.useState("");
  const [selection, setSelection] = React.useState<RowSelectionState>({});
  const [runOpen, setRunOpen] = React.useState(false);

  const str = (k: string) => filters[k] as string | undefined;
  const arr = (k: string) => (filters[k] as string[] | undefined) ?? [];

  // Seed the box from a pasted ?q=, once the URL has actually been read.
  React.useEffect(() => {
    if (hydrated) setSearch((filters.q as string) ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hydrated]);

  // Debounced so typing does not fire a request per keystroke, each of which
  // would also reset the cursor.
  React.useEffect(() => {
    if (!hydrated) return;
    const t = setTimeout(() => {
      const trimmed = search.trim();
      if (trimmed !== (str("q") ?? "")) setValue("q", trimmed || undefined);
    }, 300);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, hydrated]);

  const columns = React.useMemo(() => alertColumns(), []);
  const [visibility, setVisibility] = useColumnVisibility(
    "alerts.columns",
    columns,
    ALERT_COLUMN_DEFAULTS,
  );

  const filter: AlertFilter = React.useMemo(() => {
    const triaged = str("triaged");
    return {
      ...(arr("severity").length ? { severity: arr("severity") as Severity[] } : {}),
      ...(arr("status").length ? { status: arr("status") as AlertStatus[] } : {}),
      ...(arr("source_kind").length ? { source_kind: arr("source_kind") as SourceKind[] } : {}),
      ...(arr("sla_state").length ? { sla_state: arr("sla_state") as SLAState[] } : {}),
      ...(arr("assignee_id").length ? { assignee_id: arr("assignee_id") } : {}),
      ...(arr("team_id").length ? { team_id: arr("team_id") } : {}),
      ...(arr("tags").length ? { tags: arr("tags") } : {}),
      ...(filters.unassigned ? { unassigned: true } : {}),
      ...(str("created_from") ? { created_from: str("created_from") } : {}),
      ...(str("created_to") ? { created_to: str("created_to") } : {}),
      ...(filters.dedup_min ? { dedup_min: filters.dedup_min as number } : {}),
      ...(triaged ? { triaged: triaged === "true" } : {}),
      ...(str("q") ? { q: str("q") } : {}),
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const pager = useCursorPager(JSON.stringify(filter));

  const alertsQuery = useQuery({
    queryKey: ["alerts", filter, pager.limit, pager.cursor],
    queryFn: () =>
      AlertService.getAlerts({ ...filter, limit: pager.limit, cursor: pager.cursor }),
  });
  const summaryQuery = useQuery({
    queryKey: ["alerts-summary", {}],
    queryFn: () => AlertService.getAlertsSummary(),
  });
  const assignableQuery = useQuery({
    queryKey: ["assignable-users"],
    queryFn: () => AdminService.listAssignableUsers(),
    staleTime: 5 * 60 * 1000,
  });

  const alerts = alertsQuery.data?.entries ?? [];
  const summary = summaryQuery.data;
  const sevCount = (s: Severity) => summary?.by_severity.find((b) => b.severity === s)?.count ?? 0;
  const srcCount = (s: SourceKind) => summary?.by_source.find((b) => b.source_kind === s)?.count ?? 0;
  const usernameOf = (id: string) =>
    assignableQuery.data?.find((u) => u.id === id)?.username ?? id.slice(0, 8);

  const selectedIds = Object.keys(selection).filter((id) => selection[id]);

  const refresh = () => {
    queryClient.invalidateQueries({ queryKey: ["alerts"] });
    queryClient.invalidateQueries({ queryKey: ["alerts-summary"] });
  };

  const resetAll = () => {
    clearAll();
    setSearch("");
  };

  // Filters that live behind "More filters", counted for its badge.
  const overflowKeys = [
    "sla_state",
    "team_id",
    "tags",
    "unassigned",
    "created_from",
    "created_to",
    "dedup_min",
    "triaged",
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
      label: pillLabel(v as AlertStatus),
      onRemove: () => removeFrom("status", v),
    })),
    ...arr("source_kind").map((v) => ({
      group: "Source",
      value: v,
      label: SOURCE_LABEL[v as SourceKind],
      onRemove: () => removeFrom("source_kind", v),
    })),
    ...arr("sla_state").map((v) => ({
      group: "SLA",
      value: v,
      label: v,
      onRemove: () => removeFrom("sla_state", v),
    })),
    ...arr("assignee_id").map((v) => ({
      group: "Assignee",
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
            group: "Assignee",
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
    ...(filters.dedup_min
      ? [
          {
            group: "Recurring",
            value: String(filters.dedup_min),
            label: `seen ≥ ${filters.dedup_min}×`,
            onRemove: () => setValue("dedup_min", undefined),
          },
        ]
      : []),
    ...(str("triaged")
      ? [
          {
            group: "Triage",
            value: str("triaged")!,
            label: str("triaged") === "true" ? "triaged" : "untriaged",
            onRemove: () => setValue("triaged", undefined),
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
            options={STATUSES.map((s) => ({ value: s, label: pillLabel(s) }))}
          />
          <FilterMenu
            label="Source"
            selected={arr("source_kind")}
            onChange={(v) => setValue("source_kind", v)}
            options={SOURCE_KINDS.map((s) => ({
              value: s,
              label: SOURCE_LABEL[s],
              count: srcCount(s),
            }))}
          />
          <FilterMenu
            label="Assignee"
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
                placeholder="phishing, ransomware"
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

            <FilterField label="Seen at least">
              <input
                type="number"
                min={1}
                value={(filters.dedup_min as number | undefined) ?? ""}
                onChange={(e) =>
                  setValue("dedup_min", e.target.value ? Number(e.target.value) : undefined)
                }
                placeholder="e.g. 2 - recurring only"
                className="rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12.5px] outline-none focus:border-signal-dot"
              />
            </FilterField>

            <FilterField label="Triage">
              <div className="flex gap-1.5">
                {[
                  { v: "true", label: "Triaged" },
                  { v: "false", label: "Untriaged" },
                ].map((o) => (
                  <button
                    key={o.v}
                    onClick={() => setValue("triaged", str("triaged") === o.v ? undefined : o.v)}
                    className={cn(
                      "rounded-sm border px-2 py-1 text-[12px] font-semibold",
                      str("triaged") === o.v
                        ? "border-signal-dot bg-signal-soft text-signal-text"
                        : "border-line-strong text-ink-soft",
                    )}
                  >
                    {o.label}
                  </button>
                ))}
              </div>
            </FilterField>
          </MoreFilters>

          <ColumnPicker columns={columns} visibility={visibility} onChange={setVisibility} />

          <SearchInput
            className="ml-auto"
            placeholder="Search alerts…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        <ActiveFilters filters={active} onClearAll={resetAll} />

        {selectedIds.length > 0 && (
          <div className="flex items-center gap-3 rounded-md border border-signal-dot bg-signal-soft px-3.5 py-2.5 text-[13px]">
            <span className="font-semibold text-signal-text">
              {selectedIds.length} selected
            </span>
            <button
              onClick={() => setRunOpen(true)}
              disabled={!canExecute}
              title={canExecute ? undefined : "Needs alerts:execute"}
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
            <DataTable<Alert>
              columns={columns}
              data={alerts}
              getRowId={(a) => a.id}
              columnVisibility={visibility}
              onColumnVisibilityChange={setVisibility}
              rowSelection={selection}
              onRowSelectionChange={setSelection}
              onRowClick={(a) => router.push(`/alerts/${a.id}`)}
            />
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

      <RunPlaybookDialog
        open={runOpen}
        onOpenChange={setRunOpen}
        moduleType="alert"
        recordIds={selectedIds}
        onLaunched={() => {
          setSelection({});
          refresh();
        }}
      />
    </div>
  );
}
