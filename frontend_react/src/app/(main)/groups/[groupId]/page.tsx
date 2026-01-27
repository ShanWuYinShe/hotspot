'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { toast } from 'sonner';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { GroupService } from '@/lib/api/services/groups';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function GroupDetailPage() {
  const params = useParams();
  const router = useRouter();
  const groupId = params.groupId as string;
  const t = useTranslations('groups');
  const tCommon = useTranslations('common');
  const [password, setPassword] = useState('');
  const [passwordOpen, setPasswordOpen] = useState(false);

  const { data: group } = useQuery({
    queryKey: ['group', groupId],
    queryFn: async () => {
      const res = await GroupService.openTheDoor(groupId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!groupId,
  });

  const { data: settings } = useQuery({
    queryKey: ['group-settings', groupId],
    queryFn: async () => {
      const res = await GroupService.settings(groupId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!groupId,
  });

  const {
    data: announcementsData,
    fetchNextPage: fetchNextAnnouncements,
    hasNextPage: hasNextAnnouncements,
    isFetchingNextPage: isFetchingAnnouncements,
  } = useInfiniteQuery({
    queryKey: ['group-announcements', groupId],
    queryFn: async ({ pageParam }) => {
      const res = await GroupService.announcements(groupId, { cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!groupId,
  });

  const { ref: announcementsRef } = useInfiniteScroll({
    hasNextPage: hasNextAnnouncements,
    isFetchingNextPage: isFetchingAnnouncements,
    fetchNextPage: fetchNextAnnouncements,
  });

  const announcements = announcementsData?.pages.flatMap((page) => page.items) ?? [];

  const handleJoin = async () => {
    if (!settings) {
      return;
    }
    if (settings.settings.join_method === 'approval') {
      const res = await GroupService.apply({ group_chat_id: groupId });
      if (res.code === 0) {
        toast.success(tCommon('submit'));
      } else {
        toast.error(res.message || tCommon('retry'));
      }
      return;
    }
    if (settings.has_password) {
      setPasswordOpen(true);
      return;
    }
    const res = await GroupService.join({ group_chat_id: groupId });
    if (res.code === 0) {
      toast.success(tCommon('submit'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const handleJoinWithPassword = async () => {
    const res = await GroupService.join({ group_chat_id: groupId, password });
    if (res.code === 0) {
      toast.success(tCommon('submit'));
      setPasswordOpen(false);
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const handleLeave = async () => {
    const res = await GroupService.leave({ group_chat_id: groupId });
    if (res.code === 0) {
      toast.success(tCommon('submit'));
      router.push('/groups');
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{group?.name ?? t('title')}</h1>
        {group?.my_state?.is_joined ? (
          <Button variant="outline" onClick={handleLeave}>
            {t('leave')}
          </Button>
        ) : (
          <Button onClick={handleJoin}>{t('join')}</Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t('announcements')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-sm text-muted-foreground">{group?.description ?? tCommon('empty')}</p>
          {group?.tags?.length ? (
            <div className="flex flex-wrap gap-2">
              {group.tags.map((tag) => (
                <span key={tag} className="rounded-full bg-muted px-3 py-1 text-xs">
                  {tag}
                </span>
              ))}
            </div>
          ) : null}
          {announcements.length > 0 ? (
            <div className="space-y-2 pt-2">
              {announcements.map((item) => (
                <div key={item.announcement_id} className="rounded-lg border p-3 text-sm">
                  <div className="font-semibold">{item.title}</div>
                  <div className="text-muted-foreground">{item.content}</div>
                </div>
              ))}
              <div ref={announcementsRef} className="flex justify-center p-2">
                {isFetchingAnnouncements ? (
                  <div className="h-5 w-5 animate-spin rounded-full border-2 border-t-transparent border-current" />
                ) : null}
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{tCommon('empty')}</p>
          )}
        </CardContent>
      </Card>

      <div className="flex flex-wrap gap-3">
        <Button variant="outline" onClick={() => router.push(`/groups/${groupId}/members`)}>
          {t('members')}
        </Button>
        <Button variant="outline" onClick={() => router.push(`/groups/${groupId}/settings`)}>
          {t('settings')}
        </Button>
      </div>

      <Dialog open={passwordOpen} onOpenChange={setPasswordOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('password')}</DialogTitle>
          </DialogHeader>
          <Input value={password} onChange={(event) => setPassword(event.target.value)} placeholder={t('enterPassword')} />
          <DialogFooter>
            <Button onClick={handleJoinWithPassword}>{tCommon('confirm')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
