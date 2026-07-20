import { NextRequest, NextResponse } from "next/server";

/**
 * Presence check only — this is NOT authentication.
 *
 * The refresh token is signed with a secret that only the Go API holds, so
 * Next has no way to validate it. All this does is skip a flash of the app
 * shell for visitors who obviously have no session. AuthProvider does the real
 * work: it trades the cookie for an access token and redirects if that fails.
 *
 * A forged cookie gets past this and then fails at /refresh, which is fine —
 * the redirect happens either way, just a beat later.
 *
 */
const REFRESH_COOKIE = "ytsoar_rt";

export function middleware(request: NextRequest) {
  if (request.cookies.has(REFRESH_COOKIE)) {
    return NextResponse.next();
  }

  const loginUrl = new URL("/login", request.url);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  // Everything except the login page, Next internals, and static assets.
  matcher: ["/((?!login|_next/static|_next/image|favicon.ico|fonts|.*\\.svg$).*)"],
};