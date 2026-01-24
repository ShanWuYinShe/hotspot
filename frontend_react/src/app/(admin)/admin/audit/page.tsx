/**
 * 审计日志页面 (Server Component)
 * 服务端获取审计日志
 */

import { Suspense } from 'react';
import { getTranslations } from 'next-intl/server';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { AuditLogTable } from './audit-log-table';
import { PageHeader } from '@/components/layout';
import type { AuditLogItem, PaginatedData } from '@/types/api';

async function AuditContent() {
  const response = await serverApi.post<PaginatedData<AuditLogItem>>(
    ENDPOINTS.ADMIN.AUDIT_LOGS,
    { limit: 100 },
    { tags: ['audit-logs'], revalidate: 30 }
  );

  const logs = isServerSuccess(response) ? response.data?.items ?? [] : [];

  return <AuditLogTable logs={logs} />;
}

export default async function AdminAuditPage() {
  const t = await getTranslations('admin');

  return (
    <div className="space-y-6">
      <PageHeader title={t('audit')} description="管理员操作与变更记录" />
      
      <Suspense fallback={<LoadingState text="加载中..." />}>
        <AuditContent />
      </Suspense>
    </div>
  );
}
