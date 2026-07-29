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

const ROLE_LABEL: Record<string, string> = { owner: "群主", admin: "管理员", member: "成员" };

/**
 * Map a spec GroupItem (`{ group: GroupInfo, description, my_state, ... }`) to the
 * flat shape SectionGroups/MyGroupCard consumes. Previously the raw spec item was
 * passed straight through, so every `g.name`/`g.avatar`/`g.members` read was
 * undefined and the avatar rendered as a broken-image icon.
 */
function mapGroup(item: any) {
  const g = item?.group ?? item ?? {};
  const role = item?.my_state?.role;
  return {
    id: g.group_id ?? "",
    name: g.name ?? "",
    avatar: g.avatar_url ?? "",
    // spec GroupItem has no banner; reuse the avatar so the banner area isn't a
    // broken-image icon. A dedicated banner field can be added when the spec does.
    banner: g.banner_url ?? g.avatar_url ?? "",
    members: g.members_count ?? 0,
    online: g.online_count ?? 0,
    desc: item?.description ?? g.description ?? "",
    role: role ? (ROLE_LABEL[role] ?? role) : "成员",
    lastActive: item?.last_activity_at ?? "",
    tags: item?.tags ?? g.tags ?? [],
  };
}

/** Days elapsed since `iso`, floored to ≥1 so "今天" reads sensibly. */
function daysSince(iso?: string): number {
  if (!iso) return 1;
  const diff = Date.now() - new Date(iso).getTime();
  return Math.max(1, Math.floor(diff / 86400000));
}

/**
 * Map the three spec relationship items to the flat shapes the Section components
 * expect. The spec nests UserInfo under `user:` and uses presence_status /
 * is_mutual / is_following_back — the components previously read avatar/name/
 * online/mutual/handle/posts directly off the raw item, so every field was
 * undefined.
 */
function mapFriend(item: any) {
  const u = item?.user ?? {};
  return {
    id: u.user_id ?? "",
    name: u.nickname ?? "",
    avatar: u.avatar_url ?? "",
    online: item?.presence_status === "online",
    handle: item?.remark ? `@${item.remark}` : `@${u.user_id ?? ""}`,
    mutual: true, // friends list is bidirectional by definition
    tag: "好友",
  };
}

function mapFollowing(item: any) {
  const u = item?.user ?? {};
  return {
    id: u.user_id ?? "",
    name: u.nickname ?? "",
    avatar: u.avatar_url ?? "",
    desc: item?.remark ?? item?.bio ?? "",
    posts: item?.moments_count ?? 0,
    latestImg: item?.latest_moment_image ?? "",
    isMutual: Boolean(item?.is_mutual),
  };
}

function mapFollower(item: any) {
  const u = item?.user ?? {};
  return {
    id: u.user_id ?? "",
    name: u.nickname ?? "",
    avatar: u.avatar_url ?? "",
    handle: item?.remark ? `@${item.remark}` : `@${u.user_id ?? ""}`,
    joinedDays: daysSince(item?.followed_at),
    followBack: Boolean(item?.is_following_back),
    isMutual: Boolean(item?.is_following_back),
  };
}

export function useMyGroups() {
  const [groups, setGroups] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listMyGroups({ pagination: { limit: 20 } }).then(res => {
      if (res.code === 0) setGroups(((res.data as any).items ?? []).map(mapGroup));
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
      setFriends(f.code === 0 ? ((f.data as any).items ?? []).map(mapFriend) : []);
      setFollowing(fo.code === 0 ? ((fo.data as any).items ?? []).map(mapFollowing) : []);
      setFollowers(fr.code === 0 ? ((fr.data as any).items ?? []).map(mapFollower) : []);
    }).finally(() => setLoading(false));
  }, []);

  return { friends, following, followers, loading };
}
