/**
 * 认证状态管理
 * 使用 Zustand 管理用户认证状态
 * 
 * 核心原则：非阻塞式初始化，不阻塞页面渲染
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { MyCard, AuthData } from '@/types/api';

// Token 存储 key
const ACCESS_TOKEN_KEY = 'hotspot_access_token';
const REFRESH_TOKEN_KEY = 'hotspot_refresh_token';

// Token 操作函数
export function getAccessToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function setAccessToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(ACCESS_TOKEN_KEY, token);
}

export function getRefreshToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setRefreshToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(REFRESH_TOKEN_KEY, token);
}

export function clearTokens(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

interface AuthState {
  /** 当前用户信息 */
  user: MyCard | null;
  /** 是否已登录 */
  isAuthenticated: boolean;
  /** Token 过期时间 */
  tokenExpiresAt: string | null;
  /** 认证状态是否已经初始化（后台完成，不阻塞渲染） */
  isHydrated: boolean;
  
  /** 设置用户信息 */
  setUser: (user: MyCard | null) => void;
  /** 登录成功后设置认证数据 */
  setAuthData: (data: AuthData) => void;
  /** 登出 */
  logout: () => void;
  /** 检查 token 是否有效 */
  checkTokenValid: () => boolean;
  /** 设置 hydrated 状态 */
  setHydrated: (hydrated: boolean) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      tokenExpiresAt: null,
      isHydrated: false,

      setUser: (user) => {
        set({ user, isAuthenticated: !!user });
      },

      setAuthData: (data) => {
        // 存储 token
        setAccessToken(data.access_token);
        if (data.refresh_token) {
          setRefreshToken(data.refresh_token);
        }
        
        set({
          isAuthenticated: true,
          tokenExpiresAt: data.access_token_expires_at,
        });
      },

      logout: () => {
        clearTokens();
        set({
          user: null,
          isAuthenticated: false,
          tokenExpiresAt: null,
        });
      },

      checkTokenValid: () => {
        const token = getAccessToken();
        const { tokenExpiresAt } = get();
        
        if (!token) {
          return false;
        }
        
        // 检查 token 是否过期
        if (tokenExpiresAt) {
          const expiresAt = new Date(tokenExpiresAt);
          if (expiresAt <= new Date()) {
            return false;
          }
        }
        
        return true;
      },

      setHydrated: (hydrated) => {
        set({ isHydrated: hydrated });
      },
    }),
    {
      name: 'hotspot-auth',
      storage: createJSONStorage(() => localStorage),
      // 只持久化必要的字段
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
        tokenExpiresAt: state.tokenExpiresAt,
      }),
      onRehydrateStorage: () => (state) => {
        // 恢复后标记为已 hydrated
        state?.setHydrated(true);
      },
    }
  )
);

/**
 * 获取当前用户 ID
 */
export function getCurrentUserId(): number | null {
  const { user } = useAuthStore.getState();
  return user?.user_id ?? null;
}

/**
 * 检查是否为临时用户
 */
export function isTempUser(): boolean {
  const { user } = useAuthStore.getState();
  return user?.user_type === 'temp';
}
