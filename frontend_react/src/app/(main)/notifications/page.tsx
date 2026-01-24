/**
 * 通知页面 (Server Component)
 * 服务端获取通知列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess, isAuthError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { NotificationList } from './notification-list';
import { NotificationSettings } from './notification-settings';
import { PageHeader } from '@/components/layout';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { NotificationItem, PaginatedData } from '@/types/api';

async function NotificationsContent() {
  const response = await serverApi.post<PaginatedData<NotificationItem>>(
    ENDPOINTS.NOTIFICATIONS.LIST,
    { limit: 50 },
    { tags: ['notifications'], revalidate: 30 }
  );

  if (!isServerSuccess(response)) {
    if (isAuthError(response)) {
      return (
        <LoginRequiredCard actionName="查看通知" description="登录后才能查看通知消息。" />
      );
    }
    return (
      <ServerErrorCard
        title="加载通知失败"
        description={getServerErrorMessage(response)}
        requestId={response.request_id}
      />
    );
  }

  const notifications = response.data?.items ?? [];

  return <NotificationList notifications={notifications} />;
}

export default async function NotificationsPage() {
  const t = await getTranslations('notifications');

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <PageHeader
        title={t('title')}
        description="及时掌握互动、系统与聊天提醒"
      />

      <Suspense fallback={<LoadingState text="加载中..." />}>
        <NotificationsContent />
      </Suspense>

      <NotificationSettings />
    </div>
  );
}
