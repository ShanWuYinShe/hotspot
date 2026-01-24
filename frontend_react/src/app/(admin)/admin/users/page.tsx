/**
 * 用户管理页面 (Server Component)
 * 服务端获取用户列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { UserTable } from './user-table';
import { PageHeader } from '@/components/layout';
import type { AdminUserItem, PaginatedData } from '@/types/api';

interface AdminUsersPageProps {
  searchParams: Promise<{ q?: string }>;
}

async function UsersContent({ keyword }: { keyword: string }) {
  const response = await serverApi.post<PaginatedData<AdminUserItem>>(
    ENDPOINTS.ADMIN.USERS_SEARCH,
    { keyword: keyword || undefined, limit: 50 },
    { tags: ['admin-users'], revalidate: 60 }
  );

  const users = isServerSuccess(response) ? response.data?.items ?? [] : [];

  return <UserTable users={users} initialKeyword={keyword} />;
}

export default async function AdminUsersPage({ searchParams }: AdminUsersPageProps) {
  const t = await getTranslations('admin');
  const params = await searchParams;

  return (
    <div className="space-y-6">
      <PageHeader title={t('users')} description="检索用户并进行管理操作" />
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <UsersContent keyword={params.q || ''} />
      </Suspense>
    </div>
  );
}
