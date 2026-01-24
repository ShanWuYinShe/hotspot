'use client';

/**
 * 瞬间操作按钮 (Client Component)
 * 处理点赞、收藏等交互
 */

import Link from 'next/link';
import { Heart, MessageCircle, Bookmark } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { useRequireAuth } from '@/hooks/use-require-auth';

interface MomentActionsProps {
  momentId: string;
  likesCount: number;
  commentsCount: number;
  isLiked: boolean;
  isFavorited: boolean;
  onLike?: (momentId: string) => void;
  onUnlike?: (momentId: string) => void;
  onFavorite?: (momentId: string) => void;
  onUnfavorite?: (momentId: string) => void;
}

export function MomentActions({
  momentId,
  likesCount,
  commentsCount,
  isLiked,
  isFavorited,
  onLike,
  onUnlike,
  onFavorite,
  onUnfavorite,
}: MomentActionsProps) {
  const { requireAuth } = useRequireAuth();

  const handleLikeClick = (e: React.MouseEvent) => {
    e.preventDefault();
    if (!requireAuth(isLiked ? '取消点赞' : '点赞')) {
      return;
    }
    if (isLiked) {
      onUnlike?.(momentId);
    } else {
      onLike?.(momentId);
    }
  };

  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.preventDefault();
    if (!requireAuth(isFavorited ? '取消收藏' : '收藏')) {
      return;
    }
    if (isFavorited) {
      onUnfavorite?.(momentId);
    } else {
      onFavorite?.(momentId);
    }
  };

  return (
    <div className="flex items-center gap-6">
      <Button
        variant="ghost"
        size="sm"
        className={cn('gap-1.5', isLiked && 'text-red-500')}
        onClick={handleLikeClick}
      >
        <Heart className={cn('h-4 w-4', isLiked && 'fill-current')} />
        <span className="text-xs">{likesCount}</span>
      </Button>
      <Button variant="ghost" size="sm" className="gap-1.5" asChild>
        <Link href={`/moments/${momentId}#comments`}>
          <MessageCircle className="h-4 w-4" />
          <span className="text-xs">{commentsCount}</span>
        </Link>
      </Button>
      <Button
        variant="ghost"
        size="sm"
        className={cn('gap-1.5', isFavorited && 'text-yellow-500')}
        onClick={handleFavoriteClick}
      >
        <Bookmark className={cn('h-4 w-4', isFavorited && 'fill-current')} />
      </Button>
    </div>
  );
}
