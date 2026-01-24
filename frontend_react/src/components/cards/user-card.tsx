/**
 * 用户卡片 (混合组件)
 * 展示部分为 Server Component，交互按钮为 Client Component
 */

import Link from 'next/link';
import { MapPin, Users } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/common/user-avatar';
import { formatDistance } from '@/lib/utils';
import { FollowButton } from './follow-button';
import type { UserItem } from '@/types/api';

interface UserCardProps {
  user: UserItem;
  showFollowButton?: boolean;
}

export function UserCard({ user, showFollowButton = true }: UserCardProps) {
  const isOnline = user.is_online ?? false;
  const isMutual = user.relation?.is_mutual_follow ?? false;

  return (
    <Card className="hover:bg-accent/50 transition-colors">
      <CardContent className="p-4">
        <div className="flex items-start gap-4">
          <Link
            href={`/users/${user.user_id}`}
            className="flex flex-1 min-w-0 items-start gap-4"
          >
            <UserAvatar user={user} size="lg" showOnlineStatus isOnline={isOnline} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="font-medium truncate">{user.nickname}</h3>
                {isMutual && (
                  <Badge variant="secondary" className="text-xs">
                    <Users className="h-3 w-3 mr-1" />
                    互关
                  </Badge>
                )}
              </div>
              {user.bio && (
                <p className="text-sm text-muted-foreground line-clamp-2 mt-1">{user.bio}</p>
              )}
              <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                {user.location && user.location_meta?.visibility === 'visible' && (
                  <span className="flex items-center gap-1">
                    <MapPin className="h-3 w-3" />
                    {formatDistance(user.location.distance)}
                  </span>
                )}
                {user.tags && user.tags.length > 0 && (
                  <div className="flex items-center gap-1 flex-wrap">
                    {user.tags.slice(0, 3).map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </Link>
          {showFollowButton && (
            <FollowButton
              userId={user.user_id}
              isFollowing={user.relation?.is_following ?? false}
            />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
