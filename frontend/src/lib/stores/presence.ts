/**
 * 用户在线状态管理 (Zustand)
 *
 * 通过 WebSocket 事件 (UserOnline/UserOffline) 实时更新。
 * spec 仅定义 online/offline 两种状态，前端扩展了 idle/dnd/invisible 供未来使用。
 */

import { create } from 'zustand';
import type { UserPresenceStatus } from '@/lib/types/api';

export type ExtendedPresenceStatus = UserPresenceStatus | 'idle' | 'dnd' | 'invisible';

export interface UserPresence {
  user_id: string;
  status: ExtendedPresenceStatus;
  status_text?: string;
  updated_at: string;
  last_seen?: string;
}

interface PresenceState {
  presences: Map<string, UserPresence>;

  getPresence: (userId: string) => UserPresence | null;
  getStatus: (userId: string) => ExtendedPresenceStatus;
  isOnline: (userId: string) => boolean;
  isOffline: (userId: string) => boolean;
  getLastSeen: (userId: string) => string | undefined;
  getBatchStatuses: (userIds: string[]) => Map<string, ExtendedPresenceStatus>;
  getAllUserIds: () => string[];

  setOnline: (userId: string, status: string, statusText?: string, onlineAt?: string) => void;
  setOffline: (userId: string, offlineAt: string) => void;
  clearPresence: (userId: string) => void;
  clearBatchPresences: (userIds: string[]) => void;
  clearAll: () => void;
}

export const usePresenceStore = create<PresenceState>((set, get) => ({
  presences: new Map(),

  getPresence(userId) {
    return get().presences.get(userId) ?? null;
  },

  getStatus(userId) {
    return get().presences.get(userId)?.status ?? 'offline';
  },

  isOnline(userId) {
    const status = get().getStatus(userId);
    return status === 'online' || status === 'idle';
  },

  isOffline(userId) {
    return get().getStatus(userId) === 'offline';
  },

  getLastSeen(userId) {
    return get().presences.get(userId)?.last_seen;
  },

  getBatchStatuses(userIds) {
    const { presences } = get();
    const result = new Map<string, ExtendedPresenceStatus>();
    for (const id of userIds) {
      result.set(id, presences.get(id)?.status ?? 'offline');
    }
    return result;
  },

  getAllUserIds() {
    return Array.from(get().presences.keys());
  },

  setOnline(userId, status, statusText, onlineAt) {
    const now = new Date().toISOString();
    const updatedAt = onlineAt ?? now;
    set((state) => {
      const existing = state.presences.get(userId);
      if (
        existing?.status === status &&
        existing?.status_text === statusText &&
        existing?.updated_at === updatedAt
      ) {
        return state;
      }
      const next = new Map(state.presences);
      next.set(userId, {
        user_id: userId,
        status: status as ExtendedPresenceStatus,
        status_text: statusText,
        updated_at: updatedAt,
        last_seen: existing?.last_seen,
      });
      return { presences: next };
    });
  },

  setOffline(userId, offlineAt) {
    set((state) => {
      const existing = state.presences.get(userId);
      const nextLastSeen = existing?.last_seen ?? offlineAt;
      if (
        existing?.status === 'offline' &&
        existing?.updated_at === offlineAt &&
        existing?.last_seen === nextLastSeen
      ) {
        return state;
      }
      const next = new Map(state.presences);
      next.set(userId, {
        user_id: userId,
        status: 'offline',
        updated_at: offlineAt,
        last_seen: nextLastSeen,
      });
      return { presences: next };
    });
  },

  clearPresence(userId) {
    set((state) => {
      if (!state.presences.has(userId)) return state;
      const next = new Map(state.presences);
      next.delete(userId);
      return { presences: next };
    });
  },

  clearBatchPresences(userIds) {
    set((state) => {
      const next = new Map(state.presences);
      let changed = false;
      for (const id of userIds) {
        changed = next.delete(id) || changed;
      }
      return changed ? { presences: next } : state;
    });
  },

  clearAll() {
    set({ presences: new Map() });
  },
}));
