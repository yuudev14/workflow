// Dashboard range params, shared by every /summary endpoint.

export type Bucket = "day" | "week" | "month" | "year";

/**
 * Request bounds. `from`/`to` are RFC3339 instants with an offset, not dates:
 * a date alone cannot say whose midnight it means, so a GMT+9 viewer's daily
 * bucket would silently span 15:00-15:00 UTC. `new Date(v).toISOString()`
 * produces the right thing.
 */
export interface DateRangeParams {
  from?: string;
  to?: string;
  bucket?: Bucket;
}

/** What the server actually used, echoed back so a chart can label itself. */
export interface ResolvedRange {
  from: string;
  to: string;
  bucket: Bucket;
}

/** A figure paired with the same figure over the immediately preceding window. */
export interface WindowCount {
  current: number;
  previous: number;
}

export interface WindowRate {
  current: number;
  previous: number;
}
