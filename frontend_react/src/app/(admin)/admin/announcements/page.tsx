/**
 * 公告管理页面 (Server Component)
 * 服务端获取公告列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { AnnouncementList } from './announcement-list';
import type { AnnouncementItem, PaginatedData } from '@/types/api';

async function AnnouncementsContent() {
  const response = await serverApi.get<PaginatedData<AnnouncementItem>>(
    ENDPOINTS.ADMIN.ANNOUNCEMENTS,
    { limit: 50 },
    { tags: ['admin-announcements'], revalidate: 60 }
  );

  const announcements = isServerSuccess(response) ? response.data?.items ?? [] : [];

  return <AnnouncementList announcements={announcements} />;
}

export default async function AdminAnnouncementsPage() {
  const t = await getTranslations('admin');

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">{t('announcements')}</h1>
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <AnnouncementsContent />
      </Suspense>
    </div>
  );
}
