'use client';

import { useParams } from 'next/navigation';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { GroupService } from '@/lib/api/services/groups';
import { GroupMemberItem, PaginatedData } from '@/types/api';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';

export default function GroupMembersPage() {
  const params = useParams();
  const groupId = params.groupId as string;
  const t = useTranslations('groups');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['group-members', groupId],
    queryFn: async ({ pageParam }): Promise<PaginatedData<GroupMemberItem>> => {
      const res = await GroupService.members(groupId, { cursor: pageParam as string | undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!groupId,
  });

  const members = data?.pages.flatMap((page) => page.items) ?? [];
  const { ref } = useInfiniteScroll({
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  });

  return (
    <div className="max-w-3xl mx-auto space-y-4">
      <h1 className="text-2xl font-bold">{t('members')}</h1>
      {members.length > 0 ? (
        <>
          {members.map((member) => (
            <Card key={member.user.user_id}>
              <CardContent className="flex items-center gap-4 p-4">
                <Avatar className="h-10 w-10">
                  <AvatarImage src={member.user.avatar_url} alt={member.user.nickname} />
                  <AvatarFallback>{member.user.nickname.slice(0, 2).toUpperCase()}</AvatarFallback>
                </Avatar>
                <div className="flex-1">
                  <div className="font-semibold">{member.user.nickname}</div>
                  <div className="text-xs text-muted-foreground">{member.role}</div>
                </div>
              </CardContent>
            </Card>
          ))}
          <div ref={ref} className="flex justify-center p-4">
            {isFetchingNextPage ? (
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-t-transparent border-current" />
            ) : null}
          </div>
        </>
      ) : (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">{t('empty')}</CardContent>
        </Card>
      )}
    </div>
  );
}
