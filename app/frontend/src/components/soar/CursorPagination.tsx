"use client";

import * as React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PAGE_SIZES } from "@/hooks/useCursorPager";
import { cn } from "@/lib/utils";

/**
 * Prev/Next paging for a keyset-cursor endpoint, plus a page-size picker.
 *
 * No page numbers: the API takes an opaque cursor rather than an offset, so
 * "page 7" is not a question it can answer. `PaginationBar` is the offset-based
 * sibling used by the audit trail and the client-side tables.
 */
export function CursorPagination({
  page,
  shown,
  total,
  limit,
  onLimitChange,
  canPrev,
  canNext,
  onPrev,
  onNext,
  busy,
  className,
}: {
  /** Zero-based. */
  page: number;
  /** Rows on this page. */
  shown: number;
  total: number;
  limit: number;
  onLimitChange: (limit: number) => void;
  canPrev: boolean;
  canNext: boolean;
  onPrev: () => void;
  onNext: () => void;
  busy?: boolean;
  className?: string;
}) {
  const start = total === 0 ? 0 : page * limit + 1;
  const end = page * limit + shown;

  return (
    <div className={cn("flex flex-wrap items-center justify-between gap-3", className)}>
      <div className="flex items-center gap-2 text-[12.5px] text-ink-soft">
        <span className="tnum">
          {start}–{end} of {total}
        </span>
        <span className="text-ink-faint">·</span>
        <label htmlFor="page_size" className="text-ink-faint">
          Rows
        </label>
        <Select
          value={String(limit)}
          onValueChange={(v) => onLimitChange(Number(v))}
        >
          <SelectTrigger id="page_size" className="h-7 w-[72px] text-[12.5px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PAGE_SIZES.map((n) => (
              <SelectItem key={n} value={String(n)} className="text-[12.5px]">
                {n}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center gap-1.5">
        <button
          onClick={onPrev}
          disabled={!canPrev || busy}
          className="inline-flex items-center gap-1 rounded-sm border border-line-strong px-2.5 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken disabled:opacity-40"
        >
          <ChevronLeft className="size-3.5" /> Prev
        </button>
        <span className="px-1 text-[12.5px] text-ink-faint tnum">Page {page + 1}</span>
        <button
          onClick={onNext}
          disabled={!canNext || busy}
          className="inline-flex items-center gap-1 rounded-sm border border-line-strong px-2.5 py-1.5 text-[12.5px] font-semibold text-ink-soft hover:bg-paper-sunken disabled:opacity-40"
        >
          Next <ChevronRight className="size-3.5" />
        </button>
      </div>
    </div>
  );
}
