/**
 * 审计日志表格 (Server Component)
 * 纯展示组件
 */

import { FileText } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
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
import { formatDateTime } from '@/lib/utils';
import type { AuditLogItem } from '@/types/api';

interface AuditLogTableProps {
  logs: AuditLogItem[];
}

const actionTypeMap: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  create: { label: '创建', variant: 'default' },
  update: { label: '更新', variant: 'secondary' },
  delete: { label: '删除', variant: 'destructive' },
  login: { label: '登录', variant: 'outline' },
  logout: { label: '登出', variant: 'outline' },
};

export function AuditLogTable({ logs }: AuditLogTableProps) {
  if (logs.length === 0) {
    return (
      <EmptyState
        icon={FileText}
        title="暂无审计日志"
        description="操作日志将在这里显示"
      />
    );
  }

  return (
    <Card>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>操作者</TableHead>
              <TableHead>操作类型</TableHead>
              <TableHead>目标</TableHead>
              <TableHead>详情</TableHead>
              <TableHead>IP 地址</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {logs.map((log) => {
              const actionInfo = actionTypeMap[log.action] || { label: log.action, variant: 'outline' as const };
              return (
                <TableRow key={log.log_id}>
                  <TableCell className="text-sm">
                    {formatDateTime(log.created_at)}
                  </TableCell>
                  <TableCell className="font-medium">
                    {log.admin_nickname || `管理员 ${log.admin_user_id}`}
                  </TableCell>
                  <TableCell>
                    <Badge variant={actionInfo.variant}>{actionInfo.label}</Badge>
                  </TableCell>
                  <TableCell className="text-sm">
                    {log.target_type}: {log.target_id}
                  </TableCell>
                  <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
                    {log.details ? JSON.stringify(log.details) : '-'}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {log.ip_address || '-'}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
