'use client';

/**
 * 举报列表表格 (Client Component)
 * 处理举报操作
 */

import { useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { Flag, Check, X } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { EmptyState } from '@/components/common/empty-state';
import { useToast } from '@/hooks/use-toast';
import { adminService } from '@/lib/api/services/admin';
import { isSuccess } from '@/lib/api/client';
import { formatDateTime } from '@/lib/utils';
import type { ReportItem } from '@/types/api';

interface ReportTableProps {
  reports: ReportItem[];
}

const reportTypeMap: Record<string, string> = {
  user: '用户',
  moment: '瞬间',
  comment: '评论',
  message: '消息',
  group: '群组',
};

export function ReportTable({ reports }: ReportTableProps) {
  const { toast } = useToast();
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const handleResolve = async (reportId: string) => {
    startTransition(async () => {
      const response = await adminService.handleReport(reportId, {
        action: 'resolve',
      });
      if (isSuccess(response)) {
        toast.success('已处理', '举报已标记为已解决');
        router.refresh();
      } else {
        toast.error('处理失败', response.message);
      }
    });
  };

  const handleReject = async (reportId: string) => {
    startTransition(async () => {
      const response = await adminService.handleReport(reportId, {
        action: 'dismiss',
      });
      if (isSuccess(response)) {
        toast.success('已驳回', '举报已标记为无效');
        router.refresh();
      } else {
        toast.error('驳回失败', response.message);
      }
    });
  };

  if (reports.length === 0) {
    return (
      <EmptyState
        icon={Flag}
        title="暂无待处理举报"
        description="所有举报已处理完毕"
      />
    );
  }

  return (
    <Card>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>举报类型</TableHead>
              <TableHead>目标</TableHead>
              <TableHead>原因</TableHead>
              <TableHead>时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {reports.map((report) => (
              <TableRow key={report.report_id}>
                <TableCell>
                  <Badge variant="outline">
                    {reportTypeMap[report.target_type] || report.target_type}
                  </Badge>
                </TableCell>
                <TableCell className="font-medium">
                  {report.target_type}: {report.target_id}
                </TableCell>
                <TableCell className="max-w-xs truncate">
                  {report.reason}
                </TableCell>
                <TableCell className="text-sm">
                  {formatDateTime(report.created_at)}
                </TableCell>
                <TableCell className="text-right space-x-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleResolve(report.report_id)}
                    disabled={isPending}
                  >
                    <Check className="h-4 w-4 mr-1" />
                    处理
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleReject(report.report_id)}
                    disabled={isPending}
                  >
                    <X className="h-4 w-4 mr-1" />
                    驳回
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
