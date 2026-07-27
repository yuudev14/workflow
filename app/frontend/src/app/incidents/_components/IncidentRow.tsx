"use client";

import * as React from "react";
import Link from "next/link";
import { Bell, Layers } from "lucide-react";
import type { Incident } from "@/services/incidents/incidents.schema";
import { Glyph, InitialsAvatar, LinkChip, StatusPill } from "@/components/soar";
import { cn, relativeAge } from "@/lib/utils";
import { INCIDENT_ICON, SEV_GLYPH_TONE, SEV_STRIPE } from "./constants";

export function IncidentRow({
  incident,
  href,
}: {
  incident: Incident;
  href: string;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "flex w-full items-center gap-3 border-t border-line bg-card px-3.5 py-3 text-left first:border-t-0",
        "hover:bg-paper-sunken"
      )}
    >
      <span className={cn("h-9 w-[3px] shrink-0 rounded", SEV_STRIPE[incident.severity])} />
      <Glyph icon={INCIDENT_ICON} tone={SEV_GLYPH_TONE[incident.severity]} />
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13.5px] font-semibold">{incident.title}</div>
        <div className="mt-1 flex flex-wrap items-center gap-1.5">
          <LinkChip>
            <Bell />
            {incident.alert_count} alerts
          </LinkChip>
          <LinkChip>
            <Layers />
            {incident.run_count} runs
          </LinkChip>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2.5">
        <StatusPill variant={incident.severity} />
        <StatusPill variant={incident.status} />
        <span className="flex w-[112px] items-center gap-1.5 text-[12.5px]">
          <InitialsAvatar name={incident.assignee} />
          <span
            className={cn("truncate", incident.assignee ? "text-ink-soft" : "text-ink-faint")}
          >
            {incident.assignee ?? "Unassigned"}
          </span>
        </span>
      </div>
      <span className="w-[44px] shrink-0 text-right font-mono text-[12px] text-ink-faint">
        {relativeAge(incident.created_at)}
      </span>
    </Link>
  );
}
