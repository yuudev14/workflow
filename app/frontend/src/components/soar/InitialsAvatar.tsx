import * as React from "react";
import { cn } from "@/lib/utils";

/** Initials chip. Pass empty/undefined name for an unassigned (dashed) slot. */
export function InitialsAvatar({
  name,
  className,
  size = 24,
}: {
  name?: string | null;
  className?: string;
  size?: number;
}) {
  const initials = name
    ? name
        .replace(/[@._-]/g, " ")
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
        .map((w) => w[0]?.toUpperCase())
        .join("")
    : "";
  const unassigned = !initials;
  return (
    <span
      className={cn(
        "flex shrink-0 items-center justify-center rounded-full border text-[11px] font-bold",
        unassigned
          ? "border-dashed border-line-strong bg-transparent text-ink-faint"
          : "border-line-strong bg-paper-sunken text-ink-soft",
        className
      )}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.42) }}
      title={name ?? "Unassigned"}
    >
      {unassigned ? "·" : initials}
    </span>
  );
}
