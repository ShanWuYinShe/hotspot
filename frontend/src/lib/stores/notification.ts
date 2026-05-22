/**
 * 通知 Store (Zustand)
 * HTTP API 负责冷启动数据，WebSocket 负责实时增量
 */

import { create } from 'zustand';
import { useAuthStore } from './auth';
import { notificationsService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import { useWebSocketStore } from '@/lib/websocket/store';
import type {
  ApiResponse,
  PaginatedData,
  NotificationItem,
  NotificationListRequest,
} from '@/lib/types/api';

interface NotificationState {
  unreadCount: number;
  readSyncVersion: number;
  lastReadMutationType: 'mark_read' | 'mark_all_read' | null;
  lastReadNotificationIds: string[];

  fetchUnreadCount: () => Promise<void>;
  fetchHistory: (params?: NotificationListRequest) => Promise<ApiResponse<PaginatedData<NotificationItem>>>;
  markAsRead: (notificationIds: string[]) => Promise<boolean>;
  markAllAsRead: () => Promise<boolean>;
  initialize: () => void;
  destroy: () => void;
}

const INCREMENT_EVENTS = [
  'NewFollower',
  'MutualFollow',
  'MomentLiked',
  'MomentCommented',
  'CommentLiked',
  'CommentReply',
  'MessageMentioned',
  'GroupInviteReceived',
  'GroupApplicationReceived',
  'SystemAnnouncement',
] as const;

let fetchVersion = 0;
let cleanupFns: (() => void)[] = [];
let unsubscribeAuth: (() => void) | null = null;

export const useNotificationStore = create<NotificationState>((set, get) => ({
  unreadCount: 0,
  readSyncVersion: 0,
  lastReadMutationType: null,
  lastReadNotificationIds: [],

  async fetchUnreadCount() {
    const version = ++fetchVersion;
    const res = await notificationsService.unreadCount();
    if (version !== fetchVersion) return;
    if (isSuccess(res) && res.data) {
      set({ unreadCount: res.data.total });
    }
  },

  async fetchHistory(params) {
    return notificationsService.list(params ?? {});
  },

  async markAsRead(notificationIds) {
    if (notificationIds.length === 0) return true;
    const res = await notificationsService.markAsRead({ notification_ids: notificationIds });
    if (isSuccess(res)) {
      set((state) => ({
        unreadCount: Math.max(0, state.unreadCount - notificationIds.length),
        lastReadMutationType: 'mark_read',
        lastReadNotificationIds: [...notificationIds],
        readSyncVersion: state.readSyncVersion + 1,
      }));
      return true;
    }
    return false;
  },

  async markAllAsRead() {
    const res = await notificationsService.markAllRead();
    if (isSuccess(res)) {
      set((state) => ({
        unreadCount: 0,
        lastReadMutationType: 'mark_all_read',
        lastReadNotificationIds: [],
        readSyncVersion: state.readSyncVersion + 1,
      }));
      return true;
    }
    return false;
  },

  initialize() {
    if (typeof window === 'undefined') return;

    for (const event of INCREMENT_EVENTS) {
      cleanupFns.push(
        useWebSocketStore.getState().on(event, () => {
          set((state) => ({ unreadCount: state.unreadCount + 1 }));
        }),
      );
    }

    unsubscribeAuth = useAuthStore.subscribe((state, prevState) => {
      if (state.isAuthenticated === prevState.isAuthenticated) return;
      if (state.isAuthenticated) {
        get().fetchUnreadCount();
      } else {
        fetchVersion++;
        set({ unreadCount: 0 });
      }
    });

    if (useAuthStore.getState().isAuthenticated) {
      get().fetchUnreadCount();
    }
  },

  destroy() {
    if (unsubscribeAuth) {
      unsubscribeAuth();
      unsubscribeAuth = null;
    }
    cleanupFns.forEach((fn) => fn());
    cleanupFns = [];
  },
}));
