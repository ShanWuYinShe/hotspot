import { create } from 'zustand';
import { MyCard, UserInfo } from '@/types/api';

interface AuthState {
  accessToken: string | null;
  user: (UserInfo & Partial<MyCard>) | null;
  isAuthenticated: boolean;
  hasBootstrapped: boolean;
  setAuth: (accessToken: string, user: UserInfo & Partial<MyCard>) => void;
  setBootstrapped: (value: boolean) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: null,
  user: null,
  isAuthenticated: false,
  hasBootstrapped: false,
  setAuth: (accessToken, user) => set({ accessToken, user, isAuthenticated: true }),
  setBootstrapped: (value) => set({ hasBootstrapped: value }),
  clearAuth: () => set({ accessToken: null, user: null, isAuthenticated: false }),
}));
