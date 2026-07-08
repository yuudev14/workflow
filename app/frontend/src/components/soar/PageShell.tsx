import * as React from "react";
import { cn } from "@/lib/utils";

/** Standard wide page container + optional header. One place to tune width. */
export function PageShell({
  title,
  subtitle,
  actions,
  header,
  children,
  className,
}: {
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  actions?: React.ReactNode;
  /** fully custom header (overrides title/subtitle/actions) */
  header?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div className="flex w-full justify-center">
      <div className={cn("flex w-full flex-col gap-5 px-6 py-8", className)}>
        {header
          ? header
          : (title || actions) && (
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  {title && <h1>{title}</h1>}
                  {subtitle && <p className="mt-1 text-[15px] text-ink-soft">{subtitle}</p>}
                </div>
                {actions}
              </div>
            )}
        {children}
      </div>
    </div>
  );
}
