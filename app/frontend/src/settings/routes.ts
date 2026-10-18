
export const PUBLIC_ROUTES = ["/login", "/forgot-password", "/set-password"];

export const isPublicRoute = (pathname: string | null): boolean =>
  !!pathname && PUBLIC_ROUTES.includes(pathname);