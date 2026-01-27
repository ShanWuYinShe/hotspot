'use client';

import { useEffect, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';

import { ChatHeader } from '@/components/chat/chat-header';
import { MessageBubble } from '@/components/chat/message-bubble';
import { MessageInput } from '@/components/chat/message-input';
import { PrivateChatService, PrivateMessageService } from '@/lib/api/services/chat';
import { PrivateMessageItem, PaginatedData } from '@/types/api';
import { useChatStore } from '@/stores/chat';
import { useAuthStore } from '@/stores/auth';
import { useWebSocket } from '@/hooks/use-websocket';

const mergeMessages = (messages: PrivateMessageItem[]) => {
  const map = new Map<string, PrivateMessageItem>();
  messages.forEach((message) => map.set(message.message_id, message));
  return Array.from(map.values()).sort((a, b) => a.sent_at.localeCompare(b.sent_at));
};

export default function PrivateChatPage() {
  const params = useParams();
  const router = useRouter();
  const t = useTranslations('chat');
  const tCommon = useTranslations('common');
  const userId = Number(params.userId);
  const { subscribeUsers, sendPrivateTyping } = useWebSocket();
  const currentUserId = useAuthStore((state) => state.user?.user_id ?? 0);
  const { privateMessages, setPrivateMessages, privateTyping } = useChatStore();

  useEffect(() => {
    if (userId) {
      subscribeUsers([userId]);
    }
  }, [subscribeUsers, userId]);

  const { data: conversation } = useQuery({
    queryKey: ['private-chat', userId],
    queryFn: async () => {
      const res = await PrivateChatService.openTheDoor(userId, 20);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: Number.isFinite(userId) && userId > 0,
  });

  const { data: historyData } = useInfiniteQuery({
    queryKey: ['private-messages', userId],
    queryFn: async ({ pageParam }): Promise<PaginatedData<PrivateMessageItem>> => {
      const res = await PrivateMessageService.history({
        target_user_id: userId,
        cursor: pageParam as string | undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: Number.isFinite(userId) && userId > 0,
  });

  useEffect(() => {
    if (!historyData) {
      return;
    }
    const historyMessages = historyData.pages.flatMap((page) => page.items);
    const existing = privateMessages[userId] ?? [];
    setPrivateMessages(userId, mergeMessages([...historyMessages, ...existing]));
  }, [historyData, privateMessages, setPrivateMessages, userId]);

  const messages = useMemo(() => privateMessages[userId] ?? [], [privateMessages, userId]);
  const isTyping = privateTyping[userId];

  const handleSend = async (content: string) => {
    const response = await PrivateMessageService.send({
      target_user_id: userId,
      type: 'text',
      content,
    });
    if (response.code !== 0) {
      toast.error(response.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <ChatHeader
        title={conversation?.other_user.nickname ?? t('title')}
        subtitle={conversation?.other_user.user_id ? String(conversation.other_user.user_id) : undefined}
        onBack={() => router.back()}
      />

      <div className="space-y-3 min-h-[40vh]">
        {messages.length > 0 ? (
          messages.map((message) => (
            <MessageBubble key={message.message_id} message={message} isMine={message.sender.user_id === currentUserId} />
          ))
        ) : (
          <div className="text-center text-sm text-muted-foreground">{t('empty')}</div>
        )}
        {isTyping ? <div className="text-xs text-muted-foreground">{t('typing')}</div> : null}
      </div>

      <MessageInput onSend={handleSend} onTyping={(value) => sendPrivateTyping(userId, value)} />
    </div>
  );
}
