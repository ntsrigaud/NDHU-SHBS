import { NextRequest, NextResponse } from 'next/server';

const PROTECTED_PATHS = ['/my-listings', '/cart', '/checkout', '/orders', '/messages'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isProtected = PROTECTED_PATHS.some((p) => pathname.startsWith(p));

  if (isProtected) {
    const token = request.cookies.get('jwt')?.value;
    if (!token) {
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
