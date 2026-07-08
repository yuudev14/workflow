"use client";

import React from "react";
import { usePathname, useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";

import CreatePlaybookForm from "../_components/CreatePlaybookForm";
import MetricsService from "@/services/metrics/metrics";
import { KpiRow } from "@/components/soar";
import { cn } from "@/lib/utils";

const tabs = [
  { label: "Playbooks", path: "/playbooks" },
  { label: "Executions", path: "/playbooks/executions" },
];

const Layout = ({ children }: Readonly<{ children: React.ReactNode }>) => {
  const router = useRouter();
  const pathname = usePathname();

  const kpiQuery = useQuery({
    queryKey: ["playbook-kpis"],
    queryFn: () => MetricsService.getPlaybookKpis(),
  });

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
          <CreatePlaybookForm />
        </div>

        <KpiRow metrics={kpiQuery.data} loading={kpiQuery.isLoading} />

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
