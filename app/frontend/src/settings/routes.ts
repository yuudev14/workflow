
export const PUBLIC_ROUTES = ["/login"];

export const isPublicRoute = (pathname: string | null): boolean =>
  !!pathname && PUBLIC_ROUTES.includes(pathname);