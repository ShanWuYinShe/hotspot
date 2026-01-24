'use client';

/**
 * 用户资料头部 (Client Component)
 * 处理关注和私信交互
 */

import { useTransition, useOptimistic } from 'react';
import { useRouter } from 'next/navigation';
import { MapPin, MessageCircle, Users, Calendar, Settings } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/common/user-avatar';
import { useAuthStore } from '@/stores/auth';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { followUser, unfollowUser } from '@/lib/actions';
import { formatDateTime, formatDistance } from '@/lib/utils';
import type { UserDetailData } from '@/types/api';

interface UserProfileHeaderProps {
  user: UserDetailData;
}

export function UserProfileHeader({ user }: UserProfileHeaderProps) {
  const router = useRouter();
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  const currentUserId = useAuthStore((state) => state.currentAccount?.user_id);
  
  const isOwner = currentUserId === user.user_id;
  const isFollowing = user.relation?.is_following ?? false;
  const isMutual = user.relation?.is_mutual_follow ?? false;

  const [optimisticFollowing, setOptimisticFollowing] = useOptimistic(isFollowing);

  const handleFollow = () => {
    if (!requireAuth(optimisticFollowing ? '取消关注' : '关注用户')) {
      return;
    }
    startTransition(async () => {
      const previous = optimisticFollowing;
      const next = !previous;
      setOptimisticFollowing(next);
      const result = previous
        ? await unfollowUser(user.user_id)
        : await followUser(user.user_id);
      
      if (result.code !== 0) {
        setOptimisticFollowing(previous);
        toast.error('操作失败', result.message);
      }
    });
  };

  const handleMessage = () => {
    if (!requireAuth('发送私信')) {
      return;
    }
    router.push(`/chat/users/${user.user_id}`);
  };

  return (
    <Card>
      <CardContent className="p-6">
        {/* 头部信息 */}
        <div className="flex items-start gap-6">
          <UserAvatar
            user={user}
            size="xl"
            showOnlineStatus
            isOnline={user.is_online ?? false}
          />
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-2xl font-bold truncate">{user.nickname}</h1>
              {isMutual && (
                <Badge variant="secondary">
                  <Users className="h-3 w-3 mr-1" />
                  好友
                </Badge>
              )}
            </div>

            {user.bio && (
              <p className="text-muted-foreground mb-3">{user.bio}</p>
            )}

            {/* 标签 */}
            {user.tags && user.tags.length > 0 && (
              <div className="flex flex-wrap gap-1 mb-3">
                {user.tags.map((tag) => (
                  <Badge key={tag} variant="outline">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}

            {/* 位置和时间 */}
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              {user.location && user.location_meta?.visibility === 'visible' && (
                <span className="flex items-center gap-1">
                  <MapPin className="h-4 w-4" />
                  {formatDistance(user.location.distance)}
                </span>
              )}
              {user.created_at && (
                <span className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  {formatDateTime(user.created_at)}
                </span>
              )}
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex gap-2">
            {isOwner ? (
              <Button variant="outline" onClick={() => router.push('/settings')}>
                <Settings className="h-4 w-4 mr-2" />
                编辑资料
              </Button>
            ) : (
              <>
                <Button
                  variant={optimisticFollowing ? 'outline' : 'default'}
                  onClick={handleFollow}
                  disabled={isPending}
                >
                  {optimisticFollowing ? '已关注' : '关注'}
                </Button>
                <Button variant="outline" onClick={handleMessage}>
                  <MessageCircle className="h-4 w-4 mr-2" />
                  私信
                </Button>
              </>
            )}
          </div>
        </div>

        {/* 统计数据 */}
        {user.stats && (
          <div className="grid grid-cols-5 gap-4 mt-6 pt-6 border-t">
            <div className="text-center">
              <p className="text-2xl font-bold">{user.stats.moments_count}</p>
              <p className="text-sm text-muted-foreground">瞬间</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{user.stats.friends_count}</p>
              <p className="text-sm text-muted-foreground">好友</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{user.stats.following_count}</p>
              <p className="text-sm text-muted-foreground">关注</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{user.stats.followers_count}</p>
              <p className="text-sm text-muted-foreground">粉丝</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-bold">{user.stats.groups_count}</p>
              <p className="text-sm text-muted-foreground">群组</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
