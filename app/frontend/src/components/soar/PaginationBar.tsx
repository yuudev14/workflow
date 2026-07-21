"use client";

import * as React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

/**
 * One pagination control for both worlds: client-side tables drive it from the
 * react-table instance, the audit trail drives it from server offsets. Shown
 * whenever there is at least one row — the buttons disable on a single page so
 * the control stays visible rather than vanishing, which reads as missing.
 */
export function PaginationBar({
  page,
  pageCount,
  total,
  pageSize,
  onPrev,
  onNext,
  className,
}: {
  page: number;
  pageCount: number;
  total: number;
  pageSize: number;
  onPrev: () => void;
  onNext: () => void;
  className?: string;
}) {
  if (total === 0) return null;

  const start = page * pageSize + 1;
  const end = Math.min((page + 1) * pageSize, total);

  return (
    <div className={cn("flex items-center justify-between gap-3 text-[13px] text-ink-soft", className)}>
      <span className="tnum">
        {start}–{end} of {total}
      </span>
      <div className="flex items-center gap-1">
        <Button variant="outline" size="sm" disabled={page === 0} onClick={onPrev}>
          <ChevronLeft /> Prev
        </Button>
        <span className="px-1.5 tnum tabular-nums text-ink-faint">
          {page + 1} / {pageCount}
        </span>
        <Button variant="outline" size="sm" disabled={page >= pageCount - 1} onClick={onNext}>
          Next <ChevronRight />
        </Button>
      </div>
    </div>
  );
}
