'use client';

/**
 * 评论区组件 (Client Component)
 * 处理评论列表和新建评论
 */

import { useState, useTransition } from 'react';
import Link from 'next/link';
import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query';
import { MessageCircle, Heart, Reply, Send, ChevronDown } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { UserAvatar } from '@/components/common/user-avatar';
import { LoadingState } from '@/components/common/loading-state';
import { EmptyState } from '@/components/common/empty-state';
import { useAuthStore } from '@/stores/auth';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { commentsService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import { createComment, likeComment } from '@/lib/actions';
import { formatRelativeTime } from '@/lib/utils';
import type { CommentListItem, PaginationMeta } from '@/types/api';

interface CommentSectionProps {
  momentId: string;
  initialComments: CommentListItem[];
  commentsCount: number;
  initialPagination?: PaginationMeta;
}

export function CommentSection({
  momentId,
  initialComments,
  commentsCount,
  initialPagination,
}: CommentSectionProps) {
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const queryClient = useQueryClient();
  const [isPending, startTransition] = useTransition();
  const [newComment, setNewComment] = useState('');
  const [replyTo, setReplyTo] = useState<CommentListItem | null>(null);
  const currentUser = useAuthStore((state) => state.currentAccount);

  const resolvedPagination: PaginationMeta = initialPagination?.next_cursor
    ? { has_more: true, next_cursor: initialPagination.next_cursor }
    : { has_more: false };

  const { data, fetchNextPage, hasNextPage, isFetching } = useInfiniteQuery({
    queryKey: ['comments', momentId],
    queryFn: async ({ pageParam }) => {
      const response = await commentsService.getComments(momentId, {
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
    initialData:
      initialComments.length > 0
        ? {
            pages: [{ items: initialComments, pagination: resolvedPagination }],
            pageParams: [undefined],
          }
        : undefined,
  });

  const comments = data?.pages.flatMap((page) => page.items) ?? [];

  const handleSubmit = () => {
    if (!newComment.trim()) return;
    if (!requireAuth('发表评论')) {
      return;
    }

    startTransition(async () => {
      const result = await createComment(momentId, {
        content: newComment,
        parent_id: replyTo?.comment_id,
        reply_to_user_id: replyTo?.user_id,
      });

      if (result.code === 0) {
        setNewComment('');
        setReplyTo(null);
        queryClient.invalidateQueries({ queryKey: ['comments', momentId] });
        toast.success('评论成功');
      } else {
        toast.error('评论失败', result.message);
      }
    });
  };

  const handleLike = async (comment: CommentListItem) => {
    if (!requireAuth('点赞评论')) {
      return;
    }
    const result = await likeComment(comment.comment_id);
    if (result.code !== 0) {
      toast.error('操作失败', result.message);
    }
  };

  return (
    <Card id="comments">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MessageCircle className="h-5 w-5" />
          评论 ({commentsCount})
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* 评论输入框 */}
        {currentUser && (
          <div className="flex gap-3">
            <UserAvatar user={currentUser} size="sm" />
            <div className="flex-1 space-y-2">
              {replyTo && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <span>回复 @{replyTo?.user?.nickname || '用户'}</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-auto p-0"
                    onClick={() => setReplyTo(null)}
                  >
                    取消
                  </Button>
                </div>
              )}
              <Textarea
                placeholder={replyTo ? `回复 @${replyTo.user.nickname}` : '写下你的评论...'}
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                rows={2}
                className="resize-none"
              />
              <div className="flex justify-end">
                <Button size="sm" onClick={handleSubmit} disabled={!newComment.trim() || isPending}>
                  <Send className="h-4 w-4 mr-2" />
                  发送
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* 评论列表 */}
        {comments.length === 0 ? (
          <EmptyState icon={MessageCircle} title="暂无评论" description="成为第一个评论的人吧" />
        ) : (
          <div className="space-y-4">
            {comments.map((comment) => (
              <CommentItem
                key={comment.comment_id}
                comment={comment}
                onReply={() => setReplyTo(comment)}
                onLike={() => handleLike(comment)}
              />
            ))}
          </div>
        )}

        {/* 加载更多 */}
        {hasNextPage && (
          <Button
            variant="ghost"
            className="w-full"
            onClick={() => fetchNextPage()}
            disabled={isFetching}
          >
            {isFetching ? (
              <LoadingState size="sm" />
            ) : (
              <>
                <ChevronDown className="h-4 w-4 mr-2" />
                加载更多评论
              </>
            )}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

interface CommentItemProps {
  comment: CommentListItem;
  onReply: () => void;
  onLike: () => void;
}

function CommentItem({ comment, onReply, onLike }: CommentItemProps) {
  return (
    <div className="flex gap-3">
      <Link href={`/users/${comment.user.user_id}`}>
        <UserAvatar user={comment.user} size="sm" />
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
        <p className="text-sm mt-1 whitespace-pre-wrap">{comment.content}</p>
        <div className="flex items-center gap-4 mt-2">
          <Button
            variant="ghost"
            size="sm"
            className="h-auto p-0 text-muted-foreground hover:text-red-500"
            onClick={onLike}
          >
            <Heart className="h-3.5 w-3.5 mr-1" />
            <span className="text-xs">{comment.likes_count || ''}</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-auto p-0 text-muted-foreground"
            onClick={onReply}
          >
            <Reply className="h-3.5 w-3.5 mr-1" />
            <span className="text-xs">回复</span>
          </Button>
        </div>

        {/* 子评论预览 */}
        {comment.replies_count > 0 && (
          <Button variant="link" size="sm" className="h-auto p-0 mt-2 text-primary">
            查看 {comment.replies_count} 条回复
          </Button>
        )}
      </div>
    </div>
  );
}
