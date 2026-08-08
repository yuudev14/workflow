"use client";

import * as React from "react";

export type FilterKind = "string" | "string[]" | "boolean" | "number";

export type FilterSpec = Record<string, FilterKind>;

type Value = string | string[] | boolean | number | undefined;
export type FilterState = Record<string, Value>;

function parse(params: URLSearchParams, spec: FilterSpec): FilterState {
  const out: FilterState = {};
  for (const [key, kind] of Object.entries(spec)) {
    if (kind === "string[]") {
      const all = params.getAll(key);
      if (all.length) out[key] = all;
      continue;
    }
    const raw = params.get(key);
    if (raw === null || raw === "") continue;
    if (kind === "boolean") out[key] = raw === "true";
    else if (kind === "number") {
      const n = Number(raw);
      if (Number.isFinite(n)) out[key] = n;
    } else out[key] = raw;
  }
  return out;
}

function serialize(state: FilterState): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(state)) {
    if (value === undefined || value === "" || value === false) continue;
    if (Array.isArray(value)) {
      for (const v of value) params.append(key, v);
    } else {
      params.set(key, String(value));
    }
  }
  return params.toString();
}

/**
 * Queue filters, mirrored into the query string.
 *
 * With ten filters, losing them on a refresh or on back-from-detail is the
 * fastest way to make the queue annoying, and a shareable URL is what lets one
 * analyst hand a view to another.
 *
 * The URL is read in a mount effect rather than through `useSearchParams`,
 * which would drag in the Suspense boundary that fails the production build -
 * the same reason `login/page.tsx` reads `window.location.search` directly.
 * `history.replaceState` rather than `router.replace` keeps the write out of
 * React's render path, so typing in a filter never remounts the tree.
 *
 * `false` and empty arrays are dropped rather than written as
 * `unassigned=false`: an absent key is already how the API reads "no filter".
 */
export function useUrlFilters(spec: FilterSpec) {
  const [state, setState] = React.useState<FilterState>({});
  // Nothing is known until the mount effect runs, so a caller mirroring a
  // filter into its own state (a debounced search box) must wait for this
  // rather than reading an empty object and writing it back.
  const [hydrated, setHydrated] = React.useState(false);
  const specRef = React.useRef(spec);

  React.useEffect(() => {
    setState(parse(new URLSearchParams(window.location.search), specRef.current));
    setHydrated(true);
  }, []);

  const setFilters = React.useCallback(
    (next: FilterState | ((prev: FilterState) => FilterState)) => {
      setState((prev) => {
        const resolved = typeof next === "function" ? next(prev) : next;
        const qs = serialize(resolved);
        window.history.replaceState(
          null,
          "",
          qs ? `${window.location.pathname}?${qs}` : window.location.pathname,
        );
        return resolved;
      });
    },
    [],
  );

  const setValue = React.useCallback(
    (key: string, value: Value) => setFilters((prev) => ({ ...prev, [key]: value })),
    [setFilters],
  );

  const clearAll = React.useCallback(() => setFilters({}), [setFilters]);

  return { filters: state, hydrated, setFilters, setValue, clearAll };
}
