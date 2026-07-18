"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Check, ChevronDown, Globe, Tag } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import type { IncidentStatus } from "@/services/incidents/incidents.schema";
import {
  Glyph,
  InitialsAvatar,
  LinkChip,
  Panel,
  PanelTitle,
  StatusMenu,
  StatusPill,
  Stepper,
  Timeline,
} from "@/components/soar";
import { cn } from "@/lib/utils";
import { INCIDENT_STATUS_OPTIONS, SEV_GLYPH_TONE, SEV_STRIPE } from "../_components/constants";

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
  const [status, setStatus] = React.useState<IncidentStatus | null>(null);

  const incidentQuery = useQuery({
    queryKey: ["incident", incidentId],
    queryFn: () => IncidentService.getIncidentById(incidentId),
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

  const currentStatus = status ?? incident.status;

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
              <span>· owner</span>
              <span className="inline-flex items-center gap-1.5 text-ink-soft">
                <InitialsAvatar name={incident.owner} size={18} />
                {incident.owner ?? "unassigned"}
              </span>
              <span>· opened {incident.openedAgo}{incident.slaLeft ? ` · SLA ${incident.slaLeft}` : ""}</span>
            </div>
          </div>
          <div className="flex gap-2">
            <button className="rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken">
              Add note
            </button>
            <button className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110">
              <Check className="size-3.5" /> Mark contained
            </button>
          </div>
        </div>

        <div className="mt-6 flex flex-wrap items-center justify-between gap-3">
          <Stepper steps={STEPS} current={STEP_INDEX[currentStatus]} />
          <StatusMenu
            align="end"
            value={currentStatus}
            options={INCIDENT_STATUS_OPTIONS}
            onChange={(v) => setStatus(v as IncidentStatus)}
            prefix={<span className="text-inherit">Set status:&nbsp;</span>}
          />
        </div>

        <div className="mt-5 flex flex-col gap-3 lg:flex-row">
          <div className="flex flex-[1.6] flex-col gap-3">
            {incident.timeline && (
              <Panel>
                <PanelTitle>Timeline</PanelTitle>
                <Timeline
                  entries={incident.timeline.map((t) => ({
                    title: t.title,
                    detail: t.detail && <span className="mono">{t.detail}</span>,
                    tone: t.tone,
                  }))}
                />
              </Panel>
            )}

            {incident.linkedAlerts && (
              <Panel>
                <PanelTitle>Linked alerts ({incident.linkedAlerts.length})</PanelTitle>
                <div className="flex flex-col gap-2">
                  {incident.linkedAlerts.map((a) => (
                    <Link
                      key={a.id}
                      href={`/alerts/${a.id}`}
                      className="flex items-center gap-3 rounded-sm border border-line bg-card px-3 py-2.5 hover:bg-paper-sunken"
                    >
                      <span className={cn("h-8 w-[3px] shrink-0 rounded", SEV_STRIPE[a.severity])} />
                      <Glyph icon={Globe} tone={SEV_GLYPH_TONE[a.severity]} />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-[13px] font-semibold">{a.title}</div>
                        <div className="text-[12px] text-ink-faint">{a.source}</div>
                      </div>
                      <StatusPill variant={a.severity} />
                    </Link>
                  ))}
                </div>
              </Panel>
            )}

            {incident.runs && (
              <Panel>
                <PanelTitle>Playbook runs ({incident.runs.length})</PanelTitle>
                <div className="flex flex-col gap-2">
                  {incident.runs.map((r, i) => (
                    <Link
                      key={i}
                      href="/playbooks/executions"
                      className="flex items-center gap-3 rounded-md border border-line bg-card px-3.5 py-3 hover:bg-paper-sunken"
                    >
                      <Glyph icon={Globe} tone={r.outcome === "failed" ? "rose" : "moss"} />
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-[13.5px] font-semibold">{r.playbook}</div>
                        <div className="text-[12px] text-ink-faint">{r.detail}</div>
                      </div>
                      <StatusPill variant={r.outcome === "failed" ? "failed" : "success"} />
                    </Link>
                  ))}
                </div>
              </Panel>
            )}

            {incident.notes && (
              <Panel>
                <PanelTitle>Notes</PanelTitle>
                <div className="flex flex-col">
                  {incident.notes.map((n, i) => (
                    <div
                      key={i}
                      className="flex gap-2.5 border-t border-line py-2.5 first:border-t-0 first:pt-0"
                    >
                      <InitialsAvatar name={n.who} />
                      <div className="flex-1">
                        <div className="text-[13px] font-semibold">
                          {n.who}
                          <span className="ml-1.5 font-medium text-ink-faint">{n.when}</span>
                        </div>
                        <div className="mt-0.5 text-[13px] leading-relaxed text-ink-soft">{n.text}</div>
                      </div>
                    </div>
                  ))}
                  <div className="mt-1.5 rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13.5px] text-ink-faint">
                    Add a note…
                  </div>
                </div>
              </Panel>
            )}
          </div>

          <div className="flex w-full flex-col gap-3.5 rounded-md border border-line bg-card p-3.5 lg:w-[280px]">
            {incident.iocs && (
              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                  Indicators
                </label>
                <div className="overflow-x-auto">
                  <table className="w-full border-collapse text-[12.5px]">
                    <thead>
                      <tr>
                        <th className="pb-2 pr-2 text-left text-[11px] font-semibold uppercase tracking-wide text-ink-faint">
                          Type
                        </th>
                        <th className="pb-2 text-left text-[11px] font-semibold uppercase tracking-wide text-ink-faint">
                          Value
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {incident.iocs.map((ioc, i) => (
                        <tr key={i}>
                          <td className="border-t border-line py-2 pr-2">{ioc.type}</td>
                          <td className="border-t border-line py-2 font-mono">{ioc.value}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            {incident.tags && (
              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                  Tags
                </label>
                <div className="flex flex-wrap gap-1.5">
                  {incident.tags.map((t) => (
                    <LinkChip key={t}>
                      <Tag />
                      {t}
                    </LinkChip>
                  ))}
                </div>
              </div>
            )}
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                Severity
              </label>
              <div className="flex items-center justify-between rounded-sm border border-line-strong bg-background px-2.5 py-2">
                <StatusPill variant={incident.severity} />
                <ChevronDown className="size-4 text-ink-soft" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Page;
