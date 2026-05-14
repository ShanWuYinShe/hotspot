import React, { useState } from "react";
import { Search, MapPin, Heart, MessageCircle, Share2, Bookmark, X, TrendingUp, Sparkles } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { useNavigate } from "react-router";
import { useHoverConfig } from "../hooks/useHoverConfig";
import {
  TabletRail, DailyPickCard, StreakCard, VibeCard, SuggestedFollowsCard,
} from "../components/TabletRail";

/* ── Feed data ──────────────────────────────────────────────────── */
const FEED_ITEMS = [
  {
    id: 1, user: "Aria Sol", location: "植物园", tag: "自然",
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
    id: 3, user: "Marco", location: "老街咖啡馆", tag: "生活",
    avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "周末午后最适合窝在这里看书，点杯手冲，看窗外的行人。",
    image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 56, comments: 8, time: "昨天",
  },
  {
    id: 4, user: "Isabella", location: "云龙湖畔", tag: "活动",
    avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "湖畔的周末市集开始了！有好多有意思的手工艺品和本地小吃。",
    image: "https://images.unsplash.com/photo-1758613172074-7275c3455034?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 201, comments: 34, time: "周一",
  },
  {
    id: 5, user: "Noah", location: "彭城广场", tag: "音乐",
    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "街头音乐节现场，氛围太好了，久违的烟火气又回来了。",
    image: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 315, comments: 27, time: "3天前",
  },
  {
    id: 6, user: "Aiko", location: "汉文化景区", tag: "人文",
    avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400",
    content: "徐州的历史比很多人想象的还要厚重，这里每一块砖都有故事。",
    image: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1080",
    likes: 142, comments: 19, time: "4天前",
  },
];

const TABS  = ["热门", "附近", "活动", "关注"];
const TAGS  = ["自然", "城市", "美食", "音乐", "活动", "摄影", "生活", "旅行"];

/* ── Feed state hook ────────────────────────────────────────────── */
function useFeedState() {
  const [likedIds, setLikedIds] = useState<Set<number>>(new Set());
  const [savedIds, setSavedIds] = useState<Set<number>>(new Set());
  const toggleLike = (id: number) =>
    setLikedIds(p => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const toggleSave = (id: number) =>
    setSavedIds(p => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });
  return { likedIds, savedIds, toggleLike, toggleSave };
}

