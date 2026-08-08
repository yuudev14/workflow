"use client";

import * as React from "react";
import { Columns3 } from "lucide-react";
import type { ColumnDef, VisibilityState } from "@tanstack/react-table";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

const columnId = <T,>(c: ColumnDef<T, any>): string =>
  (c.id ?? ("accessorKey" in c ? String(c.accessorKey) : "")) as string;

/**
 * Column visibility, persisted per table.
 *
 * The default lives with the columns rather than in storage, so adding a column
 * later shows it to everyone instead of leaving it hidden behind a stale
 * localStorage blob. Only keys the caller still defines are read back.
 */
export function useColumnVisibility<T>(
  storageKey: string,
  columns: ColumnDef<T, any>[],
  defaults: VisibilityState = {},
) {
  const [visibility, setVisibility] = React.useState<VisibilityState>(defaults);

  React.useEffect(() => {
    const raw = typeof window === "undefined" ? null : window.localStorage.getItem(storageKey);
    if (!raw) return;
    try {
      const stored = JSON.parse(raw) as VisibilityState;
      const known = new Set(columns.map(columnId));
      const merged = { ...defaults };
      for (const [id, shown] of Object.entries(stored)) {
        if (known.has(id)) merged[id] = shown;
      }
      setVisibility(merged);
    } catch {
      window.localStorage.removeItem(storageKey);
    }
    // Read once on mount; the columns array is rebuilt every render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storageKey]);

  const update = React.useCallback<React.Dispatch<React.SetStateAction<VisibilityState>>>(
    (next) => {
      setVisibility((prev) => {
        const resolved = typeof next === "function" ? next(prev) : next;
        window.localStorage.setItem(storageKey, JSON.stringify(resolved));
        return resolved;
      });
    },
    [storageKey],
  );

  return [visibility, update] as const;
}

export function ColumnPicker<T>({
  columns,
  visibility,
  onChange,
  className,
}: {
  columns: ColumnDef<T, any>[];
  visibility: VisibilityState;
  onChange: React.Dispatch<React.SetStateAction<VisibilityState>>;
  className?: string;
}) {
  const toggleable = columns.filter((c) => !c.meta?.locked && columnId(c));
  const hidden = toggleable.filter((c) => visibility[columnId(c)] === false).length;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            "inline-flex items-center gap-1.5 rounded-sm border border-line-strong px-3 py-1.5 text-[13px] font-semibold text-ink-soft transition-colors hover:bg-paper-sunken",
            className,
          )}
        >
          <Columns3 className="size-3.5" />
          Columns
          {hidden > 0 && <span className="tnum text-ink-faint">· {hidden} hidden</span>}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[200px]">
        <DropdownMenuLabel className="text-[11px] uppercase tracking-wide text-ink-faint">
          Columns
        </DropdownMenuLabel>
        {toggleable.map((c) => {
          const id = columnId(c);
          return (
            <DropdownMenuCheckboxItem
              key={id}
              checked={visibility[id] !== false}
              onCheckedChange={(checked) => onChange((v) => ({ ...v, [id]: checked }))}
              onSelect={(e) => e.preventDefault()}
              className="text-[13px] font-medium text-ink-soft"
            >
              {c.meta?.title ?? id}
            </DropdownMenuCheckboxItem>
          );
        })}
        {hidden > 0 && (
          <>
            <DropdownMenuSeparator />
            <button
              onClick={() => onChange({})}
              className="w-full px-2 py-1.5 text-left text-[12.5px] font-semibold text-ink-soft hover:text-foreground"
            >
              Show all
            </button>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
