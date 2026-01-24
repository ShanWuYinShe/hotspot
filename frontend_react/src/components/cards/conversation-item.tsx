/**
 * 会话列表项 (Server Component)
 * 纯展示组件，无交互逻辑
 */

import Link from 'next/link';
import { Pin, BellOff } from '@/components/icons';
import { UserAvatar } from '@/components/common/user-avatar';
import { formatRelativeTime, cn } from '@/lib/utils';
import type { PrivateChatListItem } from '@/types/api';

interface ConversationItemProps {
  conversation: PrivateChatListItem;
}

export function ConversationItem({ conversation }: ConversationItemProps) {
  const { other_user, unread_count, last_message, is_muted, is_pinned, is_online } = conversation;

  const lastMessageText = last_message?.content ?? (last_message?.type === 'image' ? '[图片]' : '[消息]');

  return (
    <Link href={`/chat/users/${other_user.user_id}`}>
      <div
        className={cn(
          'flex items-center gap-3 p-3 hover:bg-accent/50 transition-colors rounded-lg',
          is_pinned && 'bg-accent/30'
        )}
      >
        <UserAvatar user={other_user} showOnlineStatus isOnline={is_online} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <span className="font-medium text-sm truncate">{other_user.nickname}</span>
              {is_pinned && <Pin className="h-3 w-3 text-muted-foreground" />}
              {is_muted && <BellOff className="h-3 w-3 text-muted-foreground" />}
            </div>
            {last_message && (
              <span className="text-xs text-muted-foreground">
                {formatRelativeTime(last_message.sent_at)}
              </span>
            )}
          </div>
          <div className="flex items-center justify-between mt-0.5">
            <p className="text-sm text-muted-foreground truncate">{lastMessageText}</p>
            {unread_count > 0 && (
              <span className="flex items-center justify-center min-w-[18px] h-[18px] px-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-full">
                {unread_count > 99 ? '99+' : unread_count}
              </span>
            )}
          </div>
        </div>
      </div>
    </Link>
  );
}
