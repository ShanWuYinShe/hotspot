'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/stores/auth';
import { Loader2 } from 'lucide-react';

interface ProtectedRouteProps {
  children: React.ReactNode;
  /** 自定义未登录时的重定向路径 */
  redirectTo?: string;
  /** 是否显示加载状态 */
  showLoading?: boolean;
}

/**
 * 受保护路由组件
 * 用于需要认证才能访问的页面内容
 * 
 * 注意：这是一个客户端组件，用于在页面内部进行认证检查
 * 主要的路由保护应该通过 middleware.ts 处理
 */
export function ProtectedRoute({
  children,
  redirectTo = '/login',
  showLoading = true,
}: ProtectedRouteProps) {
  const router = useRouter();
  const { isAuthenticated, isHydrated } = useAuthStore();
  const [isChecking, setIsChecking] = React.useState(true);

  React.useEffect(() => {
    if (!isHydrated) return;

    if (!isAuthenticated) {
      // 保存当前路径用于登录后重定向
      const currentPath = window.location.pathname;
      const redirectUrl = `${redirectTo}?redirect=${encodeURIComponent(currentPath)}`;
      router.replace(redirectUrl);
    } else {
      setIsChecking(false);
    }
  }, [isHydrated, isAuthenticated, router, redirectTo]);

  // 等待 hydration
  if (!isHydrated) {
    if (showLoading) {
      return (
        <div className="flex min-h-[50vh] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      );
    }
    return null;
  }

  // 正在检查或未登录
  if (isChecking || !isAuthenticated) {
    if (showLoading) {
      return (
        <div className="flex min-h-[50vh] items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      );
    }
    return null;
  }

  return <>{children}</>;
}
