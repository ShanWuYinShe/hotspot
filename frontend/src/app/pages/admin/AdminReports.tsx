import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router';
import { ArrowLeft, Flag, CheckCircle, XCircle } from 'lucide-react';
import { motion } from 'motion/react';
import { toast } from 'sonner';
import { adminReportsService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import { useDocumentTitle } from '@/lib/hooks/use-document-title';
import { usePagination } from '@/lib/hooks/use-pagination';
import type { AdminReportItem, ReportStatus } from '@/lib/types/api';

const STATUS_OPTIONS: { label: string; value: ReportStatus | 'all' }[] = [
  { label: '全部', value: 'all' },
  { label: '待处理', value: 'pending' },
  { label: '已解决', value: 'resolved' },
  { label: '已驳回', value: 'dismissed' },
];

function statusColor(status: ReportStatus): string {
  switch (status) {
    case 'pending':
      return '#F59E0B';
    case 'reviewing':
      return '#6366F1';
    case 'resolved':
      return '#10B981';
    case 'dismissed':
      return '#9CA3AF';
  }
}

function statusLabel(status: ReportStatus): string {
  switch (status) {
    case 'pending':
      return '待处理';
    case 'reviewing':
      return '审核中';
    case 'resolved':
      return '已解决';
    case 'dismissed':
      return '已驳回';
  }
}

export function AdminReports() {
  const navigate = useNavigate();
  useDocumentTitle('举报处理');

  const [statusFilter, setStatusFilter] = useState<ReportStatus | 'all'>('all');
  const page = usePagination<AdminReportItem>({ uniqueKey: 'report_id' });

  const doSearch = useCallback(
    (refresh = true) => {
      page.load(
        (cursor) =>
          adminReportsService.search({
            status: statusFilter === 'all' ? undefined : statusFilter,
            pagination: { limit: 20, cursor },
          }),
        refresh,
      );
    },
    [page, statusFilter],
  );

  const handleResolve = useCallback(
    async (report: AdminReportItem) => {
      try {
        const res = await adminReportsService.handle({
          report_id: report.report_id,
          action: 'resolve',
        });
        if (isSuccess(res)) {
          toast.success('举报已处理');
          doSearch(true);
        } else {
          toast.error('处理失败');
        }
      } catch {
        toast.error('处理请求失败');
      }
    },
    [doSearch],
  );

  const handleDismiss = useCallback(
    async (report: AdminReportItem) => {
      try {
        const res = await adminReportsService.handle({
          report_id: report.report_id,
          action: 'dismiss',
        });
        if (isSuccess(res)) {
          toast.success('举报已驳回');
          doSearch(true);
        } else {
          toast.error('驳回失败');
        }
      } catch {
        toast.error('驳回请求失败');
      }
    },
    [doSearch],
  );

  return (
    <div className="flex flex-col min-h-screen" style={{ background: 'var(--neu-bg)' }}>
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: '1px solid var(--neu-border)' }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{
            width: 36,
            height: 36,
            background: 'var(--neu-bg)',
            boxShadow:
              '3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light)',
            border: '1px solid var(--neu-border)',
          }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: 'var(--neu-text-muted)' }}
            strokeWidth={1.5}
          />
        </motion.button>
        <p style={{ fontSize: 16, fontWeight: 600, color: 'var(--neu-text)' }}>
          举报处理
        </p>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex-1 px-4 py-4"
        style={{ maxWidth: 600, marginLeft: 'auto', marginRight: 'auto', width: '100%' }}
      >
        {/* Status filter */}
        <div className="flex gap-2 mb-4 overflow-x-auto pb-1">
          {STATUS_OPTIONS.map((opt) => (
            <motion.button
              key={opt.value}
              whileTap={{ scale: 0.94 }}
              onClick={() => {
                setStatusFilter(opt.value);
              }}
              className="px-3 py-1.5 rounded-full whitespace-nowrap flex-shrink-0"
              style={{
                fontSize: 12,
                background:
                  statusFilter === opt.value
                    ? 'var(--neu-accent-primary)'
                    : 'var(--neu-bg)',
                color:
                  statusFilter === opt.value
                    ? 'var(--text-on-accent, white)'
                    : 'var(--neu-text-muted)',
                boxShadow:
                  statusFilter === opt.value
                    ? '0 2px 8px rgba(0,0,0,0.15)'
                    : '3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)',
                border: '1px solid var(--neu-border)',
              }}
            >
              {opt.label}
            </motion.button>
          ))}
        </div>

        {page.error && (
          <p className="mb-3 text-center" style={{ fontSize: 13, color: '#EF4444' }}>
            {page.error}
          </p>
        )}

        {page.loading && (
          <div className="flex items-center justify-center py-16">
            <div
              className="rounded-full"
              style={{
                width: 28,
                height: 28,
                border: '3px solid var(--neu-border)',
                borderTopColor: 'var(--neu-accent-primary)',
                animation: 'spin 0.8s linear infinite',
              }}
            />
            <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
          </div>
        )}

        {!page.loading && page.items.length === 0 && (
          <div className="flex flex-col items-center py-16">
            <Flag
              style={{ width: 40, height: 40, color: 'var(--neu-text-muted)' }}
              strokeWidth={1.2}
            />
            <p className="mt-3" style={{ fontSize: 14, color: 'var(--neu-text-muted)' }}>
              暂无举报
            </p>
          </div>
        )}

        <div className="flex flex-col gap-2">
          {page.items.map((report) => (
            <div
              key={report.report_id}
              className="px-4 py-3 rounded-[16px]"
              style={{
                background: 'var(--neu-bg)',
                boxShadow:
                  '4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)',
                border: '1px solid var(--neu-border)',
              }}
            >
              {/* Top row */}
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span
                    className="px-2 py-0.5 rounded-full"
                    style={{
                      fontSize: 11,
                      background: `${statusColor(report.status)}20`,
                      color: statusColor(report.status),
                    }}
                  >
                    {statusLabel(report.status)}
                  </span>
                  <span style={{ fontSize: 12, color: 'var(--neu-text-muted)' }}>
                    {report.target_type}
                  </span>
                </div>
                <span style={{ fontSize: 11, color: 'var(--neu-text-muted)' }}>
                  {new Date(report.created_at).toLocaleDateString()}
                </span>
              </div>

              {/* Reporter & reason */}
              <p className="mb-1" style={{ fontSize: 13, color: 'var(--neu-text)' }}>
                <span style={{ color: 'var(--neu-text-muted)' }}>举报者：</span>
                {report.reporter?.nickname ?? '未知'}
              </p>
              <p
                className="mb-2 line-clamp-2"
                style={{ fontSize: 12, color: 'var(--neu-text-muted)' }}
              >
                {report.description ?? report.reason}
              </p>

              {/* Actions */}
              {report.status === 'pending' && (
                <div className="flex items-center gap-2">
                  <motion.button
                    whileTap={{ scale: 0.94 }}
                    onClick={() => handleResolve(report)}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-full"
                    style={{
                      background: '#10B98118',
                      fontSize: 12,
                      color: '#10B981',
                    }}
                  >
                    <CheckCircle style={{ width: 13, height: 13 }} strokeWidth={1.5} />
                    处理
                  </motion.button>
                  <motion.button
                    whileTap={{ scale: 0.94 }}
                    onClick={() => handleDismiss(report)}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-full"
                    style={{
                      background: '#9CA3AF18',
                      fontSize: 12,
                      color: '#9CA3AF',
                    }}
                  >
                    <XCircle style={{ width: 13, height: 13 }} strokeWidth={1.5} />
                    驳回
                  </motion.button>
                </div>
              )}
            </div>
          ))}
        </div>

        {page.hasMore && !page.loading && page.items.length > 0 && (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => doSearch(false)}
            disabled={page.loadingMore}
            className="mt-4 py-2.5 rounded-[14px] w-full"
            style={{
              background: 'var(--neu-bg)',
              boxShadow:
                '4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)',
              border: '1px solid var(--neu-border)',
              fontSize: 13,
              color: 'var(--neu-text-muted)',
            }}
          >
            {page.loadingMore ? '加载中...' : '加载更多'}
          </motion.button>
        )}
      </motion.div>
    </div>
  );
}
