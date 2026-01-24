/**
 * 群聊详情页面 (Server Component)
 * 服务端获取初始数据，客户端处理实时通信
 */

import { notFound } from 'next/navigation';
import { serverApi, isServerSuccess, isAuthError, isNotFoundError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { GroupChatRoom } from '@/components/chat';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { GroupDetailData } from '@/types/api';

interface GroupChatPageProps {
  params: Promise<{ groupId: string }>;
}

export default async function GroupChatPage({ params }: GroupChatPageProps) {
  const { groupId } = await params;
  
  // 检查认证状态
  const authResponse = await serverApi.post<{ user_id: number }>('/auth/me', {});
  
  if (!isServerSuccess(authResponse)) {
    if (isAuthError(authResponse)) {
      return (
        <div className="max-w-lg mx-auto">
          <LoginRequiredCard actionName="开始群聊" description="登录后才能查看和发送群消息。" />
        </div>
      );
    }
    return (
      <div className="max-w-lg mx-auto">
        <ServerErrorCard
          title="加载失败"
          description={getServerErrorMessage(authResponse)}
          requestId={authResponse.request_id}
        />
      </div>
    );
  }
  
  // 获取群组信息
  const groupResponse = await serverApi.post<GroupDetailData>(
    ENDPOINTS.GROUP_CHATS.DETAIL(groupId),
    { group_chat_id: groupId }
  );
  
  if (!isServerSuccess(groupResponse) || !groupResponse.data) {
    if (isNotFoundError(groupResponse)) {
      notFound();
    }
    return (
      <div className="max-w-lg mx-auto">
        <ServerErrorCard
          title="加载群组失败"
          description={getServerErrorMessage(groupResponse)}
          requestId={groupResponse.request_id}
        />
      </div>
    );
  }
  
  return <GroupChatRoom groupId={groupId} initialGroup={groupResponse.data} />;
}
