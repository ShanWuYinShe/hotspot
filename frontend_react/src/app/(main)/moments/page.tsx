'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { MomentCard } from '@/components/cards/moment-card';
import { Card, CardContent } from '@/components/ui/card';
import { MomentService } from '@/lib/api/services/moments';
import { FeedMomentItem, PaginatedData, UserMomentListItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useAuthStore } from '@/stores/auth';

export default function MyMomentsPage() {
  const t = useTranslations('moments');
  const tCommon = useTranslations('common');
  const currentUser = useAuthStore((state) => state.user);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['my-moments'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<UserMomentListItem>> => {
      const res = await MomentService.listMyMoments({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const moments = data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('myTitle')}</h1>
      {moments.length > 0 ? (
        <>
          {moments.map((moment) => {
            const feedItem: FeedMomentItem = {
              ...moment,
              user: currentUser ?? {
                user_id: 0,
                nickname: tCommon('loading'),
                user_type: 'normal',
              },
              location: {
                ...moment.location,
                distance: 0,
                bearing_degrees: 0,
              },
              is_liked: false,
              is_favorited: false,
            };
            return <MomentCard key={moment.moment_id} moment={feedItem} />;
          })}
          <div ref={ref} className="flex justify-center p-4">
            {isFetchingNextPage ? (
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
            ) : null}
          </div>
        </>
      ) : (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">{tCommon('empty')}</CardContent>
        </Card>
      )}
    </div>
  );
}
