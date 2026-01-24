'use client';

/**
 * 私聊聊天室组件 (Client Component)
 * 处理 WebSocket、消息发送、实时更新等客户端逻辑
 */

import { useEffect, useRef, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ChatHeader } from './chat-header';
import { MessageBubble } from './message-bubble';
import { MessageInput } from './message-input';
import { LoadingState } from '@/components/common/loading-state';
import { EmptyState } from '@/components/common/empty-state';
import { MessageCircle } from '@/components/icons';
import { useAuthStore } from '@/stores/auth';
import { useWebSocket } from '@/hooks/use-websocket';
import { useToast } from '@/hooks/use-toast';
import { privateChatService } from '@/lib/api/services/private-chat';
import { uploadService } from '@/lib/api/services/upload';
import { isSuccess } from '@/lib/api/client';
import type { UserDetailData, SendMessageType, MessageReplyTo } from '@/types/api';

interface PrivateChatRoomProps {
  userId: number;
  initialUser: UserDetailData;
}

export function PrivateChatRoom({ userId, initialUser }: PrivateChatRoomProps) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  
  const currentUser = useAuthStore((state) => state.currentAccount);
  
  const [otherUser] = useState<UserDetailData>(initialUser);
  const [isTyping, setIsTyping] = useState(false);
  const [replyTo, setReplyTo] = useState<MessageReplyTo | null>(null);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);

  // WebSocket
  const { on, sendTyping, isConnected } = useWebSocket();

  // 消息列表
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ['private-messages', userId],
    queryFn: async ({ pageParam }) => {
      const res = await privateChatService.getHistory(userId, {
        cursor: pageParam,
        limit: 50,
      });
      if (isSuccess(res) && res.data) {
        return res.data;
      }
      throw new Error(res.message ?? 'Failed to load messages');
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination?.has_more ? lastPage.pagination.next_cursor : undefined,
  });

  const messages = data?.pages.flatMap((page) => page.items).reverse() ?? [];

  // 发送消息
  const sendMutation = useMutation({
    mutationFn: async ({
      type,
      content,
      file,
    }: {
      type: SendMessageType;
      content?: string;
      file?: File;
    }) => {
      let fileId: string | undefined;
      if (file) {
        const category = type === 'image' ? 'chat_image' :
                         type === 'video' ? 'chat_video' :
                         type === 'audio' ? 'chat_audio' : 'chat_file';
        const uploadResult = await uploadService.uploadFile(file, category);
        fileId = uploadResult.file_id;
      }

      const res = await privateChatService.sendMessage({
        target_user_id: userId,
        type,
        content,
        file_id: fileId,
        reply_to_message_id: replyTo?.message_id,
      });

      if (!isSuccess(res)) {
        throw new Error(res.message ?? '发送失败');
      }
      return res.data;
    },
    onSuccess: () => {
      setReplyTo(null);
    },
    onError: (error) => {
      toast.error('发送失败', error instanceof Error ? error.message : '未知错误');
    },
  });

  // 处理WebSocket消息
  useEffect(() => {
    if (!isConnected) return;

    const unsubNew = on('PrivateMessageNew', (payload) => {
      if (payload.target_user_id === userId || payload.sender.user_id === userId) {
        queryClient.invalidateQueries({ queryKey: ['private-messages', userId] });
      }
    });

    const unsubTyping = on('PrivateUserTyping', (payload) => {
      if (payload.user_id === userId) {
        setIsTyping(payload.is_typing);
      }
    });

    const unsubEdited = on('PrivateMessageEdited', (payload) => {
      if (payload.target_user_id === userId) {
        queryClient.invalidateQueries({ queryKey: ['private-messages', userId] });
      }
    });

    const unsubRecalled = on('PrivateMessageRecalled', (payload) => {
      if (payload.target_user_id === userId) {
        queryClient.invalidateQueries({ queryKey: ['private-messages', userId] });
      }
    });

    return () => {
      unsubNew();
      unsubTyping();
      unsubEdited();
      unsubRecalled();
    };
  }, [isConnected, on, userId, queryClient]);

  // 滚动到底部
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length]);

  // 处理加载更多
  const handleScroll = useCallback(() => {
    const container = messagesContainerRef.current;
    if (!container) return;

    if (container.scrollTop === 0 && hasNextPage && !isFetching) {
      fetchNextPage();
    }
  }, [hasNextPage, isFetching, fetchNextPage]);

  const handleSend = (type: SendMessageType, content?: string, file?: File) => {
    sendMutation.mutate({ type, content, file });
  };

  const handleTyping = (typing: boolean) => {
    sendTyping(userId, typing);
  };

  if (isLoading) {
    return <LoadingState text="加载消息..." />;
  }

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      <ChatHeader
        type="private"
        user={{
          ...otherUser,
          is_online: otherUser.is_online ?? false,
        }}
        isTyping={isTyping}
        onBack={() => router.back()}
        onViewProfile={() => router.push(`/users/${userId}`)}
      />

      <div
        ref={messagesContainerRef}
        className="flex-1 overflow-y-auto p-4 space-y-4"
        onScroll={handleScroll}
      >
        {isFetching && messages.length > 0 && (
          <div className="text-center">
            <LoadingState size="sm" />
          </div>
        )}

        {messages.length === 0 ? (
          <EmptyState
            icon={MessageCircle}
            title="暂无消息"
            description="发送第一条消息开始聊天吧"
          />
        ) : (
          messages.map((message, index) => {
            const isOwn = message.sender.user_id === currentUser?.user_id;
            const showAvatar = index === 0 ||
              messages[index - 1]?.sender.user_id !== message.sender.user_id;

            return (
              <MessageBubble
                key={message.message_id}
                message={message}
                isOwn={isOwn}
                showAvatar={showAvatar}
                onReply={() => setReplyTo({
                  message_id: message.message_id,
                  user_id: message.sender.user_id,
                  nickname: message.sender.nickname,
                  content: message.content ?? '',
                })}
              />
            );
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      <MessageInput
        onSend={handleSend}
        onTyping={handleTyping}
        replyTo={replyTo}
        onCancelReply={() => setReplyTo(null)}
        disabled={sendMutation.isPending}
      />
    </div>
  );
}
