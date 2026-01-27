'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { AuthService } from '@/lib/api/services/auth';
import { PaginatedData, SessionItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function SessionsPage() {
  const t = useTranslations('settings');
  const tCommon = useTranslations('common');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['sessions'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<SessionItem>> => {
      const res = await AuthService.sessions({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const sessions = data?.pages.flatMap((page) => page.items) ?? [];

  const handleRevoke = async (sessionId: string) => {
    const res = await AuthService.revokeSession({ session_id: sessionId });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-4">
      <h1 className="text-2xl font-bold">{t('sections.sessions')}</h1>
      {sessions.length > 0 ? (
        <>
          {sessions.map((session) => (
            <Card key={session.session_id}>
              <CardContent className="flex items-center justify-between p-4">
                <div>
                  <div className="font-semibold">{session.device_name ?? session.device_type ?? tCommon('unknown')}</div>
                  <div className="text-sm text-muted-foreground">{session.ip_location ?? session.ip_address}</div>
                </div>
                {!session.is_current ? (
                  <Button variant="outline" size="sm" onClick={() => handleRevoke(session.session_id)}>
                    {tCommon('delete')}
                  </Button>
                ) : (
                  <span className="text-xs text-primary">{t('labels.currentSession')}</span>
                )}
              </CardContent>
            </Card>
          ))}
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
