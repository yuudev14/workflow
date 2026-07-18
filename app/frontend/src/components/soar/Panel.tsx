import * as React from "react";
import { cn } from "@/lib/utils";

export function Panel({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "rounded-md border border-line bg-card px-4 py-3.5 shadow-sm",
        className
      )}
      {...props}
    />
  );
}

export function PanelTitle({
  className,
  children,
  aside,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { aside?: React.ReactNode }) {
  return (
    <div
      className={cn(
        "mb-3 flex items-center justify-between text-xs font-semibold text-ink-soft",
        className
      )}
      {...props}
    >
      <span>{children}</span>
      {aside && <span className="font-medium text-ink-faint tnum">{aside}</span>}
    </div>
  );
}
