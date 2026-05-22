/**
 * 管理员权限 Store (Zustand)
 * 跟随 authStore 自动加载权限
 */

import { create } from 'zustand';
import { useAuthStore } from './auth';
import { adminRbacService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import type { AdminPermissions } from '@/lib/types/api';

type AdminPermissionStatus = 'idle' | 'loading' | 'ready' | 'forbidden' | 'error';

interface AdminPermissionsState {
  status: AdminPermissionStatus;
  permissions: AdminPermissions | null;
  loadedUserId: string | null;
  isLoading: boolean;
  isAdmin: boolean;
  isForbidden: boolean;
  isError: boolean;

  ensureLoaded: (userId?: string) => Promise<void>;
  refresh: () => void;
  reset: () => void;
  initialize: () => void;
  destroy: () => void;
}

let requestVersion = 0;
let unsubscribeAuth: (() => void) | null = null;

export const useAdminPermissionsStore = create<AdminPermissionsState>((set, get) => ({
  status: 'idle',
  permissions: null,
  loadedUserId: null,
  isLoading: true,
  isAdmin: false,
  isForbidden: false,
  isError: false,

  async ensureLoaded(userId) {
    const targetUserId = userId ?? useAuthStore.getState().currentUserId;
    if (!useAuthStore.getState().isAuthenticated || !targetUserId) {
      get().reset();
      return;
    }
    const { loadedUserId, status } = get();
    if (loadedUserId === targetUserId || status === 'loading') return;

    const version = ++requestVersion;
    set({ status: 'loading', permissions: null, isLoading: true });

    try {
      const res = await adminRbacService.getMyPermissions();
      if (
        version !== requestVersion ||
        useAuthStore.getState().currentUserId !== targetUserId ||
        !useAuthStore.getState().isAuthenticated
      ) {
        return;
      }

      if (isSuccess(res) && res.data) {
        set({
          loadedUserId: targetUserId,
          permissions: res.data,
          status: 'ready',
          isLoading: false,
          isAdmin: true,
          isForbidden: false,
          isError: false,
        });
      } else {
        set({
          loadedUserId: targetUserId,
          permissions: null,
          status: 'forbidden',
          isLoading: false,
          isAdmin: false,
          isForbidden: true,
          isError: false,
        });
      }
    } catch {
      const currentUserId = useAuthStore.getState().currentUserId;
      if (
        version !== requestVersion ||
        currentUserId !== targetUserId ||
        !useAuthStore.getState().isAuthenticated
      ) {
        return;
      }
      set({
        loadedUserId: targetUserId,
        permissions: null,
        status: 'error',
        isLoading: false,
        isAdmin: false,
        isForbidden: false,
        isError: true,
      });
    }
  },

  refresh() {
    set({ loadedUserId: null, permissions: null });
    const userId = useAuthStore.getState().currentUserId;
    if (useAuthStore.getState().isAuthenticated && userId) {
      get().ensureLoaded(userId);
      return;
    }
    set({ status: 'idle', isLoading: true, isAdmin: false, isForbidden: false, isError: false });
  },

  reset() {
    requestVersion++;
    set({
      status: 'idle',
      permissions: null,
      loadedUserId: null,
      isLoading: true,
      isAdmin: false,
      isForbidden: false,
      isError: false,
    });
  },

  initialize() {
    if (typeof window === 'undefined') return;

    unsubscribeAuth = useAuthStore.subscribe((state, prevState) => {
      const authChanged =
        state.isAuthenticated !== prevState.isAuthenticated ||
        state.currentUserId !== prevState.currentUserId;
      if (!authChanged) return;

      if (!state.isAuthenticated || !state.currentUserId) {
        get().reset();
        return;
      }

      const { loadedUserId, status } = get();
      if (loadedUserId === state.currentUserId || status === 'loading') return;
      get().ensureLoaded(state.currentUserId);
    });

    const { isAuthenticated, currentUserId } = useAuthStore.getState();
    if (isAuthenticated && currentUserId) {
      get().ensureLoaded(currentUserId);
    }
  },

  destroy() {
    if (unsubscribeAuth) {
      unsubscribeAuth();
      unsubscribeAuth = null;
    }
  },
}));
