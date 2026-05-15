import React, { createContext, useState, useEffect, useContext, useCallback } from 'react';
import { authService, usersService, type User } from '@/lib/api/services';
import { api } from '@/lib/api/client';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface AuthContextType extends AuthState {
  login: (login_id: string, password: string) => Promise<boolean>;
  register: (login_id: string, nickname: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    isAuthenticated: false,
    isLoading: true,
  });

  // Restore session on mount — backend auth via HttpOnly cookie
  useEffect(() => {
    usersService.me().then((res) => {
      if (api.isSuccess(res) && res.data) {
        setState({ user: res.data, isAuthenticated: true, isLoading: false });
      } else {
        setState({ user: null, isAuthenticated: false, isLoading: false });
      }
    }).catch(() => {
      setState({ user: null, isAuthenticated: false, isLoading: false });
    });
  }, []);

  const refreshUser = useCallback(async () => {
    const res = await usersService.me();
    if (api.isSuccess(res) && res.data) {
      setState({ user: res.data, isAuthenticated: true, isLoading: false });
    }
  }, []);

  const login = useCallback(async (login_id: string, password: string): Promise<boolean> => {
    const res = await authService.login(login_id, password);
    if (api.isSuccess(res)) {
      await refreshUser();
      return true;
    }
    return false;
  }, [refreshUser]);

  const register = useCallback(async (login_id: string, nickname: string, password: string): Promise<boolean> => {
    const res = await authService.register(login_id, nickname, password);
    if (api.isSuccess(res)) {
      await refreshUser();
      return true;
    }
    return false;
  }, [refreshUser]);

  const logout = useCallback(async () => {
    await authService.logout();
    setState({ user: null, isAuthenticated: false, isLoading: false });
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, register, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
