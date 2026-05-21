import { useState, useEffect, useCallback } from 'react';
import { usersService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import type {
  MyProfile,
  MomentItem,
  GroupItem,
  FriendItem,
  FollowingItem,
  FollowerItem,
} from '@/lib/types/api';

interface ProfileData {
  profile: MyProfile | null;
  moments: MomentItem[];
  likedMoments: MomentItem[];
  groups: GroupItem[];
  friends: FriendItem[];
  following: FollowingItem[];
  followers: FollowerItem[];
  loadingProfile: boolean;
  loadingMoments: boolean;
  loadingLiked: boolean;
  loadingGroups: boolean;
  loadingFriends: boolean;
  loadingFollowing: boolean;
  loadingFollowers: boolean;
  profileError: string | null;
  refreshProfile: () => void;
  refreshMoments: () => void;
  refreshLiked: () => void;
  refreshGroups: () => void;
  refreshFriends: () => void;
  refreshFollowing: () => void;
  refreshFollowers: () => void;
  hasMoreMoments: boolean;
  loadMoreMoments: () => void;
}

export function useProfileData(): ProfileData {
  const [profile, setProfile] = useState<MyProfile | null>(null);
  const [moments, setMoments] = useState<MomentItem[]>([]);
  const [likedMoments, setLikedMoments] = useState<MomentItem[]>([]);
  const [groups, setGroups] = useState<GroupItem[]>([]);
  const [friends, setFriends] = useState<FriendItem[]>([]);
  const [following, setFollowing] = useState<FollowingItem[]>([]);
  const [followers, setFollowers] = useState<FollowerItem[]>([]);

  const [loadingProfile, setLoadingProfile] = useState(true);
  const [loadingMoments, setLoadingMoments] = useState(false);
  const [loadingLiked, setLoadingLiked] = useState(false);
  const [loadingGroups, setLoadingGroups] = useState(false);
  const [loadingFriends, setLoadingFriends] = useState(false);
  const [loadingFollowing, setLoadingFollowing] = useState(false);
  const [loadingFollowers, setLoadingFollowers] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);

  const [nextMomentCursor, setNextMomentCursor] = useState<string | undefined>();

  const loadProfile = useCallback(async () => {
    setLoadingProfile(true);
    setProfileError(null);
    try {
      const res = await usersService.getMyProfile();
      if (isSuccess(res) && res.data) {
        setProfile(res.data);
      } else {
        setProfileError('加载失败，请重试');
      }
    } catch {
      setProfileError('网络异常，请重试');
    } finally {
      setLoadingProfile(false);
    }
  }, []);

  const loadMoments = useCallback(async () => {
    setLoadingMoments(true);
    try {
      const res = await usersService.getMoments({ user_id: profile?.user_id ?? '', pagination: { limit: 20 } });
      if (isSuccess(res) && res.data) {
        setMoments(res.data.items);
        setNextMomentCursor(res.data.next_cursor);
      }
    } catch {
      // silent
    } finally {
      setLoadingMoments(false);
    }
  }, [profile?.user_id]);

  const loadLiked = useCallback(async () => {
    setLoadingLiked(true);
    try {
      const res = await usersService.getLikedMoments({ user_id: profile?.user_id ?? '', pagination: { limit: 20 } });
      if (isSuccess(res) && res.data) {
        setLikedMoments(res.data.items);
      }
    } catch {
      // silent
    } finally {
      setLoadingLiked(false);
    }
  }, [profile?.user_id]);

  const loadGroups = useCallback(async () => {
    if (!profile?.user_id) return;
    setLoadingGroups(true);
    try {
      const res = await usersService.getGroups({ user_id: profile.user_id, pagination: { limit: 50 } });
      if (isSuccess(res) && res.data) {
        setGroups(res.data.items);
      }
    } catch {
      // silent
    } finally {
      setLoadingGroups(false);
    }
  }, [profile?.user_id]);

  const loadFriends = useCallback(async () => {
    setLoadingFriends(true);
    try {
      const res = await usersService.getFriends({ pagination: { limit: 50 } });
      if (isSuccess(res) && res.data) {
        setFriends(res.data.items);
      }
    } catch {
      // silent
    } finally {
      setLoadingFriends(false);
    }
  }, []);

  const loadFollowing = useCallback(async () => {
    setLoadingFollowing(true);
    try {
      const res = await usersService.getFollowing({ pagination: { limit: 50 } });
      if (isSuccess(res) && res.data) {
        setFollowing(res.data.items);
      }
    } catch {
      // silent
    } finally {
      setLoadingFollowing(false);
    }
  }, []);

  const loadFollowers = useCallback(async () => {
    setLoadingFollowers(true);
    try {
      const res = await usersService.getFollowers({ pagination: { limit: 50 } });
      if (isSuccess(res) && res.data) {
        setFollowers(res.data.items);
      }
    } catch {
      // silent
    } finally {
      setLoadingFollowers(false);
    }
  }, []);

  const loadMoreMoments = useCallback(async () => {
    if (!nextMomentCursor || !profile?.user_id) return;
    try {
      const res = await usersService.getMoments({
        user_id: profile.user_id,
        pagination: { limit: 20, cursor: nextMomentCursor },
      });
      if (isSuccess(res) && res.data) {
        setMoments((prev) => [...prev, ...res.data!.items]);
        setNextMomentCursor(res.data.next_cursor);
      }
    } catch {
      // silent
    }
  }, [profile?.user_id, nextMomentCursor]);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  useEffect(() => {
    if (profile?.user_id) loadMoments();
  }, [profile?.user_id, loadMoments]);

  return {
    profile,
    moments,
    likedMoments,
    groups,
    friends,
    following,
    followers,
    loadingProfile,
    loadingMoments,
    loadingLiked,
    loadingGroups,
    loadingFriends,
    loadingFollowing,
    loadingFollowers,
    profileError,
    refreshProfile: loadProfile,
    refreshMoments: loadMoments,
    refreshLiked: loadLiked,
    refreshGroups: loadGroups,
    refreshFriends: loadFriends,
    refreshFollowing: loadFollowing,
    refreshFollowers: loadFollowers,
    hasMoreMoments: !!nextMomentCursor,
    loadMoreMoments,
  };
}
