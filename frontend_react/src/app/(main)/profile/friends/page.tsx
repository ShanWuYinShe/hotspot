/**
 * 我的好友列表 (Server Component)
 */

import Link from 'next/link';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { Card, CardContent } from '@/components/ui/card';
import { EmptyState } from '@/components/common/empty-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { UserAvatar } from '@/components/common/user-avatar';
import { Users } from '@/components/icons';
import { formatRelativeTime } from '@/lib/utils';
import type { FriendListItem, PaginatedData } from '@/types/api';

export default async function FriendsPage() {
  const response = await serverApi.get<PaginatedData<FriendListItem>>(
    ENDPOINTS.USER.ME_FRIENDS,
    { limit: 50 },
    { tags: ['profile-friends'], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    return (
      <div className="max-w-2xl mx-auto">
        <LoginRequiredCard actionName="查看好友列表" description="登录后才能查看好友列表。" />
      </div>
    );
  }

  const items = response.data?.items ?? [];

  if (items.length === 0) {
    return (
      <div className="max-w-2xl mx-auto">
        <EmptyState icon={Users} title="暂无好友" description="先去关注和互动吧" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-3">
      {items.map((item) => (
        <Link key={item.user.user_id} href={`/users/${item.user.user_id}`}>
          <Card className="hover:bg-accent/50 transition-colors">
            <CardContent className="p-4 flex items-center gap-3">
              <UserAvatar user={item.user} showOnlineStatus isOnline={item.is_online} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{item.user.nickname}</span>
                </div>
                <p className="text-xs text-muted-foreground">
                  成为好友于 {formatRelativeTime(item.followed_back_at)}
                </p>
              </div>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  );
}
