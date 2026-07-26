import React, { useState } from "react";
import { ArrowLeft, Search, MapPin, Heart, MessageCircle, Share2, Bookmark, X, Sparkles } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { useNavigate } from "react-router";
import { ROUTES } from "../lib/routes";
import { useHoverConfig } from "../hooks/useHoverConfig";
import {
  TabletRail, DailyPickCard, StreakCard, VibeCard, SuggestedFollowsCard,
} from "../components/TabletRail";
import { ContextRail } from "../components/layout/ContextRail";
import { useSocial, type SocialPost } from "../contexts/SocialContext";
import { BspCardCanvas } from "../components/shared/BspCardCanvas";
import { ActionIconButton } from "../components/shared/ActionIconButton";

/* ── Feed data ──────────────────────────────────────────────────── */
type FeedItem = Omit<SocialPost, "moods" | "audience">;

const FEED_ITEM_SEED: FeedItem[] = [
  {
    id: 1, user: "阿禾", location: "植物园", tag: "自然",
    avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "清晨在绿意中找到了一片宁静。黎明时分阳光穿过树叶，滤出完美的光影。",
    image: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 124, comments: 12, time: "2小时前",
  },
  {
    id: 2, user: "本地向导", location: "市中心核心区", tag: "城市",
    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "城市里那些安静的角落已经被我全部地图标注出来了。每一条小巷都有属于己的故事。",
    image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 89, comments: 5, time: "5小时前",
  },
  {
    id: 3, user: "野原", location: "老街手冲馆", tag: "生活",
    avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "周末午后最适合窝在这里看书，点杯手冲，看窗外的行人。",
    image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 56, comments: 8, time: "昨天",
  },
  {
    id: 4, user: "小满", location: "云龙湖畔", tag: "活动",
    avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "湖畔的周末市集开始了！有好多有意思的手工艺品和本地小吃。",
    image: "https://images.unsplash.com/photo-1758613172074-7275c3455034?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 201, comments: 34, time: "周一",
  },
  {
    id: 5, user: "林一", location: "彭城广场", tag: "音乐",
    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "户部山周末弹唱现场，氛围太好了，久违的烟火气又回来了。",
    image: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 315, comments: 27, time: "3天前",
  },
  {
    id: 6, user: "Mika", location: "汉文化景区", tag: "人文",
    avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "徐州的历史比很多人想象的还要厚重，这里每一块砖都有故事。",
    image: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 142, comments: 19, time: "4天前",
  },
];

// Mock pagination: real discovery feeds stream many more cards than fit on a
// screen, so the seed items are cycled with id/likes offsets to give the BSP
// canvas enough cards to exercise incremental loading + virtualization.
// Replaced by the API once pagination is wired through.
const FEED_ITEMS: FeedItem[] = Array.from({ length: 72 }, (_, i) => {
  const seed = FEED_ITEM_SEED[i % FEED_ITEM_SEED.length];
  const cycle = Math.floor(i / FEED_ITEM_SEED.length);
  return {
    ...seed,
    id: i + 1,
    user: cycle === 0 ? seed.user : `${seed.user}${cycle + 1}`,
    likes: seed.likes + cycle * 17,
    comments: seed.comments + cycle * 3,
  };
});

const TABS  = ["热门", "附近", "活动", "关注"];
const TAGS  = ["自然", "城市", "美食", "音乐", "活动", "摄影", "生活", "旅行"];


