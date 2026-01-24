/**
 * 我的粉丝列表 (Server Component)
 */

import Link from 'next/link';
import { serverApi, isServerSuccess, isAuthError, getServerErrorMessage } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/common/empty-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { UserAvatar } from '@/components/common/user-avatar';
import { Users } from '@/components/icons';
import { formatRelativeTime } from '@/lib/utils';
import { PageHeader } from '@/components/layout';
import { ServerErrorCard } from '@/components/common/server-error-card';
import type { FollowerItem, PaginatedData } from '@/types/api';

export default async function FollowersPage() {
  const response = await serverApi.post<PaginatedData<FollowerItem>>(
    ENDPOINTS.USER.ME_FOLLOWERS,
    { limit: 50 },
    { tags: ['profile-followers'], revalidate: 60 }
  );
  const header = (
    <PageHeader title="我的粉丝" description="关注你的用户列表" />
  );

  if (!isServerSuccess(response)) {
    if (isAuthError(response)) {
      return (
        <div className="max-w-2xl mx-auto space-y-6">
          {header}
          <LoginRequiredCard actionName="查看粉丝列表" description="登录后才能查看粉丝列表。" />
        </div>
      );
    }
    return (
      <div className="max-w-2xl mx-auto space-y-6">
        {header}
        <ServerErrorCard
          title="加载粉丝失败"
          description={getServerErrorMessage(response)}
          requestId={response.request_id}
        />
      </div>
    );
  }

  const items = response.data?.items ?? [];

  if (items.length === 0) {
    return (
      <div className="max-w-2xl mx-auto space-y-6">
        {header}
        <EmptyState icon={Users} title="暂无粉丝" description="分享动态吸引更多关注吧" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {header}
      <div className="space-y-3">
        {items.map((item) => (
          <Link key={item.user.user_id} href={`/users/${item.user.user_id}`}>
            <Card className="hover:bg-accent/50 transition-colors">
              <CardContent className="p-4 flex items-center gap-3">
                <UserAvatar user={item.user} showOnlineStatus isOnline={item.is_online} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{item.user.nickname}</span>
                    {item.is_following_back && <Badge variant="secondary">已回关</Badge>}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    关注于 {formatRelativeTime(item.followed_at)}
                  </p>
                </div>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  );
}
