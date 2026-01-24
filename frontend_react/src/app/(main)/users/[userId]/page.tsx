/**
 * 用户详情页面 (Server Component)
 * 服务端获取用户详情数据
 */

import { Suspense } from 'react';
import { notFound } from 'next/navigation';
import { serverApi, isServerSuccess, isNotFoundError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { UserProfileHeader } from './user-profile-header';
import { UserMomentsList } from './user-moments-list';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { UserDetailData, MomentItem, PaginatedData } from '@/types/api';

interface UserPageProps {
  params: Promise<{ userId: string }>;
}

async function UserContent({ userId }: { userId: string }) {
  const numericUserId = Number(userId);
  const [userRes, momentsRes] = await Promise.all([
    serverApi.post<UserDetailData>(
      ENDPOINTS.USER.DETAIL(numericUserId),
      { user_id: numericUserId },
      { tags: [`user-${userId}`], revalidate: 60 }
    ),
    serverApi.post<PaginatedData<MomentItem>>(
      ENDPOINTS.USER.MOMENTS(numericUserId),
      { user_id: numericUserId, limit: 20 },
      { tags: [`user-${userId}-moments`], revalidate: 60 }
    ),
  ]);

  if (!isServerSuccess(userRes)) {
    if (isNotFoundError(userRes)) {
      notFound();
    }
    return (
      <div className="max-w-2xl mx-auto">
        <ServerErrorCard
          title="加载用户失败"
          description={getServerErrorMessage(userRes)}
          requestId={userRes.request_id}
        />
      </div>
    );
  }

  const user = userRes.data;
  const moments = isServerSuccess(momentsRes) ? momentsRes.data?.items ?? [] : [];
  const momentsPagination = isServerSuccess(momentsRes) ? momentsRes.data?.pagination : undefined;

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <UserProfileHeader user={user} />
      <UserMomentsList
        userId={userId}
        initialMoments={moments}
        initialPagination={momentsPagination}
      />
    </div>
  );
}

export default async function UserPage({ params }: UserPageProps) {
  const { userId } = await params;

  return (
    <Suspense fallback={<LoadingState text="加载中..." />}>
      <UserContent userId={userId} />
    </Suspense>
  );
}
