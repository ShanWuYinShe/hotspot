import { useState, useEffect, useCallback } from "react";
import { notificationsService } from "@/lib/api/services";
import { isSuccess } from "@/lib/api/client";
import type { NotificationItem, NotificationType } from "@/lib/types/api";

interface NotificationsData {
  notifications: NotificationItem[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
  hasMore: boolean;
  loadMore: () => void;
  refresh: () => void;
  markRead: (ids: string[]) => Promise<void>;
  markAllRead: () => Promise<void>;
}

export function useNotificationsData(
  type?: NotificationType,
): NotificationsData {
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [unreadCount, setUnreadCount] = useState(0);

  const load = useCallback(
    async (cursor?: string) => {
      setLoading(true);
      setError(null);
      try {
        const [notifsRes, unreadRes] = await Promise.allSettled([
          notificationsService.list({
            type,
            pagination: { limit: 20, cursor },
          }),
          cursor ? Promise.resolve(null) : notificationsService.unreadCount(),
        ]);

        if (
          notifsRes.status === "fulfilled" &&
          isSuccess(notifsRes.value) &&
          notifsRes.value.data
        ) {
          if (cursor) {
            setNotifications((prev) => [
              ...prev,
              ...notifsRes.value.data!.items,
            ]);
          } else {
            setNotifications(notifsRes.value.data.items);
          }
          setNextCursor(notifsRes.value.data.next_cursor);
        }

        if (
          unreadRes.status === "fulfilled" &&
          unreadRes.value &&
          isSuccess(unreadRes.value) &&
          unreadRes.value.data
        ) {
          setUnreadCount(unreadRes.value.data.total);
        } else if (
          !cursor &&
          notifsRes.status === "fulfilled" &&
          notifsRes.value.data
        ) {
          setUnreadCount(
            notifsRes.value.data.items.filter((n) => !n.is_read).length,
          );
        }
      } catch {
        setError("加载失败，请重试");
      } finally {
        setLoading(false);
      }
    },
    [type],
  );

  useEffect(() => {
    load();
  }, [load]);

  const markRead = useCallback(async (ids: string[]) => {
    try {
      await notificationsService.markAsRead({ notification_ids: ids });
      setNotifications((prev) =>
        prev.map((n) =>
          ids.includes(n.notification_id) ? { ...n, is_read: true } : n,
        ),
      );
      setUnreadCount((prev) => Math.max(0, prev - ids.length));
    } catch {}
  }, []);

  const markAllRead = useCallback(async () => {
    try {
      await notificationsService.markAllRead();
      setNotifications((prev) => prev.map((n) => ({ ...n, is_read: true })));
      setUnreadCount(0);
    } catch {}
  }, []);

  return {
    notifications,
    unreadCount,
    loading,
    error,
    hasMore: !!nextCursor,
    loadMore: () => nextCursor && load(nextCursor),
    refresh: () => load(),
    markRead,
    markAllRead,
  };
}
