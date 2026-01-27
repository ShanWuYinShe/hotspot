'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { AdminReportService } from '@/lib/api/services/admin/reports';
import { PaginatedData, ReportItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function AdminReportsPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['admin-reports'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<ReportItem>> => {
      const res = await AdminReportService.list({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const reports = data?.pages.flatMap((page) => page.items) ?? [];

  const handleResolve = async (reportId: string) => {
    const res = await AdminReportService.handle({ report_id: reportId, action: 'resolve' });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('reports')}</h1>
      {reports.length > 0 ? (
        <div className="space-y-3">
          {reports.map((report) => (
            <Card key={report.report_id}>
              <CardContent className="flex items-center justify-between p-4">
                <div>
                  <div className="font-semibold">{report.reason}</div>
                  <div className="text-xs text-muted-foreground">{report.target_type}</div>
                </div>
                <Button size="sm" onClick={() => handleResolve(report.report_id)}>
                  {tCommon('confirm')}
                </Button>
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
