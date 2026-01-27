'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { AdminAuditService } from '@/lib/api/services/admin/audit';
import { AuditLogItem, PaginatedData } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function AdminAuditPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['admin-audit'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<AuditLogItem>> => {
      const res = await AdminAuditService.list({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const logs = data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('audit')}</h1>
      {logs.length > 0 ? (
        <div className="space-y-3">
          {logs.map((log) => (
            <Card key={log.log_id}>
              <CardContent className="p-4">
                <div className="font-semibold">{log.action}</div>
                <div className="text-xs text-muted-foreground">{log.target_id}</div>
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
