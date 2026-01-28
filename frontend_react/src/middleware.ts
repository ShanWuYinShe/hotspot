import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// 受保护的路由路径
const protectedPaths = [
  '/feed',
  '/chat',
  '/notifications',
  '/profile',
  '/settings',
];

// 管理员路由路径
const adminPaths = [
  '/admin',
];

// 认证页面路径（已登录用户不应访问）
const authPaths = [
  '/login',
  '/register',
];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  
  // 获取 token（从 localStorage 无法在 middleware 中访问，所以使用 cookie）
  const accessToken = request.cookies.get('hotspot_access_token')?.value;
  const isAuthenticated = !!accessToken;

  // 检查是否是受保护路由
  const isProtectedPath = protectedPaths.some(path => 
    pathname === path || pathname.startsWith(`${path}/`)
  );

  // 检查是否是管理员路由
  const isAdminPath = adminPaths.some(path => 
    pathname === path || pathname.startsWith(`${path}/`)
  );

  // 检查是否是认证页面
  const isAuthPath = authPaths.some(path => 
    pathname === path || pathname.startsWith(`${path}/`)
  );

  // 受保护路由：未登录则重定向到登录页
  if (isProtectedPath && !isAuthenticated) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // 管理员路由：未登录则重定向
  if (isAdminPath && !isAuthenticated) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  // 认证页面：已登录则重定向到首页
  if (isAuthPath && isAuthenticated) {
    return NextResponse.redirect(new URL('/', request.url));
  }

  return NextResponse.next();
}

export const config = {
  // 匹配所有路径，排除静态资源和 API
  matcher: [
    '/((?!api|_next/static|_next/image|favicon.ico|.*\\..*|__).*)',
  ],
};
