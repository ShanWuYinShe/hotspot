'use client'
import React, { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation"
import { useRouter } from "next/navigation";
import { ROUTES } from "@/lib/routes";
import { ActionIconButton } from "@/components/shared/ActionIconButton";
import { PageHeader } from "@/components/shared/PageHeader";
import { Avatar } from "@/components/shared/Avatar";
import { useDismissibleLayer } from "@/hooks/useDismissibleLayer";
import { useAuth } from "@/contexts/AuthContext";
import {
  MapPin, Heart, MessageCircle, Share2, Bookmark,
  MoreHorizontal, Send, Link2, Check,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import {
  DesktopDetailSplit, RailSection, RailUserRow, RailLocationMini,
} from "@/components/layout/DetailRail";
import {
  getMomentDetail, listComments, setReaction, createComment, favoriteMoment,
} from "@/api";

interface PostComment { id: number; user: string; avatar: string; text: string; time: string; likes: number; }

interface PostData {
  id: number; user: string; handle: string; avatar: string;
  location: string; image: string; content: string;
  time: string; likes: number; tag: string;
  comments: PostComment[];
}

const DEFAULT_POST: PostData = {
  id: 0, user: "", handle: "", avatar: "",
  location: "", image: "", content: "",
  time: "", likes: 0, tag: "",
  comments: [],
};

export function MomentDetailPage() {
  const { postId } = useParams<{ postId: string }>();
  const router = useRouter();
  const { user } = useAuth();
  const numericPostId = Number(postId);
  const [post, setPost] = useState<PostData>(DEFAULT_POST);
  const [apiComments, setApiComments] = useState<PostComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [input, setInput] = useState("");
  const [moreOpen, setMoreOpen] = useState(false);
  const [shareStatus, setShareStatus] = useState<"idle" | "copied" | "failed">("idle");
  const moreMenuRef = useRef<HTMLDivElement>(null);
  // Reaction/favorite state derived from the moment detail (spec: liked state
  // comes from reactions[].reacted_by_me; saved from my_state.is_favorited).
  const [liked, setLiked] = useState(false);
  const [saved, setSaved] = useState(false);
  const [likeEmoji, setLikeEmoji] = useState("👍");

  useEffect(() => {
    let cancelled = false;
    const momentId = String(numericPostId);
    async function fetchPost() {
      try {
        setLoading(true);
        const [detailRes, commentsRes] = await Promise.all([
          getMomentDetail(momentId).catch(() => ({ code: -1, data: null })),
          listComments(momentId).catch(() => ({ code: -1, data: { items: [] } })),
        ]);
        if (cancelled) return;

        const detailData = (detailRes.code === 0 ? detailRes.data : null) as any;
        const commentsData = (commentsRes.code === 0 ? commentsRes.data : null) as any;

        if (detailData) {
          const pd: PostData = {
            id: detailData.id ?? detailData.moment_id ?? numericPostId,
            user: detailData.user?.nickname ?? detailData.author_name ?? detailData.nickname ?? "",
            handle: detailData.handle ?? detailData.username ?? "",
            avatar: detailData.avatar ?? detailData.user?.avatar_url ?? detailData.author_avatar ?? "",
            location: detailData.location?.location_name ?? detailData.place ?? "",
            image: detailData.image ?? detailData.images?.[0]?.url ?? detailData.images?.[0] ?? "",
            content: detailData.content ?? detailData.text ?? detailData.body ?? "",
            time: detailData.time ?? detailData.created_at ?? "",
            likes: detailData.likes_count ?? 0,
            tag: detailData.tag ?? detailData.tags?.[0] ?? "",
            comments: [],
          };
          setPost(pd);
          // Derive liked from reactions[].reacted_by_me (spec); saved from my_state.
          const mine = (detailData.reactions || []).find((r: any) => r.reacted_by_me);
          if (mine) { setLiked(true); setLikeEmoji(mine.emoji ?? "👍"); } else { setLiked(false); }
          setSaved(Boolean(detailData.my_state?.is_favorited));
        }

        if (commentsData) {
          const items: PostComment[] = (commentsData.items || []).map((c: any, i: number) => ({
            id: c.id ?? i + 1,
            user: c.user?.nickname ?? c.author_name ?? c.nickname ?? "",
            avatar: c.avatar ?? c.user?.avatar_url ?? c.author_avatar ?? "",
            text: c.text ?? c.content ?? c.body ?? "",
            time: c.time ?? c.created_at ?? "",
            likes: (c.reactions || []).reduce((s: number, r: any) => s + (r.count ?? 0), 0),
          }));
          setApiComments(items);
        }
      } catch {
        // empty data is normal
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchPost();
    return () => { cancelled = true; };
  }, [numericPostId]);

  const commentCount = apiComments.length;

  const handleComment = async () => {
    const text = input.trim();
    if (!text) return;
    setInput("");
    // Optimistic append; createComment confirms server-side.
    const optimistic: PostComment = {
      id: Date.now(),
      user: user?.nickname ?? "",
      avatar: user?.avatar_url ?? "",
      text,
      time: "刚刚",
      likes: 0,
    };
    setApiComments((prev) => [...prev, optimistic]);
    try {
      await createComment({ moment_id: String(numericPostId), content: text });
    } catch {
      setApiComments((prev) => prev.filter((c) => c.id !== optimistic.id));
    }
  };

  const toggleLike = async () => {
    const next = !liked;
    setLiked(next);
    try {
      await setReaction({ target_type: "moment", target_id: String(numericPostId), emoji: likeEmoji, active: next });
    } catch {
      setLiked(!next);
    }
  };

  const toggleSave = async () => {
    const next = !saved;
    setSaved(next);
    try {
      await favoriteMoment({ moment_id: String(numericPostId), active: next });
    } catch {
      setSaved(!next);
    }
  };

  const copyPostLink = async () => {
    const postUrl = `${window.location.origin}${ROUTES.moment(numericPostId)}`;
    try {
      if (!navigator.clipboard?.writeText) throw new Error("Clipboard API unavailable");
      await navigator.clipboard.writeText(postUrl);
      setShareStatus("copied");
    } catch {
      setShareStatus("failed");
    }
    setMoreOpen(false);
    window.setTimeout(() => setShareStatus("idle"), 1800);
  };

  useDismissibleLayer({
    open: moreOpen,
    onDismiss: () => setMoreOpen(false),
    rootRef: moreMenuRef,
    closeOnOutsidePress: true,
  });

  const main = (
    <div className="flex flex-col bg-[var(--neu-bg)]" style={{ height: "100%" }}>
      {/* Header */}
      <PageHeader
        title={post.user || "详情"}
        onBack={() => router.back()}
        className="flex-shrink-0 border-b border-[var(--neu-border)]"
        actions={
          <div ref={moreMenuRef} className="relative">
            <ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => setMoreOpen((open) => !open)} />
            <AnimatePresence>
              {moreOpen && (
                <motion.div
                initial={{ opacity: 0, y: -4, scale: 0.98 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -4, scale: 0.98 }}
                role="menu"
                aria-label="动态更多操作"
                className="absolute right-0 top-[calc(100%+8px)] z-30 w-36 overflow-hidden rounded-[14px] neu-surface-flat p-1.5"
                style={{ boxShadow: "0 12px 30px var(--neu-shadow-dark), 0 0 0 1px var(--neu-border)" }}
              >
                <button
                  type="button"
                  role="menuitem"
                  onClick={copyPostLink}
                  className="flex w-full items-center gap-2 rounded-[10px] px-2.5 py-2 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] text-[var(--neu-text)] text-xs"
                >
                  <Link2 style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                  复制链接
                </button>
                <button
                  type="button"
                  role="menuitem"
                  onClick={() => setMoreOpen(false)}
                  className="flex w-full items-center gap-2 rounded-[10px] px-2.5 py-2 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] text-[var(--neu-text-muted)] text-xs"
                >
                  取消
                </button>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        }
      />

      <p aria-live="polite" className="sr-only">
        {shareStatus === "copied" ? "动态链接已复制" : shareStatus === "failed" ? "无法复制动态链接" : ""}
      </p>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto no-scrollbar">
        {/* Post image */}
        <div className="relative overflow-hidden" style={{ height: "clamp(220px, 38vw, 380px)" }}>
          {post.image ? (
            <img src={post.image} alt="post" className="w-full h-full object-cover" />
          ) : (
            <div className="flex h-full w-full flex-col items-center justify-center gap-2 bg-[color-mix(in_srgb,var(--neu-accent-primary)_12%,var(--neu-bg))]">
              <MapPin size={28} className="text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
              <span className="text-[var(--neu-text-muted)] text-xs">来自 {post.location || "附近"} 的动态</span>
            </div>
          )}
          <div className="absolute top-3 left-3 px-2.5 py-0.5 rounded-full bg-[var(--neu-accent-primary)]"
            style={{ boxShadow: "0 2px 8px var(--neu-glow)" }}>
            <span className="text-white text-[10px]">#{post.tag}</span>
          </div>
        </div>

        <div className="px-5 pt-4 pb-3"
          style={{ maxWidth: "var(--content-reading-max)", marginLeft: "auto", marginRight: "auto", width: "100%" }}>
          {/* Author */}
          <div className="flex items-center gap-3 mb-3">
            <motion.button type="button" whileTap={{ scale: 0.95 }} onClick={() => router.push(ROUTES.user(post.id))}
              aria-label={`查看${post.user}的个人主页`}
              className="rounded-full overflow-hidden flex-shrink-0 cursor-pointer neu-focus-ring focus-visible:ring-offset-[var(--neu-bg)]"
              style={{ width: 44, height: 44, padding: 0, background: "transparent", border: "2px solid var(--neu-accent-primary)" }}>
              <Avatar src={post.avatar} name={post.user} size={44} style={{ border: "none" }} />
            </motion.button>
            <div className="flex-1">
              <p className="text-[var(--neu-text)] text-[15px]">{post.user}</p>
              <div className="flex items-center gap-1.5">
                <span className="text-[var(--neu-text-muted)] text-xs">{post.handle}</span>
                <span className="text-[var(--neu-text-muted)] text-[10px]" style={{ opacity: 0.5 }}>·</span>
                <MapPin style={{ width: 10, height: 10, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                <span className="text-[var(--neu-text-muted)] text-[11px]">{post.location}</span>
              </div>
            </div>
            <span className="text-[var(--neu-text-muted)] text-[11px]">{post.time}</span>
          </div>

          {/* Content */}
          <p className="text-[var(--neu-text)] text-sm" style={{ lineHeight: 1.75 }}>{post.content}</p>

          {/* Actions */}
          <div className="flex items-center justify-between mt-4 py-3"
            style={{ borderTop: "1px solid var(--neu-border)", borderBottom: "1px solid var(--neu-border)" }}>
            <div className="flex gap-4">
              <motion.button whileTap={{ scale: 0.85 }}
                onClick={() => toggleLike()}
                className={`flex items-center gap-1.5 outline-none ${liked ? "text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"}`}>
                <Heart style={{ width: 19, height: 19 }} strokeWidth={1.5} fill={liked ? "currentColor" : "none"} />
                <span className="text-[13px]">{post.likes + (liked ? 1 : 0)}</span>
              </motion.button>
              <div className="flex items-center gap-1.5 text-[var(--neu-text-muted)]">
                <MessageCircle style={{ width: 19, height: 19 }} strokeWidth={1.5} />
                <span className="text-[13px]">{commentCount}</span>
              </div>
            </div>
            <div className="flex gap-3">
              <motion.button whileTap={{ scale: 0.85 }} onClick={() => toggleSave()}
                className={`outline-none ${saved ? "text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"}`}>
                <Bookmark style={{ width: 19, height: 19 }} strokeWidth={1.5} fill={saved ? "currentColor" : "none"} />
              </motion.button>
              <motion.button whileTap={{ scale: 0.85 }} onClick={copyPostLink} aria-label={shareStatus === "failed" ? "复制动态链接失败" : "复制动态链接"} className={`rounded-full neu-focus-ring ${shareStatus === "copied" ? "text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"}`}>
                {shareStatus === "copied" ? <Check style={{ width: 19, height: 19 }} strokeWidth={1.8} /> : <Share2 style={{ width: 19, height: 19 }} strokeWidth={1.5} />}
              </motion.button>
            </div>
          </div>

          {/* Comments */}
          <p className="uppercase mt-4 mb-3 text-[var(--neu-text-muted)] text-[10px]" style={{ letterSpacing: "0.14em" }}>
            评论 · {commentCount}
          </p>
          <div className="flex flex-col gap-4">
            {apiComments.map(c => (
              <div key={c.id} className="flex gap-3">
                <Avatar src={c.avatar} name={c.user} size={34} />
                <div className="flex-1">
                  <div className="flex items-baseline gap-2">
                    <span className="text-[var(--neu-text)] text-[13px]">{c.user}</span>
                    <span className="text-[var(--neu-text-muted)] text-[11px]">{c.time}</span>
                  </div>
                  <p className="text-[var(--neu-text-muted)] text-[13px]" style={{ lineHeight: 1.6, marginTop: 2 }}>{c.text}</p>
                  <div className="flex items-center gap-1 mt-1.5">
                    <motion.button whileTap={{ scale: 0.85 }} className="flex items-center gap-1 outline-none text-[var(--neu-text-muted)]">
                      <Heart style={{ width: 12, height: 12 }} strokeWidth={1.5} />
                      <span className="text-[11px]">{c.likes}</span>
                    </motion.button>
                    <span className="text-[var(--neu-text-muted)] text-[11px]" style={{ marginLeft: 8, cursor: "pointer" }}>回复</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div style={{ height: 80 }} />
        </div>
      </div>

      {/* Input bar */}
      <div className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--neu-bg)]"
        style={{ borderTop: "1px solid var(--neu-border)" }}>
        <Avatar src={user?.avatar_url} name={user?.nickname} size={30}
          style={{ border: "1px solid var(--neu-border)" }} />
        <div className="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}>
          <input value={input} onChange={e => setInput(e.target.value)} onKeyDown={e => e.key === "Enter" && handleComment()}
            placeholder="写下你的评论…" className="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-[13px]" />
        </div>
        <motion.button whileTap={{ scale: 0.88 }} onClick={handleComment}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{ width: 36, height: 36, background: input.trim() ? "var(--neu-accent-primary)" : "var(--neu-bg)", boxShadow: input.trim() ? "0 4px 14px var(--neu-glow)" : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", transition: "all 0.2s" }}>
          <Send style={{ width: 14, height: 14, color: input.trim() ? "white" : "var(--neu-text-muted)", transform: "translateX(1px)" }} strokeWidth={2} />
        </motion.button>
      </div>
    </div>
  );

  const rail = (
    <>
      <RailSection title="作者">
        <RailUserRow
          avatar={post.avatar}
          name={post.user}
          sub={`${post.handle} · ${post.location}`}
          onClick={() => router.push(ROUTES.user(post.id))}
          trailing={
            <motion.button
              whileTap={{ scale: 0.94 }}
              onClick={(e) => { e.stopPropagation(); router.push(ROUTES.message(post.id)); }}
              className="px-2.5 py-1 rounded-full outline-none flex-shrink-0 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]"
              style={{ fontSize: 11, border: "1px solid var(--neu-accent-soft-border)" }}
            >
              私信
            </motion.button>
          }
        />
      </RailSection>

      <RailSection title="相关位置">
        <RailLocationMini name={post.location} />
      </RailSection>
    </>
  );

  return <DesktopDetailSplit main={main} rail={rail} />;
}