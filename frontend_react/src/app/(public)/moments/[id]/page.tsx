'use client';

import * as React from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth';
import * as momentsService from '@/lib/api/services/moments';
import * as commentsService from '@/lib/api/services/comments';
import { isSuccess } from '@/lib/api/client';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Input } from '@/components/ui/input';
import {
  ArrowLeft,
  Loader2,
  MapPin,
  Heart,
  MessageCircle,
  Bookmark,
  Share2,
  Send,
  MoreHorizontal,
  ChevronDown,
} from 'lucide-react';
import type { CommentListItem } from '@/types/api';

export default function MomentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { isAuthenticated, user } = useAuthStore();
  const momentId = params.id as string;

  const [newComment, setNewComment] = React.useState('');
  const [replyTo, setReplyTo] = React.useState<CommentListItem | null>(null);

  // Fetch moment
  const momentQuery = useQuery({
    queryKey: ['moment', momentId],
    queryFn: async () => {
      const res = await momentsService.getMoment(momentId);
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  // Fetch comments
  const commentsQuery = useInfiniteQuery({
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

  // Interaction states
  const [isLiked, setIsLiked] = React.useState(false);
  const [isFavorited, setIsFavorited] = React.useState(false);
  const [likesCount, setLikesCount] = React.useState(0);

  React.useEffect(() => {
    if (momentQuery.data) {
      setIsLiked(momentQuery.data.my_state?.is_liked ?? false);
      setIsFavorited(momentQuery.data.my_state?.is_favorited ?? false);
      setLikesCount(momentQuery.data.likes_count);
    }
  }, [momentQuery.data]);

  // Like mutation
  const likeMutation = useMutation({
    mutationFn: async () => {
      if (isLiked) {
        return momentsService.unlikeMoment(momentId);
      } else {
        return momentsService.likeMoment(momentId);
      }
    },
    onMutate: () => {
      const newLiked = !isLiked;
      setIsLiked(newLiked);
      setLikesCount((prev) => prev + (newLiked ? 1 : -1));
    },
    onError: () => {
      setIsLiked(!isLiked);
      setLikesCount((prev) => prev + (isLiked ? 1 : -1));
    },
  });

  // Favorite mutation
  const favoriteMutation = useMutation({
    mutationFn: async () => {
      if (isFavorited) {
        return momentsService.unfavoriteMoment(momentId);
      } else {
        return momentsService.favoriteMoment(momentId);
      }
    },
    onMutate: () => {
      setIsFavorited(!isFavorited);
    },
    onError: () => {
      setIsFavorited(!isFavorited);
    },
  });

  // Comment mutation
  const commentMutation = useMutation({
    mutationFn: async () => {
      const res = await commentsService.createComment({
        moment_id: momentId,
        content: newComment,
        parent_id: replyTo?.comment_id,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    onSuccess: () => {
      setNewComment('');
      setReplyTo(null);
      queryClient.invalidateQueries({ queryKey: ['comments', momentId] });
    },
  });

  const moment = momentQuery.data;
  const comments = commentsQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];

  if (momentQuery.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  if (momentQuery.error || !moment) {
    return (
      <Card className="max-w-2xl mx-auto shadow-soft">
        <CardContent className="py-12 text-center">
          <h2 className="text-lg font-semibold mb-2">内容不存在</h2>
          <p className="text-muted-foreground mb-4">该瞬间可能已被删除</p>
          <Button variant="outline" onClick={() => router.back()}>
            返回
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      {/* Back Button */}
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.back()}
        className="gap-2"
      >
        <ArrowLeft className="h-4 w-4" />
        返回
      </Button>

      {/* Moment Card */}
      <Card className="shadow-soft overflow-hidden">
        <CardContent className="p-6">
          {/* Header */}
          <div className="flex items-start justify-between mb-4">
            <Link href={`/users/${moment.user.user_id}`} className="flex items-center gap-3 group">
              <Avatar className="h-12 w-12 ring-2 ring-background shadow-md">
                <AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
                <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground">
                  {moment.user.nickname.slice(0, 2).toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div>
                <p className="font-semibold group-hover:text-primary transition-colors">
                  {moment.user.nickname}
                </p>
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <span>{new Date(moment.created_at).toLocaleString('zh-CN')}</span>
                </div>
              </div>
            </Link>
            <Button variant="ghost" size="icon" className="rounded-full">
              <MoreHorizontal className="h-5 w-5" />
            </Button>
          </div>

          {/* Content */}
          <div className="space-y-4">
            <p className="text-lg whitespace-pre-wrap">{moment.description}</p>

            {/* Images */}
            {moment.images && moment.images.length > 0 && (
              <div
                className={cn(
                  'grid gap-2 rounded-xl overflow-hidden',
                  moment.images.length === 1 && 'grid-cols-1',
                  moment.images.length === 2 && 'grid-cols-2',
                  moment.images.length >= 3 && 'grid-cols-2'
                )}
              >
                {moment.images.map((image, index) => (
                  <div
                    key={index}
                    className={cn(
                      'relative bg-muted rounded-lg overflow-hidden',
                      moment.images!.length === 1 ? 'aspect-video max-h-96' : 'aspect-square'
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

            {/* Location */}
            {moment.location && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <MapPin className="h-4 w-4" />
                <span>{moment.location.location_name}</span>
              </div>
            )}

            {/* Tags */}
            {moment.tags && moment.tags.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {moment.tags.map((tag) => (
                  <Link
                    key={tag}
                    href={`/explore?tag=${encodeURIComponent(tag)}`}
                    className="px-3 py-1 rounded-full bg-primary/10 text-primary text-sm hover:bg-primary/20 transition-colors"
                  >
                    #{tag}
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between pt-6 mt-6 border-t">
            <div className="flex items-center gap-6">
              <button
                onClick={() => isAuthenticated && likeMutation.mutate()}
                disabled={!isAuthenticated || likeMutation.isPending}
                className={cn(
                  'flex items-center gap-2 transition-colors',
                  isLiked ? 'text-red-500' : 'text-muted-foreground hover:text-red-500',
                  !isAuthenticated && 'cursor-not-allowed opacity-50'
                )}
              >
                <Heart className={cn('h-5 w-5', isLiked && 'fill-current')} />
                <span className="font-medium">{likesCount}</span>
              </button>
              <div className="flex items-center gap-2 text-muted-foreground">
                <MessageCircle className="h-5 w-5" />
                <span className="font-medium">{moment.comments_count}</span>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => isAuthenticated && favoriteMutation.mutate()}
                disabled={!isAuthenticated || favoriteMutation.isPending}
                className={cn(
                  'p-2 rounded-full transition-colors',
                  isFavorited ? 'text-yellow-500' : 'text-muted-foreground hover:text-yellow-500',
                  !isAuthenticated && 'cursor-not-allowed opacity-50'
                )}
              >
                <Bookmark className={cn('h-5 w-5', isFavorited && 'fill-current')} />
              </button>
              <button className="p-2 rounded-full text-muted-foreground hover:text-primary transition-colors">
                <Share2 className="h-5 w-5" />
              </button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Comments Section */}
      <Card className="shadow-soft">
        <CardContent className="p-6">
          <h2 className="font-semibold text-lg mb-4">
            评论 ({moment.comments_count})
          </h2>

          {/* Comment Input */}
          {isAuthenticated ? (
            <div className="flex gap-3 mb-6">
              <Avatar className="h-10 w-10 shrink-0">
                <AvatarImage src={user?.avatar_url} alt={user?.nickname} />
                <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-sm">
                  {user?.nickname?.slice(0, 2).toUpperCase() || 'U'}
                </AvatarFallback>
              </Avatar>
              <div className="flex-1">
                {replyTo && (
                  <div className="flex items-center gap-2 mb-2 text-sm text-muted-foreground">
                    <span>回复 @{replyTo.user.nickname}</span>
                    <button
                      onClick={() => setReplyTo(null)}
                      className="text-xs hover:text-foreground"
                    >
                      取消
                    </button>
                  </div>
                )}
                <div className="flex gap-2">
                  <Input
                    placeholder={replyTo ? `回复 ${replyTo.user.nickname}...` : '写下你的评论...'}
                    value={newComment}
                    onChange={(e) => setNewComment(e.target.value)}
                    className="flex-1 rounded-full"
                  />
                  <Button
                    size="icon"
                    className="rounded-full shrink-0"
                    disabled={!newComment.trim() || commentMutation.isPending}
                    onClick={() => commentMutation.mutate()}
                  >
                    {commentMutation.isPending ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Send className="h-4 w-4" />
                    )}
                  </Button>
                </div>
              </div>
            </div>
          ) : (
            <div className="mb-6 p-4 rounded-lg bg-muted/50 text-center">
              <p className="text-muted-foreground text-sm mb-2">登录后即可发表评论</p>
              <Button asChild size="sm" className="rounded-full">
                <Link href={`/login?redirect=/moments/${momentId}`}>登录</Link>
              </Button>
            </div>
          )}

          {/* Comments List */}
          {commentsQuery.isLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : comments.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              暂无评论，快来抢沙发吧！
            </div>
          ) : (
            <div className="space-y-4">
              {comments.map((comment) => (
                <CommentItem
                  key={comment.comment_id}
                  comment={comment}
                  onReply={() => setReplyTo(comment)}
                />
              ))}

              {commentsQuery.hasNextPage && (
                <Button
                  variant="ghost"
                  className="w-full"
                  onClick={() => commentsQuery.fetchNextPage()}
                  disabled={commentsQuery.isFetchingNextPage}
                >
                  {commentsQuery.isFetchingNextPage ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    <ChevronDown className="mr-2 h-4 w-4" />
                  )}
                  加载更多评论
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

interface CommentItemProps {
  comment: CommentListItem;
  onReply: () => void;
}

function CommentItem({ comment, onReply }: CommentItemProps) {
  const { isAuthenticated } = useAuthStore();

  return (
    <div className="flex gap-3">
      <Link href={`/users/${comment.user.user_id}`}>
        <Avatar className="h-9 w-9">
          <AvatarImage src={comment.user.avatar_url} alt={comment.user.nickname} />
          <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-xs">
            {comment.user.nickname.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>
      </Link>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <Link
            href={`/users/${comment.user.user_id}`}
            className="font-medium text-sm hover:text-primary transition-colors"
          >
            {comment.user.nickname}
          </Link>
          <span className="text-xs text-muted-foreground">
            {new Date(comment.created_at).toLocaleDateString('zh-CN')}
          </span>
        </div>
        <p className="text-sm text-foreground">{comment.content}</p>
        <div className="flex items-center gap-4 mt-2">
          <button className="flex items-center gap-1 text-xs text-muted-foreground hover:text-red-500 transition-colors">
            <Heart className="h-3.5 w-3.5" />
            <span>{comment.likes_count}</span>
          </button>
          {isAuthenticated && (
            <button
              onClick={onReply}
              className="text-xs text-muted-foreground hover:text-primary transition-colors"
            >
              回复
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
