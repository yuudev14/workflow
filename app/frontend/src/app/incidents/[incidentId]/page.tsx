"use client";

import React from "react";
import Link from "next/link";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Bell, Check, Fingerprint, History, X, Zap } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import type { IncidentStatus } from "@/services/incidents/incidents.schema";
import { usePermission } from "@/hooks/usePermission";
import RecordExecutionsModal from "@/components/executions/RecordExecutionsModal";
import {
  AssigneeField,
  InitialsAvatar,
  NotesPanel,
  Panel,
  PanelTabs,
  RunPlaybookDialog,
  StatusPill,
  TagsField,
  StatusSelect,
  Stepper,
  Timeline,
  eventEntry,
  type PanelTab,
} from "@/components/soar";
import { SOURCE_LABEL } from "@/app/alerts/_components/alertPresentation";
import { relativeAge } from "@/lib/utils";
import { INCIDENT_STATUS_OPTIONS } from "../_components/constants";

const STEPS = [{ label: "Open" }, { label: "Investigating" }, { label: "Contained" }, { label: "Resolved" }];
const STEP_INDEX: Record<IncidentStatus, number> = {
  open: 0,
  investigating: 1,
  contained: 2,
  resolved: 3,
  closed: 3,
};

const Page: React.FC<{ params: Promise<{ incidentId: string }> }> = ({ params }) => {
  const { incidentId } = React.use(params);
  const queryClient = useQueryClient();

  const canUpdate = usePermission("incidents", "update");
  const canExecute = usePermission("incidents", "execute");

  const [tab, setTab] = React.useState("timeline");
  const [runOpen, setRunOpen] = React.useState(false);
  const [historyOpen, setHistoryOpen] = React.useState(false);
  const [draftStatus, setDraftStatus] = React.useState<IncidentStatus | null>(null);

  const incidentQuery = useQuery({
    queryKey: ["incident", incidentId],
    queryFn: () => IncidentService.getIncidentById(incidentId),
  });

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["incident", incidentId] });
    queryClient.invalidateQueries({ queryKey: ["incidents"] });
    queryClient.invalidateQueries({ queryKey: ["incidents-summary"] });
  };

  const statusMutation = useMutation({
    mutationFn: (status: IncidentStatus) =>
      IncidentService.setIncidentStatus(incidentId, status),
    onSuccess: () => {
      invalidate();
      setDraftStatus(null);
    },
  });

  const assignMutation = useMutation({
    mutationFn: (assigneeId: string | null) =>
      IncidentService.updateIncident(incidentId, { assignee_id: assigneeId }),
    onSuccess: invalidate,
  });

  const tagsMutation = useMutation({
    mutationFn: (tags: string[]) => IncidentService.updateIncident(incidentId, { tags }),
    onSuccess: invalidate,
  });

  const addNote = useMutation({
    mutationFn: (body: string) => IncidentService.addNote(incidentId, body),
    onSuccess: invalidate,
  });
  const editNote = useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) =>
      IncidentService.updateNote(incidentId, id, body),
    onSuccess: invalidate,
  });
  const removeNote = useMutation({
    mutationFn: (id: string) => IncidentService.deleteNote(incidentId, id),
    onSuccess: invalidate,
  });
  const unlinkAlert = useMutation({
    mutationFn: (alertId: string) => IncidentService.unlinkAlert(incidentId, alertId),
    onSuccess: () => {
      invalidate();
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });

  const incident = incidentQuery.data;

  if (incidentQuery.isLoading) return <div className="p-8 text-ink-faint">Loading…</div>;
  if (!incident) {
    return (
      <div className="p-8">
        <Link href="/incidents" className="text-signal-text">
          ← Back to incidents
        </Link>
        <p className="mt-4 text-ink-faint">Incident not found.</p>
      </div>
    );
  }

  const notes = incident.notes ?? [];
  const timeline = incident.timeline ?? [];
  const linked = incident.linked_alerts ?? [];
  const runs = incident.runs ?? [];
  const iocs = incident.iocs ?? [];
  const relatedCount = linked.length + iocs.length;

  const selectedStatus = draftStatus ?? incident.status;
  const statusDirty = selectedStatus !== incident.status;

  const tabs: PanelTab[] = [
    { value: "timeline", label: "Timeline", count: timeline.length },
    { value: "notes", label: "Notes", count: notes.length },
    { value: "related", label: "Related", count: relatedCount },
  ];

  return (
    <div className="flex justify-center">
      <div className="w-full px-6 py-8">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <Link
              href="/incidents"
              className="mb-2 inline-flex items-center gap-1.5 text-[13px] font-semibold text-ink-soft hover:text-foreground"
            >
              <ArrowLeft className="size-3.5" /> Incidents
            </Link>
            <h2>{incident.title}</h2>
            <div className="mt-1.5 flex flex-wrap items-center gap-2 text-[13px] text-ink-faint">
              <StatusPill variant={incident.severity} />
              <StatusPill variant={incident.status} />
              <span>· owner</span>
              <span className="inline-flex items-center gap-1.5 text-ink-soft">
                <InitialsAvatar name={incident.assignee} size={18} />
                {incident.assignee ?? "unassigned"}
              </span>
              <span>
                · opened {relativeAge(incident.created_at)} ago
                {incident.resolved_at
                  ? ` · resolved ${relativeAge(incident.resolved_at)} ago`
                  : ""}
              </span>
            </div>
          </div>
          <div className="flex h-fit gap-2">
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
              onClick={() => setRunOpen(true)}
              disabled={!canExecute}
              title={canExecute ? undefined : "Needs incidents:execute"}
              className="inline-flex items-center gap-1.5 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken disabled:opacity-50"
            >
              <Zap className="size-3.5" /> Run playbook
            </button>
            <button
              onClick={() => statusMutation.mutate("contained")}
              disabled={!canUpdate || statusMutation.isPending}
              className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
            >
              <Check className="size-3.5" /> Mark contained
            </button>
          </div>
        </div>

        <div className="mt-6">
          <Stepper steps={STEPS} current={STEP_INDEX[incident.status]} />
        </div>

        <div className="mt-5 flex flex-col gap-3 lg:flex-row">
          <div className="flex min-w-0 flex-[1.6] flex-col gap-3">
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
                  <RelatedGroup
                    label="Linked alerts"
                    count={linked.length}
                    empty="No alerts linked. Escalate an alert, or link one from the queue."
                  >
                    {linked.map((a) => (
                      <div
                        key={a.id}
                        className="flex items-center gap-2.5 rounded-sm border border-line bg-card px-3 py-2.5 hover:bg-paper-sunken"
                      >
                        <Bell className="size-3.5 shrink-0 text-signal-dot" />
                        <Link href={`/alerts/${a.id}`} className="min-w-0 flex-1">
                          <div className="truncate text-[13px] font-semibold">{a.title}</div>
                          <div className="truncate text-[12px] text-ink-faint">
                            {a.reporter ?? SOURCE_LABEL[a.source_kind]} · linked by {a.link_source}
                          </div>
                        </Link>
                        <StatusPill variant={a.severity} />
                        {canUpdate && (
                          <button
                            onClick={() => unlinkAlert.mutate(a.id)}
                            disabled={unlinkAlert.isPending}
                            className="text-ink-faint hover:text-rose-text"
                            aria-label="Unlink alert"
                          >
                            <X className="size-3.5" />
                          </button>
                        )}
                      </div>
                    ))}
                  </RelatedGroup>

                  {/* Populated once IOC extraction ships. */}
                  <RelatedGroup
                    label="Indicators"
                    count={iocs.length}
                    empty="No indicators - IOC extraction is not running yet."
                  >
                    {iocs.map((ioc, i) => (
                      <div
                        key={i}
                        className="flex items-center gap-2.5 rounded-sm border border-line bg-card px-3 py-2"
                      >
                        <Fingerprint className="size-3.5 shrink-0 text-ink-faint" />
                        <span className="w-20 shrink-0 text-[12px] text-ink-faint">{ioc.type}</span>
                        <span className="truncate font-mono text-[12.5px]">{ioc.value}</span>
                      </div>
                    ))}
                  </RelatedGroup>
                </div>
              )}
            </Panel>
          </div>

          <div className="flex w-full flex-col gap-3.5 rounded-md border border-line bg-card p-3.5 lg:w-[280px]">
            <Field label="Status">
              <StatusSelect
                value={selectedStatus}
                options={INCIDENT_STATUS_OPTIONS}
                disabled={!canUpdate || statusMutation.isPending}
                onChange={(v) => setDraftStatus(v as IncidentStatus)}
              />
              {statusDirty && (
                <div className="mt-1.5 flex gap-2">
                  <button
                    onClick={() => statusMutation.mutate(selectedStatus)}
                    disabled={statusMutation.isPending}
                    className="rounded-sm bg-primary px-3 py-1.5 text-[12.5px] font-semibold text-primary-foreground hover:brightness-110 disabled:opacity-50"
                  >
                    {statusMutation.isPending ? "Saving…" : "Update status"}
                  </button>
                  <button
                    onClick={() => setDraftStatus(null)}
                    className="rounded-sm border border-line-strong px-3 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </Field>
            <Field label="Severity">
              <div className="flex items-center rounded-sm border border-line-strong bg-background px-2.5 py-2">
                <StatusPill variant={incident.severity} />
              </div>
            </Field>
            <Field label="Owner">
              <AssigneeField
                assigneeId={incident.assignee_id}
                assignee={incident.assignee}
                canAssign={canUpdate}
                pending={assignMutation.isPending}
                onAssign={(id) => assignMutation.mutate(id)}
              />
            </Field>
            <Field label="Tags">
              <TagsField
                tags={incident.tags ?? []}
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
        moduleType="incident"
        recordIds={[incidentId]}
        onLaunched={invalidate}
      />

      <RecordExecutionsModal
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        moduleType="incident"
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

export default Page;
