"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

export interface PanelTab {
  value: string;
  label: string;
  /** Rendered as a dim suffix; omit rather than pass 0 to hide it. */
  count?: number;
}

/**
 * Segmented control for switching panels on a record page.
 *
 * Same visual idiom as the incidents queue tabs. Not `components/ui/tabs` -
 * that one is sized for page-level navigation (text-base, px-6 py-3) and reads
 * far too heavy sitting inside a panel header.
 */
export function PanelTabs({
  tabs,
  value,
  onChange,
  className,
}: {
  tabs: PanelTab[];
  value: string;
  onChange: (value: string) => void;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      className={cn(
        "flex w-fit gap-1 rounded-sm border border-line bg-paper-sunken p-[3px]",
        className
      )}
    >
      {tabs.map((t) => {
        const active = t.value === value;
        return (
          <button
            key={t.value}
            role="tab"
            aria-selected={active}
            onClick={() => onChange(t.value)}
            className={cn(
              "rounded-[6px] px-3 py-1.5 text-[13px] font-semibold transition-colors",
              active ? "bg-card text-foreground shadow-sm" : "text-ink-soft hover:text-foreground"
            )}
          >
            {t.label}
            {t.count !== undefined && (
              <span className={cn("ml-1.5", active ? "text-ink-faint" : "text-ink-faint")}>
                {t.count}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
