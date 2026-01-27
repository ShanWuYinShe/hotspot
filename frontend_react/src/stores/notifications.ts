import { create } from 'zustand';

import { NotificationItem } from '@/types/api';

interface NotificationsState {
  items: NotificationItem[];
  unreadCount: number;
  setItems: (items: NotificationItem[]) => void;
  prependItem: (item: NotificationItem) => void;
  markRead: (ids: string[]) => void;
  setUnreadCount: (count: number) => void;
  incrementUnread: () => void;
}

export const useNotificationsStore = create<NotificationsState>((set) => ({
  items: [],
  unreadCount: 0,
  setItems: (items) => set({ items }),
  prependItem: (item) =>
    set((state) => ({
      items: [item, ...state.items],
      unreadCount: state.unreadCount + (item.is_read ? 0 : 1),
    })),
  markRead: (ids) =>
    set((state) => ({
      items: state.items.map((item) =>
        ids.includes(item.notification_id) ? { ...item, is_read: true } : item,
      ),
    })),
  setUnreadCount: (count) => set({ unreadCount: count }),
  incrementUnread: () => set((state) => ({ unreadCount: state.unreadCount + 1 })),
}));
