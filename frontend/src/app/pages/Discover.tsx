import { useState } from "react";
import { Search, MapPin, Heart, MessageCircle, Share2, Bookmark, X, TrendingUp, Sparkles } from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { useNavigate } from "react-router";
import { useHoverConfig } from "../hooks/useHoverConfig";
import {
  TabletRail, DailyPickCard, StreakCard, VibeCard, SuggestedFollowsCard,
} from "../components/TabletRail";
import { useDiscoverData } from "@/lib/hooks/use-discover-data";
import { useHomeData } from "@/lib/hooks/use-home-data";
import { useFeedActions } from "@/lib/hooks/use-feed-actions";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import type { MomentItem, FeedSource } from "@/lib/types/api";

const TABS = [
  { label: "热门", source: "all" as FeedSource },
  { label: "附近", source: "nearby" as FeedSource },
  { label: "关注", source: "following" as FeedSource },
];

import { formatDistance } from "@/lib/utils";

/* ── PostCard ──────────────────────────────────────────────────── */
function PostCard({
  item, liked, saved, onLike, onSave, dense = false, onClick, hero = false,
}: {
  item: MomentItem;
  liked: boolean; saved: boolean;
  onLike: () => void; onSave: () => void;
  dense?: boolean; onClick?: () => void; hero?: boolean;
}) {
  const hover = useHoverConfig();
  const image = item.images?.[0]?.url;
  return (
    <motion.div
      whileTap={hover.tap}
      whileHover={hero ? (hover.card) : { y: -1 }}
      className={hero
        ? "rounded-[22px] overflow-hidden cursor-pointer w-full neu-surface-hero neu-card-hover"
        : "rounded-[18px] overflow-hidden cursor-pointer w-full neu-surface-flat"
      }
      onClick={onClick}
    >
      {image && (
        <div className="relative overflow-hidden" style={{ aspectRatio: "16/9" }}>
          <img src={image} alt={item.tags?.[0] ?? ''} className="w-full h-full object-cover" loading="lazy" />
          <div className="absolute inset-0"
            style={{ background: "linear-gradient(to bottom, rgba(0,0,0,0.04) 0%, rgba(0,0,0,0.44) 100%)" }} />
          {item.tags?.[0] && (
            <div className="absolute top-2.5 left-2.5 px-2 py-0.5 rounded-full"
              style={{ background: "var(--neu-accent-primary)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
              <span style={{ fontSize: 10, color: "var(--text-on-accent, white)" }}>#{item.tags[0]}</span>
            </div>
          )}
          <div className="absolute bottom-0 left-0 right-0 flex items-center gap-2 px-3 pb-2.5">
            <div className="rounded-full overflow-hidden flex-shrink-0"
              style={{ width: 22, height: 22, border: "1.5px solid rgba(255,255,255,0.65)" }}>
              {item.user.avatar_url && <img src={item.user.avatar_url} alt={item.user.nickname} className="w-full h-full object-cover" />}
            </div>
            <span className="truncate max-w-[90px]" style={{ fontSize: 11, color: "white", fontWeight: 500 }}>{item.user.nickname}</span>
            {item.location?.location_name && (
              <div className="flex items-center gap-0.5 ml-auto">
                <MapPin style={{ width: 9, height: 9, color: "rgba(255,255,255,0.8)" }} strokeWidth={1.5} />
                <span className="truncate max-w-[100px]" style={{ fontSize: 10, color: "rgba(255,255,255,0.8)" }}>{item.location.location_name}</span>
              </div>
            )}
          </div>
        </div>
      )}

      <div style={{ padding: dense ? "10px 14px 6px" : "12px 16px 8px" }}>
        <p className="line-clamp-2" style={{ fontSize: dense ? 12 : 13, color: "var(--neu-text-muted)", lineHeight: 1.6 }}>
          {item.description}
        </p>
      </div>

      <div className="flex items-center justify-between" style={{ padding: dense ? "4px 14px 12px" : "4px 16px 14px" }}>
        <div className="flex gap-3.5">
          <motion.button
            whileTap={hover.tap}
            whileHover={hover.action}
            className="flex items-center gap-1.5 outline-none"
            aria-label="点赞"
            style={{ color: liked ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}
            onClick={e => { e.stopPropagation(); onLike(); }}
          >
            <Heart style={{ width: 15, height: 15 }} strokeWidth={1.5} fill={liked ? "currentColor" : "none"} />
            <span style={{ fontSize: 11 }}>{item.likes_count + (liked ? 1 : 0)}</span>
          </motion.button>
          <motion.button
            whileTap={hover.tap}
            whileHover={hover.action}
            className="flex items-center gap-1.5 outline-none"
            aria-label="评论"
            style={{ color: "var(--neu-text-muted)" }}
            onClick={e => e.stopPropagation()}
          >
            <MessageCircle style={{ width: 15, height: 15 }} strokeWidth={1.5} />
            <span style={{ fontSize: 11 }}>{item.comments_count}</span>
          </motion.button>
        </div>
        <div className="flex gap-3">
          <motion.button
            whileTap={hover.tap}
            whileHover={hover.action}
            onClick={e => { e.stopPropagation(); onSave(); }}
            aria-label="收藏"
            className="outline-none"
            style={{ color: saved ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}
          >
            <Bookmark style={{ width: 15, height: 15 }} strokeWidth={1.5} fill={saved ? "currentColor" : "none"} />
          </motion.button>
          <motion.button
            whileTap={hover.tap}
            whileHover={hover.action}
            onClick={e => e.stopPropagation()}
            aria-label="分享"
            className="outline-none" style={{ color: "var(--neu-text-muted)" }}
          >
            <Share2 style={{ width: 15, height: 15 }} strokeWidth={1.5} />
          </motion.button>
        </div>
      </div>
    </motion.div>
  );
}

/* ── Feed skeleton ──────────────────────────────────────────────── */
function FeedSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      {[0, 1, 2].map(i => (
        <div key={i} className="rounded-[18px] overflow-hidden neu-surface-flat animate-pulse">
          <div style={{ aspectRatio: "16/9", background: "var(--neu-shadow-light)" }} />
          <div style={{ padding: "12px 16px 14px" }}>
            <div className="rounded" style={{ height: 12, width: "70%", background: "var(--neu-shadow-light)" }} />
            <div className="rounded mt-2" style={{ height: 10, width: "40%", background: "var(--neu-shadow-light)" }} />
          </div>
        </div>
      ))}
    </div>
  );
}

/* ── Shared header (tabs + search) ─────────────────────────────── */
function DiscoverHeader({
  searchVal, setSearchVal, activeTab, setActiveTab,
}: {
  searchVal: string; setSearchVal: (v: string) => void;
  activeTab: string; setActiveTab: (t: string) => void;
}) {
  return (
    <>
      <div className="flex items-center gap-3 px-4 py-2.5 rounded-[18px] neu-input-surface" style={{ marginBottom: 10 }}>
        <Search style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
        <input
          aria-label="搜索"
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
            aria-label="清除搜索"
            className="flex-shrink-0 outline-none"
          >
            <X style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={2} />
          </motion.button>
        )}
      </div>

      <div className="flex overflow-x-auto no-scrollbar" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        {TABS.map(tab => {
          const isActive = activeTab === tab.label;
          return (
            <motion.button
              key={tab.label} whileTap={{ scale: 0.94 }} onClick={() => setActiveTab(tab.label)}
              className="relative flex-shrink-0 pb-3 pt-1 outline-none"
              style={{
                paddingLeft: 18, paddingRight: 18,
                color: isActive ? "var(--neu-accent-primary)" : "var(--neu-text-muted)",
                fontWeight: isActive ? 500 : 400, fontSize: 13,
                transition: "color 0.2s",
              }}
            >
              {tab.label}
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
  tags, activeTag, setActiveTag, scrollable = false,
}: {
  tags: string[]; activeTag: string | null; setActiveTag: (t: string | null) => void; scrollable?: boolean;
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
                  fontSize: 12, fontWeight: 500,
                  borderColor: "var(--neu-accent-soft-border)",
                }
              : { fontSize: 12, color: "var(--neu-text-muted)" }}>
            #{tag}
          </motion.button>
        );
      })}
    </div>
  );
}

