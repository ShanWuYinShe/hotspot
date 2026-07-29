import React, { useEffect, useState } from "react";
import { ArrowLeft, Search, MapPin, Heart, MessageCircle, Share2, Bookmark, X, Sparkles } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { usePlatform } from "../../contexts/PlatformContext";
import { useNavigate } from "react-router";
import { ROUTES } from "../../lib/routes";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDefaultLocation } from "../../hooks/useDefaultLocation";
import { BspCardCanvas } from "../../components/shared/BspCardCanvas";
import { ActionIconButton } from "../../components/shared/ActionIconButton";
import { Avatar } from "../../components/shared/Avatar";
import { getFeedMoments, setReaction, favoriteMoment } from "../../../api";

/* ── Feed data ──────────────────────────────────────────────────── */
// FeedItem is the card shape; liked/saved are per-item interaction state held
// locally (derived from reactions/my_state on fetch, toggled via the API).
interface FeedItem {
  id: number | string;
  user: string;
  avatar: string;
  location: string;
  tag: string;
  content: string;
  image: string;
  likes: number;
  comments: number;
  time: string;
  liked: boolean;
  saved: boolean;
  likeEmoji: string;
}

const TABS  = ["热门", "附近", "活动", "关注"];

/**
 * Holds per-item liked/saved interaction state and toggles it via the API
 * (setReaction / moments/favorite). Initial state is derived from the feed
 * item's reactions/my_state at fetch time.
 */
