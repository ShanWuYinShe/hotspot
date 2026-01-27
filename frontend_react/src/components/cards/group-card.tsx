'use client';

import Link from 'next/link';
import { Users } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { FeedGroupItem, GroupSearchItem, MyGroupItem } from '@/types/api';

interface GroupCardProps {
  group: GroupSearchItem | MyGroupItem | FeedGroupItem;
  actionSlot?: React.ReactNode;
}

export function GroupCard({ group, actionSlot }: GroupCardProps) {
  const t = useTranslations('groups');
  const tCommon = useTranslations('common');

  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <div className="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center">
          <Users className="h-5 w-5 text-primary" />
        </div>
        <div className="flex-1">
          <Link href={`/groups/${group.group_chat_id}`} className="font-semibold hover:underline">
            {group.name}
          </Link>
          {group.description ? (
            <p className="text-sm text-muted-foreground line-clamp-1">{group.description}</p>
          ) : null}
          <div className="text-xs text-muted-foreground mt-1">
            {group.members_count} {t('members')}
          </div>
        </div>
        {actionSlot ?? (
          <Link href={`/groups/${group.group_chat_id}`} className="text-sm text-primary hover:underline">
            {tCommon('view')}
          </Link>
        )}
      </CardContent>
    </Card>
  );
}
