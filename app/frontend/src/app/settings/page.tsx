"use client";

import Link from "next/link";

import { Glyph, PageShell } from "@/components/soar";
import { SETTINGS_SECTIONS } from "./_components/sections";

export default function SettingsIndexPage() {
  return (
    <PageShell
      title="Settings"
      subtitle="Administer who can sign in, what they can do, and how they get here."
    >
      <div className="grid gap-3 sm:grid-cols-2">
        {SETTINGS_SECTIONS.map((section) => (
          <Link
            key={section.url}
            href={section.url}
            className="flex items-start gap-3 rounded-lg border border-line bg-card p-4 transition-colors hover:border-line-strong"
          >
            <Glyph icon={section.icon} tone="signal" />
            <div className="min-w-0">
              <div className="font-semibold">{section.title}</div>
              <p className="mt-0.5 text-[12.5px] text-ink-faint">{section.description}</p>
            </div>
          </Link>
        ))}
      </div>
    </PageShell>
  );
}
