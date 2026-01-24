'use client';

import { useEffect, useOptimistic, useTransition, useState } from 'react';
import { Compass } from '@/components/icons';
import { MomentCard } from '@/components/cards/moment-card';
import { LoadingState } from '@/components/common/loading-state';
import { EmptyState } from '@/components/common/empty-state';
import { useInfiniteScroll } from '@/hooks/use-infinite-scroll';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { likeMoment, unlikeMoment, favoriteMoment, unfavoriteMoment } from '@/lib/actions';
import { feedService } from '@/lib/api/services/feed';
import { isSuccess } from '@/lib/api/client';
import type { MomentItem } from '@/types/api';

interface FeedListProps {
  initialMoments: MomentItem[];
  hasMore?: boolean;
  nextCursor?: string;
  source: string;
}

export function FeedList({
  initialMoments,
  hasMore = false,
  nextCursor,
  source,
}: FeedListProps) {
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  const [moments, setMoments] = useState(initialMoments);
  const [cursor, setCursor] = useState(nextCursor);
  const [hasMoreState, setHasMoreState] = useState(hasMore);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  useEffect(() => {
    setMoments(initialMoments);
    setCursor(nextCursor);
    setHasMoreState(hasMore);
    setIsLoadingMore(false);
  }, [initialMoments, nextCursor, hasMore, source]);

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

  const loadMore = async () => {
    if (!cursor || isLoadingMore) return;
    setIsLoadingMore(true);
    try {
      const response = await feedService.getMoments({
        source: source as 'all' | 'following' | 'nearby',
        cursor,
        limit: 20,
      });
      if (isSuccess(response)) {
        const items = response.data?.items ?? [];
        const pagination = response.data?.pagination;
        setMoments((prev) => [...prev, ...items]);
        setCursor(pagination?.next_cursor);
        setHasMoreState(pagination?.has_more ?? false);
      } else {
        toast.error('加载失败', response.message);
      }
    } finally {
      setIsLoadingMore(false);
    }
  };

  const { loadMoreRef } = useInfiniteScroll(loadMore, { enabled: hasMoreState });

  if (optimisticMoments.length === 0) {
    return (
      <EmptyState
        icon={Compass}
        title="暂无动态"
        description="关注更多用户或去附近看看"
        className="min-h-[400px]"
      />
    );
  }

  return (
    <div className="space-y-6">
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
      <div ref={loadMoreRef} className="flex h-16 items-center justify-center">
        {(isPending || isLoadingMore) && <LoadingState size="sm" />}
      </div>
    </div>
  );
}
