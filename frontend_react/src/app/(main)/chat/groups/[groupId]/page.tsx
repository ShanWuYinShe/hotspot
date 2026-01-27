'use client';

import { useEffect, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';

import { ChatHeader } from '@/components/chat/chat-header';
import { MessageBubble } from '@/components/chat/message-bubble';
import { MessageInput } from '@/components/chat/message-input';
import { GroupMessageService } from '@/lib/api/services/chat';
import { GroupService } from '@/lib/api/services/groups';
import { GroupMessageItem, PaginatedData } from '@/types/api';
import { useChatStore } from '@/stores/chat';
import { useAuthStore } from '@/stores/auth';
import { useWebSocket } from '@/hooks/use-websocket';

const mergeMessages = (messages: GroupMessageItem[]) => {
  const map = new Map<string, GroupMessageItem>();
  messages.forEach((message) => map.set(message.message_id, message));
  return Array.from(map.values()).sort((a, b) => a.sent_at.localeCompare(b.sent_at));
};

export default function GroupChatPage() {
  const params = useParams();
  const router = useRouter();
  const t = useTranslations('chat');
  const tCommon = useTranslations('common');
  const groupId = params.groupId as string;
  const { subscribeGroups, sendGroupTyping } = useWebSocket();
  const currentUserId = useAuthStore((state) => state.user?.user_id ?? 0);
  const { groupMessages, setGroupMessages, groupTyping } = useChatStore();

  useEffect(() => {
    if (groupId) {
      subscribeGroups([groupId]);
    }
  }, [groupId, subscribeGroups]);

  const { data: group } = useQuery({
    queryKey: ['group', groupId],
    queryFn: async () => {
      const res = await GroupService.openTheDoor(groupId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!groupId,
  });

  const { data: historyData } = useInfiniteQuery({
    queryKey: ['group-messages', groupId],
    queryFn: async ({ pageParam }): Promise<PaginatedData<GroupMessageItem>> => {
      const res = await GroupMessageService.history({
        group_chat_id: groupId,
        cursor: pageParam as string | undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!groupId,
  });

  useEffect(() => {
    if (!historyData) {
      return;
    }
    const historyMessages = historyData.pages.flatMap((page) => page.items);
    const existing = groupMessages[groupId] ?? [];
    setGroupMessages(groupId, mergeMessages([...historyMessages, ...existing]));
  }, [groupId, groupMessages, historyData, setGroupMessages]);

  const messages = useMemo(() => groupMessages[groupId] ?? [], [groupId, groupMessages]);
  const typingState = groupTyping[groupId] ?? {};
  const typingUsers = Object.entries(typingState)
    .filter(([, isTyping]) => isTyping)
    .map(([userId]) => userId);

  const handleSend = async (content: string) => {
    const response = await GroupMessageService.send({
      group_chat_id: groupId,
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
        title={group?.name ?? t('title')}
        subtitle={group?.description}
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
        {typingUsers.length > 0 ? (
          <div className="text-xs text-muted-foreground">{t('typing')}</div>
        ) : null}
      </div>

      <MessageInput onSend={handleSend} onTyping={(value) => sendGroupTyping(groupId, value)} />
    </div>
  );
}
