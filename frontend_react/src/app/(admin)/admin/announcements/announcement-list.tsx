'use client';

/**
 * 公告列表组件 (Client Component)
 * 处理公告管理操作
 */

import { Plus, Megaphone, Pencil, Trash } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { EmptyState } from '@/components/common/empty-state';
import { useToast } from '@/hooks/use-toast';
import { formatDateTime } from '@/lib/utils';
import type { AnnouncementItem } from '@/types/api';

interface AnnouncementListProps {
  announcements: AnnouncementItem[];
}

export function AnnouncementList({ announcements }: AnnouncementListProps) {
  const { toast } = useToast();

  const handleCreate = () => {
    toast.info('功能开发中', '创建公告功能即将上线');
  };

  const handleEdit = (_id: string) => {
    toast.info('功能开发中', '编辑公告功能即将上线');
  };

  const handleDelete = (_id: string) => {
    toast.info('功能开发中', '删除公告功能即将上线');
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-end">
        <Button onClick={handleCreate}>
          <Plus className="h-4 w-4 mr-2" />
          发布公告
        </Button>
      </div>

      {announcements.length === 0 ? (
        <EmptyState
          icon={Megaphone}
          title="暂无公告"
          description="点击上方按钮发布第一条公告"
        />
      ) : (
        <div className="grid gap-4">
          {announcements.map((announcement) => (
            <Card key={announcement.announcement_id}>
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <CardTitle className="text-lg">{announcement.title}</CardTitle>
                    <div className="flex items-center gap-2 text-sm text-muted-foreground">
                      <span>{formatDateTime(announcement.created_at)}</span>
                      <Badge variant={announcement.is_active ? 'default' : 'secondary'}>
                        {announcement.is_active ? '生效中' : '未启用'}
                      </Badge>
                      <Badge variant={
                        announcement.severity === 'error' ? 'destructive' :
                        announcement.severity === 'warning' ? 'secondary' : 'outline'
                      }>
                        {announcement.severity === 'error' ? '紧急' :
                         announcement.severity === 'warning' ? '警告' : '通知'}
                      </Badge>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleEdit(announcement.announcement_id)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDelete(announcement.announcement_id)}
                    >
                      <Trash className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground">{announcement.content}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