function useFeedInteractions(initial: FeedItem[]) {
  const [items, setItems] = useState<FeedItem[]>(initial);
  useEffect(() => { setItems(initial); }, [initial]);

  const toggleLike = async (id: FeedItem["id"]) => {
    setItems((prev) => prev.map((it) => it.id === id ? { ...it, liked: !it.liked } : it));
    const target = items.find((it) => it.id === id);
    if (target) {
      try {
        await setReaction({ target_type: "moment", target_id: String(id), emoji: target.likeEmoji, active: !target.liked });
      } catch {
        setItems((prev) => prev.map((it) => it.id === id ? { ...it, liked: target.liked } : it));
      }
    }
  };

  const toggleSave = async (id: FeedItem["id"]) => {
    setItems((prev) => prev.map((it) => it.id === id ? { ...it, saved: !it.saved } : it));
    const target = items.find((it) => it.id === id);
    if (target) {
      try {
        await favoriteMoment({ moment_id: String(id), active: !target.saved });
      } catch {
        setItems((prev) => prev.map((it) => it.id === id ? { ...it, saved: target.saved } : it));
      }
    }
  };

  return { items, toggleLike, toggleSave };
}


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
          <Avatar src={item.avatar} name={item.user} size={22}
            style={{ border: item.image ? "1.5px solid rgba(255,255,255,0.65)" : "1.5px solid var(--neu-border)" }} />
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
  activeTag, setActiveTag, tags, scrollable = false,
}: {
  activeTag: string | null; setActiveTag: (t: string | null) => void; tags: string[]; scrollable?: boolean;
}) {
  return (
    <div className={scrollable ? "flex gap-2 overflow-x-auto no-scrollbar pb-1" : "flex gap-2 flex-wrap"}>
      {tags.map(tag => {
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
function DiscoverMobile({ apiFeedItems, tags }: { apiFeedItems: FeedItem[]; tags: string[] }) {
  const navigate = useNavigate();
  const { items: feedItems, toggleLike, toggleSave } = useFeedInteractions(apiFeedItems);
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
            <TagChips activeTag={activeTag} setActiveTag={setActiveTag} tags={tags} scrollable />
          </div>

        {/* Feed — first is hero with ambient, rest are flat */}
        <div className="flex flex-col gap-4">
          {filtered.map((item, i) => i === 0 ? (
            <div key={item.id} className="neu-ambient-glow">
              <PostCard
                item={item} hero
                liked={item.liked} saved={item.saved}
                onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                onClick={() => navigate(ROUTES.moment(item.id))}
              />
            </div>
          ) : (
            <PostCard
              key={item.id} item={item}
              liked={item.liked} saved={item.saved}
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
function DiscoverTablet({ apiFeedItems, tags }: { apiFeedItems: FeedItem[]; tags: string[] }) {
  const navigate = useNavigate();
  const { items: feedItems, toggleLike, toggleSave } = useFeedInteractions(apiFeedItems);
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
          <TagChips activeTag={activeTag} setActiveTag={setActiveTag} tags={tags} scrollable />
          </div>

          {filtered[0] && (
            <div className="neu-ambient-glow mb-4">
              <PostCard
                item={filtered[0]} hero
                liked={filtered[0].liked} saved={filtered[0].saved}
                onLike={() => toggleLike(filtered[0].id)} onSave={() => toggleSave(filtered[0].id)}
                onClick={() => navigate(ROUTES.moment(filtered[0].id))}
              />
            </div>
          )}

          <div style={{ display: "grid", gridTemplateColumns: "repeat(2, 1fr)", gap: 14 }}>
            {filtered.slice(1).map(item => (
              <PostCard
                key={item.id} item={item} dense
                liked={item.liked} saved={item.saved}
                onLike={() => toggleLike(item.id)} onSave={() => toggleSave(item.id)}
                onClick={() => navigate(ROUTES.moment(item.id))}
              />
            ))}
          </div>
        </div>
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Discover — 2-column feed + 220px sidebar
   2 cols (not 3) because content area is ~770px; 3 cols would make
   each card only ~245px wide — image becomes unreadably shallow.
══════════════════════════════════════════════════════════════════ */
function DiscoverDesktop({ apiFeedItems, tags }: { apiFeedItems: FeedItem[]; tags: string[] }) {
  const navigate = useNavigate();
  const { items: feedItems, toggleLike, toggleSave } = useFeedInteractions(apiFeedItems);
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
              <TagChips activeTag={activeTag} setActiveTag={setActiveTag} tags={tags} />
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
                  liked={item.liked} saved={item.saved}
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
export function MomentsFeedPage() {
  const { platform } = usePlatform();
  const { reference_point } = useDefaultLocation();
  const [apiFeedItems, setApiFeedItems] = useState<FeedItem[]>([]);
  const [tags, setTags] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // spec: feed/list_moments requires source + reference_point.
    // reference_point comes from ClientConfig.default_location (location denied fallback).
    if (!reference_point) return;
    let cancelled = false;
    async function fetchData() {
      try {
        setLoading(true);
        const feedRes = await getFeedMoments({ source: "nearby", reference_point }).catch(() => ({ code: -1, data: { items: [] } }));
        if (cancelled) return;

        const feedData = (feedRes.code === 0 ? feedRes.data : null) as any;

        if (feedData) {
          const items: FeedItem[] = (feedData.items || []).map((m: any, i: number) => {
            const mine = (m.reactions || []).find((r: any) => r.reacted_by_me);
            return {
              id: m.moment_id ?? m.id ?? i + 1,
              user: m.user?.nickname ?? "",
              avatar: m.user?.avatar_url ?? "",
              location: m.location?.location_name ?? "",
              tag: m.tags?.[0] ?? "",
              content: m.description ?? m.content ?? "",
              image: m.images?.[0]?.url ?? "",
              likes: m.likes_count ?? 0,
              comments: m.comments_count ?? 0,
              time: m.created_at ?? "",
              liked: Boolean(mine),
              saved: Boolean(m.my_state?.is_favorited),
              likeEmoji: mine?.emoji ?? "👍",
            };
          });
          setApiFeedItems(items);

          // Hot tags are aggregated from the feed itself — search/suggest is a
          // search-box autocomplete (needs a prefix) and must not be used to
          // browse tags. Dedupe tags across the returned moments, keep order.
          const seen = new Set<string>();
          const tagsList: string[] = [];
          for (const m of (feedData.items || [])) {
            for (const t of (m.tags || [])) {
              if (typeof t === "string" && t && !seen.has(t)) { seen.add(t); tagsList.push(t); }
            }
          }
          setTags(tagsList);
        }
      } catch {
        // empty data is normal
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchData();
    return () => { cancelled = true; };
  }, [reference_point]);

  if (platform === "desktop") return <DiscoverDesktop apiFeedItems={apiFeedItems} tags={tags} />;
  if (platform === "tablet")  return <DiscoverTablet apiFeedItems={apiFeedItems} tags={tags} />;
  return <DiscoverMobile apiFeedItems={apiFeedItems} tags={tags} />;
}