/**
 * 个人中心页面 (Server Component)
 * 服务端获取用户信息
 */

import { Suspense } from 'react';
import { serverApi, isServerSuccess, isAuthError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { ProfileContent } from './profile-content';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { MyProfileData } from '@/types/api';

async function ProfileData() {
  const response = await serverApi.post<MyProfileData>(
    ENDPOINTS.USER.ME,
    {},
    { tags: ['my-profile'], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    if (isAuthError(response)) {
      return (
        <LoginRequiredCard actionName="查看个人中心" description="登录后才能查看个人信息。" />
      );
    }
    return (
      <ServerErrorCard
        title="加载个人信息失败"
        description={getServerErrorMessage(response)}
        requestId={response.request_id}
      />
    );
  }

  return <ProfileContent profile={response.data} />;
}

export default function ProfilePage() {
  return (
    <div className="max-w-2xl mx-auto">
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <ProfileData />
      </Suspense>
    </div>
  );
}
