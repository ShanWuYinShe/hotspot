/**
 * 群成员列表页面 (Server Component)
 */

import Link from 'next/link';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/common/empty-state';
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { UserAvatar } from '@/components/common/user-avatar';
import { Users, Crown, Shield } from '@/components/icons';
import type { GroupMemberItem, PaginatedData } from '@/types/api';

interface GroupMembersPageProps {
  params: Promise<{ groupId: string }>;
}

export default async function GroupMembersPage({ params }: GroupMembersPageProps) {
  const { groupId } = await params;
  const response = await serverApi.get<PaginatedData<GroupMemberItem>>(
    ENDPOINTS.GROUP_CHATS.MEMBERS(groupId),
    { limit: 50 },
    { tags: [`group-${groupId}-members`], revalidate: 60 }
  );

  if (!isServerSuccess(response)) {
    return (
      <div className="max-w-2xl mx-auto">
        <LoginRequiredCard actionName="查看群成员" description="登录后才能查看群成员列表。" />
      </div>
    );
  }

  const members = response.data?.items ?? [];

  if (members.length === 0) {
    return (
      <div className="max-w-2xl mx-auto">
        <EmptyState icon={Users} title="暂无成员" description="该群组目前没有成员" />
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto space-y-3">
      {members.map((member) => (
        <Link key={member.user.user_id} href={`/users/${member.user.user_id}`}>
          <Card className="hover:bg-accent/50 transition-colors">
            <CardContent className="p-4 flex items-center gap-3">
              <UserAvatar user={member.user} showOnlineStatus isOnline={member.is_online} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{member.user.nickname}</span>
                  {member.role === 'owner' && (
                    <Badge variant="secondary">
                      <Crown className="h-3 w-3 mr-1" />
                      群主
                    </Badge>
                  )}
                  {member.role === 'admin' && (
                    <Badge variant="secondary">
                      <Shield className="h-3 w-3 mr-1" />
                      管理员
                    </Badge>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  );
}
