/**
 * 举报管理页面 (Server Component)
 * 服务端获取举报列表
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { ReportTable } from './report-table';
import type { ReportItem, PaginatedData } from '@/types/api';

async function ReportsContent() {
  const response = await serverApi.get<PaginatedData<ReportItem>>(
    ENDPOINTS.ADMIN.REPORTS,
    { status: 'pending', limit: 50 },
    { tags: ['admin-reports'], revalidate: 60 }
  );

  const reports = isServerSuccess(response) ? response.data?.items ?? [] : [];

  return <ReportTable reports={reports} />;
}

export default async function AdminReportsPage() {
  const t = await getTranslations('admin');

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold">{t('reports')}</h1>
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <ReportsContent />
      </Suspense>
    </div>
  );
}
