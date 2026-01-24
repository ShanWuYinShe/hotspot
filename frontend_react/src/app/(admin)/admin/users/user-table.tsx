'use client';

/**
 * 用户管理表格 (Client Component)
 * 处理搜索和操作交互
 */

import { useEffect, useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { Search, Ban, CheckCircle } from '@/components/icons';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { UserAvatar } from '@/components/common/user-avatar';
import { useToast } from '@/hooks/use-toast';
import { adminService } from '@/lib/api/services/admin';
import { isSuccess } from '@/lib/api/client';
import { formatDateTime } from '@/lib/utils';
import type { AdminUserItem } from '@/types/api';

interface UserTableProps {
  users: AdminUserItem[];
  initialKeyword?: string;
}

export function UserTable({ users, initialKeyword = '' }: UserTableProps) {
  const { toast } = useToast();
  const router = useRouter();
  const [keyword, setKeyword] = useState(initialKeyword);
  const [isPending, startTransition] = useTransition();
  const [banDialogOpen, setBanDialogOpen] = useState(false);
  const [banTarget, setBanTarget] = useState<AdminUserItem | null>(null);
  const [banReason, setBanReason] = useState('');
  const [banDays, setBanDays] = useState('');

  useEffect(() => {
    if (!banDialogOpen) {
      setBanReason('');
      setBanDays('');
      setBanTarget(null);
    }
  }, [banDialogOpen]);

  const handleSearch = () => {
    startTransition(() => {
      const params = new URLSearchParams();
      if (keyword) params.set('q', keyword);
      router.push(`/admin/users?${params.toString()}`);
    });
  };

  const handleBan = async (userId: number) => {
    const target = users.find((item) => item.user_id === userId) ?? null;
    setBanTarget(target);
    setBanDialogOpen(true);
  };

  const handleUnban = async (userId: number) => {
    startTransition(async () => {
      const response = await adminService.unbanUser(userId);
      if (isSuccess(response)) {
        toast.success('操作成功', '用户已解封');
        router.refresh();
      } else {
        toast.error('操作失败', response.message);
      }
    });
  };

  const handleConfirmBan = () => {
    if (!banTarget) return;
    let duration: number | undefined;
    if (banDays.trim()) {
      const parsed = Number(banDays);
      if (!Number.isFinite(parsed) || parsed <= 0) {
        toast.error('请输入有效的封禁天数');
        return;
      }
      duration = parsed;
    }

    startTransition(async () => {
      const response = await adminService.banUser(banTarget.user_id, {
        reason: banReason.trim() || undefined,
        duration_days: duration,
      });

      if (isSuccess(response)) {
        toast.success('操作成功', '用户已封禁');
        setBanDialogOpen(false);
        router.refresh();
      } else {
        toast.error('操作失败', response.message);
      }
    });
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>搜索用户</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="输入用户名、昵称或邮箱"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                className="pl-9"
                disabled={isPending}
              />
            </div>
            <Button onClick={handleSearch} disabled={isPending}>
              搜索
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>注册时间</TableHead>
                <TableHead>最后登录</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    暂无用户数据
                  </TableCell>
                </TableRow>
              ) : (
                users.map((user) => (
                  <TableRow key={user.user_id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <UserAvatar user={user} size="sm" />
                        <div>
                          <p className="font-medium">{user.nickname}</p>
                          <p className="text-xs text-muted-foreground">ID: {user.user_id}</p>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{user.user_type}</Badge>
                    </TableCell>
                    <TableCell>
                      {user.is_banned ? (
                        <Badge variant="destructive">已封禁</Badge>
                      ) : (
                        <Badge variant="secondary">正常</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-sm">
                      {formatDateTime(user.created_at)}
                    </TableCell>
                    <TableCell className="text-sm">
                      {user.last_login_at ? formatDateTime(user.last_login_at) : '-'}
                    </TableCell>
                    <TableCell className="text-right">
                      {user.is_banned ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleUnban(user.user_id)}
                        >
                          <CheckCircle className="h-4 w-4 mr-1" />
                          解封
                        </Button>
                      ) : (
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => handleBan(user.user_id)}
                        >
                          <Ban className="h-4 w-4 mr-1" />
                          封禁
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
      <Dialog open={banDialogOpen} onOpenChange={setBanDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>封禁用户</DialogTitle>
            <DialogDescription>
              {banTarget ? `为 ${banTarget.nickname} 设置封禁原因与期限` : '设置封禁原因与期限'}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <Textarea
              placeholder="封禁原因（可选）"
              value={banReason}
              onChange={(event) => setBanReason(event.target.value)}
              rows={3}
            />
            <Input
              type="number"
              min={1}
              placeholder="封禁天数（可选）"
              value={banDays}
              onChange={(event) => setBanDays(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setBanDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleConfirmBan} disabled={isPending}>
              {isPending ? '处理中...' : '确认封禁'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
