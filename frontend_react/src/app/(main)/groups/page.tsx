'use client';

import { useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { Plus } from 'lucide-react';

import { GroupCard } from '@/components/cards/group-card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Card, CardContent } from '@/components/ui/card';
import { GroupService } from '@/lib/api/services/groups';
import { FeedService } from '@/lib/api/services/feed';
import { useLocation } from '@/hooks/use-location';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { FeedGroupItem, MyGroupItem, PaginatedData } from '@/types/api';

export default function GroupsPage() {
  const t = useTranslations('groups');
  const tCommon = useTranslations('common');
  const [tab, setTab] = useState<'mine' | 'discover'>('mine');
  const { referencePoint, requestLocation } = useLocation();

  const {
    data: myGroupsData,
    fetchNextPage: fetchNextMyGroups,
    hasNextPage: hasNextMyGroups,
    isFetchingNextPage: isFetchingMyGroups,
  } = useInfiniteQuery({
    queryKey: ['groups', 'mine'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<MyGroupItem>> => {
      const res = await GroupService.myGroups({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const {
    data: discoverData,
    fetchNextPage: fetchNextDiscover,
    hasNextPage: hasNextDiscover,
    isFetchingNextPage: isFetchingDiscover,
  } = useInfiniteQuery({
    queryKey: ['groups', 'discover', referencePoint],
    queryFn: async ({ pageParam }): Promise<PaginatedData<FeedGroupItem>> => {
      if (!referencePoint) {
        return { items: [], pagination: { has_more: false } };
      }
      const res = await FeedService.getGroups({
        reference_point: referencePoint,
        cursor: pageParam as string | undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: tab === 'discover',
  });

  const myGroups = myGroupsData?.pages.flatMap((page) => page.items) ?? [];
  const discoverGroups = discoverData?.pages.flatMap((page) => page.items) ?? [];

  const { ref: myGroupsRef } = useInfiniteScroll({
    hasNextPage: hasNextMyGroups,
    isFetchingNextPage: isFetchingMyGroups,
    fetchNextPage: fetchNextMyGroups,
  });

  const { ref: discoverRef } = useInfiniteScroll({
    hasNextPage: hasNextDiscover,
    isFetchingNextPage: isFetchingDiscover,
    fetchNextPage: fetchNextDiscover,
  });

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{t('title')}</h1>
        <Link href="/groups/create">
          <Button size="sm">
            <Plus className="h-4 w-4 mr-2" />
            {t('create')}
          </Button>
        </Link>
      </div>

      <Tabs value={tab} onValueChange={(value) => setTab(value as typeof tab)}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="mine">{t('myGroups')}</TabsTrigger>
          <TabsTrigger value="discover">{t('discover')}</TabsTrigger>
        </TabsList>

        <TabsContent value="mine" className="space-y-3 mt-6">
          {myGroups.length > 0 ? (
            <>
              {myGroups.map((group) => (
                <GroupCard key={group.group_chat_id} group={group} />
              ))}
              <div ref={myGroupsRef} className="flex justify-center p-4">
                {isFetchingMyGroups ? (
                  <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
                ) : null}
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">{t('empty')}</CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="discover" className="space-y-3 mt-6">
          {!referencePoint ? (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground space-y-4">
                <p>{t('empty')}</p>
                <Button onClick={requestLocation}>{tCommon('confirm')}</Button>
              </CardContent>
            </Card>
          ) : discoverGroups.length > 0 ? (
            <>
              {discoverGroups.map((group) => (
                <GroupCard key={group.group_chat_id} group={group} />
              ))}
              <div ref={discoverRef} className="flex justify-center p-4">
                {isFetchingDiscover ? (
                  <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
                ) : null}
              </div>
            </>
          ) : (
            <Card>
              <CardContent className="py-12 text-center text-muted-foreground">{t('empty')}</CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
