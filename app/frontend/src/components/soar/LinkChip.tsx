import * as React from "react";
import { cn } from "@/lib/utils";

/** Small bordered chip with an optional leading icon — links, tags, counts. */
export function LinkChip({
  className,
  children,
  ...props
}: React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border border-line-strong bg-card px-2.5 py-1 text-[12px] font-semibold text-ink-soft [&_svg]:size-3",
        className
      )}
      {...props}
    >
      {children}
    </span>
  );
}
