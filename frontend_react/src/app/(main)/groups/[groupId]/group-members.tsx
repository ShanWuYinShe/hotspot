'use client';

/**
 * 群组成员列表 (Client Component)
 */

import Link from 'next/link';
import { Users, Crown, Shield } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/common/user-avatar';
import type { GroupMemberItem } from '@/types/api';

interface GroupMembersProps {
  groupId: string;
  members: GroupMemberItem[];
  totalCount: number;
}

export function GroupMembers({ groupId, members, totalCount }: GroupMembersProps) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            成员
          </CardTitle>
          <span className="text-sm text-muted-foreground">{totalCount}</span>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {members.map((member) => (
          <Link
            key={member.user.user_id}
            href={`/users/${member.user.user_id}`}
            className="flex items-center gap-3 p-2 rounded-lg hover:bg-accent/50 transition-colors"
          >
            <UserAvatar
              user={member.user}
              size="sm"
              showOnlineStatus
              isOnline={member.is_online}
            />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-medium text-sm truncate">
                  {member.user.nickname}
                </span>
                {member.role === 'owner' && (
                  <Crown className="h-3 w-3 text-yellow-500" />
                )}
                {member.role === 'admin' && (
                  <Shield className="h-3 w-3 text-blue-500" />
                )}
              </div>
            </div>
            {member.muted_until && (
              <Badge variant="outline" className="text-xs">
                禁言中
              </Badge>
            )}
          </Link>
        ))}

        {totalCount > members.length && (
          <Button variant="ghost" className="w-full" asChild>
            <Link href={`/groups/${groupId}/members`}>
              查看全部 {totalCount} 名成员
            </Link>
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
