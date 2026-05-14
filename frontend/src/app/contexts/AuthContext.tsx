import React, { createContext, useState, useEffect, useContext, useCallback } from 'react';
import { authService, type User } from '@/lib/api/services';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface AuthContextType extends AuthState {
  login: (username: string, password: string) => Promise<boolean>;
  register: (username: string, password: string, nickname: string) => Promise<boolean>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

const TOKEN_KEY = 'hotspot_token';

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    token: localStorage.getItem(TOKEN_KEY),
    isAuthenticated: false,
    isLoading: true,
  });

  // Restore session on mount
  useEffect(() => {
    if (state.token) {
      authService.me().then((res) => {
        if (res.code === 0) {
          setState((s) => ({ ...s, user: res.data, isAuthenticated: true, isLoading: false }));
        } else {
          localStorage.removeItem(TOKEN_KEY);
          setState((s) => ({ ...s, token: null, isLoading: false }));
        }
      });
    } else {
      setState((s) => ({ ...s, isLoading: false }));
    }
  }, []);

  const login = useCallback(async (username: string, password: string): Promise<boolean> => {
    const res = await authService.login(username, password);
    if (res.code === 0 && res.data?.token) {
      localStorage.setItem(TOKEN_KEY, res.data.token);
      setState({ user: res.data.user, token: res.data.token, isAuthenticated: true, isLoading: false });
      return true;
    }
    return false;
  }, []);

  const register = useCallback(async (username: string, password: string, nickname: string): Promise<boolean> => {
    const res = await authService.register(username, password, nickname);
    if (res.code === 0 && res.data?.token) {
      localStorage.setItem(TOKEN_KEY, res.data.token);
      setState({ user: res.data.user, token: res.data.token, isAuthenticated: true, isLoading: false });
      return true;
    }
    return false;
  }, []);

  const logout = useCallback(async () => {
    await authService.logout();
    localStorage.removeItem(TOKEN_KEY);
    setState({ user: null, token: null, isAuthenticated: false, isLoading: false });
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
