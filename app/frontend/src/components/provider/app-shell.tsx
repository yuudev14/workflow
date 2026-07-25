"use client";

import React from "react";
import { usePathname } from "next/navigation";

import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/sidebar/app-sidebar";
import Header from "@/components/header/header";
import { isPublicRoute } from "@/settings/routes";
import PlaybookStatusProvider from "./playbook-status-provider";


const AppShell: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const pathname = usePathname();

  if (isPublicRoute(pathname)) {
    return <>{children}</>;
  }


  return (
    <PlaybookStatusProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset>
          <Header />
          <div className="flex-1">{children}</div>
        </SidebarInset>
      </SidebarProvider>
    </PlaybookStatusProvider>
  );
};

export default AppShell;