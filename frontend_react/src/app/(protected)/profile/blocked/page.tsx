'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as usersService from '@/lib/api/services/users';
import { isSuccess } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  ArrowLeft,
  Loader2,
  Shield,
  UserX,
} from 'lucide-react';
import type { UserInfo } from '@/types/api';

export default function BlockedUsersPage() {
  const router = useRouter();

  const blockedQuery = useInfiniteQuery({
    queryKey: ['my-blocked'],
    queryFn: async ({ pageParam }) => {
      const res = await usersService.getMyBlocked({
        cursor: pageParam as string | undefined,
        limit: 20,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
  });

  const blocked = blockedQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold">黑名单</h1>
          <p className="text-muted-foreground text-sm">
            {blocked.length} 人
          </p>
        </div>
      </div>

      {/* Info */}
      <Card className="shadow-soft bg-muted/30">
        <CardContent className="py-4">
          <div className="flex items-start gap-3">
            <Shield className="h-5 w-5 text-muted-foreground mt-0.5" />
            <p className="text-sm text-muted-foreground">
              被拉黑的用户将无法查看你的内容、关注你或给你发送消息。
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Blocked List */}
      {blockedQuery.isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : blocked.length === 0 ? (
        <Card className="shadow-soft">
          <CardContent className="py-12 text-center">
            <UserX className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">黑名单为空</h3>
            <p className="text-muted-foreground text-sm">
              你没有屏蔽任何人
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {blocked.map((item) => (
            <BlockedUserCard key={item.user.user_id} user={item.user} />
          ))}

          {blockedQuery.hasNextPage && (
            <div className="flex justify-center pt-4">
              <Button
                variant="outline"
                onClick={() => blockedQuery.fetchNextPage()}
                disabled={blockedQuery.isFetchingNextPage}
                className="rounded-full"
              >
                {blockedQuery.isFetchingNextPage ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  '加载更多'
                )}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function BlockedUserCard({ user }: { user: UserInfo }) {
  const queryClient = useQueryClient();

  const unblockMutation = useMutation({
    mutationFn: async () => {
      return usersService.unblockUser(user.user_id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-blocked'] });
    },
  });

  return (
    <div className="flex items-center gap-3 p-4 rounded-xl hover:bg-muted/50 transition-colors">
      <Avatar className="h-12 w-12">
        <AvatarImage src={user.avatar_url} alt={user.nickname} />
        <AvatarFallback className="bg-muted text-muted-foreground">
          {user.nickname.slice(0, 2).toUpperCase()}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 min-w-0">
        <p className="font-medium truncate">{user.nickname}</p>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="rounded-full"
        onClick={() => unblockMutation.mutate()}
        disabled={unblockMutation.isPending}
      >
        {unblockMutation.isPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          '解除屏蔽'
        )}
      </Button>
    </div>
  );
}
