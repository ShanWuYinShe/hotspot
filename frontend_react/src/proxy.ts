import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const AUTH_FREE_PATHS = [
  '/login',
  '/register',
  '/reset-password',
  '/announcements',
  '/health',
];

const AUTH_REQUIRED_PREFIXES = [
  '/settings',
  '/admin',
];

export function proxy(request: NextRequest) {
  const refreshToken = request.cookies.get('refresh_token');
  const path = request.nextUrl.pathname;

  if (AUTH_FREE_PATHS.some((item) => path.startsWith(item))) {
    if (refreshToken && (path.startsWith('/login') || path.startsWith('/register'))) {
      return NextResponse.redirect(new URL('/feed', request.url));
    }
    return NextResponse.next();
  }

  if (AUTH_REQUIRED_PREFIXES.some((prefix) => path.startsWith(prefix))) {
    if (!refreshToken) {
      return NextResponse.redirect(new URL('/login', request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
};
