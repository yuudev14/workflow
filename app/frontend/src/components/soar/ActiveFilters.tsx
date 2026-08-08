"use client";

import * as React from "react";
import { X } from "lucide-react";

export interface ActiveFilter {
  /** Which filter this came from, e.g. "Severity". */
  group: string;
  value: string;
  label: string;
  onRemove: () => void;
}

/**
 * The current filter set, spelled out. Without this a filter buried in a
 * dropdown silently shrinks the queue and reads as missing data.
 */
export function ActiveFilters({
  filters,
  onClearAll,
}: {
  filters: ActiveFilter[];
  onClearAll: () => void;
}) {
  if (filters.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {filters.map((f) => (
        <span
          key={`${f.group}:${f.value}`}
          className="inline-flex items-center gap-1.5 rounded-sm border border-line-strong bg-card px-2 py-1 text-[12px] text-ink-soft"
        >
          <span className="text-ink-faint">{f.group}</span>
          {f.label}
          <button
            onClick={f.onRemove}
            className="text-ink-faint hover:text-rose-text"
            aria-label={`Remove ${f.group} ${f.label}`}
          >
            <X className="size-3" />
          </button>
        </span>
      ))}
      {filters.length > 1 && (
        <button
          onClick={onClearAll}
          className="px-1.5 text-[12px] font-semibold text-ink-soft hover:text-foreground"
        >
          Clear all
        </button>
      )}
    </div>
  );
}
