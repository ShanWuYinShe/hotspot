'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useEffect, useState, type ReactNode } from 'react';

import { Toaster } from '@/components/ui/toaster';
import { LoginPromptDialog } from '@/components/common/login-prompt-dialog';
import { configureApiClient, isSuccess } from '@/lib/api/client';
import { authService } from '@/lib/api/services/auth';
import { useAuthPromptStore } from '@/stores/auth-prompt';
import { useAuthStore } from '@/stores/auth';
import { clearAccessTokenCookie, setAccessTokenCookie } from '@/lib/auth/cookies';

interface ProvidersProps {
  children: ReactNode;
}

export function Providers({ children }: ProvidersProps) {
  const openPrompt = useAuthPromptStore((state) => state.openPrompt);
  const currentAccount = useAuthStore((state) => state.currentAccount);
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60 * 1000, // 1 minute
            refetchOnWindowFocus: false,
          },
        },
      })
  );

  useEffect(() => {
    configureApiClient({
      baseUrl: '',
      getAccessToken: () => useAuthStore.getState().getAccessToken(),
      onUnauthorized: () => {
        openPrompt({
          actionName: '继续操作',
          description: '登录后即可继续使用该功能。',
        });
      },
      onRefreshToken: async () => {
        const { currentAccount, updateToken, logout } = useAuthStore.getState();
        if (!currentAccount) return null;

        try {
          const response = await authService.refresh({
            client_type: 'web',
            refresh_token: currentAccount.refresh_token,
          });
          if (isSuccess(response)) {
            updateToken(response.data.access_token, response.data.access_token_expires_at);
            return response.data.access_token;
          }
        } catch {
          // 刷新失败，交由下方处理
        }

        logout();
        clearAccessTokenCookie();
        openPrompt({
          actionName: '继续操作',
          description: '登录状态已过期，请重新登录。',
        });
        return null;
      },
    });
  }, [openPrompt]);

  useEffect(() => {
    if (currentAccount?.access_token) {
      setAccessTokenCookie(currentAccount.access_token, currentAccount.access_token_expires_at);
    } else {
      clearAccessTokenCookie();
    }
  }, [currentAccount]);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <Toaster />
      <LoginPromptDialog />
    </QueryClientProvider>
  );
}
