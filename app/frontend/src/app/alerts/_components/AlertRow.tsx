"use client";

import * as React from "react";
import Link from "next/link";
import { Layers } from "lucide-react";
import type { Alert } from "@/services/alerts/alerts.schema";
import { Glyph, InitialsAvatar, LinkChip, StatusPill } from "@/components/soar";
import { cn } from "@/lib/utils";
import { SEV_STRIPE, sourceGlyph } from "./alertPresentation";

const OUTCOME_LABEL = { success: "ran", failed: "failed", running: "running" } as const;

export function AlertRow({
  alert,
  selected,
  href,
  onHover,
}: {
  alert: Alert;
  selected?: boolean;
  /** navigate to the detail page on click */
  href: string;
  /** preview the alert in the side panel on hover */
  onHover?: () => void;
}) {
  const g = sourceGlyph(alert.sourceKind);
  return (
    <Link
      href={href}
      onMouseEnter={onHover}
      className={cn(
        "flex w-full items-center gap-3 border-t border-line px-3.5 py-3 text-left first:border-t-0",
        selected ? "bg-signal-soft" : "bg-card hover:bg-paper-sunken"
      )}
    >
      <span className={cn("h-9 w-[3px] shrink-0 rounded", SEV_STRIPE[alert.severity])} />
      <Glyph icon={g.icon} tone={g.tone} />
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13.5px] font-semibold">{alert.title}</div>
        <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-[12px] text-ink-faint">
          {alert.source}
          {alert.linkedRun && (
            <>
              <LinkChip>
                <Layers />
                {alert.linkedRun.playbook}
              </LinkChip>
              <StatusPill
                variant={alert.linkedRun.outcome}
                className="px-1.5 py-0.5"
              >
                {OUTCOME_LABEL[alert.linkedRun.outcome]}
              </StatusPill>
            </>
          )}
          {!alert.linkedRun && <span>no playbook matched yet</span>}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2.5">
        <StatusPill variant={alert.severity} />
        <StatusPill variant={alert.status} />
        <InitialsAvatar name={alert.assignee} />
      </div>
      <span className="w-[52px] shrink-0 text-right font-mono text-[12px] text-ink-faint">
        {alert.age}
      </span>
    </Link>
  );
}
