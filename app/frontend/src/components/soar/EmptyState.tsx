import * as React from "react";
import { Inbox, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export function EmptyState({
  icon: Icon = Inbox,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon;
  title: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "mt-2.5 rounded-md border border-dashed border-line-strong px-5 py-9 text-center text-ink-faint",
        className
      )}
    >
      <Icon className="mx-auto mb-2 size-6" />
      <div className="text-[14px] font-semibold text-ink-soft">{title}</div>
      {description && <div className="mt-0.5 text-xs">{description}</div>}
      {action && <div className="mt-3 flex justify-center">{action}</div>}
    </div>
  );
}
