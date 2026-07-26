import React, { useRef, useState } from "react";
import { useParams, useNavigate } from "react-router";
import { ROUTES } from "../lib/routes";
import { useSocial } from "../contexts/SocialContext";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { useDismissibleLayer } from "../hooks/useDismissibleLayer";
import {
  ArrowLeft, MapPin, Heart, MessageCircle, Share2, Bookmark,
  MoreHorizontal, Send, ChevronRight, Link2, Check,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import {
  DesktopDetailSplit, RailSection, RailUserRow, RailPostThumb, RailLocationMini,
} from "../components/layout/DetailRail";

/* ── Mock related content for desktop rail ─────────────────────── */
const RELATED_POSTS = [
  { id: 41, image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=200", title: "市中心隐藏茶馆", meta: "本地向导 · 0.8km" },
  { id: 43, image: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=200", title: "云龙湖落日", meta: "小满 · 1.2km" },
  { id: 44, image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=200", title: "老街胶片漫步", meta: "鹿鸣 · 1.5km" },
];

const NEARBY_NOW = [
  { id: 2, name: "阿禾",   avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80", sub: "252m · 在线" },
  { id: 4, name: "Mika",   avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80", sub: "520m · 在线" },
  { id: 5, name: "青砚",   avatar: "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=80", sub: "614m · 在线" },
];

/* ── Mock data ─────────────────────────────────────────────────── */
const POSTS: Record<number, {
  id: number; user: string; handle: string; avatar: string;
  location: string; image: string; content: string;
  time: string; likes: number; tag: string;
  comments: Array<{ id: number; user: string; avatar: string; text: string; time: string; likes: number }>;
}> = {
  40: {
    id: 40, user: "Aria Sol", handle: "@aria.sol", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
    location: "植物园", image: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=1080",
    content: "清晨在绿意中找到了一片宁静。黎明时分阳光穿过树叶，滤出完美的光影。推荐大家周末早起来这里散步，绝对值得！这种时刻让人感觉整个世界都属于自己。",
    time: "2小时前", likes: 124, tag: "自然",
    comments: [
      { id: 1, user: "野原",   avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",    text: "太美了！我也想去",              time: "1小时前",  likes: 8 },
      { id: 2, user: "林一",    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",   text: "这光线真的绝了，什么时候去的？", time: "45分钟前", likes: 3 },
      { id: 3, user: "小满",avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",   text: "我们下次一起去吧 🌿",           time: "20分钟前", likes: 12 },
    ],
  },
  41: {
    id: 41, user: "本地向导", handle: "@local.guide", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",
    location: "市中心核心区", image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=1080",
    content: "城市里那些安静的角落已经被我全部标注出来了。每一条小巷都有属于自己的故事，等待有心人去发现。\n\n这次找到了一条被遗忘的石板路，通向一家隐藏了30年的老茶馆。",
    time: "5小时前", likes: 89, tag: "城市",
    comments: [
      { id: 1, user: "阿川",    avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80", text: "请问地址方便分享吗？",  time: "3小时前",  likes: 5 },
      { id: 2, user: "阿禾", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80", text: "老茶馆！我一定要去！", time: "2小时前",  likes: 14 },
    ],
  },
};

const DEFAULT_POST = POSTS[40];
const FEED_COMMENT_COUNTS: Record<number, number> = {
  1: 12, 2: 5, 3: 8, 4: 34, 5: 27, 6: 19,
};

/* Discover uses IDs 1–6. Keep its cards and the detail route on the same data contract. */
const DISCOVER_DETAIL_POSTS: Record<number, typeof DEFAULT_POST> = {
  1: { ...POSTS[40], id: 1 },
  2: { ...POSTS[41], id: 2 },
  3: {
    ...DEFAULT_POST, id: 3, user: "野原", handle: "@yuan.wander",
    avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",
    location: "老街手冲馆", image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=1080",
    tag: "生活", likes: 56, time: "昨天",
    content: "周末午后最适合窝在这里看书，点杯手冲，看窗外的行人。安静的座位不多，适合慢慢待上一整个下午。",
    comments: [],
  },
  4: {
    ...DEFAULT_POST, id: 4, user: "小满", handle: "@xiaoman.xz",
    location: "云龙湖畔", image: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=1080",
    tag: "社群", likes: 201, time: "周一",
    content: "湖畔的周末市集开始了！有好多有意思的手工艺品和本地小吃，傍晚还有一场小型现场演出。",
    comments: [],
  },
  5: {
    ...DEFAULT_POST, id: 5, user: "林一", handle: "@linyi.live",
    location: "彭城广场", image: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=1080",
    tag: "音乐", likes: 315, time: "3天前",
    content: "户部山周末弹唱现场，氛围太好了。久违的烟火气又回来了，路过的人会不自觉地停下来听一会儿。",
    comments: [],
  },
  6: {
    ...DEFAULT_POST, id: 6, user: "Mika", handle: "@mika.city",
    location: "汉文化景区", image: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=1080",
    tag: "人文", likes: 142, time: "4天前",
    content: "徐州的历史比很多人想象的还要厚重。沿着景区慢慢走，每一块砖和每一段城墙都有自己的故事。",
    comments: [],
  },
};

export function PostDetail() {
  const { postId } = useParams<{ postId: string }>();
  const navigate = useNavigate();
  const { profilePosts, likedIds, savedIds, commentsByPost, toggleLike, toggleSave, addComment } = useSocial();
  const numericPostId = Number(postId);
  const publishedPost = profilePosts.find((item) => item.id === numericPostId);
  const post = publishedPost
    ? {
        ...DEFAULT_POST,
        id: publishedPost.id,
        user: publishedPost.user,
        handle: "@elara.v",
        avatar: publishedPost.avatar,
        location: publishedPost.location,
        image: publishedPost.image ?? "https://images.unsplash.com/photo-1762113396314-8e9e5417ed17?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
        content: publishedPost.content,
        time: publishedPost.time,
        likes: publishedPost.likes,
        tag: publishedPost.tag,
        comments: [],
      }
    : DISCOVER_DETAIL_POSTS[numericPostId] ?? POSTS[numericPostId] ?? DEFAULT_POST;

  const [input, setInput] = useState("");
  const [moreOpen, setMoreOpen] = useState(false);
  const [shareStatus, setShareStatus] = useState<"idle" | "copied" | "failed">("idle");
  const moreMenuRef = useRef<HTMLDivElement>(null);
  const persistedComments = commentsByPost[numericPostId] ?? [];
  const comments = [...post.comments, ...persistedComments];
  const commentCount = (FEED_COMMENT_COUNTS[numericPostId] ?? post.comments.length) + persistedComments.length;
  const liked = likedIds.has(numericPostId);
  const saved = savedIds.has(numericPostId);

  const handleComment = () => {
    if (!input.trim()) return;
    addComment(numericPostId, input);
    setInput("");
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
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      {/* Header */}
      <div className="flex-shrink-0 flex items-center justify-between px-4 py-3"
        style={{ borderBottom: "1px solid var(--neu-border)", backgroundColor: "var(--neu-bg)" }}>
        <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} />
        <h1 style={{ fontSize: 16, color: "var(--neu-text)" }}>动态详情</h1>
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
                className="flex w-full items-center gap-2 rounded-[10px] px-2.5 py-2 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
                style={{ fontSize: 12, color: "var(--neu-text)" }}
              >
                <Link2 style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                复制链接
              </button>
              <button
                type="button"
                role="menuitem"
                onClick={() => setMoreOpen(false)}
                className="flex w-full items-center gap-2 rounded-[10px] px-2.5 py-2 text-left outline-none transition-colors hover:bg-[var(--neu-accent-soft)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
                style={{ fontSize: 12, color: "var(--neu-text-muted)" }}
              >
                取消
              </button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>

      <p aria-live="polite" className="sr-only">
        {shareStatus === "copied" ? "动态链接已复制" : shareStatus === "failed" ? "无法复制动态链接" : ""}
      </p>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto no-scrollbar">
        {/* Post image */}
        <div className="relative overflow-hidden" style={{ height: "clamp(220px, 38vw, 380px)" }}>
          <img src={post.image} alt="post" className="w-full h-full object-cover" />
          <div className="absolute top-3 left-3 px-2.5 py-0.5 rounded-full"
            style={{ background: "var(--neu-accent-primary)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
            <span style={{ fontSize: 10, color: "white" }}>#{post.tag}</span>
          </div>
        </div>

        <div className="px-5 pt-4 pb-3"
          style={{ maxWidth: "var(--content-reading-max)", marginLeft: "auto", marginRight: "auto", width: "100%" }}>
          {/* Author */}
          <div className="flex items-center gap-3 mb-3">
            <motion.button type="button" whileTap={{ scale: 0.95 }} onClick={() => navigate(ROUTES.user(post.id))}
              aria-label={`查看${post.user}的个人主页`}
              className="rounded-full overflow-hidden flex-shrink-0 cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)]"
              style={{ width: 44, height: 44, padding: 0, background: "transparent", border: "2px solid var(--neu-accent-primary)" }}>
              <img src={post.avatar} alt="" className="w-full h-full object-cover" />
            </motion.button>
            <div className="flex-1">
              <p style={{ fontSize: 15, color: "var(--neu-text)" }}>{post.user}</p>
              <div className="flex items-center gap-1.5">
                <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{post.handle}</span>
                <span style={{ fontSize: 10, color: "var(--neu-text-muted)", opacity: 0.5 }}>·</span>
                <MapPin style={{ width: 10, height: 10, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{post.location}</span>
              </div>
            </div>
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{post.time}</span>
          </div>

          {/* Content */}
          <p style={{ fontSize: 14, color: "var(--neu-text)", lineHeight: 1.75 }}>{post.content}</p>

          {/* Actions */}
          <div className="flex items-center justify-between mt-4 py-3"
            style={{ borderTop: "1px solid var(--neu-border)", borderBottom: "1px solid var(--neu-border)" }}>
            <div className="flex gap-4">
              <motion.button whileTap={{ scale: 0.85 }}
                onClick={() => toggleLike(numericPostId)}
                className="flex items-center gap-1.5 outline-none"
                style={{ color: liked ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}>
                <Heart style={{ width: 19, height: 19 }} strokeWidth={1.5} fill={liked ? "currentColor" : "none"} />
                <span style={{ fontSize: 13 }}>{post.likes + (liked ? 1 : 0)}</span>
              </motion.button>
              <div className="flex items-center gap-1.5" style={{ color: "var(--neu-text-muted)" }}>
                <MessageCircle style={{ width: 19, height: 19 }} strokeWidth={1.5} />
                <span style={{ fontSize: 13 }}>{commentCount}</span>
              </div>
            </div>
            <div className="flex gap-3">
              <motion.button whileTap={{ scale: 0.85 }} onClick={() => toggleSave(numericPostId)} className="outline-none"
                style={{ color: saved ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}>
                <Bookmark style={{ width: 19, height: 19 }} strokeWidth={1.5} fill={saved ? "currentColor" : "none"} />
              </motion.button>
              <motion.button whileTap={{ scale: 0.85 }} onClick={copyPostLink} aria-label={shareStatus === "failed" ? "复制动态链接失败" : "复制动态链接"} className="rounded-full outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]" style={{ color: shareStatus === "copied" ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}>
                {shareStatus === "copied" ? <Check style={{ width: 19, height: 19 }} strokeWidth={1.8} /> : <Share2 style={{ width: 19, height: 19 }} strokeWidth={1.5} />}
              </motion.button>
            </div>
          </div>

          {/* Comments */}
          <p className="uppercase mt-4 mb-3" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>
            评论 · {commentCount}
          </p>
          <div className="flex flex-col gap-4">
            {comments.map(c => (
              <div key={c.id} className="flex gap-3">
                <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 34, height: 34 }}>
                  <img src={c.avatar} alt={c.user} className="w-full h-full object-cover" />
                </div>
                <div className="flex-1">
                  <div className="flex items-baseline gap-2">
                    <span style={{ fontSize: 13, color: "var(--neu-text)" }}>{c.user}</span>
                    <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{c.time}</span>
                  </div>
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)", lineHeight: 1.6, marginTop: 2 }}>{c.text}</p>
                  <div className="flex items-center gap-1 mt-1.5">
                    <motion.button whileTap={{ scale: 0.85 }} className="flex items-center gap-1 outline-none" style={{ color: "var(--neu-text-muted)" }}>
                      <Heart style={{ width: 12, height: 12 }} strokeWidth={1.5} />
                      <span style={{ fontSize: 11 }}>{c.likes}</span>
                    </motion.button>
                    <span style={{ fontSize: 11, color: "var(--neu-text-muted)", marginLeft: 8, cursor: "pointer" }}>回复</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div style={{ height: 80 }} />
        </div>
      </div>

      {/* Input bar */}
      <div className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderTop: "1px solid var(--neu-border)" }}>
        <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 30, height: 30 }}>
          <img src="https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" alt="me" className="w-full h-full object-cover" />
        </div>
        <div className="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}>
          <input value={input} onChange={e => setInput(e.target.value)} onKeyDown={e => e.key === "Enter" && handleComment()}
            placeholder="写下你的评论…" className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }} />
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
          onClick={() => navigate(ROUTES.user(post.id))}
          trailing={
            <motion.button
              whileTap={{ scale: 0.94 }}
              onClick={(e) => { e.stopPropagation(); navigate(ROUTES.message(post.id)); }}
              className="px-2.5 py-1 rounded-full outline-none flex-shrink-0"
              style={{ fontSize: 11, background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)", border: "1px solid var(--neu-accent-soft-border)" }}
            >
              私信
            </motion.button>
          }
        />
      </RailSection>

      <RailSection title="相关位置">
        <RailLocationMini name={post.location} desc="共 12 条动态 · 6 人在此" />
      </RailSection>

      <RailSection title="此刻附近" action="查看全部" onAction={() => navigate(ROUTES.nearbyPeople)}>
        <div className="flex flex-col gap-3">
          {NEARBY_NOW.map(u => (
            <RailUserRow
              key={u.id}
              avatar={u.avatar}
              name={u.name}
              sub={u.sub}
              onClick={() => navigate(ROUTES.user(u.id))}
            />
          ))}
        </div>
      </RailSection>

      <RailSection title="更多动态" action="发现" onAction={() => navigate(ROUTES.discover)}>
        <div className="flex flex-col gap-3">
          {RELATED_POSTS.map(p => (
            <RailPostThumb
              key={p.id}
              image={p.image}
              title={p.title}
              meta={p.meta}
              onClick={() => navigate(ROUTES.moment(p.id))}
            />
          ))}
        </div>
      </RailSection>
    </>
  );

  return <DesktopDetailSplit main={main} rail={rail} />;
}