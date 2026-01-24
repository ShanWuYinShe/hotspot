/**
 * 群组卡片 (混合组件)
 * 展示部分为 Server Component，交互按钮为 Client Component
 */

import Link from 'next/link';
import { MapPin, Users, LockKeyhole } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { formatDistance } from '@/lib/utils';
import { JoinGroupButton } from './join-group-button';
import type { GroupItem } from '@/types/api';

interface GroupCardProps {
  group: GroupItem;
  showJoinButton?: boolean;
}

export function GroupCard({ group, showJoinButton = true }: GroupCardProps) {
  const isJoined = group.my_state?.is_joined ?? false;
  const hasPassword = group.has_password;

  const initials = group.name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  return (
    <Card className="hover:bg-accent/50 transition-colors">
      <CardContent className="p-4">
        <div className="flex items-start gap-4">
          <Link
            href={`/groups/${group.group_chat_id}`}
            className="flex flex-1 min-w-0 items-start gap-4"
          >
            <Avatar className="h-12 w-12">
              {group.avatar_url ? (
                <AvatarImage src={group.avatar_url} alt={group.name} />
              ) : null}
              <AvatarFallback>{initials}</AvatarFallback>
            </Avatar>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="font-medium truncate">{group.name}</h3>
                {hasPassword && (
                  <LockKeyhole className="h-4 w-4 text-muted-foreground" />
                )}
              </div>
              {group.description && (
                <p className="text-sm text-muted-foreground line-clamp-2 mt-1">{group.description}</p>
              )}
              <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                <span className="flex items-center gap-1">
                  <Users className="h-3 w-3" />
                  {group.members_count}
                </span>
                {group.location && (
                  <span className="flex items-center gap-1">
                    <MapPin className="h-3 w-3" />
                    {formatDistance(group.location.distance)}
                  </span>
                )}
              </div>
              {group.tags && group.tags.length > 0 && (
                <div className="flex items-center gap-1 flex-wrap mt-2">
                  {group.tags.slice(0, 3).map((tag) => (
                    <Badge key={tag} variant="outline" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          </Link>
          {showJoinButton && (
            <JoinGroupButton
              groupId={group.group_chat_id}
              isJoined={isJoined}
              hasPassword={hasPassword}
            />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
