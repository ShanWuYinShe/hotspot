'use client';

/**
 * 群公告列表 (Client Component)
 */

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { Megaphone, Pin, Plus, Pencil, Trash } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { EmptyState } from '@/components/common/empty-state';
import { formatRelativeTime } from '@/lib/utils';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { groupsService } from '@/lib/api/services/groups';
import { isSuccess } from '@/lib/api/client';
import type { GroupAnnouncementItem } from '@/types/api';

interface GroupAnnouncementsProps {
  groupId: string;
  announcements: GroupAnnouncementItem[];
  isAdmin: boolean;
}

export function GroupAnnouncements({
  groupId,
  announcements,
  isAdmin,
}: GroupAnnouncementsProps) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const router = useRouter();
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<GroupAnnouncementItem | null>(null);
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [isPinned, setIsPinned] = useState(false);

  const openCreateDialog = () => {
    setEditing(null);
    setTitle('');
    setContent('');
    setIsPinned(false);
    setDialogOpen(true);
  };

  const openEditDialog = (announcement: GroupAnnouncementItem) => {
    setEditing(announcement);
    setTitle(announcement.title);
    setContent(announcement.content);
    setIsPinned(announcement.is_pinned);
    setDialogOpen(true);
  };

  const handleSubmit = () => {
    if (!requireAuth(editing ? '编辑群公告' : '发布群公告')) return;
    if (!title.trim() || !content.trim()) {
      toast.error('请填写标题和内容');
      return;
    }

    startTransition(async () => {
      const response = editing
        ? await groupsService.updateAnnouncement(editing.announcement_id, {
            title: title.trim(),
            content: content.trim(),
            is_pinned: isPinned,
          })
        : await groupsService.createAnnouncement(groupId, {
            title: title.trim(),
            content: content.trim(),
            is_pinned: isPinned,
          });

      if (response.code === 0) {
        toast.success(editing ? '修改成功' : '发布成功');
        setDialogOpen(false);
        router.refresh();
      } else {
        toast.error('操作失败', response.message);
      }
    });
  };

  const handleDelete = (announcementId: string) => {
    if (!requireAuth('删除群公告')) return;
    startTransition(async () => {
      const response = await groupsService.deleteAnnouncement(announcementId);
      if (isSuccess(response)) {
        toast.success('已删除');
        router.refresh();
      } else {
        toast.error('删除失败', response.message);
      }
    });
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Megaphone className="h-5 w-5" />
            群公告
          </CardTitle>
          {isAdmin && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                openCreateDialog();
              }}
            >
              <Plus className="h-4 w-4 mr-1" />
              发布
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {announcements.length === 0 ? (
          <EmptyState
            icon={Megaphone}
            title="暂无公告"
            description="群管理员可以在这里发布公告"
          />
        ) : (
          <div className="space-y-4">
            {announcements.map((announcement) => (
              <div
                key={announcement.announcement_id}
                className="p-4 border rounded-lg space-y-2"
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <h3 className="font-medium">{announcement.title}</h3>
                    {announcement.is_pinned && (
                      <Badge variant="secondary" className="text-xs">
                        <Pin className="h-3 w-3 mr-1" />
                        置顶
                      </Badge>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">
                      {formatRelativeTime(announcement.created_at)}
                    </span>
                    {isAdmin && (
                      <>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          onClick={() => openEditDialog(announcement)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-destructive"
                          onClick={() => handleDelete(announcement.announcement_id)}
                        >
                          <Trash className="h-4 w-4" />
                        </Button>
                      </>
                    )}
                  </div>
                </div>
                <p
                  className={`text-sm text-muted-foreground ${
                    expanded !== announcement.announcement_id && 'line-clamp-2'
                  }`}
                >
                  {announcement.content}
                </p>
                {announcement.content.length > 100 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-auto p-0 text-primary"
                    onClick={() =>
                      setExpanded(
                        expanded === announcement.announcement_id
                          ? null
                          : announcement.announcement_id
                      )
                    }
                  >
                    {expanded === announcement.announcement_id ? '收起' : '展开'}
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}
      </CardContent>
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? '编辑公告' : '发布公告'}</DialogTitle>
            <DialogDescription>公告将展示给群成员</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Input
                placeholder="公告标题"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Textarea
                placeholder="公告内容"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                rows={4}
              />
            </div>
            <div className="flex items-center justify-between text-sm">
              <span>是否置顶</span>
              <Button variant={isPinned ? 'default' : 'outline'} onClick={() => setIsPinned(!isPinned)}>
                {isPinned ? '已置顶' : '不置顶'}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSubmit} disabled={isPending}>
              {isPending ? '提交中...' : '确认'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
