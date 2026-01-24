/**
 * 个人中心页面 (Server Component)
 * 服务端获取用户信息
 */

import { Suspense } from 'react';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { ProfileContent } from './profile-content';
import type { MyProfileData } from '@/types/api';

async function ProfileData() {
  const response = await serverApi.get<MyProfileData>(
    ENDPOINTS.USER.ME,
    undefined,
    { tags: ['my-profile'], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    return (
      <LoginRequiredCard actionName="查看个人中心" description="登录后才能查看个人信息。" />
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
