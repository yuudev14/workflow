"use client";

import * as React from "react";
import Link from "next/link";
import { Bell, Layers } from "lucide-react";
import type { Incident } from "@/services/incidents/incidents.schema";
import { Glyph, InitialsAvatar, LinkChip, StatusPill } from "@/components/soar";
import { cn } from "@/lib/utils";
import { INCIDENT_ICON, SEV_GLYPH_TONE, SEV_STRIPE } from "./constants";

export function IncidentRow({
  incident,
  selected,
  href,
  onHover,
}: {
  incident: Incident;
  selected?: boolean;
  href: string;
  onHover?: () => void;
}) {
  return (
    <Link
      href={href}
      onMouseEnter={onHover}
      className={cn(
        "flex w-full items-center gap-3 border-t border-line px-3.5 py-3 text-left first:border-t-0",
        selected ? "bg-signal-soft" : "bg-card hover:bg-paper-sunken"
      )}
    >
      <span className={cn("h-9 w-[3px] shrink-0 rounded", SEV_STRIPE[incident.severity])} />
      <Glyph icon={INCIDENT_ICON} tone={SEV_GLYPH_TONE[incident.severity]} />
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13.5px] font-semibold">{incident.title}</div>
        <div className="mt-1 flex flex-wrap items-center gap-1.5">
          <LinkChip>
            <Bell />
            {incident.alertCount} alerts
          </LinkChip>
          <LinkChip>
            <Layers />
            {incident.runCount} runs
          </LinkChip>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-2.5">
        <StatusPill variant={incident.status} />
        <InitialsAvatar name={incident.owner} />
      </div>
      <span className="w-[44px] shrink-0 text-right font-mono text-[12px] text-ink-faint">
        {incident.age}
      </span>
    </Link>
  );
}