/* ── Discover Content (shared between layouts) ─────────────── */
function DiscoverContent({
  moments, likedIds, savedIds, toggleLike, toggleSave, activeTag, tags, setActiveTag, navigate,
  loading, error, onRetry,
}: {
  moments: MomentItem[];
  likedIds: Set<string>; savedIds: Set<string>;
  toggleLike: (id: string) => void; toggleSave: (id: string) => void;
  activeTag: string | null; tags: string[]; setActiveTag: (t: string | null) => void;
  navigate: ReturnType<typeof useNavigate>;
  loading: boolean; error: string | null; onRetry: () => void;
}) {
  if (loading && moments.length === 0) return <FeedSkeleton />;

  if (error && moments.length === 0) {
    return (
      <div className="flex flex-col items-center gap-3 py-16">
        <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>{error}</p>
        <motion.button whileTap={{ scale: 0.94 }} onClick={onRetry}
          className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none"
          style={{ background: "var(--neu-accent-primary)", color: "var(--text-on-accent, white)", fontSize: 13 }}>
          重试
        </motion.button>
      </div>
    );
  }

  const filtered = activeTag
    ? moments.filter(m => m.tags?.includes(activeTag))
    : moments;

  return (
    <>
      <div className="mb-4">
        <TagChips tags={tags} activeTag={activeTag} setActiveTag={setActiveTag} scrollable />
      </div>

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-16">
          <Sparkles style={{ width: 32, height: 32, color: "var(--neu-text-muted)", opacity: 0.3 }} strokeWidth={1.25} />
          <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>暂无内容，下拉刷新试试</p>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {filtered.map((item, i) => i === 0 ? (
            <div key={item.moment_id} className="neu-ambient-glow">
              <PostCard
                item={item} hero
                liked={likedIds.has(item.moment_id)} saved={savedIds.has(item.moment_id)}
                onLike={() => toggleLike(item.moment_id)} onSave={() => toggleSave(item.moment_id)}
                onClick={() => navigate(`/post/${item.moment_id}`)}
              />
            </div>
          ) : (
            <PostCard
              key={item.moment_id} item={item}
              liked={likedIds.has(item.moment_id)} saved={savedIds.has(item.moment_id)}
              onLike={() => toggleLike(item.moment_id)} onSave={() => toggleSave(item.moment_id)}
              onClick={() => navigate(`/post/${item.moment_id}`)}
            />
          ))}
        </div>
      )}
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile Discover
══════════════════════════════════════════════════════════════════ */
function DiscoverMobile() {
  const navigate = useNavigate();
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const { likedIds, savedIds, toggleLike, toggleSave } = useFeedActions();

  const source = TABS.find(t => t.label === activeTab)?.source ?? "all";
  const { moments, loading, error, refresh } = useDiscoverData(source);
  const { trending } = useHomeData();
  const tags: string[] = trending?.trending_tags.map(t => t.name) ?? moments.flatMap(m => m.tags ?? []).filter((v, i, a) => a.indexOf(v) === i).slice(0, 10);

  return (
    <>
      <div className="sticky top-0 z-10 px-5 pt-3 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center justify-between mb-2">
          <h1 style={{ fontSize: 22, color: "var(--neu-text)" }}>发现</h1>
          <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
            {moments.length} 条内容
          </span>
        </div>
        <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        style={{ maxWidth: 560, marginLeft: "auto", marginRight: "auto", width: "100%", padding: "12px 16px 24px" }}>
        <DiscoverContent
          moments={moments} likedIds={likedIds} savedIds={savedIds}
          toggleLike={toggleLike} toggleSave={toggleSave}
          activeTag={activeTag} tags={tags} setActiveTag={setActiveTag} navigate={navigate}
          loading={loading} error={error} onRetry={refresh}
        />
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Tablet Discover
══════════════════════════════════════════════════════════════════ */
function DiscoverTablet() {
  const navigate = useNavigate();
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const { likedIds, savedIds, toggleLike, toggleSave } = useFeedActions();

  const source = TABS.find(t => t.label === activeTab)?.source ?? "all";
  const { moments, loading, error, refresh } = useDiscoverData(source);
  const { trending } = useHomeData();
  const tags: string[] = trending?.trending_tags.map(t => t.name) ?? moments.flatMap(m => m.tags ?? []).filter((v, i, a) => a.indexOf(v) === i).slice(0, 10);

  return (
    <>
      <div className="sticky top-0 z-10 pt-4 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex gap-6 mx-auto" style={{ maxWidth: 1124, width: "100%" }}>
          <div className="mx-auto w-full lg:mx-0 px-6 min-w-0" style={{ maxWidth: 800 }}>
            <div className="flex items-center justify-between mb-2">
              <h1 style={{ fontSize: 24, color: "var(--neu-text)" }}>发现</h1>
              <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full"
                style={{ background: "var(--neu-accent-primary)", boxShadow: "0 3px 10px var(--neu-glow)" }}>
                <Sparkles style={{ width: 11, height: 11, color: "var(--text-on-accent, white)" }} strokeWidth={2} />
                <span style={{ fontSize: 11, color: "var(--text-on-accent, white)" }}>精选</span>
              </div>
            </div>
            <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
          </div>
          <div className="hidden lg:block flex-shrink-0" style={{ width: 300 + 20 }} />
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex gap-6 mx-auto"
        style={{ maxWidth: 1124, width: "100%", padding: "16px 0 28px" }}>
        <div className="mx-auto w-full lg:mx-0 px-5 min-w-0" style={{ maxWidth: 800 }}>
          <DiscoverContent
            moments={moments} likedIds={likedIds} savedIds={savedIds}
            toggleLike={toggleLike} toggleSave={toggleSave}
            activeTag={activeTag} tags={tags} setActiveTag={setActiveTag} navigate={navigate}
            loading={loading} error={error} onRetry={refresh}
          />
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
   Desktop Discover
══════════════════════════════════════════════════════════════════ */
function DiscoverDesktop() {
  const navigate = useNavigate();
  const [searchVal, setSearchVal] = useState("");
  const [activeTab, setActiveTab] = useState("热门");
  const [activeTag, setActiveTag] = useState<string | null>(null);
  const { likedIds, savedIds, toggleLike, toggleSave } = useFeedActions();

  const source = TABS.find(t => t.label === activeTab)?.source ?? "all";
  const { moments, loading, error, refresh } = useDiscoverData(source);
  const { trending } = useHomeData();
  const tags = trending?.trending_tags.map(t => t.name) ?? [];

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0 px-6 pt-5 pb-0"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center justify-between mb-2">
          <h1 style={{ fontSize: 20, color: "var(--neu-text)" }}>发现</h1>
          <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
            {moments.length} 条内容
          </span>
        </div>
        <DiscoverHeader searchVal={searchVal} setSearchVal={setSearchVal} activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      <div className="flex flex-1 min-h-0 overflow-hidden">
        <div className="flex-1 overflow-y-auto no-scrollbar">
          <div style={{ padding: "20px 24px 28px" }}>
            <DiscoverContent
              moments={moments} likedIds={likedIds} savedIds={savedIds}
              toggleLike={toggleLike} toggleSave={toggleSave}
              activeTag={activeTag} tags={tags} setActiveTag={setActiveTag} navigate={navigate}
              loading={loading} error={error} onRetry={refresh}
            />
          </div>
        </div>

        <div className="flex-shrink-0 overflow-y-auto no-scrollbar"
          style={{ width: 220, borderLeft: "1px solid var(--neu-border)", padding: "20px 16px" }}>
          <div className="mb-6">
            <div className="flex items-center gap-1.5 mb-3">
              <TrendingUp style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={2} />
              <p className="uppercase" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>热门话题</p>
            </div>
            {trending?.trending_tags.slice(0, 5).map(tag => (
              <motion.button
                key={tag.name} whileTap={{ scale: 0.98 }}
                className="flex items-center justify-between w-full rounded-[10px] px-3 py-2 outline-none neu-surface-ghost"
              >
                <span style={{ fontSize: 13, color: "var(--neu-text)" }}>#{tag.name}</span>
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{tag.usage_count}</span>
              </motion.button>
            ))}
          </div>

          <div style={{ height: 1, background: "var(--neu-border)", marginBottom: 20 }} />

          <div>
            <p className="uppercase mb-3" style={{ fontSize: 10, letterSpacing: "0.12em", color: "var(--neu-text-muted)" }}>活跃用户</p>
            {moments.slice(0, 4).map(item => (
              <div key={item.moment_id} role="button" tabIndex={0} aria-label={item.user.nickname} onKeyDown={e => e.key === "Enter" && navigate(`/user/${item.user.user_id}`)} className="flex items-center gap-2.5 mb-3.5 cursor-pointer"
                onClick={() => navigate(`/user/${item.user.user_id}`)}>
                <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 32, height: 32 }}>
                  {item.user.avatar_url && <img src={item.user.avatar_url} alt={item.user.nickname} className="w-full h-full object-cover" />}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="truncate" style={{ fontSize: 12, color: "var(--neu-text)" }}>{item.user.nickname}</p>
                  <p className="truncate" style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
                    {item.location?.location_name ?? formatDistance(item.location?.distance)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

export function Discover() {
  const { platform } = usePlatform();
  useDocumentTitle("发现");
  if (platform === "desktop") return <DiscoverDesktop />;
  if (platform === "tablet") return <DiscoverTablet />;
  return <DiscoverMobile />;
}
