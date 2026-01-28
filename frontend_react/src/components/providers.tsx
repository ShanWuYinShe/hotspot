'use client';

import * as React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/toaster';
import { useAuthStore, getAccessToken } from '@/stores/auth';
import { ENDPOINTS } from '@/lib/api/endpoints';

// 创建 QueryClient 实例
function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // 数据在 60 秒内被视为新鲜
        staleTime: 60 * 1000,
        // 组件卸载后缓存保留 5 分钟
        gcTime: 5 * 60 * 1000,
        // 窗口聚焦时不自动刷新
        refetchOnWindowFocus: false,
        // 失败重试 1 次
        retry: 1,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined = undefined;

function getQueryClient() {
  if (typeof window === 'undefined') {
    // 服务端：总是创建新的 client
    return makeQueryClient();
  } else {
    // 浏览器：复用已有 client
    if (!browserQueryClient) browserQueryClient = makeQueryClient();
    return browserQueryClient;
  }
}

/**
 * 非阻塞式认证初始化
 * 在后台验证 token，不阻塞页面渲染
 */
function AuthProvider({ children }: { children: React.ReactNode }) {
  const { isHydrated, checkTokenValid, setUser, logout } = useAuthStore();
  const initRef = React.useRef(false);

  React.useEffect(() => {
    // 只在 hydrate 完成后初始化一次
    if (!isHydrated || initRef.current) return;
    initRef.current = true;

    const initAuth = async () => {
      // 检查是否有有效的 token
      if (!checkTokenValid()) {
        // 没有有效 token，清理状态
        logout();
        return;
      }

      // 有 token，后台获取用户信息
      try {
        const token = getAccessToken();
        if (!token) return;

        const response = await fetch(
          `${process.env.NEXT_PUBLIC_API_BASE_URL || ''}${ENDPOINTS.USER.ME.PROFILE}`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({}),
          }
        );

        const result = await response.json();
        
        if (result.code === 0 && result.data) {
          setUser(result.data);
        } else {
          // Token 无效，清理状态
          logout();
        }
      } catch {
        // 网络错误，保持当前状态（可能是离线）
        console.warn('Failed to verify auth, keeping current state');
      }
    };

    initAuth();
  }, [isHydrated, checkTokenValid, setUser, logout]);

  return <>{children}</>;
}

interface ProvidersProps {
  children: React.ReactNode;
}

export function Providers({ children }: ProvidersProps) {
  const queryClient = getQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        {children}
      </AuthProvider>
      <Toaster />
    </QueryClientProvider>
  );
}
