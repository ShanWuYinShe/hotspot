/**
 * 群组管理页面 (Server Component)
 * 服务端获取群组列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { GroupTable } from './group-table';
import { PageHeader } from '@/components/layout';
import type { AdminGroupItem, PaginatedData } from '@/types/api';

interface AdminGroupsPageProps {
  searchParams: Promise<{ q?: string }>;
}

async function GroupsContent({ keyword }: { keyword: string }) {
  const response = await serverApi.post<PaginatedData<AdminGroupItem>>(
    ENDPOINTS.ADMIN.GROUPS_SEARCH,
    { keyword: keyword || undefined, limit: 50 },
    { tags: ['admin-groups'], revalidate: 60 }
  );

  const groups = isServerSuccess(response) ? response.data?.items ?? [] : [];

  return <GroupTable groups={groups} initialKeyword={keyword} />;
}

export default async function AdminGroupsPage({ searchParams }: AdminGroupsPageProps) {
  const t = await getTranslations('admin');
  const params = await searchParams;

  return (
    <div className="space-y-6">
      <PageHeader title={t('groups')} description="检索群组并进行管理操作" />
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <GroupsContent keyword={params.q || ''} />
      </Suspense>
    </div>
  );
}
