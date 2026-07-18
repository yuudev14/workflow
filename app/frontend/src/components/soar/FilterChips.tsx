"use client";

import * as React from "react";
import { cn } from "@/lib/utils";

export interface FilterChip {
  value: string;
  label: React.ReactNode;
  /** optional accent classes when not selected (e.g. severity coloring) */
  accent?: string;
}

export function FilterChips({
  chips,
  value,
  onChange,
  className,
}: {
  chips: FilterChip[];
  value: string;
  onChange?: (v: string) => void;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {chips.map((c) => {
        const on = c.value === value;
        return (
          <button
            key={c.value}
            type="button"
            onClick={() => onChange?.(c.value)}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1.5 text-xs font-semibold transition-colors",
              on
                ? "border-foreground bg-foreground text-background"
                : cn("border-line-strong text-ink-soft hover:bg-paper-sunken", c.accent)
            )}
          >
            {c.label}
          </button>
        );
      })}
    </div>
  );
}
