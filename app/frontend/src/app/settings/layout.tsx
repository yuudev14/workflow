"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ArrowLeft, Lock } from "lucide-react";

import { usePermission } from "@/hooks/usePermission";
import { EmptyState } from "@/components/soar";
import { SETTINGS_SECTIONS } from "./_components/sections";

/**
 * Shell for the whole settings area. It carries its own section rail so the
 * main sidebar does not have to list five admin pages that most users can't
 * open anyway.
 */
export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const canRead = usePermission("settings", "read");

  if (!canRead) {
    return (
      <EmptyState
        icon={Lock}
        title="You don't have access to settings"
        description="Administering users, roles and providers needs the settings permission. Ask an administrator if you need it."
      />
    );
  }

  return (
    <div className="flex w-full flex-col lg:flex-row">
      <div className="border-b border-line lg:w-60 lg:shrink-0 lg:border-r lg:border-b-0">
        <nav
          aria-label="Settings sections"
          className="px-6 py-6 lg:sticky lg:top-16 lg:max-h-[calc(100svh-4rem)] lg:overflow-y-auto lg:py-8 group-has-data-[collapsible=icon]/sidebar-wrapper:lg:top-12"
        >
          <Link
            href="/dashboard"
            className="mb-4 inline-flex items-center gap-1.5 text-[12.5px] text-ink-faint hover:text-ink"
          >
            <ArrowLeft className="size-3.5" />
            Back to app
          </Link>

          <div className="-mx-2.5 flex gap-1 overflow-x-auto lg:flex-col lg:overflow-x-visible">
            {SETTINGS_SECTIONS.map((section) => {
              const active = pathname === section.url || pathname.startsWith(section.url + "/");
              return (
                <Link
                  key={section.url}
                  href={section.url}
                  aria-current={active ? "page" : undefined}
                  className={
                    "flex shrink-0 items-center gap-2.5 rounded-md px-2.5 py-2 text-[13.5px] transition-colors " +
                    (active
                      ? "bg-paper-sunken font-semibold text-ink"
                      : "text-ink-soft hover:bg-paper-sunken hover:text-ink")
                  }
                >
                  <section.icon className="size-4 shrink-0" />
                  {section.title}
                </Link>
              );
            })}
          </div>
        </nav>
      </div>

      <div className="min-w-0 flex-1">{children}</div>
    </div>
  );
}
