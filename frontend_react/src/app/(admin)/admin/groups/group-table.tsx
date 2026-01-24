'use client';

/**
 * 群组管理表格 (Client Component)
 * 处理搜索和操作交互
 */

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { Search, Ban, UsersRound } from '@/components/icons';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
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
import type { AdminGroupItem } from '@/types/api';

interface GroupTableProps {
  groups: AdminGroupItem[];
  initialKeyword?: string;
}

export function GroupTable({ groups, initialKeyword = '' }: GroupTableProps) {
  const { toast } = useToast();
  const router = useRouter();
  const [keyword, setKeyword] = useState(initialKeyword);
  const [isPending, startTransition] = useTransition();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [targetGroup, setTargetGroup] = useState<AdminGroupItem | null>(null);

  const handleSearch = () => {
    startTransition(() => {
      const params = new URLSearchParams();
      if (keyword) params.set('q', keyword);
      router.push(`/admin/groups?${params.toString()}`);
    });
  };

  const handleDissolve = async (groupId: string) => {
    const group = groups.find((item) => item.group_chat_id === groupId) ?? null;
    setTargetGroup(group);
    setConfirmOpen(true);
  };

  const handleConfirmDissolve = () => {
    if (!targetGroup) return;
    startTransition(async () => {
      const response = await adminService.dismissGroup(targetGroup.group_chat_id);
      if (isSuccess(response)) {
        toast.success('操作成功', '群组已解散');
        setConfirmOpen(false);
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
          <CardTitle>搜索群组</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="输入群组名称"
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

      {groups.length === 0 ? (
        <EmptyState
          icon={UsersRound}
          title="暂无群组"
          description="没有找到匹配的群组"
        />
      ) : (
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>群组</TableHead>
                  <TableHead>成员数</TableHead>
                  <TableHead>创建者</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {groups.map((group) => {
                  const initials = group.name.slice(0, 2).toUpperCase();
                  return (
                    <TableRow key={group.group_chat_id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar className="h-10 w-10">
                            {group.avatar_url && <AvatarImage src={group.avatar_url} />}
                            <AvatarFallback>{initials}</AvatarFallback>
                          </Avatar>
                          <div>
                            <p className="font-medium">{group.name}</p>
                            <p className="text-xs text-muted-foreground">
                              ID: {group.group_chat_id}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">{group.members_count}</Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        用户 {group.owner_id}
                      </TableCell>
                      <TableCell className="text-sm">
                        {formatDateTime(group.created_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={() => handleDissolve(group.group_chat_id)}
                        >
                          <Ban className="h-4 w-4 mr-1" />
                          解散
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>解散群组</DialogTitle>
            <DialogDescription>
              {targetGroup ? `确定要解散群组「${targetGroup.name}」吗？` : '确定要解散该群组吗？'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setConfirmOpen(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleConfirmDissolve} disabled={isPending}>
              {isPending ? '处理中...' : '确认解散'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
