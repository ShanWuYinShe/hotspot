'use client';

import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query';
import { useInView } from 'react-intersection-observer';
import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { CommentItem as CommentItemComponent } from './comment-item';
import { CommentInput } from './comment-input';
import { CommentListItem, PaginatedData } from '@/types/api';
import { CommentService } from '@/lib/api/services/comments';
import { useAuth } from '@/hooks/use-auth';

interface CommentListProps {
  momentId: string;
}

export function CommentList({ momentId }: CommentListProps) {
  const { ref, inView } = useInView();
  const { user: currentUser } = useAuth();
  const [replyTo, setReplyTo] = useState<{ id: string; name: string } | null>(null);
  const queryClient = useQueryClient();
  const t = useTranslations('comments');

  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    status,
  } = useInfiniteQuery({
    queryKey: ['comments', momentId],
    queryFn: async ({ pageParam }): Promise<PaginatedData<CommentListItem>> => {
      const res = await CommentService.list(momentId, { cursor: pageParam ? String(pageParam) : undefined });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.pagination.next_cursor,
    enabled: !!momentId,
  });

  useEffect(() => {
    if (inView && hasNextPage) {
      fetchNextPage();
    }
  }, [inView, hasNextPage, fetchNextPage]);

  const handleReply = (commentId: string, userNickname: string) => {
    setReplyTo({ id: commentId, name: userNickname });
  };

  const handleCommentSuccess = () => {
    setReplyTo(null);
    queryClient.invalidateQueries({ queryKey: ['comments', momentId] });
  };

  const allComments = data?.pages.flatMap((page) => page.items) ?? [];

  if (status === 'pending') {
    return (
      <div className="flex justify-center p-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <CommentInput 
        momentId={momentId}
        placeholder={t('placeholder')}
        onSuccess={handleCommentSuccess}
      />

      {replyTo && (
        <div className="ml-12 border-l-2 border-primary/20 pl-4">
          <div className="text-sm text-muted-foreground mb-2">
            {t('replyingTo')} <span className="font-semibold">{replyTo.name}</span>
          </div>
          <CommentInput
            momentId={momentId}
            parentCommentId={replyTo.id}
            placeholder={`@${replyTo.name} `}
            onSuccess={handleCommentSuccess}
            onCancel={() => setReplyTo(null)}
          />
        </div>
      )}

      <div className="space-y-4">
        {allComments.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {t('empty')}
          </div>
        ) : (
          <>
            {allComments.map((comment) => (
              <CommentItemComponent
                key={comment.comment_id}
                comment={comment}
                onReply={handleReply}
                onDelete={handleCommentSuccess}
                canDelete={comment.user.user_id === currentUser?.user_id}
              />
            ))}
            
            <div ref={ref} className="flex justify-center p-4">
              {isFetchingNextPage && (
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              )}
              {!hasNextPage && allComments.length > 0 && (
                <span className="text-sm text-muted-foreground">
                  {t('noMore')}
                </span>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
