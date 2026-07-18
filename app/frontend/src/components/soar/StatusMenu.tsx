"use client";

import * as React from "react";
import { Check, ChevronDown } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { StatusPill, pillLabel, type PillVariant } from "./StatusPill";
import { cn } from "@/lib/utils";

const DOT: Partial<Record<PillVariant, string>> = {
  new: "bg-slate-dot",
  open: "bg-rose-dot",
  investigating: "bg-amber-dot",
  contained: "bg-signal-dot",
  resolved: "bg-moss-dot",
  falsepos: "bg-slate-dot",
  closed: "bg-slate-dot",
};

export interface StatusOption {
  value: PillVariant;
  label?: string;
  /** render a divider before this option */
  divider?: boolean;
}

/** A status pill that opens a menu to change the status. */
export function StatusMenu({
  value,
  options,
  onChange,
  prefix,
  align = "start",
}: {
  value: PillVariant;
  options: StatusOption[];
  onChange?: (v: PillVariant) => void;
  prefix?: React.ReactNode;
  align?: "start" | "end";
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="outline-none">
          <StatusPill variant={value} className="cursor-pointer hover:brightness-95">
            {prefix}
            {pillLabel(value)}
            <ChevronDown className="size-3" />
          </StatusPill>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="w-[200px]">
        {options.map((o) => (
          <React.Fragment key={o.value}>
            {o.divider && <DropdownMenuSeparator />}
            <DropdownMenuItem
              onClick={() => onChange?.(o.value)}
              className="flex items-center gap-2 text-xs font-semibold text-ink-soft"
            >
              <span className={cn("size-[7px] rounded-full", DOT[o.value] ?? "bg-slate-dot")} />
              {o.label ?? pillLabel(o.value)}
              {o.value === value && <Check className="ml-auto size-3 text-moss-dot" />}
            </DropdownMenuItem>
          </React.Fragment>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
