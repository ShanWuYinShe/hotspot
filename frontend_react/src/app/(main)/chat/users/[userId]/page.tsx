/**
 * 私聊详情页面 (Server Component)
 * 服务端获取初始数据，客户端处理实时通信
 */

import { notFound } from 'next/navigation';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { PrivateChatRoom } from '@/components/chat';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import type { UserDetailData } from '@/types/api';

interface PrivateChatPageProps {
  params: Promise<{ userId: string }>;
}

export default async function PrivateChatPage({ params }: PrivateChatPageProps) {
  const { userId } = await params;
  const userIdNum = Number(userId);
  
  if (Number.isNaN(userIdNum)) {
    notFound();
  }
  
  // 检查认证状态
  const authResponse = await serverApi.get<{ user_id: number }>('/auth/me');
  
  if (!isServerSuccess(authResponse)) {
    return (
      <div className="max-w-lg mx-auto">
        <LoginRequiredCard actionName="开始聊天" description="登录后才能查看和发送消息。" />
      </div>
    );
  }
  
  // 获取用户信息
  const userResponse = await serverApi.get<UserDetailData>(
    ENDPOINTS.USER.DETAIL(userIdNum)
  );
  
  if (!isServerSuccess(userResponse) || !userResponse.data) {
    notFound();
  }
  
  return <PrivateChatRoom userId={userIdNum} initialUser={userResponse.data} />;
}
