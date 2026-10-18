import settings from "@/settings";
import apiClient, { refreshSession } from "../common/client";
import { AuthProviderSummary, LoginPayload, Me, SessionResponse } from "./auth.schema";

export default class AuthService {
  private static BASE_URL = settings.BASE_URL.AUTH_SERVICE_API + "/api/v1/auth";

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

  public static logout = async (): Promise<void> => {
    await apiClient.post(`${this.BASE_URL}/logout`);
  };

  public static me = async (): Promise<Me> => {
    const res = await apiClient.get(`${this.BASE_URL}/me`);
    return res.data;
  };

  /** Always 204, so the reply does not show if the email has an account. */
  public static forgotPassword = async (email: string): Promise<void> => {
    await apiClient.post(`${this.BASE_URL}/password/forgot`, { email });
  };

  /** Takes the token of an invite link or a reset link. */
  public static resetPassword = async (token: string, password: string): Promise<void> => {
    await apiClient.post(`${this.BASE_URL}/password/reset`, { token, password });
  };

  public static getProviders = async (): Promise<AuthProviderSummary[]> => {
    const res = await apiClient.get(`${this.BASE_URL}/providers`);
    return res.data;
  };
}
