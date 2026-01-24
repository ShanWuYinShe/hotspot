/**
 * 用户头像组件 (Server Component)
 * 纯展示组件，客户端逻辑由 Avatar 组件处理
 */

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';
import type { UserInfo } from '@/types/api';

interface UserAvatarProps {
  user: Pick<UserInfo, 'nickname' | 'avatar_url'>;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
  showOnlineStatus?: boolean;
  isOnline?: boolean;
}

const sizeMap = {
  sm: 'h-8 w-8',
  md: 'h-10 w-10',
  lg: 'h-12 w-12',
  xl: 'h-16 w-16',
};

const onlineDotSizeMap = {
  sm: 'h-2 w-2',
  md: 'h-2.5 w-2.5',
  lg: 'h-3 w-3',
  xl: 'h-3.5 w-3.5',
};

export function UserAvatar({
  user,
  size = 'md',
  className,
  showOnlineStatus = false,
  isOnline = false,
}: UserAvatarProps) {
  const initials = user.nickname
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  return (
    <div className="relative inline-block">
      <Avatar className={cn(sizeMap[size], className)}>
        {user.avatar_url ? (
          <AvatarImage src={user.avatar_url} alt={user.nickname} />
        ) : null}
        <AvatarFallback className="text-xs font-medium">{initials}</AvatarFallback>
      </Avatar>
      {showOnlineStatus && (
        <span
          className={cn(
            'absolute bottom-0 right-0 rounded-full border-2 border-background',
            onlineDotSizeMap[size],
            isOnline ? 'bg-green-500' : 'bg-gray-400'
          )}
        />
      )}
    </div>
  );
}
