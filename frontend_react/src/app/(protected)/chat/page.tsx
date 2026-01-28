'use client';

import * as React from 'react';
import Link from 'next/link';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import * as privateChatService from '@/lib/api/services/private-chat';
import { isSuccess } from '@/lib/api/client';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Search,
  Loader2,
  MessageCircle,
  X,
  Pin,
  BellOff,
} from 'lucide-react';
import type { PrivateChatListItem } from '@/types/api';

export default function ChatListPage() {
  useAuthStore();
  const [searchQuery, setSearchQuery] = React.useState('');

  // Fetch chats
  const chatsQuery = useQuery({
    queryKey: ['my-chats'],
    queryFn: async () => {
      const res = await privateChatService.getMyChats({ limit: 50 });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  const chats = chatsQuery.data?.items ?? [];
  
  // Filter and sort chats
  const filteredChats = React.useMemo(() => {
    let result = chats;
    
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter((chat) =>
        chat.other_user.nickname.toLowerCase().includes(query)
      );
    }
    
    // Sort: pinned first, then by last activity
    return result.sort((a, b) => {
      if (a.is_pinned && !b.is_pinned) return -1;
      if (!a.is_pinned && b.is_pinned) return 1;
      return new Date(b.last_activity_at).getTime() - new Date(a.last_activity_at).getTime();
    });
  }, [chats, searchQuery]);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold">消息</h1>
        <p className="text-muted-foreground text-sm mt-1">
          与好友保持联系
        </p>
      </div>

      {/* Search Bar */}
      <div className="relative">
        <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索联系人..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10 pr-10 h-11 rounded-full bg-muted/50 border-0 focus-visible:ring-2 focus-visible:ring-primary/50"
        />
        {searchQuery && (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-4 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Chat List */}
      {chatsQuery.isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : filteredChats.length === 0 ? (
        <Card className="shadow-soft">
          <CardContent className="py-12 text-center">
            <MessageCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">
              {searchQuery ? '没有找到匹配的对话' : '暂无消息'}
            </h3>
            <p className="text-muted-foreground text-sm">
              {searchQuery ? '尝试其他关键词' : '去认识新朋友吧'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {filteredChats.map((chat) => (
            <ChatItem key={chat.other_user.user_id} chat={chat} />
          ))}
        </div>
      )}
    </div>
  );
}

interface ChatItemProps {
  chat: PrivateChatListItem;
}

function ChatItem({ chat }: ChatItemProps) {
  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    if (diff < 60000) return '刚刚';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
    if (diff < 86400000) return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
    if (diff < 604800000) {
      const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
      return days[date.getDay()];
    }
    return date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' });
  };

  const getLastMessagePreview = () => {
    if (!chat.last_message) return '';
    const { type, content } = chat.last_message;
    switch (type) {
      case 'text':
        return content || '';
      case 'image':
        return '[图片]';
      case 'video':
        return '[视频]';
      case 'audio':
        return '[语音]';
      case 'file':
        return '[文件]';
      case 'location':
        return '[位置]';
      default:
        return '';
    }
  };

  return (
    <Link
      href={`/chat/${chat.other_user.user_id}`}
      className="flex items-center gap-3 p-4 rounded-xl hover:bg-muted/50 transition-colors"
    >
      {/* Avatar */}
      <div className="relative">
        <Avatar className="h-12 w-12">
          <AvatarImage src={chat.other_user.avatar_url} alt={chat.other_user.nickname} />
          <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground">
            {chat.other_user.nickname.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>
        {chat.is_online && (
          <span className="absolute bottom-0 right-0 w-3 h-3 bg-green-500 rounded-full ring-2 ring-background" />
        )}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between mb-1">
          <div className="flex items-center gap-2">
            <span className="font-medium truncate">{chat.other_user.nickname}</span>
            {chat.is_pinned && (
              <Pin className="h-3 w-3 text-muted-foreground" />
            )}
            {chat.is_muted && (
              <BellOff className="h-3 w-3 text-muted-foreground" />
            )}
          </div>
          <span className="text-xs text-muted-foreground shrink-0">
            {formatTime(chat.last_activity_at)}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground truncate">
            {getLastMessagePreview()}
          </p>
          {chat.unread_count > 0 && (
            <span className="ml-2 flex h-5 min-w-5 items-center justify-center rounded-full bg-primary px-1.5 text-xs font-medium text-primary-foreground">
              {chat.unread_count > 99 ? '99+' : chat.unread_count}
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}
