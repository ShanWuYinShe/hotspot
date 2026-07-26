import { useState, useEffect } from "react";
import {
  getAccount, getProfile, listUserMoments, listUserReactedMoments,
  listMyMoments, listMyGroups, type ApiResponse, type PaginatedData,
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
          id: m.moment_id, img: m.image_url ?? "", likes: m.like_count ?? 0, comments: m.comment_count ?? 0,
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
          id: m.moment_id, img: m.image_url ?? "", likes: m.like_count ?? 0, comments: m.comment_count ?? 0,
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
    import("../../../api").then(({ Endpoints }) => {
      fetch(`${Endpoints.Moments.ListFavorites}`, {
        method: "POST", credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pagination: { limit: 20 } }),
      }).then(r => r.json()).then((res: ApiResponse<any>) => {
        if (res.code === 0) {
          setPosts((res.data as any).items?.map((m: any) => ({
            id: m.moment_id, img: m.image_url ?? "", likes: m.like_count ?? 0, comments: m.comment_count ?? 0,
          })) ?? []);
        }
      }).finally(() => setLoading(false));
    });
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

// Backward-compatible empty defaults (used by ProfileSections until fully migrated to hooks).
// When DB is empty, these show zero state which is the correct behavior.
export const USER_AVATAR = "";
export const GALLERY_POSTS: GalleryPost[] = [];
export const LIKED_POSTS: GalleryPost[] = [];
export const SAVED_POSTS: GalleryPost[] = [];
export const STATS: { label: string; value: number }[] = [
  { label: "瞬间", value: 0 },
  { label: "好友", value: 0 },
  { label: "足迹", value: 0 },
];
export const MOCK_GROUPS: any[] = [];
export const MOCK_FRIENDS: any[] = [];
export const MOCK_FOLLOWING: any[] = [];
export const MOCK_FANS: any[] = [];
