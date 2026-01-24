'use client';

import { useCallback } from 'react';
import type { Route } from 'next';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/stores/auth';
import { useFeedbackStore } from '@/stores/feedback';
import { authService } from '@/lib/api/services/auth';
import { isSuccess } from '@/lib/api/client';
import { setAccessTokenCookie, clearAccessTokenCookie } from '@/lib/auth/cookies';
import type { AuthData } from '@/types/api';

export function useAuth() {
  const router = useRouter();
  const {
    isAuthenticated,
    isLoading,
    currentAccount,
    accounts,
    login: storeLogin,
    logout: storeLogout,
    switchAccount: storeSwitchAccount,
    removeAccount: storeRemoveAccount,
    updateToken,
    getAccessToken,
    isTokenExpired,
  } = useAuthStore();
  const { success, error } = useFeedbackStore();

  const login = useCallback(
    async (loginId: string, password: string, redirectTo?: string) => {
      try {
        const response = await authService.login({
          login_id: loginId,
          password,
          client_type: 'web',
        });

        if (isSuccess(response)) {
          storeLogin(response.data);
          setAccessTokenCookie(response.data.access_token, response.data.access_token_expires_at);
          success('登录成功', `欢迎回来，${response.data.nickname}`);
          router.push((redirectTo || '/') as Route);
          return true;
        } else {
          error('登录失败', response.message ?? '请检查用户名和密码');
          return false;
        }
      } catch (err) {
        error('登录失败', err instanceof Error ? err.message : '网络错误');
        return false;
      }
    },
    [storeLogin, router, success, error]
  );

  const register = useCallback(
    async (loginId: string, nickname: string, password: string, redirectTo?: string) => {
      try {
        const response = await authService.register({
          login_id: loginId,
          nickname,
          password,
          client_type: 'web',
        });

        if (isSuccess(response)) {
          storeLogin(response.data);
          setAccessTokenCookie(response.data.access_token, response.data.access_token_expires_at);
          success('注册成功', `欢迎加入，${response.data.nickname}`);
          router.push((redirectTo || '/') as Route);
          return true;
        } else {
          error('注册失败', response.message ?? '请稍后重试');
          return false;
        }
      } catch (err) {
        error('注册失败', err instanceof Error ? err.message : '网络错误');
        return false;
      }
    },
    [storeLogin, router, success, error]
  );

  const guestLogin = useCallback(
    async (nickname: string, redirectTo?: string) => {
      try {
        const response = await authService.guest({
          nickname,
          client_type: 'web',
        });

        if (isSuccess(response)) {
          storeLogin(response.data);
          setAccessTokenCookie(response.data.access_token, response.data.access_token_expires_at);
          success('登录成功', '您正在使用访客模式');
          router.push((redirectTo || '/') as Route);
          return true;
        } else {
          error('登录失败', response.message ?? '请稍后重试');
          return false;
        }
      } catch (err) {
        error('登录失败', err instanceof Error ? err.message : '网络错误');
        return false;
      }
    },
    [storeLogin, router, success, error]
  );

  const logout = useCallback(async () => {
    try {
      await authService.logout();
    } catch {
      // 忽略登出请求失败
    }
    storeLogout();
    clearAccessTokenCookie();
    router.push('/');
  }, [storeLogout, router]);

  const refreshToken = useCallback(async (): Promise<string | null> => {
    if (!currentAccount) return null;

    try {
      const response = await authService.refresh({
        client_type: 'web',
        refresh_token: currentAccount.refresh_token,
      });

      if (isSuccess(response)) {
        const data = response.data as AuthData;
        updateToken(data.access_token, data.access_token_expires_at);
        setAccessTokenCookie(data.access_token, data.access_token_expires_at);
        return data.access_token;
      }
    } catch {
      // Token 刷新失败
    }

    // 刷新失败，退出登录
    storeLogout();
    clearAccessTokenCookie();
    return null;
  }, [currentAccount, updateToken, storeLogout]);

  const switchAccount = useCallback(
    (userId: number) => {
      storeSwitchAccount(userId);
      // 切换账户后可能需要重新加载数据
      router.refresh();
    },
    [storeSwitchAccount, router]
  );

  const removeAccount = useCallback(
    (userId: number) => {
      storeRemoveAccount(userId);
    },
    [storeRemoveAccount]
  );

  return {
    // State
    isAuthenticated,
    isLoading,
    currentAccount,
    accounts,
    userId: currentAccount?.user_id ?? null,
    nickname: currentAccount?.nickname ?? null,

    // Actions
    login,
    register,
    guestLogin,
    logout,
    refreshToken,
    switchAccount,
    removeAccount,
    getAccessToken,
    isTokenExpired,
  };
}
