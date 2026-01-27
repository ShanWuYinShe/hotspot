'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Card, CardContent } from '@/components/ui/card';
import { PrivateChatListItem } from '@/types/api';

interface PrivateChatItemProps {
  chat: PrivateChatListItem;
}

export function PrivateChatItem({ chat }: PrivateChatItemProps) {
  const t = useTranslations('chat');
  const lastMessage = chat.last_message?.content ?? t('inputPlaceholder');

  return (
    <Link href={`/chat/users/${chat.other_user.user_id}`}>
      <Card className="hover:bg-accent/40 transition-colors cursor-pointer">
        <CardContent className="flex items-center gap-4 p-4">
          <Avatar className="h-12 w-12">
            <AvatarImage src={chat.other_user.avatar_url} alt={chat.other_user.nickname} />
            <AvatarFallback>{chat.other_user.nickname.slice(0, 2).toUpperCase()}</AvatarFallback>
          </Avatar>
          <div className="flex-1">
            <div className="flex items-center justify-between">
              <h3 className="font-semibold">{chat.other_user.nickname}</h3>
              {chat.unread_count > 0 ? (
                <span className="rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">
                  {chat.unread_count}
                </span>
              ) : null}
            </div>
            <p className="text-sm text-muted-foreground line-clamp-1">{lastMessage}</p>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
