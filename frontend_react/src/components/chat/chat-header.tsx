'use client';

/**
 * 聊天头部组件
 */

import { ArrowLeft, Phone, Video, MoreVertical, Users } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/common/user-avatar';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import type { UserInfo, GroupInfo } from '@/types/api';

interface ChatHeaderProps {
  type: 'private' | 'group';
  user?: UserInfo & { is_online?: boolean };
  group?: GroupInfo;
  isTyping?: boolean;
  onBack?: () => void;
  onViewProfile?: () => void;
}

export function ChatHeader({
  type,
  user,
  group,
  isTyping,
  onBack,
  onViewProfile,
}: ChatHeaderProps) {
  const title = type === 'private' ? user?.nickname : group?.name;
  const subtitle = type === 'private'
    ? isTyping
      ? '正在输入...'
      : user?.is_online
        ? '在线'
        : '离线'
    : `${group?.members_count ?? 0} 名成员`;

  return (
    <header className="flex items-center gap-3 px-4 py-3 border-b bg-background">
      <Button variant="ghost" size="icon" onClick={onBack}>
        <ArrowLeft className="h-5 w-5" />
      </Button>

      <button
        className="flex items-center gap-3 flex-1 min-w-0"
        onClick={onViewProfile}
      >
        {type === 'private' && user && (
          <UserAvatar
            user={user}
            showOnlineStatus
            isOnline={user.is_online}
          />
        )}
        {type === 'group' && group && (
          <Avatar>
            {group.avatar_url && <AvatarImage src={group.avatar_url} />}
            <AvatarFallback>
              <Users className="h-5 w-5" />
            </AvatarFallback>
          </Avatar>
        )}

        <div className="flex-1 min-w-0 text-left">
          <h1 className="font-semibold truncate">{title}</h1>
          <p className="text-xs text-muted-foreground">{subtitle}</p>
        </div>
      </button>

      <div className="flex items-center gap-1">
        <Button variant="ghost" size="icon">
          <Phone className="h-5 w-5" />
        </Button>
        <Button variant="ghost" size="icon">
          <Video className="h-5 w-5" />
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon">
              <MoreVertical className="h-5 w-5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onViewProfile}>
              查看{type === 'private' ? '资料' : '群信息'}
            </DropdownMenuItem>
            <DropdownMenuItem>搜索聊天记录</DropdownMenuItem>
            <DropdownMenuItem>清空聊天</DropdownMenuItem>
            {type === 'private' && (
              <DropdownMenuItem className="text-destructive">
                拉黑用户
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
