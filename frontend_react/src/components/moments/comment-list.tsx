'use client';

import * as React from 'react';
import Link from 'next/link';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Heart, Reply, Loader2 } from 'lucide-react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn, formatRelativeTime } from '@/lib/utils';
import { isSuccess } from '@/lib/api/client';
import * as commentsService from '@/lib/api/services/comments';
import { useAuthStore } from '@/stores/auth';
import { toast } from '@/hooks/use-toast';
import type { CommentListItem } from '@/types/api';

interface CommentListProps {
  momentId: string;
}

export function CommentList({ momentId }: CommentListProps) {
  const queryClient = useQueryClient();
  const { isAuthenticated } = useAuthStore();
  const [replyTo, setReplyTo] = React.useState<CommentListItem | null>(null);
  const [newComment, setNewComment] = React.useState('');
  const inputRef = React.useRef<HTMLInputElement>(null);

  const {
    data,
    isLoading,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    queryKey: ['comments', momentId],
    queryFn: async ({ pageParam }) => {
      const res = await commentsService.getComments({
        moment_id: momentId,
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

  const comments = data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  const createMutation = useMutation({
    mutationFn: async (content: string) => {
      const res = await commentsService.createComment({
        moment_id: momentId,
        content,
        parent_id: replyTo?.comment_id,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    onSuccess: () => {
      setNewComment('');
      setReplyTo(null);
      queryClient.invalidateQueries({ queryKey: ['comments', momentId] });
      toast({ title: '评论成功' });
    },
    onError: (error) => {
      toast({
        title: '评论失败',
        description: error.message,
        variant: 'destructive',
      });
    },
  });

  const likeMutation = useMutation({
    mutationFn: async (commentId: string) => {
      const res = await commentsService.likeComment(commentId);
      if (!isSuccess(res)) throw new Error(res.message);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', momentId] });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newComment.trim()) return;
    createMutation.mutate(newComment.trim());
  };

  const handleReply = (comment: CommentListItem) => {
    setReplyTo(comment);
    inputRef.current?.focus();
  };

  return (
    <div className="space-y-4">
      <h3 className="font-semibold">评论</h3>

      {isAuthenticated && (
        <form onSubmit={handleSubmit} className="space-y-2">
          {replyTo && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>回复 @{replyTo.user.nickname}</span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setReplyTo(null)}
              >
                取消
              </Button>
            </div>
          )}
          <div className="flex gap-2">
            <Input
              ref={inputRef}
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder={replyTo ? `回复 ${replyTo.user.nickname}...` : '写下你的评论...'}
              disabled={createMutation.isPending}
            />
            <Button
              type="submit"
              disabled={!newComment.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                '发送'
              )}
            </Button>
          </div>
        </form>
      )}

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : comments.length === 0 ? (
        <div className="py-8 text-center text-muted-foreground">
          暂无评论，来发表第一条吧
        </div>
      ) : (
        <div className="space-y-4">
          {comments.map((comment) => (
            <CommentItem
              key={comment.comment_id}
              comment={comment}
              onReply={() => handleReply(comment)}
              onLike={() => likeMutation.mutate(comment.comment_id)}
            />
          ))}

          {hasNextPage && (
            <div className="flex justify-center">
              <Button
                variant="ghost"
                onClick={() => fetchNextPage()}
                disabled={isFetchingNextPage}
              >
                {isFetchingNextPage ? (
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

interface CommentItemProps {
  comment: CommentListItem;
  onReply: () => void;
  onLike: () => void;
}

function CommentItem({ comment, onReply, onLike }: CommentItemProps) {
  const { isAuthenticated } = useAuthStore();

  return (
    <div className="flex gap-3">
      <Link href={`/users/${comment.user.user_id}`}>
        <Avatar className="h-8 w-8">
          <AvatarImage src={comment.user.avatar_url} alt={comment.user.nickname} />
          <AvatarFallback>
            {comment.user.nickname.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>
      </Link>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <Link
            href={`/users/${comment.user.user_id}`}
            className="font-medium text-sm hover:underline"
          >
            {comment.user.nickname}
          </Link>
          <span className="text-xs text-muted-foreground">
            {formatRelativeTime(comment.created_at)}
          </span>
        </div>
        <p className="mt-1 text-sm whitespace-pre-wrap break-words">
          {comment.content}
        </p>
        <div className="mt-2 flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            className={cn('h-auto p-0 text-xs text-muted-foreground')}
            onClick={onLike}
            disabled={!isAuthenticated}
          >
            <Heart className="mr-1 h-3 w-3" />
            {comment.likes_count > 0 ? comment.likes_count : '赞'}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-auto p-0 text-xs text-muted-foreground"
            onClick={onReply}
            disabled={!isAuthenticated}
          >
            <Reply className="mr-1 h-3 w-3" />
            回复
          </Button>
        </div>
      </div>
    </div>
  );
}
