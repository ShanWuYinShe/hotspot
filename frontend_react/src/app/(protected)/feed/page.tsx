'use client';

import * as React from 'react';
import Link from 'next/link';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import { useLocation } from '@/hooks/use-location';
import * as feedService from '@/lib/api/services/feed';
import * as momentsService from '@/lib/api/services/moments';
import { isSuccess } from '@/lib/api/client';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Loader2,
  MapPin,
  Heart,
  MessageCircle,
  Bookmark,
  Plus,
  Users,
  TrendingUp,
  Compass,
} from 'lucide-react';
import type { FeedMomentItem, FeedSource } from '@/types/api';

export default function FeedPage() {
  const { user } = useAuthStore();
  const { location } = useLocation();
  const [activeSource, setActiveSource] = React.useState<FeedSource>('following');

  // Fetch feed
  const feedQuery = useInfiniteQuery({
    queryKey: ['feed', activeSource, location?.latitude, location?.longitude],
    queryFn: async ({ pageParam }) => {
      const res = await feedService.getMomentsFeed({
        source: activeSource,
        reference_point: location ? {
          latitude: location.latitude,
          longitude: location.longitude,
        } : undefined,
        radius: activeSource === 'nearby' ? 10 : undefined,
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

  const moments = feedQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  const sources: { value: FeedSource; label: string; icon: React.ElementType }[] = [
    { value: 'following', label: '关注', icon: Users },
    { value: 'nearby', label: '附近', icon: MapPin },
    { value: 'all', label: '推荐', icon: TrendingUp },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">动态</h1>
          <p className="text-muted-foreground text-sm mt-1">
            欢迎回来，{user?.nickname}
          </p>
        </div>
        <Button asChild className="rounded-full gap-2 shadow-md">
          <Link href="/moments/create">
            <Plus className="h-4 w-4" />
            发布
          </Link>
        </Button>
      </div>

      {/* Source Tabs */}
      <div className="flex items-center gap-2">
        {sources.map((source) => (
          <button
            key={source.value}
            onClick={() => setActiveSource(source.value)}
            className={cn(
              'flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all',
              activeSource === source.value
                ? 'bg-primary text-primary-foreground shadow-md'
                : 'bg-muted/50 text-muted-foreground hover:bg-muted'
            )}
          >
            <source.icon className="h-4 w-4" />
            {source.label}
          </button>
        ))}
      </div>

      {/* Content */}
      {feedQuery.isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : moments.length === 0 ? (
        <Card className="shadow-soft">
          <CardContent className="py-12 text-center">
            <Compass className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">
              {activeSource === 'following' ? '还没有关注的人' : '暂无内容'}
            </h3>
            <p className="text-muted-foreground text-sm mb-4">
              {activeSource === 'following'
                ? '去探索发现有趣的人吧'
                : '附近还没有人发布瞬间'}
            </p>
            <Button asChild variant="outline" className="rounded-full">
              <Link href="/explore">去探索</Link>
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {moments.map((moment) => (
            <MomentCard key={moment.moment_id} moment={moment} />
          ))}

          {/* Load More */}
          {feedQuery.hasNextPage && (
            <div className="flex justify-center pt-4">
              <Button
                variant="outline"
                onClick={() => feedQuery.fetchNextPage()}
                disabled={feedQuery.isFetchingNextPage}
                className="rounded-full"
              >
                {feedQuery.isFetchingNextPage ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    加载中...
                  </>
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

interface MomentCardProps {
  moment: FeedMomentItem;
}

function MomentCard({ moment }: MomentCardProps) {
  const [isLiked, setIsLiked] = React.useState(moment.is_liked);
  const [isFavorited, setIsFavorited] = React.useState(moment.is_favorited);
  const [likesCount, setLikesCount] = React.useState(moment.likes_count);

  const handleLike = async () => {
    const newLiked = !isLiked;
    setIsLiked(newLiked);
    setLikesCount((prev) => prev + (newLiked ? 1 : -1));

    try {
      if (newLiked) {
        await momentsService.likeMoment(moment.moment_id);
      } else {
        await momentsService.unlikeMoment(moment.moment_id);
      }
    } catch {
      setIsLiked(!newLiked);
      setLikesCount((prev) => prev + (newLiked ? -1 : 1));
    }
  };

  const handleFavorite = async () => {
    const newFavorited = !isFavorited;
    setIsFavorited(newFavorited);

    try {
      if (newFavorited) {
        await momentsService.favoriteMoment(moment.moment_id);
      } else {
        await momentsService.unfavoriteMoment(moment.moment_id);
      }
    } catch {
      setIsFavorited(!newFavorited);
    }
  };

  return (
    <Card className="shadow-soft hover:shadow-lg transition-shadow overflow-hidden">
      <CardContent className="p-5">
        {/* Recommendation Context */}
        {moment.recommendation && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground mb-3 pb-3 border-b">
            {moment.recommendation.type === 'friend_liked' && (
              <>
                {moment.recommendation.friends_preview && (
                  <div className="flex -space-x-1">
                    {moment.recommendation.friends_preview.slice(0, 2).map((friend) => (
                      <Avatar key={friend.user_id} className="h-5 w-5 ring-1 ring-background">
                        <AvatarImage src={friend.avatar_url} />
                        <AvatarFallback className="text-[8px]">
                          {friend.nickname.slice(0, 1)}
                        </AvatarFallback>
                      </Avatar>
                    ))}
                  </div>
                )}
                <span>你的好友喜欢了这条瞬间</span>
              </>
            )}
            {moment.recommendation.type === 'trending' && (
              <>
                <TrendingUp className="h-3 w-3" />
                <span>热门内容</span>
              </>
            )}
            {moment.recommendation.type === 'nearby' && (
              <>
                <MapPin className="h-3 w-3" />
                <span>附近的瞬间</span>
              </>
            )}
          </div>
        )}

        {/* Header */}
        <div className="flex items-start justify-between mb-4">
          <Link href={`/users/${moment.user.user_id}`} className="flex items-center gap-3 group">
            <Avatar className="h-10 w-10 ring-2 ring-background shadow">
              <AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
              <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-sm">
                {moment.user.nickname.slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div>
              <p className="font-medium group-hover:text-primary transition-colors">
                {moment.user.nickname}
              </p>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span>{new Date(moment.created_at).toLocaleDateString('zh-CN')}</span>
                {moment.location && (
                  <>
                    <span>·</span>
                    <span className="flex items-center gap-0.5">
                      <MapPin className="h-3 w-3" />
                      {moment.location.distance < 1
                        ? `${Math.round(moment.location.distance * 1000)}m`
                        : `${moment.location.distance.toFixed(1)}km`}
                    </span>
                  </>
                )}
              </div>
            </div>
          </Link>
        </div>

        {/* Content */}
        <Link href={`/moments/${moment.moment_id}`} className="block">
          <p className="text-foreground mb-3 whitespace-pre-wrap">{moment.description}</p>

          {/* Images */}
          {moment.images && moment.images.length > 0 && (
            <div
              className={cn(
                'grid gap-2 mb-3 rounded-xl overflow-hidden',
                moment.images.length === 1 && 'grid-cols-1',
                moment.images.length === 2 && 'grid-cols-2',
                moment.images.length >= 3 && 'grid-cols-3'
              )}
            >
              {moment.images.slice(0, 9).map((image, index) => (
                <div
                  key={index}
                  className={cn(
                    'relative bg-muted rounded-lg overflow-hidden',
                    moment.images!.length === 1 ? 'aspect-video' : 'aspect-square'
                  )}
                >
                  <img
                    src={image.url}
                    alt=""
                    className="w-full h-full object-cover"
                  />
                </div>
              ))}
            </div>
          )}

          {/* Tags */}
          {moment.tags && moment.tags.length > 0 && (
            <div className="flex flex-wrap gap-1.5 mb-3">
              {moment.tags.map((tag) => (
                <span
                  key={tag}
                  className="px-2 py-0.5 rounded-full bg-primary/10 text-primary text-xs"
                >
                  #{tag}
                </span>
              ))}
            </div>
          )}
        </Link>

        {/* Actions */}
        <div className="flex items-center gap-4 pt-3 border-t">
          <button
            onClick={handleLike}
            className={cn(
              'flex items-center gap-1.5 text-sm transition-colors',
              isLiked ? 'text-red-500' : 'text-muted-foreground hover:text-red-500'
            )}
          >
            <Heart className={cn('h-4 w-4', isLiked && 'fill-current')} />
            <span>{likesCount}</span>
          </button>
          <Link
            href={`/moments/${moment.moment_id}`}
            className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-primary transition-colors"
          >
            <MessageCircle className="h-4 w-4" />
            <span>{moment.comments_count}</span>
          </Link>
          <button
            onClick={handleFavorite}
            className={cn(
              'flex items-center gap-1.5 text-sm transition-colors',
              isFavorited ? 'text-yellow-500' : 'text-muted-foreground hover:text-yellow-500'
            )}
          >
            <Bookmark className={cn('h-4 w-4', isFavorited && 'fill-current')} />
          </button>
        </div>
      </CardContent>
    </Card>
  );
}
