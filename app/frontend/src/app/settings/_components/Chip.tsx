import * as React from "react";
import { cn } from "@/lib/utils";

export function Chip({ className, ...props }: React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-sm border border-line-strong px-1.5 py-0.5 text-[11.5px] text-ink-soft",
        className,
      )}
      {...props}
    />
  );
}
