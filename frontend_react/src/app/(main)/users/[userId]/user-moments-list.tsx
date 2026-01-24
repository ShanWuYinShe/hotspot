'use client';

/**
 * 用户瞬间列表 (Client Component)
 * 处理无限滚动和交互
 */

import { useEffect, useOptimistic, useTransition } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { Compass } from '@/components/icons';
import { MomentCard } from '@/components/cards/moment-card';
import { LoadingState } from '@/components/common/loading-state';
import { EmptyState } from '@/components/common/empty-state';
import { LocationRequiredCard } from '@/components/common/location-required-card';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useToast } from '@/hooks/use-toast';
import { useLocation } from '@/hooks/use-location';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { momentsService } from '@/lib/api/services/moments';
import { isSuccess } from '@/lib/api/client';
import { likeMoment, unlikeMoment, favoriteMoment, unfavoriteMoment } from '@/lib/actions';
import type { MomentItem, PaginationMeta } from '@/types/api';

interface UserMomentsListProps {
  userId: string;
  initialMoments: MomentItem[];
  initialPagination?: PaginationMeta;
}

export function UserMomentsList({ userId, initialMoments, initialPagination }: UserMomentsListProps) {
  const { toast } = useToast();
  const { currentLocation, refreshLocation, isLoading: locationLoading } = useLocation();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    if (!currentLocation) {
      refreshLocation(false);
    }
  }, [currentLocation, refreshLocation]);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ['user', userId, 'moments', currentLocation?.latitude, currentLocation?.longitude],
    queryFn: async ({ pageParam }) => {
      const response = await momentsService.getUserMoments(Number(userId), {
        reference_point: currentLocation ?? { latitude: 0, longitude: 0 },
        cursor: pageParam,
        limit: 20,
      });
      if (isSuccess(response)) {
        return response.data;
      }
      throw new Error(response.message);
    },
    enabled: !!currentLocation,
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
    initialData:
      initialMoments.length > 0
        ? {
            pages: [
              {
                items: initialMoments,
                pagination: initialPagination?.next_cursor
                  ? { has_more: true, next_cursor: initialPagination.next_cursor }
                  : { has_more: false },
              },
            ],
            pageParams: [undefined],
          }
        : undefined,
  });

  const { loadMoreRef } = useInfiniteScroll(
    () => {
      if (hasNextPage && !isFetching) {
        fetchNextPage();
      }
    },
    { enabled: hasNextPage }
  );

  const moments = data?.pages.flatMap((page) => page.items) ?? [];

  const [optimisticMoments, updateOptimisticMoments] = useOptimistic(
    moments,
    (state, update: { id: string; type: 'like' | 'unlike' | 'favorite' | 'unfavorite' }) => {
      return state.map((moment) => {
        if (moment.moment_id !== update.id) return moment;
        switch (update.type) {
          case 'like':
            return { ...moment, is_liked: true, likes_count: moment.likes_count + 1 };
          case 'unlike':
            return { ...moment, is_liked: false, likes_count: moment.likes_count - 1 };
          case 'favorite':
            return { ...moment, is_favorited: true };
          case 'unfavorite':
            return { ...moment, is_favorited: false };
          default:
            return moment;
        }
      });
    }
  );

  const handleLike = async (momentId: string) => {
    if (!requireAuth('点赞')) {
      return;
    }
    startTransition(async () => {
      updateOptimisticMoments({ id: momentId, type: 'like' });
      const result = await likeMoment(momentId);
      if (result.code !== 0) {
        updateOptimisticMoments({ id: momentId, type: 'unlike' });
        toast.error('操作失败', result.message);
      }
    });
  };

  const handleUnlike = async (momentId: string) => {
    if (!requireAuth('取消点赞')) {
      return;
    }
    startTransition(async () => {
      updateOptimisticMoments({ id: momentId, type: 'unlike' });
      const result = await unlikeMoment(momentId);
      if (result.code !== 0) {
        updateOptimisticMoments({ id: momentId, type: 'like' });
        toast.error('操作失败', result.message);
      }
    });
  };

  const handleFavorite = async (momentId: string) => {
    if (!requireAuth('收藏')) {
      return;
    }
    startTransition(async () => {
      updateOptimisticMoments({ id: momentId, type: 'favorite' });
      const result = await favoriteMoment(momentId);
      if (result.code !== 0) {
        updateOptimisticMoments({ id: momentId, type: 'unfavorite' });
        toast.error('操作失败', result.message);
      }
    });
  };

  const handleUnfavorite = async (momentId: string) => {
    if (!requireAuth('取消收藏')) {
      return;
    }
    startTransition(async () => {
      updateOptimisticMoments({ id: momentId, type: 'unfavorite' });
      const result = await unfavoriteMoment(momentId);
      if (result.code !== 0) {
        updateOptimisticMoments({ id: momentId, type: 'favorite' });
        toast.error('操作失败', result.message);
      }
    });
  };

  if (locationLoading || isLoading) {
    return <LoadingState text="加载中..." />;
  }

  if (!currentLocation) {
    return (
      <LocationRequiredCard
        title="需要位置信息"
        description="请开启位置权限以查看该用户的瞬间"
      />
    );
  }

  if (optimisticMoments.length === 0) {
    return (
      <EmptyState
        icon={Compass}
        title="暂无瞬间"
        description="该用户还没有发布瞬间"
      />
    );
  }

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">瞬间</h2>
      {optimisticMoments.map((moment) => (
        <MomentCard
          key={moment.moment_id}
          moment={moment}
          onLike={handleLike}
          onUnlike={handleUnlike}
          onFavorite={handleFavorite}
          onUnfavorite={handleUnfavorite}
        />
      ))}
      <div ref={loadMoreRef} className="h-10">
        {(isFetching || isPending) && <LoadingState size="sm" />}
      </div>
    </div>
  );
}
