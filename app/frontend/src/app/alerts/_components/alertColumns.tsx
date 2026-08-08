"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Copy } from "lucide-react";

import type { Alert } from "@/services/alerts/alerts.schema";
import { Glyph, InitialsAvatar, LinkChip, StatusPill } from "@/components/soar";
import { cn, readableDate, relativeAge } from "@/lib/utils";
import { SEV_STRIPE, SOURCE_LABEL, sourceGlyph } from "./alertPresentation";

/** Columns hidden until an analyst turns them on. */
export const ALERT_COLUMN_DEFAULTS = {
  team_id: false,
  tags: false,
  last_seen: false,
  triaged_at: false,
  sla_deadline: false,
  run_count: false,
};

export const selectColumn = <T,>(): ColumnDef<T, any> => ({
  id: "select",
  meta: { locked: true },
  header: ({ table }) => (
    <input
      type="checkbox"
      aria-label="Select all on this page"
      checked={table.getIsAllRowsSelected()}
      ref={(el) => {
        if (el) el.indeterminate = table.getIsSomeRowsSelected() && !table.getIsAllRowsSelected();
      }}
      onChange={table.getToggleAllRowsSelectedHandler()}
      className="size-3.5 cursor-pointer accent-[var(--signal-dot)]"
    />
  ),
  cell: ({ row }) => (
    <input
      type="checkbox"
      aria-label="Select row"
      checked={row.getIsSelected()}
      onChange={row.getToggleSelectedHandler()}
      // The row itself navigates; a click on the box must only select.
      onClick={(e) => e.stopPropagation()}
      className="size-3.5 cursor-pointer accent-[var(--signal-dot)]"
    />
  ),
});

export function alertColumns(): ColumnDef<Alert, any>[] {
  return [
    selectColumn<Alert>(),
    {
      id: "title",
      accessorKey: "title",
      header: "Alert",
      meta: { locked: true, title: "Alert" },
      cell: ({ row }) => {
        const a = row.original;
        const g = sourceGlyph(a.source_kind);
        return (
          <div className="flex items-center gap-2.5">
            <span className={cn("h-8 w-[3px] shrink-0 rounded", SEV_STRIPE[a.severity])} />
            <Glyph icon={g.icon} tone={g.tone} />
            <div className="min-w-0">
              {/* A real link so middle-click and copy-link still work, even
                  though the whole row navigates. */}
              <Link
                href={`/alerts/${a.id}`}
                onClick={(e) => e.stopPropagation()}
                className="block max-w-[420px] truncate font-semibold hover:underline"
              >
                {a.title}
              </Link>
              <div className="mt-0.5 flex items-center gap-1.5 text-[12px] text-ink-faint">
                <span>{SOURCE_LABEL[a.source_kind]}</span>
                {a.reporter && <span>· {a.reporter}</span>}
                {a.dedup_count > 1 && (
                  <LinkChip>
                    <Copy />
                    seen {a.dedup_count}×
                  </LinkChip>
                )}
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
      header: "Assignee",
      meta: { title: "Assignee" },
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
      id: "dedup_count",
      accessorKey: "dedup_count",
      header: "Seen",
      meta: { title: "Seen", align: "right" },
      cell: ({ row }) => <span className="tnum text-ink-soft">{row.original.dedup_count}</span>,
    },
    {
      id: "run_count",
      accessorKey: "run_count",
      header: "Runs",
      meta: { title: "Runs", align: "right" },
      cell: ({ row }) => <span className="tnum text-ink-soft">{row.original.run_count}</span>,
    },
    {
      id: "last_seen",
      accessorKey: "last_seen",
      header: "Last seen",
      meta: { title: "Last seen", align: "right" },
      cell: ({ row }) => (
        <span className="font-mono text-[12px] text-ink-faint">
          {relativeAge(row.original.last_seen)}
        </span>
      ),
    },
    {
      id: "triaged_at",
      accessorKey: "triaged_at",
      header: "Triaged",
      meta: { title: "Triaged", align: "right" },
      cell: ({ row }) => (
        <span className="font-mono text-[12px] text-ink-faint">
          {row.original.triaged_at ? readableDate(row.original.triaged_at, "MMM D, h:mm a") : "-"}
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
