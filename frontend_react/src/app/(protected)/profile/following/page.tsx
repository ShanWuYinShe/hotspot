'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as usersService from '@/lib/api/services/users';
import { isSuccess } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  ArrowLeft,
  Search,
  Loader2,
  Users,
  X,
  UserMinus,
} from 'lucide-react';
import type { UserInfo } from '@/types/api';

export default function FollowingPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = React.useState('');

  const followingQuery = useInfiniteQuery({
    queryKey: ['my-following'],
    queryFn: async ({ pageParam }) => {
      const res = await usersService.getMyFollowing({
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

  const following = followingQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];
  
  const filteredFollowing = React.useMemo(() => {
    if (!searchQuery) return following;
    const query = searchQuery.toLowerCase();
    return following.filter((f) =>
      f.user.nickname.toLowerCase().includes(query)
    );
  }, [following, searchQuery]);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold">关注</h1>
          <p className="text-muted-foreground text-sm">
            关注了 {following.length} 人
          </p>
        </div>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索关注的人..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10 pr-10 h-11 rounded-full bg-muted/50 border-0"
        />
        {searchQuery && (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-4 top-1/2 -translate-y-1/2 text-muted-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      {/* Following List */}
      {followingQuery.isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : filteredFollowing.length === 0 ? (
        <Card className="shadow-soft">
          <CardContent className="py-12 text-center">
            <Users className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">
              {searchQuery ? '没有找到' : '暂未关注任何人'}
            </h3>
            <p className="text-muted-foreground text-sm">
              {searchQuery ? '尝试其他关键词' : '去发现有趣的人吧'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {filteredFollowing.map((item) => (
            <FollowingCard key={item.user.user_id} user={item.user} />
          ))}

          {followingQuery.hasNextPage && (
            <div className="flex justify-center pt-4">
              <Button
                variant="outline"
                onClick={() => followingQuery.fetchNextPage()}
                disabled={followingQuery.isFetchingNextPage}
                className="rounded-full"
              >
                {followingQuery.isFetchingNextPage ? (
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

function FollowingCard({ user }: { user: UserInfo }) {
  const queryClient = useQueryClient();
  const [isFollowing, setIsFollowing] = React.useState(true);

  const unfollowMutation = useMutation({
    mutationFn: async () => {
      return usersService.unfollowUser(user.user_id);
    },
    onMutate: () => {
      setIsFollowing(false);
    },
    onError: () => {
      setIsFollowing(true);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-following'] });
    },
  });

  return (
    <div className="flex items-center gap-3 p-4 rounded-xl hover:bg-muted/50 transition-colors">
      <Link href={`/users/${user.user_id}`}>
        <Avatar className="h-12 w-12">
          <AvatarImage src={user.avatar_url} alt={user.nickname} />
          <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground">
            {user.nickname.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>
      </Link>
      <div className="flex-1 min-w-0">
        <Link href={`/users/${user.user_id}`} className="hover:text-primary transition-colors">
          <p className="font-medium truncate">{user.nickname}</p>
        </Link>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="rounded-full"
        onClick={() => unfollowMutation.mutate()}
        disabled={unfollowMutation.isPending}
      >
        {unfollowMutation.isPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : isFollowing ? (
          <>
            <UserMinus className="h-4 w-4 mr-1" />
            取消关注
          </>
        ) : (
          '已取消'
        )}
      </Button>
    </div>
  );
}
