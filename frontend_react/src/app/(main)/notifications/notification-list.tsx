'use client';

/**
 * 通知列表组件 (Client Component)
 * 处理通知交互（标记已读等）
 */

import { useTransition } from 'react';
import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { Check, Bell } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { EmptyState } from '@/components/common/empty-state';
import { formatRelativeTime, cn } from '@/lib/utils';
import { notificationsService } from '@/lib/api/services/notifications';
import { isSuccess } from '@/lib/api/client';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import type { NotificationItem } from '@/types/api';

interface NotificationListProps {
  notifications: NotificationItem[];
}

export function NotificationList({ notifications }: NotificationListProps) {
  const t = useTranslations('notifications');
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();

  const hasUnread = notifications.some((n) => !n.is_read);

  const handleMarkAllRead = async () => {
    if (!requireAuth('标记通知已读')) return;
    startTransition(async () => {
      const response = await notificationsService.markAllRead();
      if (isSuccess(response)) {
        router.refresh();
      } else {
        toast.error('操作失败', response.message);
      }
    });
  };

  if (notifications.length === 0) {
    return (
      <EmptyState
        icon={Bell}
        title={t('noNotifications')}
        description="暂时没有新通知"
      />
    );
  }

  return (
    <>
      {hasUnread && (
        <div className="flex justify-end mb-4">
          <Button
            variant="outline"
            size="sm"
            onClick={handleMarkAllRead}
            disabled={isPending}
          >
            <Check className="h-4 w-4 mr-2" />
            {t('markAllRead')}
          </Button>
        </div>
      )}

      <ScrollArea className="h-[calc(100vh-200px)]">
        <div className="space-y-2">
          {notifications.map((notification) => (
            <Card
              key={notification.notification_id}
              className={cn(!notification.is_read && 'bg-accent/50')}
            >
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <div
                    className={cn(
                      'mt-1 h-2 w-2 rounded-full shrink-0',
                      notification.is_read ? 'bg-transparent' : 'bg-primary'
                    )}
                  />
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-sm">{notification.title}</p>
                    <p className="text-sm text-muted-foreground mt-0.5">
                      {notification.content}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      {formatRelativeTime(notification.created_at)}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </ScrollArea>
    </>
  );
}
