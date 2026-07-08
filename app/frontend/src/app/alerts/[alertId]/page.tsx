"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, ArrowLeft, Tag, Zap } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import type { AlertStatus } from "@/services/alerts/alerts.schema";
import {
  FieldGrid,
  InitialsAvatar,
  JsonTree,
  LinkChip,
  Panel,
  PanelTitle,
  StatusMenu,
  StatusPill,
  Timeline,
} from "@/components/soar";
import { ALERT_STATUS_OPTIONS } from "../_components/constants";

const Page: React.FC<{ params: Promise<{ alertId: string }> }> = ({ params }) => {
  const { alertId } = React.use(params);
  const [status, setStatus] = React.useState<AlertStatus | null>(null);

  const alertQuery = useQuery({
    queryKey: ["alert", alertId],
    queryFn: () => AlertService.getAlertById(alertId),
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
              <StatusMenu
                value={status ?? alert.status}
                options={ALERT_STATUS_OPTIONS}
                onChange={(v) => setStatus(v as AlertStatus)}
              />
              <span>· reported by {alert.reporter ?? alert.source} · {alert.age} ago</span>
            </div>
          </div>
          <div className="flex gap-2">
            <Link
              href="/incidents"
              className="inline-flex items-center gap-1.5 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              <AlertTriangle className="size-3.5" /> Escalate
            </Link>
            <button className="inline-flex items-center gap-1.5 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110">
              <Zap className="size-3.5" /> Run playbook
            </button>
          </div>
        </div>

        <div className="mt-6 flex flex-col gap-3 lg:flex-row">
          <div className="flex flex-[1.6] flex-col gap-3">
            {alert.fields && (
              <Panel>
                <PanelTitle>Alert fields</PanelTitle>
                <FieldGrid items={alert.fields} />
              </Panel>
            )}

            {alert.linkedRun && alert.timeline && (
              <Panel>
                <PanelTitle
                  aside={<StatusPill variant={alert.linkedRun.outcome === "success" ? "success" : alert.linkedRun.outcome} />}
                >
                  Playbook run — {alert.linkedRun.playbook}
                </PanelTitle>
                <Timeline
                  entries={alert.timeline.map((t) => ({
                    title: t.title,
                    detail: t.detail && <span className="mono">{t.detail}</span>,
                    tone: t.tone,
                  }))}
                />
              </Panel>
            )}

            {alert.payload && (
              <Panel>
                <PanelTitle>Raw payload</PanelTitle>
                <JsonTree data={alert.payload} />
              </Panel>
            )}
          </div>

          <div className="flex w-full flex-col gap-3.5 rounded-md border border-line bg-card p-3.5 lg:w-[300px]">
            <Field label="Assignee">
              <div className="flex items-center gap-2 rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13.5px]">
                <InitialsAvatar name={alert.assignee} size={20} />
                {alert.assignee ?? "Unassigned"}
              </div>
            </Field>
            <Field label="Linked incident">
              <p className="text-[12px] text-ink-faint">Not yet escalated</p>
            </Field>
            {alert.relatedAlerts && (
              <Field label="Related alerts — same host, last 24h">
                <div className="flex flex-col gap-1.5">
                  {alert.relatedAlerts.map((r, i) => (
                    <LinkChip key={i} className="justify-between">
                      <span className="flex items-center gap-1.5">
                        <AlertTriangle />
                        {r.title}
                      </span>
                      <span className="font-mono text-ink-faint">{r.age}</span>
                    </LinkChip>
                  ))}
                </div>
              </Field>
            )}
            {alert.tags && (
              <Field label="Tags">
                <div className="flex flex-wrap gap-1.5">
                  {alert.tags.map((t) => (
                    <LinkChip key={t}>
                      <Tag />
                      {t}
                    </LinkChip>
                  ))}
                </div>
              </Field>
            )}
          </div>
        </div>
      </div>
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

export default Page;
