import React, { useMemo, useState } from "react";
import { ArrowLeft, Map, MapPin, Radar, Search, SlidersHorizontal, UserCheck, UserPlus, Users } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate } from "react-router";
import { ROUTES } from "../lib/routes";
import { usePlatform } from "../contexts/PlatformContext";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { MoodCheckinCard, NearbyPulseCard, SuggestedFollowsCard, TabletRail, VibeCard } from "../components/TabletRail";
import { ContextRail } from "../components/layout/ContextRail";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { BspCardCanvas } from "../components/shared/BspCardCanvas";

const ALL_USER_SEED = [
  { name: "鹿鸣", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400", tag: "摄影", note: "刚在回龙窝拍完一卷胶片" },
  { name: "阿禾", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=400", tag: "旅行", note: "在找周末短途搭子" },
  { name: "野原", avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=400", tag: "音乐", note: "晚点会去弹唱角" },
  { name: "Mika", avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=400", tag: "美食", note: "想试试黄河路的新店" },
  { name: "南枝", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=400", tag: "艺术", note: "在美术馆看展" },
  { name: "阿川", avatar: "https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=400", tag: "运动", note: "云龙湖夜跑，缺一位" },
  { name: "宋悦", avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=400", tag: "阅读", note: "最近在读《山茶文具店》" },
  { name: "林一", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400", tag: "游戏", note: "晚上开一局双人成行" },
  { name: "小满", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=400", tag: "电影", note: "想找人看周末首映" },
  { name: "青砚", avatar: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=400", tag: "科技", note: "在做一台口袋电台" },
  { name: "Yuki", avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=400", tag: "时尚", note: "逛完展想喝杯咖啡" },
  { name: "陈芊", avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=400", tag: "户外", note: "计划周日去泉山" },
];

const DISTANCE_TICKS = ["246m", "252m", "388m", "520m", "820m", "910m", "1.1km", "1.3km", "1.6km", "2.1km", "2.4km", "3.0km", "3.4km", "3.8km", "4.2km"];

// Mock pagination: real backends return dozens–hundreds of nearby users, so the
// seed pool is cycled with id/name offsets to produce a list large enough to
// exercise the BSP canvas's incremental-load + virtualization path. Replaced by
// the API once pagination is wired through.
const ALL_USERS = Array.from({ length: 96 }, (_, i) => {
  const seed = ALL_USER_SEED[i % ALL_USER_SEED.length];
  const cycle = Math.floor(i / ALL_USER_SEED.length);
  return {
    id: i + 1,
    name: cycle === 0 ? seed.name : `${seed.name}${cycle + 1}`,
    distance: DISTANCE_TICKS[i % DISTANCE_TICKS.length],
    avatar: seed.avatar,
    status: i % 3 === 2 ? "离线" : "在线",
    tag: seed.tag,
    mutual: (i * 7) % 9,
    note: seed.note,
  };
});

const SORT_OPTIONS = ["距离", "在线", "共同好友"];
const FILTER_TAGS = ["全部", "摄影", "旅行", "音乐", "美食", "运动", "阅读", "艺术"];

function OnlineStrip({ users, onNavigate }: { users: typeof ALL_USERS; onNavigate: (path: string) => void }) {
  return (
    <section className="mb-5" aria-label="正在附近">
      <div className="flex items-center gap-2 mb-3">
        <motion.span animate={{ scale: [1, 1.45, 1], opacity: [1, 0.58, 1] }} transition={{ duration: 2, repeat: Infinity }} className="rounded-full" style={{ width: 7, height: 7, background: "var(--semantic-online)" }} />
        <span className="neu-eyebrow">正在附近</span>
        <span className="px-1.5 py-0.5 rounded-full" style={{ fontSize: 10, background: "color-mix(in srgb, var(--semantic-online) 14%, transparent)", color: "var(--semantic-online)" }}>{users.length} 人在线</span>
      </div>
      <div className="flex gap-3 overflow-x-auto no-scrollbar pb-1">
        {users.map((user) => (
          <motion.button key={user.id} whileHover={{ y: -2 }} whileTap={{ scale: 0.93 }} onClick={() => onNavigate(`/user/${user.id}`)} className="flex-shrink-0 flex flex-col items-center gap-1.5 outline-none">
            <div className="relative">
              <img src={user.avatar} alt={user.name} className="w-14 h-14 rounded-full object-cover" style={{ border: "2px solid var(--semantic-online)", boxShadow: "0 0 0 2px var(--neu-bg)" }} />
              <span className="absolute bottom-0 right-0 rounded-full" style={{ width: 11, height: 11, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }} />
            </div>
            <span className="max-w-14 truncate" style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{user.name}</span>
          </motion.button>
        ))}
      </div>
    </section>
  );
}

function UserCard({ user, followed, onFollow, onNavigate, format = "standard" }: { user: typeof ALL_USERS[number]; followed: boolean; onFollow: () => void; onNavigate: () => void; format?: "focus" | "portrait" | "standard" }) {
  const hover = useHoverConfig();
  const [hovered, setHovered] = useState(false);
  return (
    <motion.article whileHover={hover.card as any} whileTap={hover.tap as any} onHoverStart={() => setHovered(true)} onHoverEnd={() => setHovered(false)} onClick={onNavigate} role="button" tabIndex={0} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") onNavigate(); }} className="flex h-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-surface-flat neu-card-hover outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]" style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <div className="relative min-h-[160px] flex-1 overflow-hidden">
        <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
        <div className="absolute inset-x-0 bottom-0 h-3/5" style={{ background: "linear-gradient(to top, rgba(0,0,0,.74), transparent)" }} />
        <div className="absolute top-2.5 right-2.5 flex items-center gap-1 px-1.5 py-0.5 rounded-full" style={{ background: "var(--overlay-strong)", backdropFilter: "blur(8px)" }}>
          <MapPin style={{ width: 9, height: 9, color: "white" }} strokeWidth={1.5} /><span style={{ fontSize: 10, color: "white" }}>{user.distance}</span>
        </div>
        <div className="absolute bottom-0 inset-x-0 flex items-end justify-between gap-2 px-3 pb-2.5">
          <div className="min-w-0"><p className="truncate" style={{ fontSize: 14, color: "white", fontWeight: 600 }}>{user.name}</p><p className="truncate" style={{ fontSize: 10, color: "rgba(255,255,255,.78)", marginTop: 2 }}>{user.note}</p></div>
          {user.status === "在线" && <span className="flex-shrink-0 rounded-full" title="在线" style={{ width: 9, height: 9, background: "var(--semantic-online)", boxShadow: "0 0 0 2px rgba(255,255,255,.62)" }} />}
        </div>
      </div>
      <div className="px-3 pt-2.5 pb-3">
        <div className="flex items-center justify-between gap-2 mb-2.5"><span className="px-1.5 py-0.5 rounded-full" style={{ fontSize: 10, background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)", border: "1px solid var(--neu-accent-soft-border)" }}>#{user.tag}</span><span className="truncate" style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{user.mutual ? `${user.mutual} 位共同好友` : "等待认识"}</span></div>
        <div className="flex items-center justify-between gap-2"><span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{followed ? "已关注，可以发消息" : "感兴趣就打个招呼"}</span><motion.button whileTap={{ scale: 0.9 }} onClick={(event) => { event.stopPropagation(); onFollow(); }} className="flex items-center justify-center rounded-full outline-none flex-shrink-0" aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`} title={followed ? "已关注" : "关注"} style={followed ? { width: 30, height: 30, background: "var(--neu-accent-soft)", border: "1px solid var(--neu-accent-soft-border)", color: "var(--neu-accent-primary)" } : { width: 30, height: 30, background: "var(--neu-accent-primary)", color: "white", boxShadow: "0 2px 8px var(--neu-glow)" }}>{followed ? <UserCheck size={15} /> : <UserPlus size={15} />}</motion.button></div>
      </div>
    </motion.article>
  );
}

export function NearbyAll() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("距离");
  const [tagFilter, setTagFilter] = useState("全部");
  const [followedIds, setFollowedIds] = useState<Set<number>>(new Set());
  const [visibleCount, setVisibleCount] = useState(12);
  const isDesktop = platform === "desktop";
  const isTablet = platform === "tablet";
  const showRail = isTablet;

  const list = useMemo(() => {
    const next = ALL_USERS.filter((user) => (tagFilter === "全部" || user.tag === tagFilter) && (!search || `${user.name}${user.tag}${user.note}`.toLowerCase().includes(search.toLowerCase())));
    if (sort === "在线") return [...next].sort((a, b) => Number(b.status === "在线") - Number(a.status === "在线"));
    if (sort === "共同好友") return [...next].sort((a, b) => b.mutual - a.mutual);
    return next;
  }, [search, sort, tagFilter]);

  const onlineUsers = ALL_USERS.filter((user) => user.status === "在线");
  const onlineCount = list.filter((user) => user.status === "在线").length;
  const contentStyle = isDesktop ? { width: "100%" } : isTablet ? { maxWidth: 964, width: "100%" } : { maxWidth: 680, width: "100%" };
  const gridColumns = platform === "mobile" ? 2 : isDesktop ? 3 : 2;
  const toggleFollow = (id: number) => setFollowedIds((previous) => { const next = new Set(previous); next.has(id) ? next.delete(id) : next.add(id); return next; });

  return <>
    <header className="sticky top-0 z-10" style={{ background: "var(--neu-bg)", backdropFilter: "blur(16px)", borderBottom: "1px solid var(--neu-border)" }}>
      <div className={isDesktop ? "px-7 py-3" : "px-5 py-3 mx-auto"} style={contentStyle}>
        <div className="flex items-center gap-3">
          <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} size={34} />
          <div className="min-w-0 flex-1"><h1 style={{ fontSize: 18, color: "var(--neu-text)" }}>附近的人</h1><p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>徐州 · 5km 内 · 每 3 分钟刷新</p></div>
          <div className="hidden sm:flex items-center gap-2 flex-1 max-w-sm px-3 py-2 rounded-[14px] neu-input-surface"><Search size={14} color="var(--neu-text-muted)" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索昵称、兴趣或此刻状态" className="min-w-0 flex-1 bg-transparent outline-none" style={{ fontSize: 13, color: "var(--neu-text)" }} /></div>
          <ActionIconButton icon={Map} label="在地图中查看" onClick={() => navigate(ROUTES.map)} variant="ghost" tone="accent" size={34} />
        </div>
        <div className="sm:hidden mt-3 flex items-center gap-2 px-3 py-2 rounded-[14px] neu-input-surface"><Search size={14} color="var(--neu-text-muted)" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索昵称、兴趣或状态" className="min-w-0 flex-1 bg-transparent outline-none" style={{ fontSize: 13, color: "var(--neu-text)" }} /></div>
      </div>
    </header>

    <div className={isDesktop ? "px-7 py-6" : "px-5 py-5 mx-auto"} style={contentStyle}>
      <div className={showRail ? "grid gap-6 context-rail-grid" : ""}>
        <div className="min-w-0">
          <OnlineStrip users={onlineUsers} onNavigate={navigate} />
          <section className="neu-surface-hero rounded-[22px] p-4 mb-5 flex items-center gap-4">
            <div className="flex items-center justify-center rounded-full flex-shrink-0" style={{ width: 46, height: 46, background: "var(--neu-accent-primary)", boxShadow: "0 4px 14px var(--neu-glow)" }}><Radar size={20} color="white" /></div>
            <div className="min-w-0 flex-1"><p style={{ fontSize: 14, fontWeight: 600, color: "var(--neu-text)" }}>此刻的徐州，正有 {onlineCount} 位新朋友在附近</p><p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 3 }}>已按你的位置与兴趣筛出 {list.length} 人；打个招呼，从共同话题开始。</p></div>
            <button type="button" onClick={() => navigate(ROUTES.map)} className="hidden md:flex flex-shrink-0 items-center gap-1.5 px-3 py-2 rounded-full neu-surface-ghost outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]" style={{ fontSize: 11, color: "var(--neu-accent-primary)" }}><Map size={13} />查看地图</button>
          </section>
          <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-3 mb-5">
            <div className="flex items-center gap-1 p-1 rounded-full neu-input-surface w-fit">{SORT_OPTIONS.map((option) => <button key={option} type="button" onClick={() => setSort(option)} aria-pressed={sort === option} className={`px-3.5 py-1 rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${sort === option ? "" : "neu-surface-ghost"}`} style={sort === option ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, fontWeight: 600 } : { color: "var(--neu-text-muted)", fontSize: 12 }}>{option}</button>)}</div>
            <div className="flex gap-1.5 overflow-x-auto no-scrollbar">{FILTER_TAGS.map((tag) => <button key={tag} type="button" onClick={() => setTagFilter(tag)} aria-pressed={tagFilter === tag} className="flex-shrink-0 px-3 py-1 rounded-full outline-none neu-surface-ghost focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]" style={tagFilter === tag ? { background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)", border: "1px solid var(--neu-accent-soft-border)", fontSize: 12, fontWeight: 600 } : { color: "var(--neu-text-muted)", fontSize: 12 }}>{tag}</button>)}</div>
          </div>
          <div className="flex items-center justify-between mb-3"><span className="neu-eyebrow">为你排序</span><span className="flex items-center gap-1" style={{ fontSize: 11, color: "var(--neu-text-muted)" }}><SlidersHorizontal size={12} />{list.length} 位可认识</span></div>
          <AnimatePresence mode="wait">{list.length ? <motion.div key={`${sort}-${tagFilter}-${search}`} initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
            {isDesktop ? (
              <BspCardCanvas
                items={list.slice(0, visibleCount)}
                gap={24}
                minRatio={0.68}
                maxRatio={1.48}
                minCardWidth={160}
                minCardHeight={250}
                minCanvasHeight={560}
                ariaLabel="附近用户的不规则满铺布局"
                hasMore={visibleCount < list.length}
                onLoadMore={() => setVisibleCount((c) => Math.min(c + 8, list.length))}
                renderItem={(user, index) => <UserCard user={user} format={index % 4 === 1 ? "portrait" : "standard"} followed={followedIds.has(user.id)} onFollow={() => toggleFollow(user.id)} onNavigate={() => navigate(ROUTES.user(user.id))} />}
              />
            ) : (
              <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(${gridColumns}, minmax(0, 1fr))` }}>
                {list.map((user, index) => <UserCard key={user.id} user={user} format={index === 0 ? "focus" : index % 5 === 2 ? "portrait" : "standard"} followed={followedIds.has(user.id)} onFollow={() => toggleFollow(user.id)} onNavigate={() => navigate(ROUTES.user(user.id))} />)}
              </div>
            )}
          </motion.div> : <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex flex-col items-center gap-3 py-20"><Users size={40} color="var(--neu-text-muted)" opacity={0.4} /><p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>没有找到匹配的附近用户</p></motion.div>}</AnimatePresence>
        </div>
        {showRail && <ContextRail className="hidden lg:block"><TabletRail><NearbyPulseCard /><MoodCheckinCard /><SuggestedFollowsCard /><VibeCard /></TabletRail></ContextRail>}
      </div>
    </div>
  </>;
}
