"use client";

import * as React from "react";

import { Label } from "@/components/ui/label";
import { SearchInput } from "@/components/soar";

/** Structural - satisfied by both `Role` and the lighter `RoleRef`. */
type PickableRole = { id: string; name: string };

/**
 * Searchable role multi-select, shared by the user and team dialogs so both
 * behave the same way. Filtering is client-side because `listRoles` returns the
 * whole (small, unpaginated) set in one response.
 */
export function RolePicker({
  roles,
  selected,
  onToggle,
  label = "Roles",
  hint,
  emptyMessage = "No roles yet",
}: {
  roles: PickableRole[];
  selected: string[];
  onToggle: (id: string) => void;
  label?: string;
  hint?: string;
  emptyMessage?: string;
}) {
  const [query, setQuery] = React.useState("");

  const visible = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return roles;
    // A selected role stays listed even when it doesn't match, so a search can
    // never hide a grant you already made - or strand it as undeselectable.
    return roles.filter((r) => r.name.toLowerCase().includes(q) || selected.includes(r.id));
  }, [roles, query, selected]);

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline justify-between">
        <Label>{label}</Label>
        <span className="text-[12px] text-ink-faint tnum">{selected.length} selected</span>
      </div>

      {roles.length === 0 ? (
        <span className="text-xs text-ink-faint">{emptyMessage}</span>
      ) : (
        <>
          <SearchInput
            placeholder="Search roles…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <div className="flex max-h-40 flex-wrap gap-1.5 overflow-y-auto">
            {visible.map((role) => (
              <button
                key={role.id}
                type="button"
                onClick={() => onToggle(role.id)}
                aria-pressed={selected.includes(role.id)}
                className={
                  selected.includes(role.id)
                    ? "rounded-sm border border-signal-dot bg-paper-sunken px-2 py-1 text-[12.5px] font-semibold"
                    : "rounded-sm border border-line-strong px-2 py-1 text-[12.5px] text-ink-soft hover:bg-paper-sunken"
                }
              >
                {role.name}
              </button>
            ))}
            {visible.length === 0 && (
              <span className="text-xs text-ink-faint">No roles match “{query}”</span>
            )}
          </div>
        </>
      )}

      {hint && <span className="text-[12px] text-ink-faint">{hint}</span>}
    </div>
  );
}
