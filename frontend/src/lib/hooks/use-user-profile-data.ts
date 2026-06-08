import { useState, useEffect, useCallback } from "react";
import { usersService } from "@/lib/api/services/users";
import { isSuccess, TOKEN_INVALID_CODE } from "@/lib/api/client";
import type { UserItem, MomentItem } from "@/lib/types/api";

interface UserProfileData {
  user: UserItem | null;
  moments: MomentItem[];
  loading: boolean;
  error: string | null;
  isFollowing: boolean;
  followLoading: boolean;
  hasMoreMoments: boolean;
  loadMoreMoments: () => void;
  refresh: () => void;
  toggleFollow: () => void;
}

export function useUserProfileData(
  userId: string | undefined,
): UserProfileData {
  const [user, setUser] = useState<UserItem | null>(null);
  const [moments, setMoments] = useState<MomentItem[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFollowing, setIsFollowing] = useState(false);
  const [followLoading, setFollowLoading] = useState(false);

  const loadUser = useCallback(async () => {
    if (!userId) return;
    setLoading(true);
    setError(null);

    try {
      const res = await usersService.getUser({ user_id: userId });
      if (isSuccess(res) && res.data) {
        setUser(res.data);
        setIsFollowing(res.data.relation?.is_following ?? false);
      } else if (res.code !== TOKEN_INVALID_CODE) {
        setError("加载失败，请重试");
      }
    } catch (e) {
      setError("网络异常，请重试");
    } finally {
      setLoading(false);
    }
  }, [userId]);

  const loadMoments = useCallback(
    async (cursor?: string) => {
      if (!userId) return;

      try {
        const res = await usersService.getMoments({
          user_id: userId,
          pagination: { limit: 20, cursor },
        });

        if (isSuccess(res) && res.data) {
          if (cursor) {
            setMoments((prev) => [...prev, ...res.data!.items]);
          } else {
            setMoments(res.data.items);
          }
          setNextCursor(res.data.next_cursor);
        }
      } catch (error) {
        console.warn("Failed to load user moments", error);
      }
    },
    [userId],
  );

  useEffect(() => {
    loadUser();
    loadMoments();
  }, [loadUser, loadMoments]);

  const toggleFollow = useCallback(async () => {
    if (!userId || followLoading) return;
    setFollowLoading(true);

    try {
      const res = isFollowing
        ? await usersService.unfollow({ user_id: userId })
        : await usersService.follow({ user_id: userId });

      if (isSuccess(res)) {
        setIsFollowing((prev) => !prev);
        setUser((prev) =>
          prev
            ? {
                ...prev,
                stats: prev.stats
                  ? {
                      ...prev.stats,
                      followers_count:
                        prev.stats.followers_count + (isFollowing ? -1 : 1),
                    }
                  : prev.stats,
              }
            : prev,
        );
      }
    } catch {
    } finally {
      setFollowLoading(false);
    }
  }, [userId, isFollowing, followLoading]);

  return {
    user,
    moments,
    loading,
    error,
    isFollowing,
    followLoading,
    hasMoreMoments: !!nextCursor,
    loadMoreMoments: () => nextCursor && loadMoments(nextCursor),
    refresh: () => {
      loadUser();
      loadMoments();
    },
    toggleFollow,
  };
}