/* ══════════════════════════════════════════════════════════════════
   PostCard
   — always fills its container width (no internal maxWidth)
   — `dense` prop reduces padding/text size for 2-col grids
══════════════════════════════════════════════════════════════════ */
function PostCard({
  item, liked, saved, onLike, onSave, dense = false, onClick, hero = false, mediaRatio, edition,
}: {
  item: FeedItem;
  liked: boolean; saved: boolean;
  onLike: () => void; onSave: () => void;
  dense?: boolean; onClick?: () => void; hero?: boolean; mediaRatio?: string; edition?: string;
}) {
  const hover = useHoverConfig();
  return (
    <motion.div
      whileTap={hover.tap as any}
      whileHover={hover.card as any}
      className={hero
        ? "flex h-full flex-col rounded-[22px] overflow-hidden cursor-pointer w-full neu-surface-hero neu-card-hover"
        : "flex h-full flex-col rounded-[18px] overflow-hidden cursor-pointer w-full neu-surface-flat neu-card-hover"
      }
      onClick={onClick}
    >
      {/* Image — always 16/9 regardless of density */}
      <div className="relative min-h-[170px] flex-1 overflow-hidden" style={{ aspectRatio: mediaRatio ?? "16/9" }}>
        {item.image ? (
          <img src={item.image} alt={item.tag} className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full w-full flex-col justify-between bg-[color-mix(in_srgb,var(--neu-accent-primary)_12%,var(--neu-bg))] p-4">
            <MapPin size={20} className="text-[var(--neu-accent-primary)]" strokeWidth={1.5} />
            <p className="max-w-[78%] text-[15px] leading-6 text-[var(--neu-text)]">来自 {item.location} 的附近动态</p>
          </div>
        )}
        {/* Gradient overlay on photographs only */}
        {item.image && <div className="absolute inset-0" style={{ background: "var(--surface-image-scrim)" }} />}
        {/* Editorial position marker: a small constant makes different card sizes read as one collection. */}
        {edition && (
          <div className="absolute right-3 top-3 z-[1] flex h-7 min-w-7 items-center justify-center rounded-full border border-white/35 bg-black/20 px-1.5 font-mono text-[9px] tracking-[0.12em] text-white backdrop-blur-sm">
            {edition}
          </div>
        )}
        {/* Tag chip */}
        <div
          className="absolute top-2.5 left-2.5 px-2 py-0.5 rounded-full"
          style={{ background: "var(--neu-accent-primary)", boxShadow: "0 2px 8px var(--neu-glow)" }}
        >
          <span style={{ fontSize: 10, color: "white" }}>{item.tag}</span>
        </div>
        {/* Author + location on image */}
        <div className="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-3 pb-2.5">
          <div
            className="rounded-full overflow-hidden flex-shrink-0"
            style={{ width: 22, height: 22, border: item.image ? "1.5px solid rgba(255,255,255,0.65)" : "1.5px solid var(--neu-border)" }}
          >
            <img src={item.avatar} alt={item.user} className="w-full h-full object-cover" />
          </div>
          <span style={{ fontSize: 11, color: item.image ? "white" : "var(--neu-text)", fontWeight: 500 }}>{item.user}</span>
          <div className="flex items-center gap-0.5 ml-auto">
            <MapPin style={{ width: 9, height: 9, color: item.image ? "rgba(255,255,255,0.8)" : "var(--neu-text-muted)" }} strokeWidth={1.5} />
            <span style={{ fontSize: 10, color: item.image ? "rgba(255,255,255,0.8)" : "var(--neu-text-muted)" }}>{item.location}</span>
          </div>
        </div>
      </div>

      {/* Text content */}
      <div style={{ padding: dense ? "10px 14px 6px" : "12px 16px 8px" }}>
        <p
          className="line-clamp-2"
          style={{
            fontSize: dense ? 12 : 13,
            color: "var(--neu-text-muted)",
            lineHeight: 1.6,
          }}
        >
          {item.content}
        </p>
      </div>

      {/* Actions */}
      <div
        className="flex items-center justify-between"
        style={{ padding: dense ? "4px 14px 12px" : "4px 16px 14px" }}
      >
        <div className="flex gap-3.5">
          <motion.button
            whileTap={hover.tap as any}
            whileHover={hover.action as any}
            className="flex items-center gap-1.5 outline-none"
            style={{ color: liked ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}
            onClick={e => { e.stopPropagation(); onLike(); }}
          >
            <Heart style={{ width: 15, height: 15 }} strokeWidth={1.5} fill={liked ? "currentColor" : "none"} />
            <span style={{ fontSize: 11 }}>{item.likes + (liked ? 1 : 0)}</span>
          </motion.button>
          <motion.button
            whileTap={hover.tap as any}
            whileHover={hover.action as any}
            className="flex items-center gap-1.5 outline-none"
            style={{ color: "var(--neu-text-muted)" }}
            onClick={e => e.stopPropagation()}
          >
            <MessageCircle style={{ width: 15, height: 15 }} strokeWidth={1.5} />
            <span style={{ fontSize: 11 }}>{item.comments}</span>
          </motion.button>
        </div>
        <div className="flex gap-3">
          <motion.button
            whileTap={hover.tap as any}
            whileHover={hover.action as any}
            onClick={e => { e.stopPropagation(); onSave(); }}
            className="outline-none"
            style={{ color: saved ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}
          >
            <Bookmark style={{ width: 15, height: 15 }} strokeWidth={1.5} fill={saved ? "currentColor" : "none"} />
          </motion.button>
          <motion.button
            whileTap={hover.tap as any}
            whileHover={hover.action as any}
            onClick={e => e.stopPropagation()}
            className="outline-none" style={{ color: "var(--neu-text-muted)" }}
          >
            <Share2 style={{ width: 15, height: 15 }} strokeWidth={1.5} />
          </motion.button>
        </div>
      </div>
    </motion.div>
  );
}

/* ── Shared header (tabs + search) ─────────────────────────────── */
function DiscoverHeader({
  searchVal, setSearchVal, activeTab, setActiveTab,
}: {
  searchVal: string; setSearchVal: (v: string) => void;
  activeTab: string; setActiveTab: (t: string) => void;
}) {
  const query = searchVal.trim().toLowerCase();
  return (
    <>
      {/* Search bar */}
      <div
        className="flex items-center gap-3 px-4 py-2.5 rounded-[18px] neu-input-surface"
        style={{ marginBottom: 10 }}
      >
        <Search style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
        <input
          value={searchVal} onChange={e => setSearchVal(e.target.value)}
          placeholder="地点、人物、关键词…"
          className="flex-1 bg-transparent outline-none"
          style={{ fontSize: 13, color: "var(--neu-text)" }}
        />
        {searchVal.length > 0 && (
          <motion.button
            initial={{ opacity: 0, scale: 0.7 }} animate={{ opacity: 1, scale: 1 }}
            whileTap={{ scale: 0.85 }}
            onClick={() => setSearchVal("")}
            className="flex-shrink-0 outline-none"
          >
            <X style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={2} />
          </motion.button>
        )}
      </div>

      {/* Tab bar */}
      <div
        className="flex overflow-x-auto no-scrollbar"
        style={{ borderBottom: "1px solid var(--neu-border)" }}
      >
        {TABS.map(tab => {
          const isActive = activeTab === tab;
          return (
            <motion.button
              key={tab} whileTap={{ scale: 0.94 }} onClick={() => setActiveTab(tab)}
              className="relative flex-shrink-0 pb-3 pt-1 outline-none"
              style={{
                paddingLeft: 18, paddingRight: 18,
                color: isActive ? "var(--neu-accent-primary)" : "var(--neu-text-muted)",
                fontWeight: isActive ? 500 : 400, fontSize: 13,
                transition: "color 0.2s",
              }}
            >
              {tab}
              {isActive && (
                <motion.div
                  layoutId="discover-tab-indicator"
                  style={{
                    position: "absolute", bottom: -1,
                    left: "14%", right: "14%",
                    height: 2, borderRadius: 1,
                    background: "var(--neu-accent-primary)",
                  }}
                  transition={{ type: "spring", stiffness: 420, damping: 34 }}
                />
              )}
            </motion.button>
          );
        })}
      </div>
    </>
  );
}

/* ── Tag chips strip ────────────────────────────────────────────── */
function TagChips({
  activeTag, setActiveTag, scrollable = false,
}: {
  activeTag: string | null; setActiveTag: (t: string | null) => void; scrollable?: boolean;
}) {
  return (
    <div className={scrollable ? "flex gap-2 overflow-x-auto no-scrollbar pb-1" : "flex gap-2 flex-wrap"}>
      {TAGS.map(tag => {
        const active = activeTag === tag;
        return (
          <motion.button
            key={tag} whileTap={{ scale: 0.94 }}
            onClick={() => setActiveTag(active ? null : tag)}
            className="flex-shrink-0 px-3 py-1 rounded-full outline-none transition-all neu-surface-ghost"
            style={active
              ? {
                  background: "var(--neu-accent-soft)",
                  color: "var(--neu-accent-primary)",
                  fontSize: 12,
                  fontWeight: 500,
                  borderColor: "var(--neu-accent-soft-border)",
                }
              : { fontSize: 12, color: "var(--neu-text-muted)" }}>
            {tag}
          </motion.button>
        );
      })}
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile Discover — single-column feed, max-width 560px centred
══════════════════════════════════════════════════════════════════ */
function DiscoverMobile() {
  const navigate = useNavigate();
  const { discoverPosts, likedIds, savedIds, commentsByPost, toggleLike, toggleSave } = useSocial();
  const feedItems: FeedItem[] = [...discoverPosts, ...FEED_ITEMS].map((item) => ({
    ...item,
    comments: item.comments + (commentsByPost[item.id]?.length ?? 0),
  }));
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const q = searchVal.trim().toLowerCase();
  const filtered = feedItems.filter(i =>
    (!activeTag || i.tag === activeTag) &&
    (!q || i.user.toLowerCase().includes(q) || i.content.toLowerCase().includes(q) || i.location.toLowerCase().includes(q))
  );

  return (
    <>
      {/* Sticky header — full width */}
      <div
        className="sticky top-0 z-10 px-5 pt-3 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center justify-between mb-2">
          <h1 style={{ fontSize: 22, color: "var(--neu-text)" }}>发现</h1>
          <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
            {filtered.length} 条内容
          </span>
        </div>
        <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      {/* Content — centred column, max 560px */}
      <motion.div
        initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        style={{ maxWidth: 560, marginLeft: "auto", marginRight: "auto", width: "100%", padding: "12px 16px 24px" }}
      >
        {/* Tag chips */}
        <div className="mb-4">
          <TagChips activeTag={activeTag} setActiveTag={setActiveTag} scrollable />
        </div>

        {/* Feed — first is hero with ambient, rest are flat */}
        <div className="flex flex-col gap-4">
          {filtered.map((item, i) => i === 0 ? (
            <div key={item.id} className="neu-ambient-glow">
              <PostCard
                item={item} hero
                liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
                onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                onClick={() => navigate(ROUTES.moment(item.id))}
              />
            </div>
          ) : (
            <PostCard
              key={item.id} item={item}
              liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
              onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
              onClick={() => navigate(ROUTES.moment(item.id))}
            />
          ))}
        </div>
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Tablet Discover — 2-column grid, max-width 800px centred
   Handles 600-1279px — both portrait and landscape tablets
══════════════════════════════════════════════════════════════════ */
function DiscoverTablet() {
  const navigate = useNavigate();
  const { discoverPosts, likedIds, savedIds, commentsByPost, toggleLike, toggleSave } = useSocial();
  const feedItems: FeedItem[] = [...discoverPosts, ...FEED_ITEMS].map((item) => ({
    ...item,
    comments: item.comments + (commentsByPost[item.id]?.length ?? 0),
  }));
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const q = searchVal.trim().toLowerCase();
  const filtered = feedItems.filter(i =>
    (!activeTag || i.tag === activeTag) &&
    (!q || i.user.toLowerCase().includes(q) || i.content.toLowerCase().includes(q) || i.location.toLowerCase().includes(q))
  );

  return (
    <>
      {/* Sticky header — wide shell so it aligns with main + rail below */}
      <div
        className="sticky top-0 z-10 pt-4 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex gap-6 mx-auto" style={{ width: "100%" }}>
          <div className="mx-auto w-full lg:mx-0 px-6 min-w-0" style={{ width: "100%" }}>
            <div className="flex items-center justify-between mb-2">
              <h1 style={{ fontSize: 24, color: "var(--neu-text)" }}>发现</h1>
              <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full"
                style={{ background: "var(--neu-accent-primary)", boxShadow: "0 3px 10px var(--neu-glow)" }}>
                <Sparkles style={{ width: 11, height: 11, color: "white" }} strokeWidth={2} />
                <span style={{ fontSize: 11, color: "white" }}>精选</span>
              </div>
            </div>
            <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
          </div>
          <div className="hidden lg:block flex-shrink-0" style={{ width: "var(--context-rail-width)" }} />
        </div>
      </div>

      {/* Body — 2-column shell: main 800 + gap 24 + rail 300 = 1124 */}
      <motion.div
        initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex gap-6 mx-auto"
        style={{ maxWidth: 1124, width: "100%", padding: "16px 0 28px" }}
      >
        <div className="mx-auto w-full lg:mx-0 px-5 min-w-0" style={{ width: "100%" }}>
          <div className="mb-5">
            <TagChips activeTag={activeTag} setActiveTag={setActiveTag} scrollable />
          </div>

          {filtered[0] && (
            <div className="neu-ambient-glow mb-4">
              <PostCard
                item={filtered[0]} hero
                liked={likedIds.has(filtered[0].id)} saved={savedIds.has(filtered[0].id)}
                onLike={() => toggleLike(filtered[0].id)} onSave={() => toggleSave(filtered[0].id)}
                onClick={() => navigate(ROUTES.moment(filtered[0].id))}
              />
            </div>
          )}

          <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 14 }}>
            {filtered.slice(1).map(item => (
              <PostCard
                key={item.id} item={item} dense
                liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
                onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                onClick={() => navigate(ROUTES.moment(item.id))}
              />
            ))}
          </div>
        </div>

        <ContextRail className="hidden lg:block">
          <TabletRail>
            <DailyPickCard />
            <StreakCard />
            <VibeCard />
            <SuggestedFollowsCard />
          </TabletRail>
        </ContextRail>
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Discover — 2-column feed + 220px sidebar
   2 cols (not 3) because content area is ~770px; 3 cols would make
   each card only ~245px wide — image becomes unreadably shallow.
══════════════════════════════════════════════════════════════════ */
function DiscoverDesktop() {
  const navigate = useNavigate();
  const { discoverPosts, likedIds, savedIds, commentsByPost, toggleLike, toggleSave } = useSocial();
  const feedItems: FeedItem[] = [...discoverPosts, ...FEED_ITEMS].map((item) => ({
    ...item,
    comments: item.comments + (commentsByPost[item.id]?.length ?? 0),
  }));
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const [visibleCount, setVisibleCount] = useState(12);
  const q = searchVal.trim().toLowerCase();
  const filtered = feedItems.filter(i =>
    (!activeTag || i.tag === activeTag) &&
    (!q || i.user.toLowerCase().includes(q) || i.content.toLowerCase().includes(q) || i.location.toLowerCase().includes(q))
  );

  return (
    <div className="content-mode-wide flex flex-col h-full">
      {/* Directory header: fixed to the local feed viewport and opaque over scrolling content. */}
      <header
        className="sticky top-0 z-20 flex-shrink-0 px-6 pt-3 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center gap-3 mb-2">
          <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} size={34} />
          <div className="min-w-0 flex-1">
            <h1 style={{ fontSize: 18, color: "var(--neu-text)" }}>附近瞬间</h1>
            <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>徐州 · 附近正在发生的事</p>
          </div>
          <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{filtered.length} 条内容</span>
        </div>
        <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
      </header>

      {/* Body: main feed + right sidebar */}
      <div className="flex flex-1 min-h-0 overflow-hidden">

        {/* ── Main feed ─────────────────────────────────────────── */}
        <div className="flex-1 overflow-y-auto no-scrollbar">
          <div style={{ padding: "20px 24px 28px" }}>
            {/* Tag chips */}
            <div className="mb-5">
              <TagChips activeTag={activeTag} setActiveTag={setActiveTag} />
            </div>

            {/* Responsive BSP field — remeasured whenever the desktop workspace changes width. */}
            <BspCardCanvas
              items={filtered.slice(0, visibleCount)}
              gap={24}
              minRatio={0.68}
              maxRatio={1.48}
              minCardWidth={160}
              minCardHeight={260}
              minCanvasHeight={520}
              ariaLabel="发现内容的不规则满铺布局"
              hasMore={visibleCount < filtered.length}
              onLoadMore={() => setVisibleCount((c) => Math.min(c + 8, filtered.length))}
              renderItem={(item, index) => (
                <PostCard
                  item={item} dense edition={String(index + 1).padStart(2, "0")}
                  liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
                  onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                  onClick={() => navigate(ROUTES.moment(item.id))}
                />
              )}
            />
          </div>
        </div>

      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Entry
══════════════════════════════════════════════════════════════════ */
export function Discover() {
  const { platform } = usePlatform();
  if (platform === "desktop") return <DiscoverDesktop />;
  if (platform === "tablet")  return <DiscoverTablet />;
  return <DiscoverMobile />;
}