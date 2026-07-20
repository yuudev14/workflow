"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { ScrollText } from "lucide-react";

import AdminService from "@/services/admin/admin";
import { AuditLog } from "@/services/admin/admin.schema";
import { EmptyState, JsonTree, PageShell } from "@/components/soar";
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

const PAGE_SIZE = 50;

/** Read-only by design: an editable audit trail is not an audit trail. */
export default function AuditPage() {
  const [module, setModule] = React.useState("");
  const [action, setAction] = React.useState("");
  const [page, setPage] = React.useState(0);
  const [detail, setDetail] = React.useState<AuditLog | null>(null);

  const filter = {
    module: module || undefined,
    action: action || undefined,
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

  const applyFilter = (setter: (value: string) => void) => (value: string) => {
    setter(value);
    setPage(0);
  };

  return (
    <PageShell title="Audit" subtitle={`${total} recorded ${total === 1 ? "event" : "events"}`}>
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="audit_module">Module</Label>
          <Input
            id="audit_module"
            className="w-48"
            placeholder="settings"
            value={module}
            onChange={(e) => applyFilter(setModule)(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="audit_action">Action</Label>
          <Input
            id="audit_action"
            className="w-48"
            placeholder="user.created"
            value={action}
            onChange={(e) => applyFilter(setAction)(e.target.value)}
          />
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
        <div className="overflow-x-auto rounded-md border border-line">
          <table className="w-full text-[13.5px]">
            <thead className="bg-paper-sunken text-[12px] uppercase tracking-wide text-ink-soft">
              <tr>
                <th className="px-3 py-2 text-left font-semibold">When</th>
                <th className="px-3 py-2 text-left font-semibold">Actor</th>
                <th className="px-3 py-2 text-left font-semibold">Module</th>
                <th className="px-3 py-2 text-left font-semibold">Action</th>
                <th className="px-3 py-2 text-left font-semibold">Entity</th>
                <th className="px-3 py-2 text-right font-semibold" />
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id} className="border-t border-line">
                  <td className="whitespace-nowrap px-3 py-2 text-ink-soft tnum">
                    {new Date(log.created_at).toLocaleString()}
                  </td>
                  {/* Null actor: a system-generated row, or a user removed since. */}
                  <td className="px-3 py-2">{log.actor_username ?? "system"}</td>
                  <td className="px-3 py-2 text-ink-soft">{log.module}</td>
                  <td className="px-3 py-2 font-medium">{log.action}</td>
                  <td className="px-3 py-2 text-ink-faint">{log.entity_id ?? "—"}</td>
                  <td className="px-3 py-2 text-right">
                    {log.detail != null && (
                      <Button variant="ghost" size="sm" onClick={() => setDetail(log)}>
                        Detail
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {total > PAGE_SIZE && (
        <div className="flex items-center justify-end gap-2 text-[13px] text-ink-soft">
          <span className="tnum">
            {page * PAGE_SIZE + 1}–{Math.min((page + 1) * PAGE_SIZE, total)} of {total}
          </span>
          <Button variant="outline" size="sm" disabled={page === 0} onClick={() => setPage(page - 1)}>
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= lastPage}
            onClick={() => setPage(page + 1)}
          >
            Next
          </Button>
        </div>
      )}

      <Dialog open={detail !== null} onOpenChange={(next) => !next && setDetail(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {detail?.module}.{detail?.action}
            </DialogTitle>
          </DialogHeader>
          <JsonTree data={detail?.detail} />
        </DialogContent>
      </Dialog>
    </PageShell>
  );
}
