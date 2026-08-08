"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { AlertTriangle } from "lucide-react";

import type { Incident } from "@/services/incidents/incidents.schema";
import { InitialsAvatar, LinkChip, StatusPill } from "@/components/soar";
import { selectColumn } from "@/app/alerts/_components/alertColumns";
import { SEV_STRIPE } from "@/app/alerts/_components/alertPresentation";
import { cn, relativeAge } from "@/lib/utils";

export const INCIDENT_COLUMN_DEFAULTS = {
  team_id: false,
  tags: false,
  resolved_at: false,
  sla_deadline: false,
};

export function incidentColumns(): ColumnDef<Incident, any>[] {
  return [
    selectColumn<Incident>(),
    {
      id: "title",
      accessorKey: "title",
      header: "Incident",
      meta: { locked: true, title: "Incident" },
      cell: ({ row }) => {
        const i = row.original;
        return (
          <div className="flex items-center gap-2.5">
            <span className={cn("h-8 w-[3px] shrink-0 rounded", SEV_STRIPE[i.severity])} />
            <AlertTriangle className="size-4 shrink-0 text-rose-dot" />
            <div className="min-w-0">
              <Link
                href={`/incidents/${i.id}`}
                onClick={(e) => e.stopPropagation()}
                className="block max-w-[420px] truncate font-semibold hover:underline"
              >
                {i.title}
              </Link>
              <div className="mt-0.5 text-[12px] text-ink-faint">
                {i.alert_count} alert{i.alert_count === 1 ? "" : "s"}
              </div>
            </div>
          </div>
        );
      },
    },
    {
      id: "severity",
      accessorKey: "severity",
      header: "Severity",
      meta: { title: "Severity" },
      cell: ({ row }) => <StatusPill variant={row.original.severity} />,
    },
    {
      id: "status",
      accessorKey: "status",
      header: "Status",
      meta: { title: "Status" },
      cell: ({ row }) => <StatusPill variant={row.original.status} />,
    },
    {
      id: "sla_state",
      accessorKey: "sla_state",
      header: "SLA",
      meta: { title: "SLA" },
      cell: ({ row }) => (
        <span
          className={cn(
            "text-[12.5px] font-semibold",
            row.original.sla_state === "breached"
              ? "text-rose-text"
              : row.original.sla_state === "warning"
                ? "text-amber-text"
                : "text-ink-faint",
          )}
        >
          {row.original.sla_state}
        </span>
      ),
    },
    {
      id: "assignee",
      accessorKey: "assignee",
      header: "Owner",
      meta: { title: "Owner" },
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5 text-[12.5px]">
          <InitialsAvatar name={row.original.assignee} />
          <span className={row.original.assignee ? "text-ink-soft" : "text-ink-faint"}>
            {row.original.assignee ?? "Unassigned"}
          </span>
        </span>
      ),
    },
    {
      id: "team_id",
      accessorKey: "team_id",
      header: "Team",
      meta: { title: "Team" },
      cell: ({ row }) => (
        <span className="font-mono text-[11.5px] text-ink-faint">
          {row.original.team_id ? row.original.team_id.slice(0, 8) : "-"}
        </span>
      ),
    },
    {
      id: "tags",
      accessorKey: "tags",
      header: "Tags",
      meta: { title: "Tags" },
      cell: ({ row }) => {
        const tags = row.original.tags ?? [];
        if (tags.length === 0) return <span className="text-ink-faint">-</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {tags.slice(0, 3).map((t) => (
              <LinkChip key={t}>{t}</LinkChip>
            ))}
            {tags.length > 3 && <span className="text-[11.5px] text-ink-faint">+{tags.length - 3}</span>}
          </div>
        );
      },
    },
    {
      id: "alert_count",
      accessorKey: "alert_count",
      header: "Alerts",
      meta: { title: "Alerts", align: "right" },
      cell: ({ row }) => <span className="tnum text-ink-soft">{row.original.alert_count}</span>,
    },
    {
      id: "run_count",
      accessorKey: "run_count",
      header: "Runs",
      meta: { title: "Runs", align: "right" },
      cell: ({ row }) => <span className="tnum text-ink-soft">{row.original.run_count}</span>,
    },
    {
      id: "resolved_at",
      accessorKey: "resolved_at",
      header: "Resolved",
      meta: { title: "Resolved", align: "right" },
      cell: ({ row }) => (
        <span className="font-mono text-[12px] text-ink-faint">
          {row.original.resolved_at ? relativeAge(row.original.resolved_at) : "-"}
        </span>
      ),
    },
    {
      id: "sla_deadline",
      accessorKey: "sla_deadline",
      header: "SLA due",
      meta: { title: "SLA due", align: "right" },
      cell: ({ row }) => (
        <span className="font-mono text-[12px] text-ink-faint">
          {row.original.sla_deadline ? relativeAge(row.original.sla_deadline) : "-"}
        </span>
      ),
    },
    {
      id: "created_at",
      accessorKey: "created_at",
      header: "Age",
      meta: { title: "Age", align: "right" },
      cell: ({ row }) => (
        <span className="font-mono text-[12px] text-ink-faint">
          {relativeAge(row.original.created_at)}
        </span>
      ),
    },
  ];
}
