"use client";

import * as React from "react";
import { Check } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  ACTION_LABELS,
  MODULE_LABELS,
  PERMISSION_ACTIONS,
  PERMISSION_MODULES,
} from "@/settings/permissions";

export type Matrix = Record<string, string[]>;

const has = (matrix: Matrix, module: string, action: string) =>
  matrix[module]?.includes(action) ?? false;

/**
 * The matrix is the whole grant set for a role — it is sent wholesale on save,
 * never patched, so what is on screen is exactly what the role will hold.
 */
export function PermissionMatrix({
  value,
  onChange,
  readOnly,
}: {
  value: Matrix;
  onChange: (next: Matrix) => void;
  readOnly?: boolean;
}) {
  const toggle = (module: string, action: string) => {
    const current = value[module] ?? [];
    const next = current.includes(action)
      ? current.filter((a) => a !== action)
      : [...current, action];

    const updated = { ...value };
    if (next.length === 0) {
      delete updated[module];
    } else {
      updated[module] = next;
    }
    onChange(updated);
  };

  const toggleRow = (module: string) => {
    const full = (value[module] ?? []).length === PERMISSION_ACTIONS.length;
    const updated = { ...value };
    if (full) {
      delete updated[module];
    } else {
      updated[module] = [...PERMISSION_ACTIONS];
    }
    onChange(updated);
  };

  return (
    <div className="overflow-x-auto rounded-md border border-line">
      <table className="w-full text-[13px]">
        <thead className="bg-paper-sunken text-[12px] uppercase tracking-wide text-ink-soft">
          <tr>
            <th className="px-3 py-2 text-left font-semibold">Module</th>
            {PERMISSION_ACTIONS.map((action) => (
              <th key={action} className="px-3 py-2 text-center font-semibold">
                {ACTION_LABELS[action]}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {PERMISSION_MODULES.map((module) => (
            <tr key={module} className="border-t border-line hover:bg-paper-sunken/60">
              <td className="px-3 py-2">
                <button
                  type="button"
                  disabled={readOnly}
                  onClick={() => toggleRow(module)}
                  className="font-semibold disabled:cursor-not-allowed"
                  title={readOnly ? "Built-in roles can't be edited" : "Toggle the whole row"}
                >
                  {MODULE_LABELS[module]}
                </button>
              </td>
              {PERMISSION_ACTIONS.map((action) => {
                const checked = has(value, module, action);
                return (
                  <td key={action} className="p-0 text-center">
                    <button
                      type="button"
                      disabled={readOnly}
                      onClick={() => toggle(module, action)}
                      aria-pressed={checked}
                      aria-label={`${module}.${action}`}
                      title={readOnly ? "Built-in roles can't be edited" : undefined}
                      className="flex h-10 w-full items-center justify-center disabled:cursor-not-allowed"
                    >
                      <span
                        className={cn(
                          "flex size-4 items-center justify-center rounded-sm border transition-colors",
                          checked
                            ? "border-signal-dot bg-signal-dot"
                            : "border-line-strong bg-transparent"
                        )}
                      >
                        {checked && <Check className="size-3 text-signal-on" strokeWidth={3} />}
                      </span>
                    </button>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
