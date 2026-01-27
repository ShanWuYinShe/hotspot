'use client';

import { useEffect } from 'react';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { NotificationService } from '@/lib/api/services/notifications';
import { PaginatedData, NotificationItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useNotificationsStore } from '@/stores/notifications';

export default function NotificationsPage() {
  const t = useTranslations('notifications');
  const tCommon = useTranslations('common');
  const { setItems, setUnreadCount, markRead } = useNotificationsStore();

  const { data: unreadData } = useQuery({
    queryKey: ['notifications-unread'],
    queryFn: async () => {
      const res = await NotificationService.unreadCount();
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['notifications'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<NotificationItem>> => {
      const res = await NotificationService.list({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  useEffect(() => {
    if (data) {
      setItems(data.pages.flatMap((page) => page.items));
    }
  }, [data, setItems]);

  useEffect(() => {
    if (unreadData) {
      setUnreadCount(unreadData.total);
    }
  }, [setUnreadCount, unreadData]);

  const { ref } = useInfiniteScroll({
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  });

  const handleMarkAllRead = async () => {
    const res = await NotificationService.markRead();
    if (res.code === 0) {
      markRead(data?.pages.flatMap((page) => page.items.map((item) => item.notification_id)) ?? []);
      setUnreadCount(0);
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const items = data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('title')}</h1>
        <Button variant="outline" onClick={handleMarkAllRead}>
          {t('markAllRead')}
        </Button>
      </div>

      {items.length > 0 ? (
        <div className="space-y-3">
          {items.map((item) => (
            <Card key={item.notification_id}>
              <CardContent className="p-4 space-y-1">
                <div className="flex items-center justify-between">
                  <div className="font-semibold">{item.title ?? item.type}</div>
                  {!item.is_read ? <span className="text-xs text-primary">{t('new')}</span> : null}
                </div>
                {item.content ? <p className="text-sm text-muted-foreground">{item.content}</p> : null}
              </CardContent>
            </Card>
          ))}
          <div ref={ref} className="flex justify-center p-4">
            {isFetchingNextPage ? (
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
            ) : null}
          </div>
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">{t('empty')}</CardContent>
        </Card>
      )}
    </div>
  );
}
