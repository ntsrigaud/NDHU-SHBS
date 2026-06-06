import { NextRequest, NextResponse } from 'next/server';

const PROTECTED_PATHS = ['/my-listings', '/cart', '/checkout', '/orders', '/messages'];

/**
 * Route-protection middleware.
 *
 * Architecture note: The Go API is hosted on a different origin from the
 * Next.js frontend. The httpOnly `jwt` cookie set by the API server is
 * scoped to the API domain and is therefore NOT visible to this Edge
 * middleware (which runs on the Next.js / Vercel domain).
 *
 * Instead, after a successful login the client JS writes a lightweight,
 * non-httpOnly `session=1` cookie on the frontend domain (same origin as
 * this middleware). This cookie is used purely as a presence hint so the
 * middleware can redirect unauthenticated users without a round-trip.
 *
 * The actual API authorisation still relies on the JWT sent as an
 * Authorization: Bearer header (stored in localStorage, applied by the
 * api.ts singleton). The `session` cookie is NOT the auth credential.
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isProtected = PROTECTED_PATHS.some((p) => pathname.startsWith(p));

  if (isProtected) {
    const hasSession = request.cookies.has('session');
    if (!hasSession) {
      const loginUrl = new URL('/login', request.url);
      loginUrl.searchParams.set('next', pathname);
      return NextResponse.redirect(loginUrl);
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/my-listings/:path*',
    '/cart/:path*',
    '/checkout/:path*',
    '/orders/:path*',
    '/messages/:path*',
  ],
};
