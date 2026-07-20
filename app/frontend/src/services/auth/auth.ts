import settings from "@/settings";
import apiClient, { refreshSession } from "../common/client";
import { LoginPayload, Me, SessionResponse } from "./auth.schema";

export default class AuthService {
  private static BASE_URL = settings.BASE_URL.AUTH_SERVICE_API + "/api/auth/v1";

  /**
   * Exchange credentials for a session. The API replies with httpOnly cookies;
   * the tokens in the body are for non-browser clients and are ignored here.
   */
  public static login = async (payload: LoginPayload): Promise<SessionResponse> => {
    const res = await apiClient.post(`${this.BASE_URL}/login`, payload);
    return res.data;
  };

  /**
   * Rotate the session cookies. Delegates to the shared single-flight helper:
   * refreshing retires the previous refresh token, so two concurrent calls
   * would leave the second one replaying a dead token.
   */
  public static refresh = async (): Promise<void> => {
    return refreshSession();
  };

  /** Revokes the session server-side and clears both cookies. */
  public static logout = async (): Promise<void> => {
    await apiClient.post(`${this.BASE_URL}/logout`);
  };

  /** Profile, role names, and the permission map that drives UI gating. */
  public static me = async (): Promise<Me> => {
    const res = await apiClient.get(`${this.BASE_URL}/me`);
    return res.data;
  };
}
