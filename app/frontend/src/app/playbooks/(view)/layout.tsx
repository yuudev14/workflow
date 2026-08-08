"use client";

import React from "react";
import { usePathname, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import CreatePlaybookForm from "../_components/CreatePlaybookForm";
import PlaybookService from "@/services/playbooks/playbooks";
import { KpiCard } from "@/components/soar";
import { PermissionGate } from "@/components/auth/PermissionGate";
import { countDelta, percentDelta } from "@/lib/delta";
import { cn } from "@/lib/utils";

const tabs = [
  { label: "Playbooks", path: "/playbooks" },
  { label: "Executions", path: "/playbooks/executions" },
];

const Layout = ({ children }: Readonly<{ children: React.ReactNode }>) => {
  const router = useRouter();
  const pathname = usePathname();

  // No range picker here - this is a header above a list, not a dashboard, so
  // it takes the server's default window.
  const summaryQuery = useQuery({
    queryKey: ["playbooks-summary", {}],
    queryFn: () => PlaybookService.getPlaybooksSummary(),
  });

  const summary = summaryQuery.data;
  const playbooks = countDelta(summary?.playbooks);
  const runs = countDelta(summary?.runs);
  const failed = countDelta(summary?.failed);
  const successRate = percentDelta(summary?.success_rate);

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-6 px-6 py-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1>Playbooks</h1>
            <p className="mt-1 text-[15px] text-ink-soft">
              Every playbook you can run, with the status of its last execution.
            </p>
          </div>
          <PermissionGate module="playbooks" action="create">
            <CreatePlaybookForm />
          </PermissionGate>
        </div>

        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <KpiCard
            label="Playbooks"
            value={summary?.playbooks.current ?? "-"}
            delta={playbooks?.label}
            deltaDirection={playbooks?.direction}
          />
          <KpiCard
            label="Runs"
            value={summary?.runs.current ?? "-"}
            delta={runs?.label}
            deltaDirection={runs?.direction}
          />
          <KpiCard
            label="Failed runs"
            value={summary?.failed.current ?? "-"}
            delta={failed?.label}
            deltaDirection={failed?.direction}
            deltaNegative
          />
          <KpiCard
            label="Success rate"
            value={summary ? `${Math.round(summary.success_rate.current * 100)}%` : "-"}
            delta={successRate?.label}
            deltaDirection={successRate?.direction}
          />
        </div>

        <div className="flex w-fit gap-1 rounded-sm border border-line bg-paper-sunken p-[3px]">
          {tabs.map((t) => {
            const active = pathname === t.path;
            return (
              <button
                key={t.path}
                onClick={() => router.push(t.path)}
                className={cn(
                  "rounded-[6px] px-3 py-1.5 text-[13.5px] font-semibold transition-colors",
                  active
                    ? "bg-card text-foreground shadow-sm"
                    : "text-ink-soft hover:text-foreground"
                )}
              >
                {t.label}
              </button>
            );
          })}
        </div>

        <div className="flex-1">{children}</div>
      </div>
    </div>
  );
};

export default Layout;
