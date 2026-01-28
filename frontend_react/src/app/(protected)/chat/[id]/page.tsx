'use client';

import * as React from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import * as privateChatService from '@/lib/api/services/private-chat';
import { isSuccess } from '@/lib/api/client';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Input } from '@/components/ui/input';
import {
  ArrowLeft,
  Loader2,
  Send,
  MoreVertical,
  Phone,
  Video,
  Image as ImageIcon,
  Smile,
} from 'lucide-react';
import type { PrivateMessageItem } from '@/types/api';

export default function ChatDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuthStore();
  const otherUserId = Number(params.id);

  const [newMessage, setNewMessage] = React.useState('');
  const messagesEndRef = React.useRef<HTMLDivElement>(null);

  // Fetch chat info
  const chatQuery = useQuery({
    queryKey: ['chat', otherUserId],
    queryFn: async () => {
      const res = await privateChatService.openChat(otherUserId);
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  // Fetch messages
  const messagesQuery = useInfiniteQuery({
    queryKey: ['messages', otherUserId],
    queryFn: async ({ pageParam }) => {
      const res = await privateChatService.getMessageHistory({
        target_user_id: otherUserId,
        cursor: pageParam as string | undefined,
        limit: 30,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
  });

  // Send message mutation
  const sendMutation = useMutation({
    mutationFn: async (content: string) => {
      const res = await privateChatService.sendMessage({
        target_user_id: otherUserId,
        type: 'text',
        content,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    onSuccess: () => {
      setNewMessage('');
      queryClient.invalidateQueries({ queryKey: ['messages', otherUserId] });
      queryClient.invalidateQueries({ queryKey: ['my-chats'] });
    },
  });

  const otherUser = chatQuery.data?.other_user;
  const messages = messagesQuery.data?.pages.flatMap((page) => page?.items ?? []).reverse() ?? [];

  // Scroll to bottom on new messages
  React.useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length]);

  const handleSend = () => {
    if (!newMessage.trim()) return;
    sendMutation.mutate(newMessage.trim());
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (chatQuery.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  if (!otherUser) {
    return (
      <div className="text-center py-20">
        <p className="text-muted-foreground">无法加载对话</p>
        <Button variant="outline" onClick={() => router.back()} className="mt-4">
          返回
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)] max-w-2xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between pb-4 border-b">
        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.back()}
            className="shrink-0"
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <Link href={`/users/${otherUser.user_id}`} className="flex items-center gap-3 group">
            <div className="relative">
              <Avatar className="h-10 w-10">
                <AvatarImage src={otherUser.avatar_url} alt={otherUser.nickname} />
                <AvatarFallback className="bg-linear-to-br from-primary/80 to-primary text-primary-foreground">
                  {otherUser.nickname.slice(0, 2).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              {(otherUser as { is_online?: boolean }).is_online && (
                <span className="absolute bottom-0 right-0 w-2.5 h-2.5 bg-green-500 rounded-full ring-2 ring-background" />
              )}
            </div>
            <div>
              <p className="font-medium group-hover:text-primary transition-colors">
                {otherUser.nickname}
              </p>
              <p className="text-xs text-muted-foreground">
                {(otherUser as { is_online?: boolean }).is_online ? '在线' : '离线'}
              </p>
            </div>
          </Link>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" className="rounded-full">
            <Phone className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="rounded-full">
            <Video className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="rounded-full">
            <MoreVertical className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto py-4 space-y-4">
        {/* Load More Button */}
        {messagesQuery.hasNextPage && (
          <div className="flex justify-center">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => messagesQuery.fetchNextPage()}
              disabled={messagesQuery.isFetchingNextPage}
            >
              {messagesQuery.isFetchingNextPage ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                '加载更早的消息'
              )}
            </Button>
          </div>
        )}

        {messagesQuery.isLoading ? (
          <div className="flex justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : messages.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <p>开始新的对话吧</p>
          </div>
        ) : (
          messages.map((message, index) => (
            <MessageBubble
              key={message.message_id}
              message={message}
              isOwn={message.sender.user_id === user?.user_id}
              showAvatar={
                index === 0 ||
                messages[index - 1]?.sender.user_id !== message.sender.user_id
              }
              otherUser={otherUser}
            />
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="pt-4 border-t">
        <div className="flex items-end gap-2">
          <div className="flex gap-1">
            <Button variant="ghost" size="icon" className="rounded-full shrink-0">
              <ImageIcon className="h-5 w-5 text-muted-foreground" />
            </Button>
            <Button variant="ghost" size="icon" className="rounded-full shrink-0">
              <Smile className="h-5 w-5 text-muted-foreground" />
            </Button>
          </div>
          <div className="flex-1 flex gap-2">
            <Input
              placeholder="输入消息..."
              value={newMessage}
              onChange={(e) => setNewMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              className="flex-1 rounded-full"
            />
            <Button
              size="icon"
              className="rounded-full shrink-0"
              disabled={!newMessage.trim() || sendMutation.isPending}
              onClick={handleSend}
            >
              {sendMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

interface MessageBubbleProps {
  message: PrivateMessageItem;
  isOwn: boolean;
  showAvatar: boolean;
  otherUser: {
    user_id: number;
    nickname: string;
    avatar_url?: string;
  };
}

function MessageBubble({ message, isOwn, showAvatar, otherUser }: MessageBubbleProps) {
  const formatTime = (dateStr: string) => {
    return new Date(dateStr).toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const renderContent = () => {
    if (message.is_recalled) {
      return (
        <span className="text-muted-foreground italic text-sm">
          消息已撤回
        </span>
      );
    }

    switch (message.type) {
      case 'text':
        return <p className="whitespace-pre-wrap break-words">{message.content}</p>;
      case 'image':
        return message.media?.url ? (
          <img
            src={message.media.url}
            alt=""
            className="max-w-[240px] rounded-lg"
          />
        ) : null;
      case 'video':
        return (
          <div className="flex items-center gap-2 text-sm">
            <Video className="h-4 w-4" />
            <span>[视频消息]</span>
          </div>
        );
      case 'location':
        return (
          <div className="text-sm">
            📍 {message.location?.location.location_name || '位置'}
          </div>
        );
      default:
        return <span className="text-sm">[未知消息类型]</span>;
    }
  };

  return (
    <div
      className={cn(
        'flex gap-2',
        isOwn ? 'flex-row-reverse' : 'flex-row'
      )}
    >
      {/* Avatar */}
      <div className="w-8 shrink-0">
        {showAvatar && !isOwn && (
          <Avatar className="h-8 w-8">
            <AvatarImage src={otherUser.avatar_url} alt={otherUser.nickname} />
            <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-xs">
              {otherUser.nickname.slice(0, 1)}
            </AvatarFallback>
          </Avatar>
        )}
      </div>

      {/* Message */}
      <div
        className={cn(
          'max-w-[70%] space-y-1',
          isOwn ? 'items-end' : 'items-start'
        )}
      >
        <div
          className={cn(
            'px-4 py-2 rounded-2xl',
            isOwn
              ? 'bg-primary text-primary-foreground rounded-br-md'
              : 'bg-muted rounded-bl-md'
          )}
        >
          {renderContent()}
        </div>
        <p
          className={cn(
            'text-[10px] text-muted-foreground px-2',
            isOwn ? 'text-right' : 'text-left'
          )}
        >
          {formatTime(message.sent_at)}
        </p>
      </div>
    </div>
  );
}
