// Shape of a single KPI tile. Mirrors the eventual aggregate-endpoint payload
// so Phase 3 only swaps the loader body, not these types or the UI.
export interface KpiMetric {
  key: string;
  label: string;
  value: string;
  delta?: string;
  deltaDirection?: "up" | "down";
  /** an "up" delta is bad (e.g. failures rising) */
  deltaNegative?: boolean;
  spark: number[];
  sparkTone?: "signal" | "rose" | "moss" | "amber";
}
