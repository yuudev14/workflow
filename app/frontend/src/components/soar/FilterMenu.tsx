"use client";

import * as React from "react";
import { ChevronDown } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

export interface FilterOption {
  value: string;
  label: string;
  /** Shown dim after the label, e.g. a bucket count from the summary. */
  count?: number;
}

/**
 * Multi-select filter as a checkbox dropdown. Selecting nothing means "any",
 * which is also how the API reads an absent filter key, so there is no separate
 * "All" entry to keep in sync.
 */
export function FilterMenu({
  label,
  options,
  selected,
  onChange,
  className,
}: {
  label: string;
  options: FilterOption[];
  selected: string[];
  onChange: (selected: string[]) => void;
  className?: string;
}) {
  const toggle = (value: string) => {
    onChange(
      selected.includes(value) ? selected.filter((v) => v !== value) : [...selected, value]
    );
  };

  const active = selected.length > 0;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-sm border px-3 py-1.5 text-[13px] font-semibold transition-colors",
            active
              ? "border-signal-dot bg-signal-soft text-signal-text"
              : "border-line-strong text-ink-soft hover:bg-paper-sunken",
            className
          )}
        >
          {label}
          {active && <span className="tnum">· {selected.length}</span>}
          <ChevronDown className="size-3.5" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[210px]">
        <DropdownMenuLabel className="text-[11px] uppercase tracking-wide text-ink-faint">
          {label}
        </DropdownMenuLabel>
        {options.map((o) => (
          <DropdownMenuCheckboxItem
            key={o.value}
            checked={selected.includes(o.value)}
            onCheckedChange={() => toggle(o.value)}
            onSelect={(e) => e.preventDefault()}
            className="text-[13px] font-medium text-ink-soft"
          >
            <span className="flex-1">{o.label}</span>
            {o.count !== undefined && (
              <span className="ml-2 text-ink-faint tnum">{o.count}</span>
            )}
          </DropdownMenuCheckboxItem>
        ))}
        {active && (
          <>
            <DropdownMenuSeparator />
            <button
              onClick={() => onChange([])}
              className="w-full px-2 py-1.5 text-left text-[12.5px] font-semibold text-ink-soft hover:text-foreground"
            >
              Clear
            </button>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
