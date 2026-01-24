/**
 * 聊天页面 (Server Component)
 * 服务端获取会话列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { MessageCircle } from '@/components/icons';
import { serverApi, isServerSuccess, isAuthError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Card, CardContent } from '@/components/ui/card';
import { LoadingState } from '@/components/common/loading-state';
import { EmptyState } from '@/components/common/empty-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { ConversationItem } from '@/components/cards/conversation-item';
import { PageHeader } from '@/components/layout';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { PrivateChatListItem, PaginatedData } from '@/types/api';

async function ConversationList() {
  const t = await getTranslations('chat');
  
  const response = await serverApi.post<PaginatedData<PrivateChatListItem>>(
    ENDPOINTS.PRIVATE_CHATS.LIST,
    { limit: 50 },
    { tags: ['private-chats'], revalidate: 30 }
  );

  if (!isServerSuccess(response)) {
    if (isAuthError(response)) {
      return (
        <LoginRequiredCard actionName="查看聊天列表" description="登录后才能查看会话列表。" />
      );
    }
    return (
      <ServerErrorCard
        title="加载聊天失败"
        description={getServerErrorMessage(response)}
        requestId={response.request_id}
      />
    );
  }

  const conversations = response.data?.items ?? [];

  if (conversations.length === 0) {
    return (
      <Card>
        <CardContent className="p-6">
          <EmptyState
            icon={MessageCircle}
            title={t('noConversations')}
            description="开始和朋友聊天吧"
          />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent className="p-2">
        <ScrollArea className="h-[calc(100vh-200px)]">
          <div className="space-y-1">
            {conversations.map((conversation) => (
              <ConversationItem
                key={conversation.other_user.user_id}
                conversation={conversation}
              />
            ))}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}

export default async function ChatPage() {
  const t = await getTranslations('chat');

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeader
        title={t('title')}
        description="掌握最近的消息与未读提醒"
      />

      <Suspense fallback={<LoadingState text="加载中..." />}>
        <ConversationList />
      </Suspense>
    </div>
  );
}
