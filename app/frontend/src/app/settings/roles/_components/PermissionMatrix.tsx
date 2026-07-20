"use client";

import * as React from "react";
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
            <tr key={module} className="border-t border-line">
              <td className="px-3 py-2">
                <button
                  type="button"
                  disabled={readOnly}
                  onClick={() => toggleRow(module)}
                  className="font-semibold disabled:cursor-default"
                  title={readOnly ? undefined : "Toggle the whole row"}
                >
                  {MODULE_LABELS[module]}
                </button>
              </td>
              {PERMISSION_ACTIONS.map((action) => (
                <td key={action} className="px-3 py-2 text-center">
                  <input
                    type="checkbox"
                    className="size-4 accent-primary"
                    disabled={readOnly}
                    checked={has(value, module, action)}
                    onChange={() => toggle(module, action)}
                    aria-label={`${module}.${action}`}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
