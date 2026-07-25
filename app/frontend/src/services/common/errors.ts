import { AxiosError } from "axios";

/**
 * The api reports failures as `{"error": <string|object>}`. Fall back to the
 * axios message so a network failure still says something useful.
 */
export function apiErrorMessage(error: unknown): string {
  const detail = (error as AxiosError<{ error?: unknown }>)?.response?.data?.error;
  if (typeof detail === "string" && detail) return detail;
  if (detail) return JSON.stringify(detail);
  return (error as Error)?.message ?? "Something went wrong";
}
