import { useState, useEffect, useCallback } from "react";
import { momentsService } from "@/lib/api/services/moments";
import { commentsService } from "@/lib/api/services/comments";
import { isSuccess, TOKEN_INVALID_CODE } from "@/lib/api/client";
import type { MomentItem, CommentItem } from "@/lib/types/api";

interface PostDetailData {
  post: MomentItem | null;
  comments: CommentItem[];
  loading: boolean;
  error: string | null;
  isLiked: boolean;
  isFavorited: boolean;
  likesCount: number;
  commentsCount: number;
  hasMoreComments: boolean;
  actionLoading: boolean;
  loadMoreComments: () => void;
  refresh: () => void;
  toggleLike: () => void;
  toggleFavorite: () => void;
  addComment: (content: string) => Promise<boolean>;
  toggleCommentLike: (commentId: string, isLiked: boolean) => void;
}

export function usePostDetailData(
  momentId: string | undefined,
): PostDetailData {
  const [post, setPost] = useState<MomentItem | null>(null);
  const [comments, setComments] = useState<CommentItem[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isLiked, setIsLiked] = useState(false);
  const [isFavorited, setIsFavorited] = useState(false);
  const [likesCount, setLikesCount] = useState(0);
  const [commentsCount, setCommentsCount] = useState(0);
  const [actionLoading, setActionLoading] = useState(false);

  const loadPost = useCallback(async () => {
    if (!momentId) return;
    setLoading(true);
    setError(null);

    try {
      const res = await momentsService.getMoment({ moment_id: momentId });
      if (isSuccess(res) && res.data) {
        const m = res.data;
        setPost(m);
        setIsLiked(m.my_state?.is_liked ?? false);
        setIsFavorited(m.my_state?.is_favorited ?? false);
        setLikesCount(m.likes_count);
        setCommentsCount(m.comments_count);
      } else if (res.code !== TOKEN_INVALID_CODE) {
        setError("加载失败，请重试");
      }
    } catch (e) {
      setError("网络异常，请重试");
    } finally {
      setLoading(false);
    }
  }, [momentId]);

  const loadComments = useCallback(
    async (cursor?: string) => {
      if (!momentId) return;

      try {
        const res = await commentsService.getComments({
          moment_id: momentId,
          pagination: { limit: 20, cursor },
        });

        if (isSuccess(res) && res.data) {
          if (cursor) {
            setComments((prev) => [...prev, ...res.data!.items]);
          } else {
            setComments(res.data.items);
          }
          setNextCursor(res.data.next_cursor);
        }
      } catch {}
    },
    [momentId],
  );

  useEffect(() => {
    loadPost();
    loadComments();
  }, [loadPost, loadComments]);

  const toggleLike = useCallback(async () => {
    if (!momentId || actionLoading) return;
    setActionLoading(true);

    const wasLiked = isLiked;
    setIsLiked(!wasLiked);
    setLikesCount((c) => c + (wasLiked ? -1 : 1));

    try {
      const res = wasLiked
        ? await momentsService.unlike({ moment_id: momentId })
        : await momentsService.like({ moment_id: momentId });

      if (!isSuccess(res)) {
        setIsLiked(wasLiked);
        setLikesCount((c) => c + (wasLiked ? 1 : -1));
      }
    } catch {
      setIsLiked(wasLiked);
      setLikesCount((c) => c + (wasLiked ? 1 : -1));
    } finally {
      setActionLoading(false);
    }
  }, [momentId, isLiked, actionLoading]);

  const toggleFavorite = useCallback(async () => {
    if (!momentId || actionLoading) return;
    setActionLoading(true);

    const wasFav = isFavorited;
    setIsFavorited(!wasFav);

    try {
      const res = wasFav
        ? await momentsService.unfavorite({ moment_id: momentId })
        : await momentsService.favorite({ moment_id: momentId });

      if (!isSuccess(res)) {
        setIsFavorited(wasFav);
      }
    } catch {
      setIsFavorited(wasFav);
    } finally {
      setActionLoading(false);
    }
  }, [momentId, isFavorited, actionLoading]);

  const addComment = useCallback(
    async (content: string): Promise<boolean> => {
      if (!momentId || !content.trim()) return false;

      try {
        const res = await commentsService.create({
          moment_id: momentId,
          content: content.trim(),
        });

        if (isSuccess(res) && res.data) {
          setComments((prev) => [...prev, res.data!]);
          setCommentsCount((c) => c + 1);
          return true;
        }
        return false;
      } catch {
        return false;
      }
    },
    [momentId],
  );

  const toggleCommentLike = useCallback(
    async (commentId: string, wasLiked: boolean) => {
      setComments((prev) =>
        prev.map((c) => {
          if (c.comment_id !== commentId) return c;
          return {
            ...c,
            is_liked: !wasLiked,
            likes_count: c.likes_count + (wasLiked ? -1 : 1),
          };
        }),
      );
      try {
        const res = wasLiked
          ? await commentsService.unlike({ comment_id: commentId })
          : await commentsService.like({ comment_id: commentId });
        if (!isSuccess(res)) {
          setComments((prev) =>
            prev.map((c) => {
              if (c.comment_id !== commentId) return c;
              return {
                ...c,
                is_liked: wasLiked,
                likes_count: c.likes_count + (wasLiked ? 1 : -1),
              };
            }),
          );
        }
      } catch {
        setComments((prev) =>
          prev.map((c) => {
            if (c.comment_id !== commentId) return c;
            return {
              ...c,
              is_liked: wasLiked,
              likes_count: c.likes_count + (wasLiked ? 1 : -1),
            };
          }),
        );
      }
    },
    [],
  );

  return {
    post,
    comments,
    loading,
    error,
    isLiked,
    isFavorited,
    likesCount,
    commentsCount,
    hasMoreComments: !!nextCursor,
    actionLoading,
    loadMoreComments: () => nextCursor && loadComments(nextCursor),
    refresh: () => {
      loadPost();
      loadComments();
    },
    toggleLike,
    toggleFavorite,
    addComment,
    toggleCommentLike,
  };
}
