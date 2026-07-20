import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";
import settings from "@/settings";

/**
 * Tokens never pass through JavaScript. The API sets both as httpOnly cookies,
 * which the browser attaches to every same-origin request on its own — so
 * there is nothing here to store, read, or forward.
 *
 * withCredentials is what makes that work for cross-origin dev setups; behind
 * nginx everything is same-origin anyway.
 */
const apiClient = axios.create({
  withCredentials: true,
});

const AUTH_BASE = settings.BASE_URL.AUTH_SERVICE_API + "/api/auth/v1";
const REFRESH_URL = `${AUTH_BASE}/refresh`;

// A 401 from these is final: bad credentials, or the refresh itself failed.
// Retrying them would loop.
const AUTH_ENDPOINTS = [`${AUTH_BASE}/login`, REFRESH_URL, `${AUTH_BASE}/logout`];

type RetriableRequest = InternalAxiosRequestConfig & { _retried?: boolean };

/**
 * Only one refresh runs at a time, process-wide.
 *
 * Refreshing rotates the refresh token, so concurrent calls leave the losers
 * presenting a token the server has already retired — which the backend
 * correctly reads as a replayed token. Sharing one promise turns every extra
 * caller into a waiter on the first.
 */
let refreshInFlight: Promise<void> | null = null;

export const refreshSession = (): Promise<void> => {
  if (!refreshInFlight) {
    // Plain axios, not apiClient: this must not re-enter the interceptor.
    refreshInFlight = axios
      .post(REFRESH_URL, null, { withCredentials: true })
      .then(() => undefined)
      .finally(() => {
        refreshInFlight = null;
      });
  }
  return refreshInFlight;
};

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const request = error.config as RetriableRequest | undefined;

    const canRetry =
      error.response?.status === 401 &&
      request &&
      !request._retried &&
      !AUTH_ENDPOINTS.some((url) => request.url?.startsWith(url));

    if (!canRetry) {
      return Promise.reject(error);
    }

    try {
      // On success the API has already set a fresh access cookie, so the
      // retry carries it without us touching anything.
      await refreshSession();
      request._retried = true;
      return apiClient(request);
    } catch {
      // The session is genuinely over. A full navigation rather than a router
      // push, so every cached query is dropped too.
      if (typeof window !== "undefined" && window.location.pathname !== "/login") {
        window.location.assign("/login");
      }
      return Promise.reject(error);
    }
  },
);

export default apiClient;
