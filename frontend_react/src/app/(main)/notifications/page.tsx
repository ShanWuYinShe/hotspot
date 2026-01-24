/**
 * 通知页面 (Server Component)
 * 服务端获取通知列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { NotificationList } from './notification-list';
import { NotificationSettings } from './notification-settings';
import type { NotificationItem, PaginatedData } from '@/types/api';

async function NotificationsContent() {
  const response = await serverApi.get<PaginatedData<NotificationItem>>(
    ENDPOINTS.NOTIFICATIONS.LIST,
    { limit: 50 },
    { tags: ['notifications'], revalidate: 30 }
  );

  if (!isServerSuccess(response)) {
    return (
      <LoginRequiredCard actionName="查看通知" description="登录后才能查看通知消息。" />
    );
  }

  const notifications = response.data?.items ?? [];

  return <NotificationList notifications={notifications} />;
}

export default async function NotificationsPage() {
  const t = await getTranslations('notifications');

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <NotificationsContent />
      </Suspense>

      <NotificationSettings />
    </div>
  );
}
