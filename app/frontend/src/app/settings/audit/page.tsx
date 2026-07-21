"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { createColumnHelper } from "@tanstack/react-table";
import { ScrollText } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { AuditLog } from "@/services/admin/admin.schema";
import {
  DataTable,
  EmptyState,
  FilterChips,
  InitialsAvatar,
  JsonTree,
  PageShell,
  PaginationBar,
  StatusPill,
} from "@/components/soar";
import type { PillVariant } from "@/components/soar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const PAGE_SIZE = 50;

const MODULE_CHIPS = [
  { value: "", label: "All modules" },
  { value: "settings", label: "Settings" },
  { value: "auth", label: "Auth" },
];

// Mirrors the audit actions written in app/ytsoar/internal/application/auth/*.go.
const AUDIT_ACTIONS = [
  "admin_seeded",
  "login",
  "login_failed",
  "logout",
  "refresh_reuse",
  "role_created",
  "role_updated",
  "role_deleted",
  "role_permissions_changed",
  "team_created",
  "team_updated",
  "team_deleted",
  "team_members_changed",
  "user_created",
  "user_updated",
  "user_deactivated",
  "user_roles_changed",
  "user_password_reset",
];

function humanizeAction(action: string) {
  const label = action.replace(/_/g, " ");
  return label.charAt(0).toUpperCase() + label.slice(1);
}

function actionVariant(action: string): PillVariant {
  if (action.endsWith("_deleted") || action === "login_failed" || action === "refresh_reuse") {
    return "failed";
  }
  if (action === "logout" || action.endsWith("_deactivated")) {
    return "neutral";
  }
  if (action.endsWith("_created") || action === "login" || action === "admin_seeded") {
    return "success";
  }
  return "medium";
}

const columnHelper = createColumnHelper<AuditLog>();

// The api parses from/to as strict RFC3339 with no fractional seconds
// (2006-01-02T15:04:05Z07:00), so a plain toISOString() would be rejected.
// A date input gives us YYYY-MM-DD; pin it to the day's bounds in UTC.
const dayStart = (date: string) => (date ? `${date}T00:00:00Z` : undefined);
const dayEnd = (date: string) => (date ? `${date}T23:59:59Z` : undefined);

/** Read-only by design: an editable audit trail is not an audit trail. */
export default function AuditPage() {
  const [module, setModule] = React.useState("");
  const [action, setAction] = React.useState("all");
  const [actor, setActor] = React.useState("all");
  const [from, setFrom] = React.useState("");
  const [to, setTo] = React.useState("");
  const [page, setPage] = React.useState(0);
  const [detail, setDetail] = React.useState<AuditLog | null>(null);

  const usersQuery = useQuery({
    queryKey: ["users", "audit-actors"],
    queryFn: () => AdminService.listUsers({ limit: 100 }),
  });

  const filter = {
    module: module || undefined,
    action: action === "all" ? undefined : action,
    actor_id: actor === "all" ? undefined : actor,
    from: dayStart(from),
    to: dayEnd(to),
    offset: page * PAGE_SIZE,
    limit: PAGE_SIZE,
  };

  const logsQuery = useQuery({
    queryKey: ["audit", filter],
    queryFn: () => AdminService.listAuditLogs(filter),
  });

  const logs = logsQuery.data?.entries ?? [];
  const total = logsQuery.data?.total ?? 0;
  const lastPage = Math.max(0, Math.ceil(total / PAGE_SIZE) - 1);

  const columns = React.useMemo(
    () => [
      columnHelper.accessor("created_at", {
        header: "When",
        cell: ({ getValue }) => (
          <span className="whitespace-nowrap text-ink-soft tnum">
            {new Date(getValue()).toLocaleString()}
          </span>
        ),
      }),
      columnHelper.accessor("actor_username", {
        header: "Actor",
        cell: ({ getValue }) => (
          <div className="flex items-center gap-2">
            <InitialsAvatar name={getValue()} size={22} />
            {/* Null actor: a system-generated row, or a user removed since. */}
            <span>{getValue() ?? "system"}</span>
          </div>
        ),
      }),
      columnHelper.accessor("module", {
        header: "Module",
        cell: ({ getValue }) => <span className="text-ink-soft">{getValue()}</span>,
      }),
      columnHelper.accessor("action", {
        header: "Action",
        cell: ({ getValue }) => (
          <StatusPill variant={actionVariant(getValue())}>{humanizeAction(getValue())}</StatusPill>
        ),
      }),
      columnHelper.accessor("entity_id", {
        header: "Entity",
        cell: ({ getValue }) => <span className="text-ink-faint">{getValue() ?? "—"}</span>,
      }),
      columnHelper.display({
        id: "detail",
        header: "",
        meta: { align: "right" },
        cell: ({ row: { original: log } }) =>
          log.detail != null && (
            <Button variant="ghost" size="sm" onClick={() => setDetail(log)}>
              Detail
            </Button>
          ),
      }),
    ],
    []
  );

  return (
    <PageShell title="Audit" subtitle={`${total} recorded ${total === 1 ? "event" : "events"}`}>
      <div className="flex flex-wrap items-end justify-between gap-3">
        <FilterChips
          value={module}
          onChange={(v) => {
            setModule(v);
            setPage(0);
          }}
          chips={MODULE_CHIPS}
        />
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="audit_from">From</Label>
            <Input
              id="audit_from"
              type="date"
              value={from}
              max={to || undefined}
              className="w-40"
              onChange={(e) => {
                setFrom(e.target.value);
                setPage(0);
              }}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="audit_to">To</Label>
            <Input
              id="audit_to"
              type="date"
              value={to}
              min={from || undefined}
              className="w-40"
              onChange={(e) => {
                setTo(e.target.value);
                setPage(0);
              }}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="audit_actor">Actor</Label>
            <Select
              value={actor}
              onValueChange={(v) => {
                setActor(v);
                setPage(0);
              }}
            >
              <SelectTrigger id="audit_actor" className="w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All actors</SelectItem>
                {usersQuery.data?.entries.map((u) => (
                  <SelectItem key={u.id} value={u.id}>
                    {u.username}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="audit_action">Action</Label>
            <Select
              value={action}
              onValueChange={(v) => {
                setAction(v);
                setPage(0);
              }}
            >
              <SelectTrigger id="audit_action" className="w-56">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All actions</SelectItem>
                {AUDIT_ACTIONS.map((a) => (
                  <SelectItem key={a} value={a}>
                    {humanizeAction(a)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      {logsQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-11 rounded-md" />
          ))}
        </div>
      ) : logs.length === 0 ? (
        <EmptyState icon={ScrollText} title="No audit entries" description="Nothing matches this filter." />
      ) : (
        <DataTable columns={columns} data={logs} getRowId={(log) => log.id} />
      )}

      <PaginationBar
        page={page}
        pageCount={lastPage + 1}
        total={total}
        pageSize={PAGE_SIZE}
        onPrev={() => setPage(page - 1)}
        onNext={() => setPage(page + 1)}
      />

      <Dialog open={detail !== null} onOpenChange={(next) => !next && setDetail(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {detail?.module} · {detail && humanizeAction(detail.action)}
            </DialogTitle>
          </DialogHeader>
          <JsonTree data={detail?.detail} />
        </DialogContent>
      </Dialog>
    </PageShell>
  );
}
