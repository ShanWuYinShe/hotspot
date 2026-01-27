'use client';

import { useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { AdminGroupService } from '@/lib/api/services/admin/groups';
import { AdminGroupItem, PaginatedData } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function AdminGroupsPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');
  const [keyword, setKeyword] = useState('');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['admin-groups', keyword],
    queryFn: async ({ pageParam }): Promise<PaginatedData<AdminGroupItem>> => {
      const res = await AdminGroupService.search({
        cursor: pageParam as string | undefined,
        name: keyword || undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const groups = data?.pages.flatMap((page) => page.items) ?? [];

  const handleDismiss = async (groupChatId: string) => {
    const res = await AdminGroupService.dismiss({ group_chat_id: groupChatId });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('groups')}</h1>
      <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder={t('labels.search')} />
      {groups.length > 0 ? (
        <div className="space-y-3">
          {groups.map((group) => (
            <Card key={group.group_chat_id}>
              <CardContent className="flex items-center justify-between p-4">
                <div>
                  <div className="font-semibold">{group.name}</div>
                  <div className="text-xs text-muted-foreground">{group.group_chat_id}</div>
                </div>
                <Button variant="destructive" size="sm" onClick={() => handleDismiss(group.group_chat_id)}>
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
