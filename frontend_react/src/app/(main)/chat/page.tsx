/**
 * 聊天页面 (Server Component)
 * 服务端获取会话列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { MessageCircle } from '@/components/icons';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { ScrollArea } from '@/components/ui/scroll-area';
import { LoadingState } from '@/components/common/loading-state';
import { EmptyState } from '@/components/common/empty-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { ConversationItem } from '@/components/cards/conversation-item';
import type { PrivateChatListItem, PaginatedData } from '@/types/api';

async function ConversationList() {
  const t = await getTranslations('chat');
  
  const response = await serverApi.get<PaginatedData<PrivateChatListItem>>(
    ENDPOINTS.PRIVATE_CHATS.LIST,
    { limit: 50 },
    { tags: ['private-chats'], revalidate: 30 }
  );

  if (!isServerSuccess(response)) {
    return (
      <LoginRequiredCard actionName="查看聊天列表" description="登录后才能查看会话列表。" />
    );
  }

  const conversations = response.data?.items ?? [];

  if (conversations.length === 0) {
    return (
      <EmptyState
        icon={MessageCircle}
        title={t('noConversations')}
        description="开始和朋友聊天吧"
      />
    );
  }

  return (
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
  );
}

export default async function ChatPage() {
  const t = await getTranslations('chat');

  return (
    <div className="max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <ConversationList />
      </Suspense>
    </div>
  );
}
