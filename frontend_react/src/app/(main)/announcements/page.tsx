'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { AnnouncementService } from '@/lib/api/services/announcements';
import { PaginatedData, SystemAnnouncementItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function AnnouncementsPage() {
  const t = useTranslations('notifications');
  const tCommon = useTranslations('common');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['announcements'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<SystemAnnouncementItem>> => {
      const res = await AnnouncementService.list({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const items = data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      {items.length > 0 ? (
        <div className="space-y-3">
          {items.map((item) => (
            <Card key={item.announcement_id}>
              <CardContent className="p-4 space-y-2">
                <div className="font-semibold">{item.title}</div>
                <p className="text-sm text-muted-foreground">{item.content}</p>
              </CardContent>
            </Card>
          ))}
          <div ref={ref} className="flex justify-center p-4">
            {isFetchingNextPage ? (
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
            ) : null}
          </div>
        </div>
      ) : (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">{tCommon('empty')}</CardContent>
        </Card>
      )}
    </div>
  );
}
