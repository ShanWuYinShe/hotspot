'use client';

import { useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { AdminAnnouncementService } from '@/lib/api/services/admin/announcements';
import { PaginatedData, SystemAnnouncementItem } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function AdminAnnouncementsPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [severity, setSeverity] = useState<'info' | 'warning' | 'error'>('info');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['admin-announcements'],
    queryFn: async ({ pageParam }): Promise<PaginatedData<SystemAnnouncementItem>> => {
      const res = await AdminAnnouncementService.list({ cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const items = data?.pages.flatMap((page) => page.items) ?? [];

  const handleCreate = async () => {
    const res = await AdminAnnouncementService.create({ title, content, severity });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
      setTitle('');
      setContent('');
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const handleDelete = async (announcementId: string) => {
    const res = await AdminAnnouncementService.delete({ announcement_id: announcementId });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('announcements')}</h1>

      <Card>
        <CardContent className="space-y-3 p-4">
          <Input value={title} onChange={(event) => setTitle(event.target.value)} placeholder={t('labels.title')} />
          <Textarea value={content} onChange={(event) => setContent(event.target.value)} placeholder={t('labels.content')} />
          <Select value={severity} onValueChange={(value) => setSeverity(value as 'info' | 'warning' | 'error')}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="info">{t('severity.info')}</SelectItem>
              <SelectItem value="warning">{t('severity.warning')}</SelectItem>
              <SelectItem value="error">{t('severity.error')}</SelectItem>
            </SelectContent>
          </Select>
          <Button onClick={handleCreate}>{tCommon('create')}</Button>
        </CardContent>
      </Card>

      {items.length > 0 ? (
        <div className="space-y-3">
          {items.map((item) => (
            <Card key={item.announcement_id}>
              <CardContent className="flex items-center justify-between p-4">
                <div>
                  <div className="font-semibold">{item.title}</div>
                  <div className="text-xs text-muted-foreground">{item.severity}</div>
                </div>
                <Button variant="destructive" size="sm" onClick={() => handleDelete(item.announcement_id)}>
                  {tCommon('delete')}
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
