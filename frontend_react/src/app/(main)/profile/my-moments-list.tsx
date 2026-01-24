'use client';

/**
 * 我的瞬间列表 (Client Component)
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
import { LoginRequiredCard } from '@/components/common/login-required-card';
import { momentsService } from '@/lib/api/services/moments';
import { isSuccess } from '@/lib/api/client';
import { likeMoment, unlikeMoment, favoriteMoment, unfavoriteMoment } from '@/lib/actions';
import type { MomentItem } from '@/types/api';

interface MyMomentsListProps {
  initialMoments: MomentItem[];
}

export function MyMomentsList({ initialMoments }: MyMomentsListProps) {
  const { toast } = useToast();
  const { currentLocation, refreshLocation, isLoading: locationLoading } = useLocation();
  const { requireAuth, isAuthenticated } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    if (!isAuthenticated) return;
    if (!currentLocation) {
      refreshLocation(false);
    }
  }, [isAuthenticated, currentLocation, refreshLocation]);

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isLoading,
  } = useInfiniteQuery({
    queryKey: ['my-moments', currentLocation?.latitude, currentLocation?.longitude],
    queryFn: async ({ pageParam }) => {
      const response = await momentsService.getMyMoments({
        reference_point: currentLocation ?? { latitude: 0, longitude: 0 },
        cursor: pageParam,
        limit: 20,
      });
      if (isSuccess(response)) {
        return response.data;
      }
      throw new Error(response.message);
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
    enabled: isAuthenticated && !!currentLocation,
    initialData: initialMoments.length > 0
      ? {
          pages: [{ items: initialMoments, pagination: { has_more: false } }],
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
    if (!requireAuth('点赞')) return;
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
    if (!requireAuth('取消点赞')) return;
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
    if (!requireAuth('收藏')) return;
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
    if (!requireAuth('取消收藏')) return;
    startTransition(async () => {
      updateOptimisticMoments({ id: momentId, type: 'unfavorite' });
      const result = await unfavoriteMoment(momentId);
      if (result.code !== 0) {
        updateOptimisticMoments({ id: momentId, type: 'favorite' });
        toast.error('操作失败', result.message);
      }
    });
  };

  if (!isAuthenticated) {
    return (
      <LoginRequiredCard actionName="查看我的瞬间" description="登录后才能查看你的瞬间。" />
    );
  }

  if (locationLoading || isLoading) {
    return <LoadingState text="加载中..." />;
  }

  if (!currentLocation) {
    return (
      <LocationRequiredCard
        title="需要位置信息"
        description="请开启位置权限以查看你的瞬间"
      />
    );
  }

  if (optimisticMoments.length === 0) {
    return (
      <EmptyState
        icon={Compass}
        title="暂无瞬间"
        description="发布你的第一条瞬间吧"
      />
    );
  }

  return (
    <div className="space-y-4">
      {optimisticMoments.map((moment) => (
        <MomentCard
          key={moment.moment_id}
          moment={moment}
          onLike={handleLike}
          onUnlike={handleUnlike}
          onFavorite={handleFavorite}
          onUnfavorite={handleUnfavorite}
          isOwner
        />
      ))}
      <div ref={loadMoreRef} className="h-10">
        {(isFetching || isPending) && <LoadingState size="sm" />}
      </div>
    </div>
  );
}
