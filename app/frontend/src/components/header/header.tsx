"use client";

import React from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";

import { SidebarTrigger } from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

// Human labels for the known top-level segments; anything else (an id) is
// shown truncated as-is.
const LABELS: Record<string, string> = {
  playbooks: "Playbooks",
  connectors: "Connectors",
  alerts: "Alerts",
  incidents: "Incidents",
  executions: "Executions",
  history: "History",
  dashboard: "Dashboard",
};

function label(seg: string) {
  if (LABELS[seg]) return LABELS[seg];
  if (seg.length > 12) return seg.slice(0, 8) + "…";
  return seg.charAt(0).toUpperCase() + seg.slice(1);
}

const Header = () => {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  const crumbs = segments.map((seg, i) => ({
    label: label(seg),
    href: "/" + segments.slice(0, i + 1).join("/"),
    last: i === segments.length - 1,
  }));

  return (
    <header className="fixed z-10 flex h-16 w-full shrink-0 items-center gap-2 border-b border-line bg-background transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-12">
      <div className="flex items-center gap-2 px-4">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mr-2 h-4" />
        <Breadcrumb>
          <BreadcrumbList>
            {crumbs.length === 0 && (
              <BreadcrumbItem>
                <BreadcrumbPage>YTSoar</BreadcrumbPage>
              </BreadcrumbItem>
            )}
            {crumbs.map((c) => (
              <React.Fragment key={c.href}>
                <BreadcrumbItem>
                  {c.last ? (
                    <BreadcrumbPage>{c.label}</BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink asChild>
                      <Link href={c.href}>{c.label}</Link>
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
                {!c.last && <BreadcrumbSeparator />}
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </Breadcrumb>
      </div>
    </header>
  );
};

export default Header;
