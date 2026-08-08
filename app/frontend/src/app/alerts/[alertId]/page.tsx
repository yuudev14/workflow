"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ArrowLeft, Bell, Copy, History, Zap } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import type { AlertStatus } from "@/services/alerts/alerts.schema";
import { usePermission } from "@/hooks/usePermission";
import RecordExecutionsModal from "@/components/executions/RecordExecutionsModal";
import {
  FieldGrid,
  AssigneeField,
  JsonTree,
  LinkChip,
  NotesPanel,
  Panel,
  PanelTabs,
  PanelTitle,
  RunPlaybookDialog,
  StatusPill,
  StatusSelect,
  TagsField,
  Timeline,
  eventEntry,
  type PanelTab,
} from "@/components/soar";
import { readableDate, relativeAge } from "@/lib/utils";
import { SOURCE_LABEL } from "../_components/alertPresentation";
import { ALERT_STATUS_OPTIONS } from "../_components/constants";

/** Statuses that close an alert out, and so want a disposition note. */
const CLOSING: AlertStatus[] = ["resolved", "falsepos", "closed"];

const Page: React.FC<{ params: Promise<{ alertId: string }> }> = ({ params }) => {
  const { alertId } = React.use(params);
  const router = useRouter();
  const queryClient = useQueryClient();

  const canUpdate = usePermission("alerts", "update");
  const canEscalate = usePermission("incidents", "create");
  const canExecute = usePermission("alerts", "execute");

  const [tab, setTab] = React.useState("related");
  const [draftStatus, setDraftStatus] = React.useState<AlertStatus | null>(null);
  const [closureNote, setClosureNote] = React.useState("");
  const [runOpen, setRunOpen] = React.useState(false);
  const [historyOpen, setHistoryOpen] = React.useState(false);

  const alertQuery = useQuery({
    queryKey: ["alert", alertId],
    queryFn: () => AlertService.getAlertById(alertId),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["alert", alertId] });
    queryClient.invalidateQueries({ queryKey: ["alerts"] });
    queryClient.invalidateQueries({ queryKey: ["alerts-summary"] });
  };

  const statusMutation = useMutation({
    mutationFn: (payload: { status: AlertStatus; closure_note?: string }) =>
      AlertService.setAlertStatus(alertId, payload),
    onSuccess: () => {
      invalidate();
      setDraftStatus(null);
      setClosureNote("");
    },
  });

  const assignMutation = useMutation({
    mutationFn: (assigneeId: string | null) =>
      AlertService.updateAlert(alertId, { assignee_id: assigneeId }),
    onSuccess: invalidate,
  });

  const tagsMutation = useMutation({
    mutationFn: (tags: string[]) => AlertService.updateAlert(alertId, { tags }),
    onSuccess: invalidate,
  });

  const escalateMutation = useMutation({
    mutationFn: () => AlertService.escalateAlert(alertId),
    onSuccess: (incident) => {
      invalidate();
      queryClient.invalidateQueries({ queryKey: ["incidents"] });
      router.push(`/incidents/${incident.id}`);
    },
  });

  const addNote = useMutation({
    mutationFn: (body: string) => AlertService.addAlertNote(alertId, body),
    onSuccess: invalidate,
  });
  const editNote = useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) =>
      AlertService.updateAlertNote(alertId, id, body),
    onSuccess: invalidate,
  });
  const removeNote = useMutation({
    mutationFn: (id: string) => AlertService.deleteAlertNote(alertId, id),
    onSuccess: invalidate,
  });

  const alert = alertQuery.data;

  if (alertQuery.isLoading) {
    return <div className="p-8 text-ink-faint">Loading…</div>;
  }
  if (!alert) {
    return (
      <div className="p-8">
        <Link href="/alerts" className="text-signal-text">
          ← Back to alerts
        </Link>
        <p className="mt-4 text-ink-faint">Alert not found.</p>
      </div>
    );
  }

  const fields = [
    { k: "Source", v: SOURCE_LABEL[alert.source_kind] },
    { k: "Reporter", v: alert.reporter ?? "-" },
    { k: "First seen", v: readableDate(alert.created_at, "MMM D, h:mm a") },
    { k: "Last seen", v: readableDate(alert.last_seen, "MMM D, h:mm a") },
    { k: "Occurrences", v: String(alert.dedup_count) },
    { k: "Triaged", v: alert.triaged_at ? readableDate(alert.triaged_at, "MMM D, h:mm a") : "-" },
  ];

  const notes = alert.notes ?? [];
  const timeline = alert.timeline ?? [];
  const incidents = alert.linked_incidents ?? [];
  const related = alert.related_alerts ?? [];
  const runs = alert.runs ?? [];
  const relatedCount = incidents.length + related.length;

  const selectedStatus = draftStatus ?? alert.status;
  const statusDirty = selectedStatus !== alert.status;
  const needsNote = statusDirty && CLOSING.includes(selectedStatus);

  const tabs: PanelTab[] = [
    { value: "timeline", label: "Timeline", count: timeline.length },
    { value: "notes", label: "Notes", count: notes.length },
    { value: "related", label: "Related", count: relatedCount },
    { value: "payload", label: "Payload" },
  ];

  return (
    <div className="flex justify-center">
      <div className="w-full px-6 py-8">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <Link
              href="/alerts"
              className="mb-2 inline-flex items-center gap-1.5 text-[13px] font-semibold text-ink-soft hover:text-foreground"
            >
              <ArrowLeft className="size-3.5" /> Alerts queue
            </Link>
            <h2>{alert.title}</h2>
            <div className="mt-1.5 flex flex-wrap items-center gap-2 text-[13px] text-ink-faint">
              <StatusPill variant={alert.severity} />
              <StatusPill variant={alert.status} />
              <span>
                · reported by {alert.reporter ?? SOURCE_LABEL[alert.source_kind]} ·{" "}
                {relativeAge(alert.created_at)} ago
              </span>
              {alert.dedup_count > 1 && (
                <LinkChip>
                  <Copy />
                  seen {alert.dedup_count}×
                </LinkChip>
              )}
            </div>
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => setHistoryOpen(true)}
              className="inline-flex items-center gap-1.5 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              <History className="size-3.5" /> Executions
              {runs.length > 0 && (
                <span className="rounded-full bg-paper-sunken px-1.5 text-[11.5px] tabular-nums">
                  {runs.length}
                </span>
              )}
            </button>
            <button
              onClick={() => escalateMutation.mutate()}
              disabled={!canEscalate || escalateMutation.isPending}
              className="inline-flex items-center gap-1.5 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken disabled:opacity-50"
            >
              <AlertTriangle className="size-3.5" />
              {escalateMutation.isPending ? "Escalating…" : "Escalate"}
            </button>
            <button
              onClick={() => setRunOpen(true)}
              disabled={!canExecute}
              title={canExecute ? undefined : "Needs alerts:execute"}
              className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              <Zap className="size-3.5" /> Run playbook
            </button>
          </div>
        </div>

        <div className="mt-6 flex flex-col gap-3 lg:flex-row">
          <div className="flex min-w-0 flex-[1.6] flex-col gap-3">
            <Panel>
              <PanelTitle>Alert fields</PanelTitle>
              <FieldGrid items={fields} />
            </Panel>

            {alert.closure_note && (
              <div className="rounded-md border border-line bg-paper-sunken px-3.5 py-2.5 text-[13px]">
                <span className="font-semibold text-ink-soft">Closure note · </span>
                <span className="text-ink-soft">{alert.closure_note}</span>
              </div>
            )}

            <Panel>
              <PanelTabs tabs={tabs} value={tab} onChange={setTab} className="mb-3.5" />

              {tab === "timeline" &&
                (timeline.length === 0 ? (
                  <Empty>Nothing recorded yet.</Empty>
                ) : (
                  <Timeline
                    entries={timeline.map((e) => {
                      const entry = eventEntry(e);
                      return {
                        title: entry.title,
                        detail: entry.detail && <span className="mono">{entry.detail}</span>,
                        tone: entry.tone,
                      };
                    })}
                  />
                ))}

              {tab === "notes" && (
                <NotesPanel
                  embedded
                  notes={notes}
                  canWrite={canUpdate}
                  pending={addNote.isPending || editNote.isPending || removeNote.isPending}
                  onAdd={(body) => addNote.mutate(body)}
                  onEdit={(id, body) => editNote.mutate({ id, body })}
                  onDelete={(id) => removeNote.mutate(id)}
                />
              )}

              {tab === "related" && (
                <div className="flex flex-col gap-4">
                  <RelatedGroup label="Linked incidents" count={incidents.length}>
                    {incidents.map((inc) => (
                      <RelatedRow
                        key={inc.id}
                        href={`/incidents/${inc.id}`}
                        icon={<AlertTriangle className="size-3.5 text-rose-dot" />}
                        title={inc.title}
                        meta={`${inc.status} · linked by ${inc.link_source}`}
                        aside={<StatusPill variant={inc.severity} />}
                      />
                    ))}
                  </RelatedGroup>

                  {/* Populated once alert correlation ships; the API returns [] today. */}
                  <RelatedGroup
                    label="Related alerts"
                    count={related.length}
                    empty="No correlated alerts - correlation is not running yet."
                  >
                    {related.map((r) => (
                      <RelatedRow
                        key={r.id}
                        href={`/alerts/${r.id}`}
                        icon={<Bell className="size-3.5 text-signal-dot" />}
                        title={r.title}
                        meta={
                          r.shared_iocs.length > 0
                            ? `shares ${r.shared_iocs.join(", ")}`
                            : `${relativeAge(r.created_at)} ago`
                        }
                        aside={
                          <span className="font-mono text-[12px] text-ink-faint">
                            {r.score.toFixed(2)}
                          </span>
                        }
                      />
                    ))}
                  </RelatedGroup>
                </div>
              )}

              {tab === "payload" &&
                (alert.payload ? (
                  <JsonTree
                    data={alert.payload}
                    className="max-h-[360px] overflow-y-auto p-3 text-[11.5px] leading-[1.6]"
                  />
                ) : (
                  <Empty>No payload was sent with this alert.</Empty>
                ))}
            </Panel>
          </div>

          <div className="flex w-full flex-col gap-3.5 rounded-md border border-line bg-card p-3.5 lg:w-[300px]">
            <Field label="Status">
              <StatusSelect
                value={selectedStatus}
                options={ALERT_STATUS_OPTIONS}
                disabled={!canUpdate || statusMutation.isPending}
                onChange={(v) => setDraftStatus(v as AlertStatus)}
              />
              {needsNote && (
                <textarea
                  value={closureNote}
                  onChange={(e) => setClosureNote(e.target.value)}
                  rows={3}
                  placeholder="Closure note - why is this being closed?"
                  className="mt-1.5 w-full resize-y rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13px] outline-none placeholder:text-ink-faint focus:border-signal-dot"
                />
              )}
              {statusDirty && (
                <div className="mt-1.5 flex gap-2">
                  <button
                    onClick={() =>
                      statusMutation.mutate({
                        status: selectedStatus,
                        ...(needsNote && closureNote.trim()
                          ? { closure_note: closureNote.trim() }
                          : {}),
                      })
                    }
                    disabled={statusMutation.isPending}
                    className="rounded-sm bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
                  >
                    {statusMutation.isPending ? "Saving…" : "Update status"}
                  </button>
                  <button
                    onClick={() => {
                      setDraftStatus(null);
                      setClosureNote("");
                    }}
                    className="rounded-sm border border-line-strong px-3 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </Field>
            <Field label="Assignee">
              <AssigneeField
                assigneeId={alert.assignee_id}
                assignee={alert.assignee}
                canAssign={canUpdate}
                pending={assignMutation.isPending}
                onAssign={(id) => assignMutation.mutate(id)}
              />
            </Field>
            <Field label="Tags">
              <TagsField
                tags={alert.tags ?? []}
                canEdit={canUpdate}
                pending={tagsMutation.isPending}
                onChange={(tags) => tagsMutation.mutate(tags)}
              />
            </Field>
          </div>
        </div>
      </div>

      <RunPlaybookDialog
        open={runOpen}
        onOpenChange={setRunOpen}
        moduleType="alert"
        recordIds={[alertId]}
        onLaunched={invalidate}
      />

      <RecordExecutionsModal
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        moduleType="alert"
        runs={runs}
        action={
          canExecute && (
            <button
              onClick={() => {
                setHistoryOpen(false);
                setRunOpen(true);
              }}
              className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:brightness-110"
            >
              <Zap className="size-3.5" /> Run playbook
            </button>
          )
        }
      />
    </div>
  );
};

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
        {label}
      </label>
      {children}
    </div>
  );
}

function Empty({ children }: { children: React.ReactNode }) {
  return <p className="py-2 text-[13px] text-ink-faint">{children}</p>;
}

function RelatedGroup({
  label,
  count,
  empty,
  children,
}: {
  label: string;
  count: number;
  empty?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
        {label} {count > 0 && <span className="text-ink-faint">({count})</span>}
      </label>
      {count === 0 ? (
        <Empty>{empty ?? "None."}</Empty>
      ) : (
        <div className="flex flex-col gap-2">{children}</div>
      )}
    </div>
  );
}

function RelatedRow({
  href,
  icon,
  title,
  meta,
  aside,
}: {
  href: string;
  icon: React.ReactNode;
  title: string;
  meta: string;
  aside?: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className="flex items-center gap-2.5 rounded-sm border border-line bg-card px-3 py-2.5 hover:bg-paper-sunken"
    >
      {icon}
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13px] font-semibold">{title}</div>
        <div className="truncate text-[12px] text-ink-faint">{meta}</div>
      </div>
      {aside}
    </Link>
  );
}

export default Page;
