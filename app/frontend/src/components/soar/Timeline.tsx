import * as React from "react";
import { cn } from "@/lib/utils";

export type TimelineTone = "signal" | "moss" | "rose" | "amber";

const NODE: Record<TimelineTone, string> = {
  signal: "bg-signal-dot",
  moss: "bg-moss-dot",
  rose: "bg-rose-dot",
  amber: "bg-amber-dot",
};

export interface TimelineEntry {
  title: React.ReactNode;
  detail?: React.ReactNode;
  tone?: TimelineTone;
}

export function Timeline({
  entries,
  className,
}: {
  entries: TimelineEntry[];
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col", className)}>
      {entries.map((e, i) => {
        const last = i === entries.length - 1;
        return (
          <div key={i} className={cn("relative flex gap-2.5", last ? "pb-0" : "pb-3.5")}>
            {!last && (
              <span className="absolute left-[5px] top-4 -bottom-0.5 w-px bg-line" />
            )}
            <span
              className={cn(
                "z-[1] mt-[3px] size-[11px] shrink-0 rounded-full",
                NODE[e.tone ?? "signal"]
              )}
            />
            <div>
              <div className="text-xs font-semibold">{e.title}</div>
              {e.detail && (
                <div className="mt-0.5 text-[12px] text-ink-faint [&_.mono]:text-ink-soft">
                  {e.detail}
                </div>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
