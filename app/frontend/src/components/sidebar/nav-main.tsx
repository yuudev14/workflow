"use client";

import { type LucideIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useAuth } from "@/components/provider/auth-provider";

export interface NavItem {
  title: string;
  url: string;
  icon: LucideIcon;
  /** Hide the item unless the user holds this grant. */
  permission?: { module: string; action: string };
}

export interface NavSection {
  label: string;
  items: NavItem[];
}

function isActive(pathname: string, url: string) {
  if (url === "/") return pathname === "/";
  return pathname === url || pathname.startsWith(url + "/");
}

export function NavMain({ sections }: { sections: NavSection[] }) {
  const pathname = usePathname();
  const { hasPermission } = useAuth();

  const visible = sections
    .map((section) => ({
      ...section,
      items: section.items.filter(
        (item) => !item.permission || hasPermission(item.permission.module, item.permission.action),
      ),
    }))
    .filter((section) => section.items.length > 0);

  return (
    <>
      {visible.map((section) => (
        <SidebarGroup key={section.label}>
          <SidebarGroupLabel>{section.label}</SidebarGroupLabel>
          <SidebarMenu>
            {section.items.map((item) => (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton
                  asChild
                  tooltip={item.title}
                  isActive={isActive(pathname, item.url)}
                >
                  <Link href={item.url}>
                    <item.icon />
                    <span>{item.title}</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      ))}
    </>
  );
}
