import * as React from "react";
import { cn } from "@/lib/utils";

export interface KV {
  k: React.ReactNode;
  v: React.ReactNode;
}

/** Dashed key/value rows — parameters, run inputs. */
export function KVList({ items, className }: { items: KV[]; className?: string }) {
  return (
    <div className={cn("flex flex-col gap-2", className)}>
      {items.map((it, i) => (
        <div
          key={i}
          className="flex justify-between gap-2.5 border-b border-dashed border-line pb-1.5 text-xs last:border-0 last:pb-0"
        >
          <span className="text-ink-faint">{it.k}</span>
          <span className="text-right font-mono">{it.v}</span>
        </div>
      ))}
    </div>
  );
}

/** Two-column field grid — record detail (host / user / technique …). */
export function FieldGrid({ items, className }: { items: KV[]; className?: string }) {
  return (
    <div className={cn("grid grid-cols-2 gap-x-4 gap-y-2.5", className)}>
      {items.map((it, i) => (
        <div key={i}>
          <div className="text-[11.5px] font-semibold uppercase tracking-wide text-ink-faint">
            {it.k}
          </div>
          <div className="mt-0.5 font-mono text-[13.5px]">{it.v}</div>
        </div>
      ))}
    </div>
  );
}
