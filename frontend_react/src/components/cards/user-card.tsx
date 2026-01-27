'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { UserSearchItem } from '@/types/api';

interface UserCardProps {
  user: UserSearchItem;
  actionSlot?: React.ReactNode;
}

export function UserCard({ user, actionSlot }: UserCardProps) {
  const t = useTranslations('common');

  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <Avatar className="h-12 w-12">
          <AvatarImage src={user.avatar_url} alt={user.nickname} />
          <AvatarFallback>{user.nickname.slice(0, 2).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div className="flex-1">
          <Link href={`/users/${user.user_id}`} className="font-semibold hover:underline">
            {user.nickname}
          </Link>
          {user.bio ? <p className="text-sm text-muted-foreground line-clamp-1">{user.bio}</p> : null}
          {user.location?.location_name ? (
            <p className="text-xs text-muted-foreground mt-1">{user.location.location_name}</p>
          ) : null}
        </div>
        {actionSlot ?? (
          <Link href={`/users/${user.user_id}`} className="text-sm text-primary hover:underline">
            {t('view')}
          </Link>
        )}
      </CardContent>
    </Card>
  );
}
