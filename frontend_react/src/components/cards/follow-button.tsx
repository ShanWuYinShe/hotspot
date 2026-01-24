'use client';

/**
 * 关注按钮 (Client Component)
 * 处理关注/取消关注交互
 */

import { useTransition, useOptimistic } from 'react';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { followUser, unfollowUser } from '@/lib/actions';

interface FollowButtonProps {
  userId: number;
  isFollowing: boolean;
}

export function FollowButton({ userId, isFollowing: initialIsFollowing }: FollowButtonProps) {
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  const [optimisticIsFollowing, setOptimisticIsFollowing] = useOptimistic(initialIsFollowing);

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!requireAuth(optimisticIsFollowing ? '取消关注' : '关注用户')) {
      return;
    }

    startTransition(async () => {
      const previous = optimisticIsFollowing;
      const next = !previous;
      setOptimisticIsFollowing(next);

      const result = previous
        ? await unfollowUser(userId)
        : await followUser(userId);

      if (result.code !== 0) {
        setOptimisticIsFollowing(previous);
        toast.error('操作失败', result.message);
      }
    });
  };

  return (
    <Button
      variant={optimisticIsFollowing ? 'outline' : 'default'}
      size="sm"
      onClick={handleClick}
      disabled={isPending}
    >
      {optimisticIsFollowing ? '已关注' : '关注'}
    </Button>
  );
}
