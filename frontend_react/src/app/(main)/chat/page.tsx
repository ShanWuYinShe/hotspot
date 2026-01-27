'use client';

import { useTranslations } from 'next-intl';
import { useInfiniteQuery } from '@tanstack/react-query';

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { PrivateChatItem } from '@/components/chat/private-chat-item';
import { GroupChatItem } from '@/components/chat/group-chat-item';
import { Card, CardContent } from '@/components/ui/card';
import { PrivateChatService } from '@/lib/api/services/chat';
import { GroupService } from '@/lib/api/services/groups';
import { PaginatedData, PrivateChatListItem, MyGroupItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function ChatPage() {
  const t = useTranslations('chat');
  const tGroups = useTranslations('groups');
  const tCommon = useTranslations('common');

  const {
    data: privateChatsData,
    fetchNextPage: fetchNextPrivate,
    hasNextPage: hasNextPrivate,
    isFetchingNextPage: isFetchingPrivate,
  } = useInfiniteQuery({
    queryKey: ['private-chats'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<PrivateChatListItem>> => {
      const res = await PrivateChatService.list({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const {
    data: groupsData,
    fetchNextPage: fetchNextGroups,
    hasNextPage: hasNextGroups,
    isFetchingNextPage: isFetchingGroups,
  } = useInfiniteQuery({
    queryKey: ['group-chats'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<MyGroupItem>> => {
      const res = await GroupService.myGroups({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const privateChats = privateChatsData?.pages.flatMap((page) => page.items) ?? [];
  const groups = groupsData?.pages.flatMap((page) => page.items) ?? [];

  const { ref: privateRef } = useInfiniteScroll({
    hasNextPage: hasNextPrivate,
    isFetchingNextPage: isFetchingPrivate,
    fetchNextPage: fetchNextPrivate,
  });

  const { ref: groupRef } = useInfiniteScroll({
    hasNextPage: hasNextGroups,
    isFetchingNextPage: isFetchingGroups,
    fetchNextPage: fetchNextGroups,
  });

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      <Tabs defaultValue="private">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="private">{t('title')}</TabsTrigger>
          <TabsTrigger value="groups">{tGroups('title')}</TabsTrigger>
        </TabsList>

        <TabsContent value="private" className="space-y-3 mt-6">
          {privateChats.length > 0 ? (
            <>
              {privateChats.map((chat) => (
                <PrivateChatItem key={chat.other_user.user_id} chat={chat} />
              ))}
              <div ref={privateRef} className="flex justify-center p-4">
                {isFetchingPrivate ? (
                  <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
                ) : null}
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                {t('empty')}
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="groups" className="space-y-3 mt-6">
          {groups.length > 0 ? (
            <>
              {groups.map((group) => (
                <GroupChatItem key={group.group_chat_id} group={group} />
              ))}
              <div ref={groupRef} className="flex justify-center p-4">
                {isFetchingGroups ? (
                  <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
                ) : null}
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">
                {tCommon('empty')}
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
