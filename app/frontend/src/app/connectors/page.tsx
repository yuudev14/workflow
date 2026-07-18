"use client";

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { Upload } from "lucide-react";

import ConnectorService from "@/services/connectors/connectors";
import { ConnectorInfo } from "@/services/connectors/connectors.schema";
import { Skeleton } from "@/components/ui/skeleton";
import { Glyph, connectorGlyph } from "@/components/soar";
import { cn } from "@/lib/utils";

// info.json carries a `runtime` (and often a version); the shared type keeps
// unknown fields as a passthrough map, so read them defensively here.
type MarketConnector = ConnectorInfo & { runtime?: string; version?: string };

function runtimeBadge(runtime?: string) {
  const r = (runtime ?? "python").toLowerCase();
  if (r === "go") return { label: "go", cls: "bg-slate-soft text-slate-text" };
  if (r === "node" || r === "js" || r === "ts")
    return { label: "node", cls: "bg-moss-soft text-moss-text" };
  return { label: "python", cls: "bg-signal-soft text-signal-text" };
}

function Toggle({ on, onToggle }: { on: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={cn(
        "relative h-5 w-[34px] shrink-0 rounded-full transition-colors",
        on ? "bg-moss-dot" : "bg-line-strong"
      )}
    >
      <span
        className={cn(
          "absolute top-0.5 size-4 rounded-full bg-white transition-all",
          on ? "left-4" : "left-0.5"
        )}
      />
    </button>
  );
}

function ConnectorCard({ connector }: { connector: MarketConnector }) {
  // Local-only enable state — persistence is Phase 2/3 (endpoint not wired yet).
  const [enabled, setEnabled] = React.useState(true);
  const g = connectorGlyph(connector.name);
  const badge = runtimeBadge(connector.runtime);
  return (
    <div className="flex flex-col gap-2.5 rounded-md border border-line bg-card p-3.5">
      <div className="flex items-start justify-between gap-2">
        <div className="flex gap-2.5">
          <Glyph icon={g.icon} tone={g.tone} size="lg" />
          <div>
            <div className="text-[14px] font-semibold capitalize">{connector.name}</div>
            <div className="font-mono text-[12px] text-ink-faint">
              {connector.version ?? "installed"}
            </div>
          </div>
        </div>
        <Toggle on={enabled} onToggle={() => setEnabled((v) => !v)} />
      </div>
      <div className="flex items-center gap-2 text-[12px] text-ink-faint">
        <span
          className={cn(
            "rounded-[5px] px-1.5 py-0.5 text-[11px] font-bold uppercase tracking-wide",
            badge.cls
          )}
        >
          {badge.label}
        </span>
        {connector.operations?.length ?? 0} operations
      </div>
    </div>
  );
}

export default function Page() {
  const connectorsQuery = useQuery({
    queryKey: ["connectors-marketplace"],
    queryFn: () => ConnectorService.getConnectors(),
  });

  const connectors = (connectorsQuery.data ?? []) as MarketConnector[];

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-6 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Connectors</h1>
            <p className="mt-1 text-[15px] text-ink-soft">
              {connectors.length} installed · browse, enable, and upload connector packages.
            </p>
          </div>
          <button className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110">
            <Upload className="size-4" /> Install connector
          </button>
        </div>

        <label className="cursor-pointer rounded-md border-[1.5px] border-dashed border-line-strong bg-paper-sunken px-6 py-7 text-center text-ink-faint hover:border-signal-dot">
          <input type="file" accept=".zip" className="hidden" />
          <Upload className="mx-auto size-6" />
          <div className="mt-1.5 text-[14px] font-semibold text-ink-soft">
            Drop a connector .zip, or browse
          </div>
          <div className="mt-0.5 text-[12.5px]">
            Must include info.json + a python/node entry file — max 25 MB
          </div>
        </label>

        {connectorsQuery.isLoading ? (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-[104px] rounded-md" />
            ))}
          </div>
        ) : connectors.length === 0 ? (
          <div className="rounded-md border border-dashed border-line-strong px-6 py-10 text-center text-ink-faint">
            No connectors installed yet.
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {connectors.map((c) => (
              <ConnectorCard key={c.id} connector={c} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
