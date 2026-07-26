import React, { createContext, useContext, useEffect, useMemo, useState } from "react";

export type PostAudience = "nearby" | "friends" | "private";

export type SocialComment = {
  id: number;
  user: string;
  avatar: string;
  text: string;
  time: string;
  likes: number;
};

export type SocialPost = {
  id: number;
  user: string;
  location: string;
  tag: string;
  avatar: string;
  content: string;
  image?: string;
  likes: number;
  comments: number;
  time: string;
  moods: string[];
  audience: PostAudience;
};

type CreatePostInput = {
  content: string;
  moods: string[];
  location: string;
  audience: PostAudience;
  hasImage: boolean;
};

type SocialContextValue = {
  profilePosts: SocialPost[];
  discoverPosts: SocialPost[];
  likedIds: Set<number>;
  savedIds: Set<number>;
  commentsByPost: Record<number, SocialComment[]>;
  createPost: (input: CreatePostInput) => void;
  toggleLike: (postId: number) => void;
  toggleSave: (postId: number) => void;
  addComment: (postId: number, text: string) => void;
};

const STORAGE_KEY = "hotspot.local-posts.v1";
const PREFERENCES_KEY = "hotspot.post-preferences.v1";
const USER_AVATAR =
  "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400";
const COMPOSER_IMAGE =
  "https://images.unsplash.com/photo-1762113396314-8e9e5417ed17?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1000";

const SocialContext = createContext<SocialContextValue | null>(null);

function readStored<T>(key: string, fallback: T): T {
  try {
    const stored = window.localStorage.getItem(key);
    return stored ? JSON.parse(stored) as T : fallback;
  } catch {
    return fallback;
  }
}

function readPosts(): SocialPost[] {
  const parsed = readStored<unknown>(STORAGE_KEY, []);
  return Array.isArray(parsed) ? parsed as SocialPost[] : [];
}

function readPreferences() {
  const parsed = readStored<{ likedIds?: number[]; savedIds?: number[]; commentsByPost?: Record<string, SocialComment[]> }>(PREFERENCES_KEY, {});
  const commentsByPost = Object.fromEntries(
    Object.entries(parsed.commentsByPost ?? {}).filter(([, comments]) => Array.isArray(comments))
  ) as Record<number, SocialComment[]>;
  return {
    likedIds: new Set(Array.isArray(parsed.likedIds) ? parsed.likedIds : []),
    savedIds: new Set(Array.isArray(parsed.savedIds) ? parsed.savedIds : []),
    commentsByPost,
  };
}

/** Client-side content store. It preserves the full publishing contract until a backend is connected. */
export function SocialProvider({ children }: { children: React.ReactNode }) {
  const [profilePosts, setProfilePosts] = useState<SocialPost[]>(() =>
    typeof window === "undefined" ? [] : readPosts()
  );
  const [likedIds, setLikedIds] = useState<Set<number>>(() =>
    typeof window === "undefined" ? new Set() : readPreferences().likedIds
  );
  const [savedIds, setSavedIds] = useState<Set<number>>(() =>
    typeof window === "undefined" ? new Set() : readPreferences().savedIds
  );
  const [commentsByPost, setCommentsByPost] = useState<Record<number, SocialComment[]>>(() =>
    typeof window === "undefined" ? {} : readPreferences().commentsByPost
  );

  useEffect(() => {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(profilePosts));
    } catch {
      // Storage can be unavailable in private or constrained browser contexts; the in-memory feed still works.
    }
  }, [profilePosts]);

  useEffect(() => {
    try {
      window.localStorage.setItem(PREFERENCES_KEY, JSON.stringify({
        likedIds: Array.from(likedIds),
        savedIds: Array.from(savedIds),
        commentsByPost,
      }));
    } catch {
      // The interaction still remains usable in memory if persistence is unavailable.
    }
  }, [commentsByPost, likedIds, savedIds]);

  const value = useMemo<SocialContextValue>(() => ({
    profilePosts,
    discoverPosts: profilePosts.filter((post) => post.audience !== "private"),
    likedIds,
    savedIds,
    commentsByPost,
    createPost: ({ content, moods, location, audience, hasImage }) => {
      const post: SocialPost = {
        id: Date.now(),
        user: "鹿鸣",
        location,
        tag: moods[0] ?? "生活",
        avatar: USER_AVATAR,
        content: content.trim(),
        image: hasImage ? COMPOSER_IMAGE : undefined,
        likes: 0,
        comments: 0,
        time: "刚刚",
        moods,
        audience,
      };
      setProfilePosts((current) => [post, ...current]);
    },
    toggleLike: (postId) => setLikedIds((current) => {
      const next = new Set(current);
      next.has(postId) ? next.delete(postId) : next.add(postId);
      return next;
    }),
    toggleSave: (postId) => setSavedIds((current) => {
      const next = new Set(current);
      next.has(postId) ? next.delete(postId) : next.add(postId);
      return next;
    }),
    addComment: (postId, text) => {
      const trimmedText = text.trim();
      if (!trimmedText) return;
      const comment: SocialComment = {
        id: Date.now(),
        user: "鹿鸣",
        avatar: USER_AVATAR,
        text: trimmedText,
        time: "刚刚",
        likes: 0,
      };
      setCommentsByPost((current) => ({
        ...current,
        [postId]: [...(current[postId] ?? []), comment],
      }));
    },
  }), [commentsByPost, likedIds, profilePosts, savedIds]);

  return <SocialContext.Provider value={value}>{children}</SocialContext.Provider>;
}

export function useSocial() {
  const context = useContext(SocialContext);
  if (!context) throw new Error("useSocial must be used within SocialProvider");
  return context;
}
