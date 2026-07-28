import { useState, useEffect } from "react";
import {
  getAccount, getProfile, listUserMoments, listUserReactedMoments,
  listMyMoments, getFavoritePosts, listMyGroups,
  listFriends, listFollowing, listFollowers,
  type ApiResponse, type PaginatedData,
} from "../../../api";

export interface GalleryPost {
  id: string; img: string; likes: number; comments: number;
}

export interface ProfileStats {
  moments_count: number; friends_count: number;
  followers_count: number; following_count: number; groups_count: number;
}

export function useAccountData() {
  const [avatar, setAvatar] = useState("");
  const [nickname, setNickname] = useState("");
  const [stats, setStats] = useState<ProfileStats>({ moments_count: 0, friends_count: 0, followers_count: 0, following_count: 0, groups_count: 0 });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getAccount().then(res => {
      if (res.code === 0 && res.data) {
        const d = res.data as any;
        setAvatar(d.avatar_url ?? "");
        setNickname(d.nickname ?? "");
        setStats(d.stats ?? stats);
      }
    }).finally(() => setLoading(false));
  }, []);

  return { avatar, nickname, stats, loading };
}

export function useGalleryPosts(type: "my" | "user" = "my", userId?: string) {
  const [posts, setPosts] = useState<GalleryPost[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetcher = type === "my" ? listMyMoments : () => listUserMoments(userId!);
    fetcher({ pagination: { limit: 20 } }).then(res => {
      if (res.code === 0 && res.data) {
        setPosts((res.data as any).items?.map((m: any) => ({
          id: m.moment_id, img: m.images?.[0]?.url ?? "", likes: m.likes_count ?? 0, comments: m.comments_count ?? 0,
        })) ?? []);
      }
    }).finally(() => setLoading(false));
  }, [type, userId]);

  return { posts, loading };
}

export function useReactedPosts(userId?: string) {
  const [posts, setPosts] = useState<GalleryPost[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listUserReactedMoments(userId ?? "", { pagination: { limit: 20 } }).then(res => {
      if (res.code === 0 && res.data) {
        setPosts((res.data as any).items?.map((m: any) => ({
          id: m.moment_id, img: m.images?.[0]?.url ?? m.images?.[0] ?? "", likes: m.likes_count ?? 0, comments: m.comments_count ?? 0,
        })) ?? []);
      }
    }).finally(() => setLoading(false));
  }, [userId]);

  return { posts, loading };
}

export function useFavoritePosts() {
  const [posts, setPosts] = useState<GalleryPost[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getFavoritePosts({ pagination: { limit: 20 } }).then(res => {
      if (res.code === 0 && res.data) {
        setPosts((res.data as any).items?.map((m: any) => ({
          id: m.moment_id, img: m.images?.[0]?.url ?? m.images?.[0] ?? "", likes: m.likes_count ?? 0, comments: m.comments_count ?? 0,
        })) ?? []);
      }
    }).finally(() => setLoading(false));
  }, []);

  return { posts, loading };
}

export function useMyGroups() {
  const [groups, setGroups] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listMyGroups({ pagination: { limit: 20 } }).then(res => {
      if (res.code === 0) setGroups((res.data as any).items ?? []);
    }).finally(() => setLoading(false));
  }, []);

  return { groups, loading };
}

/** Relationship lists — friends / following / followers. */
export function useRelationships() {
  const [friends, setFriends] = useState<any[]>([]);
  const [following, setFollowing] = useState<any[]>([]);
  const [followers, setFollowers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      listFriends({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
      listFollowing({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
      listFollowers({ pagination: { limit: 50 } }).catch(() => ({ code: -1, data: { items: [] } })),
    ]).then(([f, fo, fr]) => {
      setFriends(f.code === 0 ? (f.data as any).items ?? [] : []);
      setFollowing(fo.code === 0 ? (fo.data as any).items ?? [] : []);
      setFollowers(fr.code === 0 ? (fr.data as any).items ?? [] : []);
    }).finally(() => setLoading(false));
  }, []);

  return { friends, following, followers, loading };
}
