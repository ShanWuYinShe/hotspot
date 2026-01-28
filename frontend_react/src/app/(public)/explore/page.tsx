'use client';

import * as React from 'react';
import Link from 'next/link';
import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import { useLocation } from '@/hooks/use-location';
import * as feedService from '@/lib/api/services/feed';
import * as momentsService from '@/lib/api/services/moments';
import * as tagsService from '@/lib/api/services/tags';
import { isSuccess } from '@/lib/api/client';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Search,
  Loader2,
  TrendingUp,
  MapPin,
  Heart,
  MessageCircle,
  Bookmark,
  X,
  Compass,
  Flame,
  UserPlus,
} from 'lucide-react';
import type { FeedMomentItem } from '@/types/api';

type TabType = 'trending' | 'nearby';

export default function ExplorePage() {
  const { isAuthenticated } = useAuthStore();
  const { location, error: locationError } = useLocation();
  const [activeTab, setActiveTab] = React.useState<TabType>('trending');
  const [searchQuery, setSearchQuery] = React.useState('');
  const [debouncedSearch, setDebouncedSearch] = React.useState('');

  // Debounce search
  React.useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(searchQuery);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // 热门内容查询
  const trendingQuery = useInfiniteQuery({
    queryKey: ['trending-moments'],
    queryFn: async ({ pageParam }) => {
      const res = await feedService.getTrendingMoments({
        cursor: pageParam as string | undefined,
        limit: 20,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
    enabled: activeTab === 'trending' && !debouncedSearch,
  });

  // 附近内容查询
  const nearbyQuery = useInfiniteQuery({
    queryKey: ['nearby-moments', location?.latitude, location?.longitude],
    queryFn: async ({ pageParam }) => {
      if (!location) throw new Error('需要位置信息');
      const res = await feedService.getMomentsFeed({
        source: 'nearby',
        reference_point: {
          latitude: location.latitude,
          longitude: location.longitude,
        },
        radius: 10,
        cursor: pageParam as string | undefined,
        limit: 20,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
    enabled: activeTab === 'nearby' && !!location && !debouncedSearch,
  });

  // 热门标签查询
  const tagsQuery = useQuery({
    queryKey: ['trending-tags'],
    queryFn: async () => {
      const res = await tagsService.getTrendingTags({ limit: 10 });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  const getCurrentQuery = () => {
    if (activeTab === 'trending') return trendingQuery;
    return nearbyQuery;
  };

  const currentQuery = getCurrentQuery();
  const moments = currentQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  const handleLoadMore = () => {
    if (currentQuery.hasNextPage && !currentQuery.isFetchingNextPage) {
      currentQuery.fetchNextPage();
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">探索</h1>
          <p className="text-muted-foreground text-sm mt-1">发现精彩内容</p>
        </div>
      </div>

      {/* Search Bar */}
      <div className="relative">
        <Search className="absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索瞬间、用户、标签..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10 pr-10 h-12 rounded-full bg-muted/50 border-0 focus-visible:ring-2 focus-visible:ring-primary/50"
        />
        {searchQuery && (
          <button
            onClick={() => setSearchQuery('')}
            className="absolute right-4 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Tabs */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => setActiveTab('trending')}
              className={cn(
                'flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all',
                activeTab === 'trending'
                  ? 'bg-primary text-primary-foreground shadow-md'
                  : 'bg-muted/50 text-muted-foreground hover:bg-muted'
              )}
            >
              <TrendingUp className="h-4 w-4" />
              热门
            </button>
            <button
              onClick={() => setActiveTab('nearby')}
              className={cn(
                'flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all',
                activeTab === 'nearby'
                  ? 'bg-primary text-primary-foreground shadow-md'
                  : 'bg-muted/50 text-muted-foreground hover:bg-muted'
              )}
            >
              <MapPin className="h-4 w-4" />
              附近
            </button>
          </div>

          {/* Location Error for Nearby Tab */}
          {activeTab === 'nearby' && locationError && (
            <Card className="border-destructive/50 bg-destructive/5">
              <CardContent className="py-4">
                <div className="flex items-center gap-3">
                  <MapPin className="h-5 w-5 text-destructive" />
                  <div>
                    <p className="font-medium text-destructive">无法获取位置</p>
                    <p className="text-sm text-muted-foreground">{locationError}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Content */}
          {currentQuery.isLoading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            </div>
          ) : moments.length === 0 ? (
            <Card className="shadow-soft">
              <CardContent className="py-12 text-center">
                <Compass className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-medium mb-2">暂无内容</h3>
                <p className="text-muted-foreground text-sm">
                  {activeTab === 'nearby'
                    ? '附近还没有人分享瞬间，成为第一个吧！'
                    : '暂时没有热门内容'}
                </p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-4">
              {moments.map((moment) => (
                <MomentCard key={moment.moment_id} moment={moment} />
              ))}

              {/* Load More */}
              {currentQuery.hasNextPage && (
                <div className="flex justify-center pt-4">
                  <Button
                    variant="outline"
                    onClick={handleLoadMore}
                    disabled={currentQuery.isFetchingNextPage}
                    className="rounded-full"
                  >
                    {currentQuery.isFetchingNextPage ? (
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

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Trending Tags */}
          <Card className="shadow-soft">
            <CardContent className="p-5">
              <div className="flex items-center gap-2 mb-4">
                <Flame className="h-5 w-5 text-primary" />
                <h3 className="font-semibold">热门标签</h3>
              </div>
              {tagsQuery.isLoading ? (
                <div className="flex justify-center py-4">
                  <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {tagsQuery.data?.map((tag, index) => (
                    <Link
                      key={tag.tag}
                      href={`/explore?tag=${encodeURIComponent(tag.tag)}`}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-muted/50 hover:bg-muted text-sm transition-colors"
                    >
                      <span className="text-muted-foreground text-xs">#{index + 1}</span>
                      <span>{tag.tag}</span>
                      {tag.trend === 'rising' && (
                        <TrendingUp className="h-3 w-3 text-green-500" />
                      )}
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Join CTA for Guests */}
          {!isAuthenticated && (
            <Card className="bg-linear-to-br from-primary/10 to-primary/5 border-primary/20 shadow-soft">
              <CardContent className="p-5 text-center">
                <div className="w-12 h-12 rounded-full bg-primary/20 flex items-center justify-center mx-auto mb-4">
                  <UserPlus className="h-6 w-6 text-primary" />
                </div>
                <h3 className="font-semibold mb-2">加入 HotSpot</h3>
                <p className="text-sm text-muted-foreground mb-4">
                  注册后可以发布瞬间、关注好友、参与互动
                </p>
                <Button asChild className="w-full rounded-full">
                  <Link href="/register">立即注册</Link>
                </Button>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}

interface MomentCardProps {
  moment: FeedMomentItem;
}

function MomentCard({ moment }: MomentCardProps) {
  const { isAuthenticated } = useAuthStore();
  const [isLiked, setIsLiked] = React.useState(moment.is_liked);
  const [isFavorited, setIsFavorited] = React.useState(moment.is_favorited);
  const [likesCount, setLikesCount] = React.useState(moment.likes_count);

  const handleLike = async () => {
    if (!isAuthenticated) return;
    
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
      // Rollback
      setIsLiked(!newLiked);
      setLikesCount((prev) => prev + (newLiked ? -1 : 1));
    }
  };

  const handleFavorite = async () => {
    if (!isAuthenticated) return;
    
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
            disabled={!isAuthenticated}
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
            disabled={!isAuthenticated}
          >
            <Bookmark className={cn('h-4 w-4', isFavorited && 'fill-current')} />
          </button>
        </div>
      </CardContent>
    </Card>
  );
}
