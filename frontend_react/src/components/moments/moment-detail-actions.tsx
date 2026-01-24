'use client';

/**
 * 瞬间详情交互区 (Client Component)
 * 处理点赞、收藏、分享等交互
 */

import { useOptimistic, useTransition } from 'react';
import { Heart, MessageCircle, Bookmark, Share } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { likeMoment, unlikeMoment, favoriteMoment, unfavoriteMoment } from '@/lib/actions';
import { cn } from '@/lib/utils';

interface MomentDetailActionsProps {
  momentId: string;
  likesCount: number;
  commentsCount: number;
  isLiked: boolean;
  isFavorited: boolean;
  shareTitle: string;
  shareText: string;
}

export function MomentDetailActions({
  momentId,
  likesCount,
  commentsCount,
  isLiked,
  isFavorited,
  shareTitle,
  shareText,
}: MomentDetailActionsProps) {
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  const [optimisticState, setOptimisticState] = useOptimistic(
    { isLiked, isFavorited, likesCount },
    (state, action: { type: 'like' | 'unlike' | 'favorite' | 'unfavorite' }) => {
      switch (action.type) {
        case 'like':
          return { ...state, isLiked: true, likesCount: state.likesCount + 1 };
        case 'unlike':
          return { ...state, isLiked: false, likesCount: state.likesCount - 1 };
        case 'favorite':
          return { ...state, isFavorited: true };
        case 'unfavorite':
          return { ...state, isFavorited: false };
        default:
          return state;
      }
    }
  );

  const handleLike = () => {
    if (!requireAuth(optimisticState.isLiked ? '取消点赞' : '点赞')) {
      return;
    }
    startTransition(async () => {
      const action = optimisticState.isLiked ? 'unlike' : 'like';
      setOptimisticState({ type: action });

      const result = optimisticState.isLiked
        ? await unlikeMoment(momentId)
        : await likeMoment(momentId);

      if (result.code !== 0) {
        setOptimisticState({ type: action === 'like' ? 'unlike' : 'like' });
        toast.error('操作失败', result.message);
      }
    });
  };

  const handleFavorite = () => {
    if (!requireAuth(optimisticState.isFavorited ? '取消收藏' : '收藏')) {
      return;
    }
    startTransition(async () => {
      const action = optimisticState.isFavorited ? 'unfavorite' : 'favorite';
      setOptimisticState({ type: action });

      const result = optimisticState.isFavorited
        ? await unfavoriteMoment(momentId)
        : await favoriteMoment(momentId);

      if (result.code !== 0) {
        setOptimisticState({ type: action === 'favorite' ? 'unfavorite' : 'favorite' });
        toast.error('操作失败', result.message);
      }
    });
  };

  const handleShare = async () => {
    if (typeof location === 'undefined') {
      return;
    }
    const url = location.href;
    const text = shareText || '查看瞬间';

    if (navigator?.share) {
      try {
        await navigator.share({ title: shareTitle, text, url });
        return;
      } catch {
        // 用户取消或不支持
      }
    }

    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(url);
        toast.success('链接已复制');
        return;
      }
    } catch {
      // 复制失败兜底
    }

    toast.error('复制失败', '请手动复制链接');
  };

  return (
    <div className="flex items-center justify-between w-full">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          className={cn('gap-2', optimisticState.isLiked && 'text-red-500')}
          onClick={handleLike}
          disabled={isPending}
          aria-pressed={optimisticState.isLiked}
        >
          <Heart className={cn('h-5 w-5', optimisticState.isLiked && 'fill-current')} />
          <span>{optimisticState.likesCount}</span>
        </Button>

        <Button variant="ghost" size="sm" className="gap-2" asChild>
          <a href="#comments">
            <MessageCircle className="h-5 w-5" />
            <span>{commentsCount}</span>
          </a>
        </Button>

        <Button
          variant="ghost"
          size="sm"
          className={cn('gap-2', optimisticState.isFavorited && 'text-yellow-500')}
          onClick={handleFavorite}
          disabled={isPending}
          aria-pressed={optimisticState.isFavorited}
        >
          <Bookmark className={cn('h-5 w-5', optimisticState.isFavorited && 'fill-current')} />
        </Button>
      </div>

      <Button
        variant="ghost"
        size="icon"
        onClick={handleShare}
        aria-label="分享瞬间"
      >
        <Share className="h-5 w-5" />
      </Button>
    </div>
  );
}
