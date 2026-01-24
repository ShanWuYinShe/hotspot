'use client';

/**
 * 登录拦截 Hook
 * 对需要登录的操作进行细粒度提示
 */

import { useMemo } from 'react';
import { usePathname, useSearchParams } from 'next/navigation';
import { useAuthStore } from '@/stores/auth';
import { useAuthPromptStore } from '@/stores/auth-prompt';

interface RequireAuthOptions {
  actionName?: string;
  description?: string;
  redirectPath?: string;
}

export function useRequireAuth() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { openPrompt } = useAuthPromptStore();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const currentPath = useMemo(() => {
    const query = searchParams?.toString();
    return query ? `${pathname}?${query}` : pathname;
  }, [pathname, searchParams]);

  const requireAuth = (actionName?: string, options?: RequireAuthOptions) => {
    if (isAuthenticated) return true;

    openPrompt({
      actionName: options?.actionName ?? actionName,
      description: options?.description,
      redirectPath: options?.redirectPath ?? currentPath,
    });
    return false;
  };

  return {
    isAuthenticated,
    requireAuth,
  };
}
