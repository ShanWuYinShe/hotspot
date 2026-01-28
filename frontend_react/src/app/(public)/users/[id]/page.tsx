'use client';

import * as React from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useInfiniteQuery, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import * as usersService from '@/lib/api/services/users';
import { isSuccess } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  ArrowLeft,
  Loader2,
  MapPin,
  Users,
  Heart,
  MessageCircle,
  UserPlus,
  UserMinus,
  UserCheck,
  Clock,
  Image as ImageIcon,
} from 'lucide-react';
import type { UserMomentListItem } from '@/types/api';

export default function UserDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { isAuthenticated, user: currentUser } = useAuthStore();
  const userId = Number(params.id);

  const isOwnProfile = currentUser?.user_id === userId;

  // Fetch user
  const userQuery = useQuery({
    queryKey: ['user', userId],
    queryFn: async () => {
      const res = await usersService.getUser(userId);
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  // Fetch user moments
  const momentsQuery = useInfiniteQuery({
    queryKey: ['user-moments', userId],
    queryFn: async ({ pageParam }) => {
      const res = await usersService.getUserMoments({
        user_id: userId,
        cursor: pageParam as string | undefined,
        limit: 12,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
  });

  // Follow mutation
  const followMutation = useMutation({
    mutationFn: async () => {
      if (userQuery.data?.relation?.is_following) {
        return usersService.unfollowUser(userId);
      } else {
        return usersService.followUser({ user_id: userId });
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', userId] });
    },
  });

  const user = userQuery.data;
  const moments = momentsQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  if (userQuery.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  if (userQuery.error || !user) {
    return (
      <Card className="max-w-2xl mx-auto shadow-soft">
        <CardContent className="py-12 text-center">
          <h2 className="text-lg font-semibold mb-2">用户不存在</h2>
          <p className="text-muted-foreground mb-4">该用户可能已注销</p>
          <Button variant="outline" onClick={() => router.back()}>
            返回
          </Button>
        </CardContent>
      </Card>
    );
  }

  const renderFollowButton = () => {
    if (!isAuthenticated || isOwnProfile) return null;

    const isFollowing = user.relation?.is_following;
    const isMutual = user.relation?.is_mutual_follow;

    return (
      <Button
        onClick={() => followMutation.mutate()}
        disabled={followMutation.isPending}
        variant={isFollowing ? 'outline' : 'default'}
        className="rounded-full gap-2"
      >
        {followMutation.isPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : isFollowing ? (
          <>
            {isMutual ? <UserCheck className="h-4 w-4" /> : <UserMinus className="h-4 w-4" />}
            {isMutual ? '互相关注' : '已关注'}
          </>
        ) : (
          <>
            <UserPlus className="h-4 w-4" />
            关注
          </>
        )}
      </Button>
    );
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Back Button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.back()}
        className="gap-2"
      >
        <ArrowLeft className="h-4 w-4" />
        返回
      </Button>

      {/* Profile Card */}
      <Card className="shadow-soft overflow-hidden">
        {/* Cover */}
        <div className="h-32 bg-gradient-to-br from-primary/30 via-primary/20 to-primary/10" />

        <CardContent className="relative pt-0 pb-6">
          {/* Avatar */}
          <div className="absolute -top-16 left-6">
            <Avatar className="h-32 w-32 ring-4 ring-background shadow-xl">
              <AvatarImage src={user.avatar_url} alt={user.nickname} />
              <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-3xl font-semibold">
                {user.nickname.slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>
          </div>

          {/* Actions (desktop position) */}
          <div className="flex justify-end pt-4">
            {renderFollowButton()}
            {isOwnProfile && (
              <Button asChild variant="outline" className="rounded-full">
                <Link href="/settings/profile">编辑资料</Link>
              </Button>
            )}
          </div>

          {/* User Info */}
          <div className="mt-6 space-y-4">
            <div>
              <h1 className="text-2xl font-bold">{user.nickname}</h1>
              {user.bio && (
                <p className="text-muted-foreground mt-1">{user.bio}</p>
              )}
            </div>

            {/* Meta Info */}
            <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground">
              {user.location && (
                <div className="flex items-center gap-1.5">
                  <MapPin className="h-4 w-4" />
                  <span>{user.location.location_name}</span>
                </div>
              )}
              {user.is_online ? (
                <div className="flex items-center gap-1.5">
                  <span className="h-2 w-2 rounded-full bg-green-500" />
                  <span>在线</span>
                </div>
              ) : user.last_online && (
                <div className="flex items-center gap-1.5">
                  <Clock className="h-4 w-4" />
                  <span>
                    {new Date(user.last_online).toLocaleDateString('zh-CN')} 活跃
                  </span>
                </div>
              )}
            </div>

            {/* Tags */}
            {user.tags && user.tags.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {user.tags.map((tag) => (
                  <span
                    key={tag}
                    className="px-3 py-1 rounded-full bg-primary/10 text-primary text-sm"
                  >
                    #{tag}
                  </span>
                ))}
              </div>
            )}

            {/* Stats */}
            <div className="flex items-center gap-6 pt-4 border-t">
              <div className="text-center">
                <p className="text-2xl font-bold">{user.stats.moments_count}</p>
                <p className="text-sm text-muted-foreground">瞬间</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold">{user.stats.followers_count}</p>
                <p className="text-sm text-muted-foreground">粉丝</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold">{user.stats.following_count}</p>
                <p className="text-sm text-muted-foreground">关注</p>
              </div>
              <div className="text-center">
                <p className="text-2xl font-bold">{user.stats.friends_count}</p>
                <p className="text-sm text-muted-foreground">好友</p>
              </div>
            </div>

            {/* Social Context */}
            {user.social_context && user.social_context.mutual_friends_count > 0 && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground bg-muted/50 rounded-lg p-3">
                <Users className="h-4 w-4" />
                <span>{user.social_context.mutual_friends_count} 个共同好友</span>
                {user.social_context.mutual_friends_preview && (
                  <div className="flex -space-x-2 ml-2">
                    {user.social_context.mutual_friends_preview.slice(0, 3).map((friend) => (
                      <Avatar key={friend.user_id} className="h-6 w-6 ring-2 ring-background">
                        <AvatarImage src={friend.avatar_url} />
                        <AvatarFallback className="text-xs">
                          {friend.nickname.slice(0, 1)}
                        </AvatarFallback>
                      </Avatar>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Moments Section */}
      <Card className="shadow-soft">
        <CardContent className="p-6">
          <h2 className="font-semibold text-lg mb-4">
            Ta 的瞬间 ({user.stats.moments_count})
          </h2>

          {momentsQuery.isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : moments.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              <ImageIcon className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>暂无瞬间</p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Grid Layout for Images */}
              <div className="grid grid-cols-3 gap-2">
                {moments
                  .filter((m) => m.images && m.images.length > 0)
                  .slice(0, 9)
                  .map((moment) => (
                    <Link
                      key={moment.moment_id}
                      href={`/moments/${moment.moment_id}`}
                      className="relative aspect-square rounded-lg overflow-hidden bg-muted group"
                    >
                      <img
                        src={moment.images?.[0]?.url}
                        alt=""
                        className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                      />
                      <div className="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
                        <div className="flex items-center gap-4 text-white">
                          <div className="flex items-center gap-1">
                            <Heart className="h-4 w-4" />
                            <span className="text-sm">{moment.likes_count}</span>
                          </div>
                          <div className="flex items-center gap-1">
                            <MessageCircle className="h-4 w-4" />
                            <span className="text-sm">{moment.comments_count}</span>
                          </div>
                        </div>
                      </div>
                    </Link>
                  ))}
              </div>

              {/* Text-only Moments */}
              {moments
                .filter((m) => !m.images || m.images.length === 0)
                .slice(0, 5)
                .map((moment) => (
                  <MomentPreview key={moment.moment_id} moment={moment} />
                ))}

              {/* Load More */}
              {momentsQuery.hasNextPage && (
                <div className="flex justify-center pt-4">
                  <Button
                    variant="outline"
                    onClick={() => momentsQuery.fetchNextPage()}
                    disabled={momentsQuery.isFetchingNextPage}
                    className="rounded-full"
                  >
                    {momentsQuery.isFetchingNextPage ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        加载中...
                      </>
                    ) : (
                      '查看更多'
                    )}
                  </Button>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface MomentPreviewProps {
  moment: UserMomentListItem;
}

function MomentPreview({ moment }: MomentPreviewProps) {
  return (
    <Link
      href={`/moments/${moment.moment_id}`}
      className="block p-4 rounded-lg bg-muted/50 hover:bg-muted transition-colors"
    >
      <p className="line-clamp-2 text-sm">{moment.description}</p>
      <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
        <span>{new Date(moment.created_at).toLocaleDateString('zh-CN')}</span>
        <div className="flex items-center gap-1">
          <Heart className="h-3 w-3" />
          <span>{moment.likes_count}</span>
        </div>
        <div className="flex items-center gap-1">
          <MessageCircle className="h-3 w-3" />
          <span>{moment.comments_count}</span>
        </div>
      </div>
    </Link>
  );
}
