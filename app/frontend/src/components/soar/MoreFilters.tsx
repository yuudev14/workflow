"use client";

import * as React from "react";
import { SlidersHorizontal } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

/**
 * Overflow for the less-used filters. Ten controls in one row is unusable, and
 * the ones that stay on the bar are the ones an analyst reaches for every shift.
 *
 * Content is kept mounted while open and closes only on the trigger, because
 * typing in a text field inside a Radix menu otherwise dismisses it.
 */
export function MoreFilters({
  activeCount,
  onClear,
  children,
  className,
}: {
  activeCount: number;
  onClear?: () => void;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-sm border px-3 py-1.5 text-[13px] font-semibold transition-colors",
            activeCount > 0
              ? "border-signal-dot bg-signal-soft text-signal-text"
              : "border-line-strong text-ink-soft hover:bg-paper-sunken",
            className,
          )}
        >
          <SlidersHorizontal className="size-3.5" />
          More filters
          {activeCount > 0 && <span className="tnum">· {activeCount}</span>}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        className="w-[320px] p-3"
        onKeyDown={(e) => e.stopPropagation()}
      >
        <DropdownMenuLabel className="px-0 text-[11px] uppercase tracking-wide text-ink-faint">
          More filters
        </DropdownMenuLabel>
        <div className="flex flex-col gap-3" onClick={(e) => e.stopPropagation()}>
          {children}
        </div>
        {activeCount > 0 && onClear && (
          <>
            <DropdownMenuSeparator />
            <button
              onClick={onClear}
              className="w-full px-0 py-1 text-left text-[12.5px] font-semibold text-ink-soft hover:text-foreground"
            >
              Clear these
            </button>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function FilterField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-[11.5px] font-semibold uppercase tracking-wide text-ink-faint">
        {label}
      </label>
      {children}
    </div>
  );
}
