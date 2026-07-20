"use client";

import * as React from "react";
import { AlertTriangle, Bell, LayoutDashboard, Layers, LayoutGrid } from "lucide-react";
import Link from "next/link";

import { NavMain, type NavSection } from "@/components/sidebar/nav-main";
import { NavUser } from "@/components/sidebar/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import ModeToggle from "./toggle-dark-theme";
import { useAuth } from "@/components/provider/auth-provider";

const sections: NavSection[] = [
  {
    label: "Platform",
    items: [
      { title: "Dashboard", url: "/dashboard", icon: LayoutDashboard },
      { title: "Playbooks", url: "/playbooks", icon: Layers },
      { title: "Connectors", url: "/connectors", icon: LayoutGrid },
    ],
  },
  {
    label: "Monitor",
    items: [
      { title: "Alerts", url: "/alerts", icon: Bell },
      { title: "Incidents", url: "/incidents", icon: AlertTriangle },
    ],
  },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { user } = useAuth();

  const navUser = {
    name: user?.username ?? "",
    email: user?.email ?? "",
    avatar: "",
  };

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild size="lg" tooltip="YTSoar">
              <Link href="/playbooks">
                <span className="flex aspect-square size-8 items-center justify-center rounded-md bg-foreground text-[14px] font-bold text-background">
                  Y
                </span>
                <div className="grid flex-1 text-left leading-tight">
                  <span className="truncate text-[16px] font-semibold">YTSoar</span>
                  <span className="truncate text-[12px] text-ink-faint">SOAR platform</span>
                </div>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain sections={sections} />
      </SidebarContent>
      <SidebarFooter>
        <ModeToggle />
        <NavUser user={navUser} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
