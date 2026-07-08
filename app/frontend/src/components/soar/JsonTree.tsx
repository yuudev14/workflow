import * as React from "react";
import { cn } from "@/lib/utils";

// Syntax-colored, read-only JSON view. Replaces raw JSON.stringify <pre> dumps.
function renderValue(value: unknown, indent: number): React.ReactNode {
  const pad = "  ".repeat(indent);
  const padClose = "  ".repeat(indent - 1);

  if (value === null) return <span className="text-ink-faint">null</span>;
  if (typeof value === "boolean" || typeof value === "number")
    return <span className="text-amber-text">{String(value)}</span>;
  if (typeof value === "string")
    return <span className="text-moss-text">&quot;{value}&quot;</span>;

  if (Array.isArray(value)) {
    if (value.length === 0) return <span>[]</span>;
    return (
      <>
        <span>[</span>
        {value.map((v, i) => (
          <div key={i}>
            {pad}
            {renderValue(v, indent + 1)}
            {i < value.length - 1 ? "," : ""}
          </div>
        ))}
        <div>{padClose}]</div>
      </>
    );
  }

  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    if (entries.length === 0) return <span>{"{}"}</span>;
    return (
      <>
        <span>{"{"}</span>
        {entries.map(([k, v], i) => (
          <div key={k}>
            {pad}
            <span className="text-signal-text">&quot;{k}&quot;</span>: {renderValue(v, indent + 1)}
            {i < entries.length - 1 ? "," : ""}
          </div>
        ))}
        <div>{padClose}{"}"}</div>
      </>
    );
  }
  return <span>{String(value)}</span>;
}

export function JsonTree({
  data,
  className,
}: {
  data: unknown;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "rounded-sm bg-paper-sunken p-2.5 font-mono text-[12.5px] leading-[1.7]",
        className
      )}
    >
      {renderValue(data, 1)}
    </div>
  );
}
