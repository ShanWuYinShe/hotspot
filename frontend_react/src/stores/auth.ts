/**
 * 认证状态管理
 * 支持多账户管理
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { StoredAccount, AuthState } from '@/types/client';
import type { AuthData, AuthTokenPairApp } from '@/types/api';

interface AuthStore extends AuthState {
  // Actions
  setLoading: (isLoading: boolean) => void;
  login: (authData: AuthData) => void;
  logout: () => void;
  switchAccount: (userId: number) => void;
  removeAccount: (userId: number) => void;
  updateToken: (accessToken: string, expiresAt: string) => void;
  getAccessToken: () => string | null;
  isTokenExpired: () => boolean;
}

function isAppAuthData(data: AuthData): data is AuthTokenPairApp {
  return 'refresh_token' in data;
}

function authDataToStoredAccount(data: AuthData): StoredAccount {
  const base: StoredAccount = {
    user_id: data.user_id,
    nickname: data.nickname,
    access_token: data.access_token,
    access_token_expires_at: data.access_token_expires_at,
    refresh_token_expires_at: data.refresh_token_expires_at,
    last_active_at: new Date().toISOString(),
  };

  if (isAppAuthData(data)) {
    base.refresh_token = data.refresh_token;
  }

  return base;
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      // State
      isAuthenticated: false,
      isLoading: true,
      currentAccount: null,
      accounts: [],

      // Actions
      setLoading: (isLoading) => set({ isLoading }),

      login: (authData) => {
        const newAccount = authDataToStoredAccount(authData);
        
        set((state) => {
          // 检查是否已存在该账户
          const existingIndex = state.accounts.findIndex(
            (a) => a.user_id === newAccount.user_id
          );

          let accounts: StoredAccount[];
          if (existingIndex >= 0) {
            // 更新现有账户
            accounts = [...state.accounts];
            accounts[existingIndex] = newAccount;
          } else {
            // 添加新账户
            accounts = [...state.accounts, newAccount];
          }

          return {
            isAuthenticated: true,
            isLoading: false,
            currentAccount: newAccount,
            accounts,
          };
        });
      },

      logout: () => {
        const { currentAccount, accounts } = get();
        
        if (currentAccount) {
          // 移除当前账户
          const remainingAccounts = accounts.filter(
            (a) => a.user_id !== currentAccount.user_id
          );

          if (remainingAccounts.length > 0) {
            // 切换到第一个剩余账户
            set({
              isAuthenticated: true,
              currentAccount: remainingAccounts[0],
              accounts: remainingAccounts,
            });
          } else {
            // 没有剩余账户
            set({
              isAuthenticated: false,
              currentAccount: null,
              accounts: [],
            });
          }
        }
      },

      switchAccount: (userId) => {
        const { accounts } = get();
        const account = accounts.find((a) => a.user_id === userId);
        
        if (account) {
          // 更新 last_active_at
          const updatedAccount = {
            ...account,
            last_active_at: new Date().toISOString(),
          };

          set((state) => ({
            isAuthenticated: true,
            currentAccount: updatedAccount,
            accounts: state.accounts.map((a) =>
              a.user_id === userId ? updatedAccount : a
            ),
          }));
        }
      },

      removeAccount: (userId) => {
        const { currentAccount, accounts } = get();
        const remainingAccounts = accounts.filter((a) => a.user_id !== userId);

        if (currentAccount?.user_id === userId) {
          // 如果移除的是当前账户
          if (remainingAccounts.length > 0) {
            set({
              currentAccount: remainingAccounts[0],
              accounts: remainingAccounts,
            });
          } else {
            set({
              isAuthenticated: false,
              currentAccount: null,
              accounts: [],
            });
          }
        } else {
          set({ accounts: remainingAccounts });
        }
      },

      updateToken: (accessToken, expiresAt) => {
        const { currentAccount } = get();
        
        if (currentAccount) {
          const updatedAccount: StoredAccount = {
            ...currentAccount,
            access_token: accessToken,
            access_token_expires_at: expiresAt,
          };

          set((state) => ({
            currentAccount: updatedAccount,
            accounts: state.accounts.map((a) =>
              a.user_id === currentAccount.user_id ? updatedAccount : a
            ),
          }));
        }
      },

      getAccessToken: () => {
        const { currentAccount } = get();
        return currentAccount?.access_token ?? null;
      },

      isTokenExpired: () => {
        const { currentAccount } = get();
        if (!currentAccount) return true;
        
        const expiresAt = new Date(currentAccount.access_token_expires_at);
        // 提前 60 秒判定为过期，预留刷新时间
        return expiresAt.getTime() - 60000 < Date.now();
      },
    }),
    {
      name: 'hotspot-auth',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        currentAccount: state.currentAccount,
        accounts: state.accounts,
        isAuthenticated: state.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        if (state) {
          state.setLoading(false);
        }
      },
    }
  )
);
