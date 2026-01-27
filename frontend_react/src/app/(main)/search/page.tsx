'use client';

import { useEffect, useState, startTransition } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { Search, ImageIcon, Users } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useTranslations } from 'next-intl';

import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { MomentCard } from '@/components/cards/moment-card';
import { GroupCard } from '@/components/cards/group-card';
import { UserCard } from '@/components/cards/user-card';
import { FeedMomentItem, GroupSearchItem, MomentSearchItem, PaginatedData, UserSearchItem } from '@/types/api';
import { SearchService } from '@/lib/api/services/search';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

const createSchema = (message: string) =>
  z.object({
    query: z.string().min(1, message),
  });

export default function SearchPage() {
  const t = useTranslations('searchPage');
  const tCommon = useTranslations('common');
  const [activeTab, setActiveTab] = useState<'all' | 'moments' | 'users' | 'groups'>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema(tCommon('empty'))),
    defaultValues: {
      query: '',
    },
  });

  const handleSubmit = (values: z.infer<ReturnType<typeof createSchema>>) => {
    setSearchQuery(values.query);
  };

  const {
    data: momentsData,
    fetchNextPage: fetchNextMomentPage,
    hasNextPage: hasNextMomentPage,
    isFetchingNextPage: isFetchingNextMomentPage,
  } = useInfiniteQuery({
    queryKey: ['search', 'moments', searchQuery],
    queryFn: async ({ pageParam }): Promise<PaginatedData<MomentSearchItem>> => {
      if (!searchQuery) {
        return { items: [], pagination: { has_more: false } };
      }
      const res = await SearchService.searchMomentsByKeyword({
        kw: searchQuery,
        cursor: pageParam ? String(pageParam) : undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!searchQuery,
  });

  const {
    data: usersData,
    fetchNextPage: fetchNextUserPage,
    hasNextPage: hasNextUserPage,
    isFetchingNextPage: isFetchingNextUserPage,
  } = useInfiniteQuery({
    queryKey: ['search', 'users', searchQuery],
    queryFn: async ({ pageParam }): Promise<PaginatedData<UserSearchItem>> => {
      if (!searchQuery) {
        return { items: [], pagination: { has_more: false } };
      }
      const res = await SearchService.searchUsersByKeyword({
        kw: searchQuery,
        cursor: pageParam ? String(pageParam) : undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!searchQuery,
  });

  const {
    data: groupsData,
    fetchNextPage: fetchNextGroupPage,
    hasNextPage: hasNextGroupPage,
    isFetchingNextPage: isFetchingNextGroupPage,
  } = useInfiniteQuery({
    queryKey: ['search', 'groups', searchQuery],
    queryFn: async ({ pageParam }): Promise<PaginatedData<GroupSearchItem>> => {
      if (!searchQuery) {
        return { items: [], pagination: { has_more: false } };
      }
      const res = await SearchService.searchGroupsByKeyword({
        kw: searchQuery,
        cursor: pageParam ? String(pageParam) : undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!searchQuery,
  });

  const moments = momentsData?.pages.flatMap((page) => page.items) ?? [];
  const users = usersData?.pages.flatMap((page) => page.items) ?? [];
  const groups = groupsData?.pages.flatMap((page) => page.items) ?? [];

  const { ref: momentRef } = useInfiniteScroll({
    hasNextPage: hasNextMomentPage,
    isFetchingNextPage: isFetchingNextMomentPage,
    fetchNextPage: fetchNextMomentPage,
  });

  const { ref: userRef } = useInfiniteScroll({
    hasNextPage: hasNextUserPage,
    isFetchingNextPage: isFetchingNextUserPage,
    fetchNextPage: fetchNextUserPage,
  });

  const { ref: groupRef } = useInfiniteScroll({
    hasNextPage: hasNextGroupPage,
    isFetchingNextPage: isFetchingNextGroupPage,
    fetchNextPage: fetchNextGroupPage,
  });

  useEffect(() => {
    if (!searchQuery) {
      startTransition(() => {
        setActiveTab('all');
      });
    }
  }, [searchQuery]);

  const renderMoments = () => (
    <>
      {moments.map((moment) => {
        const feedItem: FeedMomentItem = {
          ...moment,
          is_liked: false,
          is_favorited: false,
          location: {
            ...moment.location,
            distance: moment.location.distance ?? 0,
            bearing_degrees: 0,
          },
        };
        return <MomentCard key={moment.moment_id} moment={feedItem} />;
      })}
      <div ref={momentRef} className="flex justify-center p-4">
        {isFetchingNextMomentPage ? (
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
        ) : null}
      </div>
    </>
  );

  const renderUsers = () => (
    <>
      {users.map((user) => (
        <UserCard key={user.user_id} user={user} />
      ))}
      <div ref={userRef} className="flex justify-center p-4">
        {isFetchingNextUserPage ? (
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
        ) : null}
      </div>
    </>
  );

  const renderGroups = () => (
    <>
      {groups.map((group) => (
        <GroupCard key={group.group_chat_id} group={group} />
      ))}
      <div ref={groupRef} className="flex justify-center p-4">
        {isFetchingNextGroupPage ? (
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
        ) : null}
      </div>
    </>
  );

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">{t('title')}</h1>

      <form onSubmit={form.handleSubmit(handleSubmit)} className="mb-6">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('placeholder')}
            {...form.register('query')}
            className="pl-10"
          />
        </div>
      </form>

      {searchQuery ? (
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)}>
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="all">{t('tabs.all')}</TabsTrigger>
            <TabsTrigger value="moments">{t('tabs.moments')}</TabsTrigger>
            <TabsTrigger value="users">{t('tabs.users')}</TabsTrigger>
            <TabsTrigger value="groups">{t('tabs.groups')}</TabsTrigger>
          </TabsList>

          <TabsContent value="all" className="mt-6 space-y-8">
            {moments.length > 0 ? (
              <div>
                <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <ImageIcon className="h-5 w-5" />
                  {t('tabs.moments')}
                </h2>
                {renderMoments()}
              </div>
            ) : null}

            {users.length > 0 ? (
              <div>
                <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Users className="h-5 w-5" />
                  {t('tabs.users')}
                </h2>
                <div className="space-y-3">{renderUsers()}</div>
              </div>
            ) : null}

            {groups.length > 0 ? (
              <div>
                <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                  <Users className="h-5 w-5" />
                  {t('tabs.groups')}
                </h2>
                <div className="space-y-3">{renderGroups()}</div>
              </div>
            ) : null}

            {moments.length === 0 && users.length === 0 && groups.length === 0 ? (
              <Card>
                <CardContent className="py-12 text-center text-muted-foreground">
                  {t('empty')} &quot;{searchQuery}&quot;
                </CardContent>
              </Card>
            ) : null}
          </TabsContent>

          <TabsContent value="moments" className="mt-6 space-y-4">
            {moments.length > 0 ? (
              renderMoments()
            ) : (
              <Card>
                <CardContent className="py-12 text-center text-muted-foreground">
                  {t('empty')} &quot;{searchQuery}&quot;
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="users" className="mt-6 space-y-3">
            {users.length > 0 ? (
              renderUsers()
            ) : (
              <Card>
                <CardContent className="py-12 text-center text-muted-foreground">
                  {t('empty')} &quot;{searchQuery}&quot;
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="groups" className="mt-6 space-y-3">
            {groups.length > 0 ? (
              renderGroups()
            ) : (
              <Card>
                <CardContent className="py-12 text-center text-muted-foreground">
                  {t('empty')} &quot;{searchQuery}&quot;
                </CardContent>
              </Card>
            )}
          </TabsContent>
        </Tabs>
      ) : (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            <Search className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>{t('placeholder')}</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
