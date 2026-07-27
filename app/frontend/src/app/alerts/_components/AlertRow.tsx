"use client";

import * as React from "react";
import Link from "next/link";
import { Copy } from "lucide-react";
import type { Alert } from "@/services/alerts/alerts.schema";
import { Glyph, InitialsAvatar, LinkChip, StatusPill } from "@/components/soar";
import { cn, relativeAge } from "@/lib/utils";
import { SEV_STRIPE, SOURCE_LABEL, sourceGlyph } from "./alertPresentation";

export function AlertRow({
  alert,
  href,
}: {
  alert: Alert;
  /** navigate to the detail page on click */
  href: string;
}) {
  const g = sourceGlyph(alert.source_kind);
  return (
    <Link
      href={href}
      className={cn(
        "flex w-full items-center gap-3 border-t border-line bg-card px-3.5 py-3 text-left first:border-t-0",
        "hover:bg-paper-sunken"
      )}
    >
      <span className={cn("h-9 w-[3px] shrink-0 rounded", SEV_STRIPE[alert.severity])} />
      <Glyph icon={g.icon} tone={g.tone} />
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13.5px] font-semibold">{alert.title}</div>
        <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-[12px] text-ink-faint">
          <span>{SOURCE_LABEL[alert.source_kind]}</span>
          {alert.reporter && <span>· {alert.reporter}</span>}
          {alert.dedup_count > 1 && (
            <LinkChip>
              <Copy />
              seen {alert.dedup_count}×
            </LinkChip>
          )}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2.5">
        <StatusPill variant={alert.severity} />
        <StatusPill variant={alert.status} />
        <span className="flex w-[112px] items-center gap-1.5 text-[12.5px]">
          <InitialsAvatar name={alert.assignee} />
          <span className={cn("truncate", alert.assignee ? "text-ink-soft" : "text-ink-faint")}>
            {alert.assignee ?? "Unassigned"}
          </span>
        </span>
      </div>
      <span className="w-[52px] shrink-0 text-right font-mono text-[12px] text-ink-faint">
        {relativeAge(alert.created_at)}
      </span>
    </Link>
  );
}
