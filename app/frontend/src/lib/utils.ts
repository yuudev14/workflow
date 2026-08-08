import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import moment from "moment"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * convert date into a readable date
 * @param isoDateString 
 * @param formatString 
 * @returns 
 */
export const readableDate = (isoDateString: string, formatString: string = 'MMMM Do YYYY, h:mm:ss a') => {
  return moment(isoDateString).format(formatString)
}

/**
 * Compact elapsed time, e.g. "12m", "3h", "2d".
 *
 * Computed client-side on purpose: the API sends `created_at` and nothing else,
 * because a pre-rendered "12m ago" is wrong the moment the response is cached.
 */
export const relativeAge = (isoDateString?: string | null): string => {
  if (!isoDateString) return "-";
  const seconds = moment().diff(moment(isoDateString), "seconds");
  if (seconds < 60) return `${Math.max(seconds, 0)}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}

/** Seconds as a duration, e.g. "2h 14m". Used for the MTTR readouts. */
export const humanDuration = (seconds: number): string => {
  if (!seconds || seconds <= 0) return "-";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.round((seconds % 3600) / 60);
  return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
}

