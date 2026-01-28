'use client';

import * as React from 'react';
import { useAuthStore } from '@/stores/auth';
import { login, logout, register, registerGuest, fetchCurrentUser } from '@/lib/api/services/auth';
import type { ApiResponse, AuthData } from '@/types/api';
import { toast } from '@/hooks/use-toast';

/**
 * 认证 Hook
 * 提供认证相关的状态和操作
 */
export function useAuth() {
  const {
    user,
    isAuthenticated,
  } = useAuthStore();

  const [isLoading, setIsLoading] = React.useState(false);

  /**
   * 登录
   */
  const handleLogin = async (
    loginId: string,
    password: string
  ): Promise<ApiResponse<AuthData>> => {
    setIsLoading(true);
    try {
      const result = await login(loginId, password);
      if (result.code !== 0) {
        toast({
          title: '登录失败',
          description: result.message || '请检查用户名和密码',
          variant: 'destructive',
        });
      }
      return result;
    } finally {
      setIsLoading(false);
    }
  };

  /**
   * 注册
   */
  const handleRegister = async (
    loginId: string,
    password: string,
    nickname: string
  ): Promise<ApiResponse<AuthData>> => {
    setIsLoading(true);
    try {
      const result = await register(loginId, password, nickname);
      if (result.code !== 0) {
        toast({
          title: '注册失败',
          description: result.message || '请稍后重试',
          variant: 'destructive',
        });
      }
      return result;
    } finally {
      setIsLoading(false);
    }
  };

  /**
   * 访客登录（临时用户）
   * 访客是一种临时用户，拥有登录 token，但只能通过此接口获取
   */
  const handleGuestLogin = async (
    nickname: string
  ): Promise<ApiResponse<AuthData>> => {
    setIsLoading(true);
    try {
      const result = await registerGuest(nickname);
      if (result.code !== 0) {
        toast({
          title: '访客登录失败',
          description: result.message || '请稍后重试',
          variant: 'destructive',
        });
      }
      return result;
    } finally {
      setIsLoading(false);
    }
  };

  /**
   * 登出
   */
  const handleLogout = async (): Promise<void> => {
    await logout();
    toast({
      title: '已退出登录',
    });
  };

  /**
   * 刷新用户信息
   */
  const refreshUser = async (): Promise<void> => {
    if (isAuthenticated) {
      await fetchCurrentUser();
    }
  };

  return {
    user,
    isAuthenticated,
    isLoading,
    login: handleLogin,
    register: handleRegister,
    guestLogin: handleGuestLogin,
    logout: handleLogout,
    refreshUser,
  };
}
