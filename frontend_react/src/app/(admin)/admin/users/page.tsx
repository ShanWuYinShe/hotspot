'use client';

import { useState } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { AdminUserService } from '@/lib/api/services/admin/users';
import { AdminUserItem, PaginatedData } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function AdminUsersPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');
  const [nickname, setNickname] = useState('');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['admin-users', nickname],
    queryFn: async ({ pageParam }): Promise<PaginatedData<AdminUserItem>> => {
      const res = await AdminUserService.search({
        cursor: pageParam as string | undefined,
        nickname: nickname || undefined,
      });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
  });

  const { ref } = useInfiniteScroll({ hasNextPage, isFetchingNextPage, fetchNextPage });
  const users = data?.pages.flatMap((page) => page.items) ?? [];

  const handleBan = async (userId: number) => {
    const res = await AdminUserService.ban({ user_id: userId });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const handleUnban = async (userId: number) => {
    const res = await AdminUserService.unban({ user_id: userId });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('users')}</h1>
      <Input value={nickname} onChange={(event) => setNickname(event.target.value)} placeholder={t('labels.search')} />
      {users.length > 0 ? (
        <div className="space-y-3">
          {users.map((user) => (
            <Card key={user.user_id}>
              <CardContent className="flex items-center justify-between p-4">
                <div>
                  <div className="font-semibold">{user.nickname}</div>
                  <div className="text-xs text-muted-foreground">{user.login_id}</div>
                </div>
                {user.status === 'banned' ? (
                  <Button variant="outline" size="sm" onClick={() => handleUnban(user.user_id)}>
                    {tCommon('edit')}
                  </Button>
                ) : (
                  <Button variant="destructive" size="sm" onClick={() => handleBan(user.user_id)}>
                    {tCommon('delete')}
                  </Button>
                )}
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
