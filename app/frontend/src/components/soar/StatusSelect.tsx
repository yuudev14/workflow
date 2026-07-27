"use client";

import * as React from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { pillLabel, type PillVariant } from "./StatusPill";
import type { StatusOption } from "./StatusMenu";
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

/**
 * Status picker as a full-width form control, for record pages where the status
 * is a field being edited rather than a badge being glanced at. `StatusMenu` is
 * the pill-shaped variant used in list side-panels.
 */
export function StatusSelect({
  value,
  options,
  onChange,
  disabled,
  id,
}: {
  value: PillVariant;
  options: StatusOption[];
  onChange: (value: PillVariant) => void;
  disabled?: boolean;
  id?: string;
}) {
  return (
    <Select value={value} disabled={disabled} onValueChange={(v) => onChange(v as PillVariant)}>
      <SelectTrigger id={id} className="text-[13.5px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value} className="text-[13px]">
            <span className="flex items-center gap-2">
              <span className={cn("size-[7px] rounded-full", DOT[o.value] ?? "bg-slate-dot")} />
              {o.label ?? pillLabel(o.value)}
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
