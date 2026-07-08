"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, ChevronRight, RefreshCw, Zap } from "lucide-react";

import AlertService from "@/services/alerts/alerts";
import type { AlertStatus, Severity } from "@/services/alerts/alerts.schema";
import {
  FilterChips,
  InitialsAvatar,
  JsonTree,
  StatusMenu,
} from "@/components/soar";
import { Skeleton } from "@/components/ui/skeleton";
import { AlertRow } from "./_components/AlertRow";
import { ALERT_STATUS_OPTIONS } from "./_components/constants";

export default function Page() {
  const router = useRouter();
  const [severity, setSeverity] = React.useState<Severity | "all">("all");
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [statusOverride, setStatusOverride] = React.useState<Record<string, AlertStatus>>({});

  const alertsQuery = useQuery({ queryKey: ["alerts"], queryFn: () => AlertService.getAlerts() });
  const alerts = alertsQuery.data ?? [];

  const filtered = alerts.filter((a) => severity === "all" || a.severity === severity);
  const selected =
    alerts.find((a) => a.id === selectedId) ?? filtered[0] ?? alerts[0];

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-5 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Alerts</h1>
            <p className="mt-1 text-[15px] text-ink-soft">47 open · 5 critical</p>
          </div>
          <div className="flex items-center gap-2">
            <Link
              href="/alerts/dashboard"
              className="inline-flex items-center gap-2 rounded-sm border border-line-strong px-3.5 py-2 text-[13.5px] font-semibold text-ink-soft hover:bg-paper-sunken"
            >
              Dashboard
            </Link>
            <button className="flex size-9 items-center justify-center rounded-sm border border-line-strong text-ink-soft hover:bg-paper-sunken">
              <RefreshCw className="size-4" />
            </button>
          </div>
        </div>

        <FilterChips
          value={severity}
          onChange={(v) => setSeverity(v as Severity | "all")}
          chips={[
            { value: "all", label: "All (47)" },
            { value: "critical", label: "Critical (5)", accent: "text-rose-text border-rose-dot" },
            { value: "high", label: "High (14)", accent: "text-amber-text border-amber-dot" },
            { value: "medium", label: "Medium" },
            { value: "low", label: "Low" },
          ]}
        />

        <div className="flex gap-3.5">
          <div className="flex-[1.35]">
            {alertsQuery.isLoading ? (
              <div className="flex flex-col gap-2">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-[62px] rounded-md" />
                ))}
              </div>
            ) : (
              <div className="overflow-hidden rounded-md border border-line">
                {filtered.map((a) => (
                  <AlertRow
                    key={a.id}
                    alert={{ ...a, status: statusOverride[a.id] ?? a.status }}
                    selected={selected?.id === a.id}
                    href={`/alerts/${a.id}`}
                    onHover={() => setSelectedId(a.id)}
                  />
                ))}
              </div>
            )}
          </div>

          {selected && (
            <div className="flex w-[320px] shrink-0 flex-col gap-2.5 rounded-md border border-line bg-card p-3.5">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="text-[15px] font-semibold">{selected.title}</div>
                  <div className="mt-1 text-[12.5px] text-ink-faint">
                    {selected.severity[0].toUpperCase() + selected.severity.slice(1)} · from{" "}
                    {selected.reporter ?? selected.source} · {selected.age} ago
                  </div>
                </div>
                <Link
                  href={`/alerts/${selected.id}`}
                  className="inline-flex items-center gap-1 text-[13px] font-semibold text-ink-soft hover:text-foreground"
                >
                  Open <ChevronRight className="size-3.5" />
                </Link>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                  Status
                </label>
                <div>
                  <StatusMenu
                    value={statusOverride[selected.id] ?? selected.status}
                    options={ALERT_STATUS_OPTIONS}
                    onChange={(v) =>
                      setStatusOverride((prev) => ({ ...prev, [selected.id]: v as AlertStatus }))
                    }
                  />
                </div>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                  Assignee
                </label>
                <div className="flex items-center gap-2 rounded-sm border border-line-strong bg-background px-2.5 py-2 text-[13.5px]">
                  <InitialsAvatar name={selected.assignee} size={20} />
                  {selected.assignee ?? "Unassigned"}
                </div>
              </div>

              {selected.payload && (
                <div className="flex flex-col gap-1.5">
                  <label className="text-[12px] font-semibold uppercase tracking-wide text-ink-soft">
                    Payload
                  </label>
                  <JsonTree data={selected.payload} />
                </div>
              )}

              <div className="mt-1 flex gap-2">
                <button
                  onClick={() => router.push("/incidents")}
                  className="flex flex-1 items-center justify-center gap-1.5 rounded-sm border border-line-strong px-3 py-2 text-[13px] font-semibold text-ink-soft hover:bg-paper-sunken"
                >
                  <AlertTriangle className="size-3.5" /> Escalate
                </button>
                <button className="flex flex-1 items-center justify-center gap-1.5 rounded-sm bg-primary px-3 py-2 text-[13px] font-semibold text-primary-foreground hover:brightness-110">
                  <Zap className="size-3.5" /> Run playbook
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
