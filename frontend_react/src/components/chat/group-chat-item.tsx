'use client';

import Link from 'next/link';
import { Users } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { MyGroupItem } from '@/types/api';

interface GroupChatItemProps {
  group: MyGroupItem;
}

export function GroupChatItem({ group }: GroupChatItemProps) {
  const t = useTranslations('groups');

  return (
    <Link href={`/chat/groups/${group.group_chat_id}`}>
      <Card className="hover:bg-accent/40 transition-colors cursor-pointer">
        <CardContent className="flex items-center gap-4 p-4">
          <div className="h-12 w-12 rounded-lg bg-primary/10 flex items-center justify-center">
            <Users className="h-5 w-5 text-primary" />
          </div>
          <div className="flex-1">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold">{group.name}</h3>
              {group.unread_count > 0 ? (
                <span className="rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">
                  {group.unread_count}
                </span>
              ) : null}
            </div>
            <p className="text-sm text-muted-foreground line-clamp-1">{t('members')} {group.members_count}</p>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
