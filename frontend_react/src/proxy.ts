/**
 * Next.js Proxy
 * 处理认证保护和路由重定向
 */

import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// 管理后台路由前缀
const ADMIN_ROUTES = ['/admin'];

// 认证页面（已登录用户不应访问）
const AUTH_ROUTES = ['/login', '/register'];

// Cookie key
const ACCESS_TOKEN_KEY = 'access_token';

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const accessToken = request.cookies.get(ACCESS_TOKEN_KEY)?.value;

  // 检查是否为静态资源或 API 路由
  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/api') ||
    pathname.includes('.') // 静态文件
  ) {
    return NextResponse.next();
  }

  // 检查是否为认证页面
  const isAuthRoute = AUTH_ROUTES.some(
    (route) => pathname === route || pathname.startsWith(`${route}/`)
  );

  // 检查是否为管理后台路由
  const isAdminRoute = ADMIN_ROUTES.some(
    (route) => pathname === route || pathname.startsWith(`${route}/`)
  );

  // 已登录用户访问认证页面，重定向到首页
  if (isAuthRoute && accessToken) {
    return NextResponse.redirect(new URL('/feed', request.url));
  }

  // 未登录用户访问管理后台，重定向到登录页
  if (isAdminRoute && !accessToken) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // 管理后台需要额外的权限检查（在页面级别通过 API 验证）
  // 这里只做基本的 Token 检查

  return NextResponse.next();
}

export const config = {
  matcher: [
    /*
     * 匹配所有路径，排除:
     * - api (API 路由)
     * - _next/static (静态文件)
     * - _next/image (图片优化)
     * - favicon.ico (网站图标)
     */
    '/((?!api|_next/static|_next/image|favicon.ico).*)',
  ],
};