/* ══════════════════════════════════════════════════════════════════
   PostCard
   — always fills its container width (no internal maxWidth)
   — `dense` prop reduces padding/text size for 2-col grids
══════════════════════════════════════════════════════════════════ */
function PostCard({
  item, liked, saved, onLike, onSave, dense = false, onClick, hero = false,
}: {
  item: typeof FEED_ITEMS[0];
  liked: boolean; saved: boolean;
  onLike: () => void; onSave: () => void;
  dense?: boolean; onClick?: () => void; hero?: boolean;
}) {
  const hover = useHoverConfig();
  return (
    <motion.div
      whileTap={hover.tap as any}
      whileHover={hero ? (hover.card as any) : { y: -1 }}
      className={hero
        ? "rounded-[22px] overflow-hidden cursor-pointer w-full neu-surface-hero neu-card-hover"
        : "rounded-[18px] overflow-hidden cursor-pointer w-full neu-surface-flat"
      }
      onClick={onClick}
    >
      {/* Image — always 16/9 regardless of density */}
      <div className="relative overflow-hidden" style={{ aspectRatio: "16/9" }}>
        <img src={item.image} alt={item.tag} className="w-full h-full object-cover" />
        {/* Gradient overlay */}
        <div
          className="absolute inset-0"
          style={{ background: "linear-gradient(to bottom, rgba(0,0,0,0.04) 0%, rgba(0,0,0,0.44) 100%)" }}
        />
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
            style={{ width: 22, height: 22, border: "1.5px solid rgba(255,255,255,0.65)" }}
          >
            <img src={item.avatar} alt={item.user} className="w-full h-full object-cover" />
          </div>
          <span style={{ fontSize: 11, color: "white", fontWeight: 500 }}>{item.user}</span>
          <div className="flex items-center gap-0.5 ml-auto">
            <MapPin style={{ width: 9, height: 9, color: "rgba(255,255,255,0.8)" }} strokeWidth={1.5} />
            <span style={{ fontSize: 10, color: "rgba(255,255,255,0.8)" }}>{item.location}</span>
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
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const { likedIds, savedIds, toggleLike, toggleSave } = useFeedState();
  const q = searchVal.trim().toLowerCase();
  const filtered = FEED_ITEMS.filter(i =>
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
                onClick={() => navigate(`/post/${item.id}`)}
              />
            </div>
          ) : (
            <PostCard
              key={item.id} item={item}
              liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
              onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
              onClick={() => navigate(`/post/${item.id}`)}
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
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const { likedIds, savedIds, toggleLike, toggleSave } = useFeedState();
  const q = searchVal.trim().toLowerCase();
  const filtered = FEED_ITEMS.filter(i =>
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
        <div className="flex gap-6 mx-auto" style={{ maxWidth: 1124, width: "100%" }}>
          <div className="mx-auto w-full lg:mx-0 px-6 min-w-0" style={{ maxWidth: 800 }}>
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
          <div className="hidden lg:block flex-shrink-0" style={{ width: 300 + 20 }} />
        </div>
      </div>

      {/* Body — 2-column shell: main 800 + gap 24 + rail 300 = 1124 */}
      <motion.div
        initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex gap-6 mx-auto"
        style={{ maxWidth: 1124, width: "100%", padding: "16px 0 28px" }}
      >
        <div className="mx-auto w-full lg:mx-0 px-5 min-w-0" style={{ maxWidth: 800 }}>
          <div className="mb-5">
            <TagChips activeTag={activeTag} setActiveTag={setActiveTag} scrollable />
          </div>

          {filtered[0] && (
            <div className="neu-ambient-glow mb-4">
              <PostCard
                item={filtered[0]} hero
                liked={likedIds.has(filtered[0].id)} saved={savedIds.has(filtered[0].id)}
                onLike={() => toggleLike(filtered[0].id)} onSave={() => toggleSave(filtered[0].id)}
                onClick={() => navigate(`/post/${filtered[0].id}`)}
              />
            </div>
          )}

          <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 14 }}>
            {filtered.slice(1).map(item => (
              <PostCard
                key={item.id} item={item} dense
                liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
                onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                onClick={() => navigate(`/post/${item.id}`)}
              />
            ))}
          </div>
        </div>

        <aside className="hidden lg:block flex-shrink-0 pr-5" style={{ width: 300 + 20 }}>
          <div style={{ width: 300 }}>
            <TabletRail>
              <DailyPickCard />
              <StreakCard />
              <VibeCard />
              <SuggestedFollowsCard />
            </TabletRail>
          </div>
        </aside>
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
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const { likedIds, savedIds, toggleLike, toggleSave } = useFeedState();
  const q = searchVal.trim().toLowerCase();
  const filtered = FEED_ITEMS.filter(i =>
    (!activeTag || i.tag === activeTag) &&
    (!q || i.user.toLowerCase().includes(q) || i.content.toLowerCase().includes(q) || i.location.toLowerCase().includes(q))
  );

  return (
    <div className="flex flex-col h-full">
      {/* Sticky header */}
      <div
        className="flex-shrink-0 px-6 pt-5 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center justify-between mb-2">
          <h1 style={{ fontSize: 20, color: "var(--neu-text)" }}>发现</h1>
          <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
            {filtered.length} 条内容
          </span>
        </div>
        <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      {/* Body: main feed + right sidebar */}
      <div className="flex flex-1 min-h-0 overflow-hidden">

        {/* ── Main feed ─────────────────────────────────────────── */}
        <div className="flex-1 overflow-y-auto no-scrollbar">
          <div style={{ padding: "20px 24px 28px" }}>
            {/* Tag chips */}
            <div className="mb-5">
              <TagChips activeTag={activeTag} setActiveTag={setActiveTag} />
            </div>

            {/*
              2-column grid.
              Content area ≈ 770px (desktop main card − sidebar − padding).
              2 cols → each ≈ 377px → 16/9 image → 212px tall ✓
              3 cols → each ≈ 247px → 16/9 image → 139px tall ✗ (too short)
            */}
            {/* Hero first card */}
            {filtered[0] && (
              <div className="neu-ambient-glow mb-4" style={{ maxWidth: 800 }}>
                <PostCard
                  item={filtered[0]} hero
                  liked={likedIds.has(filtered[0].id)} saved={savedIds.has(filtered[0].id)}
                  onLike={() => toggleLike(filtered[0].id)} onSave={() => toggleSave(filtered[0].id)}
                  onClick={() => navigate(`/post/${filtered[0].id}`)}
                />
              </div>
            )}

            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(2, 1fr)",
                gap: 14,
                maxWidth: 800,
              }}
            >
              {filtered.slice(1).map(item => (
                <PostCard
                  key={item.id} item={item} dense
                  liked={likedIds.has(item.id)} saved={savedIds.has(item.id)}
                  onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                  onClick={() => navigate(`/post/${item.id}`)}
                />
              ))}
            </div>
          </div>
        </div>

        {/* ── Right sidebar (220px) ──────────────────────────────── */}
        <div
          className="flex-shrink-0 overflow-y-auto no-scrollbar"
          style={{ width: 220, borderLeft: "1px solid var(--neu-border)", padding: "20px 16px" }}
        >
          {/* Trending topics */}
          <div className="mb-6">
            <div className="flex items-center gap-1.5 mb-3">
              <TrendingUp style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={2} />
              <p className="uppercase" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>热门话题</p>
            </div>
            {["#城市探索", "#周末去哪", "#隐藏咖啡馆", "#街头摄影", "#本地美食"].map((tag, i) => (
              <motion.button
                key={tag} whileTap={{ scale: 0.98 }}
                className="flex items-center justify-between w-full rounded-[10px] px-3 py-2 outline-none neu-surface-ghost"
              >
                <span style={{ fontSize: 13, color: "var(--neu-text)" }}>{tag}</span>
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                  {[89, 64, 52, 48, 41][i]}
                </span>
              </motion.button>
            ))}
          </div>

          {/* Divider */}
          <div style={{ height: 1, background: "var(--neu-border)", marginBottom: 20 }} />

          {/* Active users */}
          <div>
            <p className="uppercase mb-3" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>活跃用户</p>
            {FEED_ITEMS.slice(0, 4).map(item => (
              <div key={item.id} className="flex items-center gap-2.5 mb-3.5">
                <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 32, height: 32 }}>
                  <img src={item.avatar} alt={item.user} className="w-full h-full object-cover" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="truncate" style={{ fontSize: 12, color: "var(--neu-text)" }}>{item.user}</p>
                  <p className="truncate" style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{item.location}</p>
                </div>
                <motion.button
                  whileTap={{ scale: 0.93 }}
                  className="px-2.5 py-1 rounded-full outline-none flex-shrink-0"
                  style={{ background: "var(--neu-accent-primary)", fontSize: 10, color: "white", boxShadow: "0 2px 6px var(--neu-glow)" }}
                >
                  关注
                </motion.button>
              </div>
            ))}
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