import type { Bucket, WindowCount, WindowRate } from "@/services/common/range";

/**
 * A KPI delta against the immediately preceding window of equal length.
 *
 * That baseline is the whole point: the deleted metrics fixture rendered a bare
 * "+2" with no period and no comparison, so the number was undefined rather
 * than merely wrong. Every figure the API returns is now paired with the same
 * figure over the previous window, and this turns the pair into a label.
 */
export interface Delta {
  label: string;
  direction: "up" | "down";
}

export type DeltaMode = "count" | "percent";

/**
 * Counts render as absolute (`+2`), rates and durations as percent (`+12%`).
 *
 * A zero baseline has no meaningful percentage - "up from nothing" is infinite,
 * not 100% - so percent mode returns nothing rather than inventing a figure.
 */
export function deltaOf(
  current: number,
  previous: number,
  mode: DeltaMode = "count",
): Delta | undefined {
  const diff = current - previous;
  if (diff === 0) return undefined;

  const direction: Delta["direction"] = diff > 0 ? "up" : "down";
  const sign = diff > 0 ? "+" : "-";

  if (mode === "count") {
    return { label: `${sign}${Math.abs(Math.round(diff))}`, direction };
  }

  if (previous === 0) return undefined;
  const pct = Math.round((diff / Math.abs(previous)) * 100);
  if (pct === 0) return undefined;
  return { label: `${sign}${Math.abs(pct)}%`, direction };
}

/** Convenience wrappers so dashboards read as one line per tile. */
export const countDelta = (w?: WindowCount) => (w ? deltaOf(w.current, w.previous, "count") : undefined);
export const percentDelta = (w?: WindowCount | WindowRate) =>
  w ? deltaOf(w.current, w.previous, "percent") : undefined;

const BUCKET_FORMAT: Record<Bucket, Intl.DateTimeFormatOptions> = {
  day: { month: "short", day: "numeric" },
  week: { month: "short", day: "numeric" },
  month: { month: "short", year: "numeric" },
  year: { year: "numeric" },
};

/**
 * Axis label for one series point. The bucket is a request parameter, so the
 * same series can be days or years and the format has to follow it.
 */
export function bucketLabel(iso: string, bucket: Bucket): string {
  const d = new Date(iso);
  const formatted = d.toLocaleDateString(undefined, BUCKET_FORMAT[bucket]);
  return bucket === "week" ? `w/c ${formatted}` : formatted;
}
