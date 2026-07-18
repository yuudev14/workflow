"use client";

import * as React from "react";
import { Search } from "lucide-react";
import { cn } from "@/lib/utils";

export function SearchInput({
  className,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div
      className={cn(
        "flex min-w-[220px] items-center gap-2 rounded-sm border border-line-strong bg-card px-2.5 py-1.5 text-[13.5px] text-ink-faint focus-within:border-signal-dot",
        className
      )}
    >
      <Search className="size-4 shrink-0" />
      <input
        className="w-full bg-transparent text-foreground outline-none placeholder:text-ink-faint"
        {...props}
      />
    </div>
  );
}
