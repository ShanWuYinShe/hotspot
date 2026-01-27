'use client';

import { useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { Loader2, Plus } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';

import { FeedService } from '@/lib/api/services/feed';
import { MomentCard } from '@/components/cards/moment-card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { FeedSource } from '@/types/api';
import { useAuth } from '@/hooks/use-auth';
import { Card, CardContent } from '@/components/ui/card';
import { useLocation } from '@/hooks/use-location';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useLoginModal } from '@/providers/login-modal-provider';

export default function FeedPage() {
  const t = useTranslations('feed');
  const tCommon = useTranslations('common');
  const router = useRouter();
  const [source, setSource] = useState<FeedSource>('all');
  const { requireAuth } = useLoginModal();
  const { isAuthenticated } = useAuth();
  const { referencePoint, requestLocation } = useLocation();

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    status,
  } = useInfiniteQuery({
    queryKey: ['feed', source, referencePoint],
    queryFn: async ({ pageParam }: { pageParam?: string }) => {
      const res = await FeedService.getMoments({
        source,
        cursor: pageParam,
        reference_point: source === 'following' ? undefined : referencePoint ?? undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: source === 'following' || !!referencePoint,
  });

  const { ref } = useInfiniteScroll({
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  });

  return (
    <div className="max-w-2xl mx-auto pb-20 space-y-6">
      <div className="flex items-center justify-between sticky top-0 bg-background/95 backdrop-blur z-10 py-4">
        <h1 className="text-2xl font-bold">{t('title')}</h1>
        <Button size="sm" onClick={() => requireAuth(() => router.push('/moments/create'))}>
          <Plus className="h-4 w-4 mr-2" />
          {t('newMoment')}
        </Button>
      </div>

      <Tabs defaultValue="all" className="mb-2" onValueChange={(v) => setSource(v as FeedSource)}>
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="all">{t('tabs.all')}</TabsTrigger>
          <TabsTrigger value="following">{t('tabs.following')}</TabsTrigger>
          <TabsTrigger value="nearby">{t('tabs.nearby')}</TabsTrigger>
        </TabsList>
      </Tabs>

      {(source === 'all' || source === 'nearby') && !referencePoint ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground space-y-4">
            <p>{t('locationRequired')}</p>
            <Button onClick={requestLocation}>{tCommon('confirm')}</Button>
          </CardContent>
        </Card>
      ) : status === 'pending' ? (
        <div className="flex justify-center p-8">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : status === 'error' ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">{tCommon('retry')}</CardContent>
        </Card>
      ) : (
        <div className="space-y-6">
          {data?.pages.map((page, i) => (
            <div key={i} className="space-y-6">
              {page.items.map((moment) => (
                <MomentCard key={moment.moment_id} moment={moment} />
              ))}
            </div>
          ))}
          <div ref={ref} className="flex justify-center p-4">
            {isFetchingNextPage ? (
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            ) : null}
            {!hasNextPage && data?.pages[0]?.items.length ? (
              <span className="text-sm text-muted-foreground">{tCommon('empty')}</span>
            ) : null}
          </div>
        </div>
      )}
    </div>
  );
}
