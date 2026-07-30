import React, { useEffect, useMemo, useState } from "react";
import { ArrowLeft, Map, MapPin, Radar, Search, SlidersHorizontal, UserCheck, UserPlus, Users } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate } from "react-router";
import { ROUTES } from "../../lib/routes";
import { usePlatform } from "../../contexts/PlatformContext";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDefaultLocation } from "../../hooks/useDefaultLocation";
import { ActionIconButton } from "../../components/shared/ActionIconButton";
import { BspCardCanvas } from "../../components/shared/BspCardCanvas";
import { getNearbyUsers } from "../../../api";

interface NearbyUser {
  id: number; name: string; distance: string; avatar: string;
  status: string; tag: string; mutual: number; note: string;
}

const SORT_OPTIONS = ["距离", "在线", "共同好友"];
const FILTER_TAGS = ["全部", "摄影", "旅行", "音乐", "美食", "运动", "阅读", "艺术"];

function OnlineStrip({ users, onNavigate }: { users: NearbyUser[]; onNavigate: (path: string) => void }) {
  return (
    <section className="mb-5" aria-label="正在附近">
      <div className="flex items-center gap-2 mb-3">
        <motion.span animate={{ scale: [1, 1.45, 1], opacity: [1, 0.58, 1] }} transition={{ duration: 2, repeat: Infinity }} className="rounded-full" style={{ width: 7, height: 7, background: "var(--semantic-online)" }} />
        <span className="neu-eyebrow">正在附近</span>
        <span className="px-1.5 py-0.5 rounded-full bg-[color-mix(in_srgb,var(--semantic-online)_14%,transparent)] text-[var(--semantic-online)]" style={{ fontSize: 10 }}>{users.length} 人在线</span>
      </div>
      <div className="flex gap-3 overflow-x-auto no-scrollbar pb-1">
        {users.map((user) => (
          <motion.button key={user.id} whileHover={{ y: -2 }} whileTap={{ scale: 0.93 }} onClick={() => onNavigate(`/user/${user.id}`)} className="flex-shrink-0 flex flex-col items-center gap-1.5 outline-none">
            <div className="relative">
              <img src={user.avatar} alt={user.name} className="w-14 h-14 rounded-full object-cover" style={{ border: "2px solid var(--semantic-online)", boxShadow: "0 0 0 2px var(--neu-bg)" }} />
              <span className="absolute bottom-0 right-0 rounded-full" style={{ width: 11, height: 11, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }} />
            </div>
            <span className="max-w-14 truncate text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>{user.name}</span>
          </motion.button>
        ))}
      </div>
    </section>
  );
}

function UserCard({ user, followed, onFollow, onNavigate, format = "standard" }: { user: NearbyUser; followed: boolean; onFollow: () => void; onNavigate: () => void; format?: "focus" | "portrait" | "standard" }) {
  const hover = useHoverConfig();
  const [hovered, setHovered] = useState(false);
  return (
    <motion.article whileHover={hover.card as any} whileTap={hover.tap as any} onHoverStart={() => setHovered(true)} onHoverEnd={() => setHovered(false)} onClick={onNavigate} role="button" tabIndex={0} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") onNavigate(); }} className="flex h-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring" style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <div className="relative min-h-[160px] flex-1 overflow-hidden">
        <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
        <div className="absolute inset-x-0 bottom-0 h-3/5" style={{ background: "linear-gradient(to top, rgba(0,0,0,.74), transparent)" }} />
        <div className="absolute top-2.5 right-2.5 flex items-center gap-1 px-1.5 py-0.5 rounded-full" style={{ background: "var(--overlay-strong)", backdropFilter: "blur(8px)" }}>
          <MapPin style={{ width: 9, height: 9, color: "white" }} strokeWidth={1.5} /><span className="text-white" style={{ fontSize: 10 }}>{user.distance}</span>
        </div>
        <div className="absolute bottom-0 inset-x-0 flex items-end justify-between gap-2 px-3 pb-2.5">
          <div className="min-w-0"><p className="truncate text-white" style={{ fontSize: 14, fontWeight: 600 }}>{user.name}</p><p className="truncate text-white/78" style={{ fontSize: 10, marginTop: 2 }}>{user.note}</p></div>
          {user.status === "在线" && <span className="flex-shrink-0 rounded-full" title="在线" style={{ width: 9, height: 9, background: "var(--semantic-online)", boxShadow: "0 0 0 2px rgba(255,255,255,.62)" }} />}
        </div>
      </div>
      <div className="px-3 pt-2.5 pb-3">
        <div className="flex items-center justify-between gap-2 mb-2.5"><span className="px-1.5 py-0.5 rounded-full bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]" style={{ fontSize: 10, border: "1px solid var(--neu-accent-soft-border)" }}>#{user.tag}</span><span className="truncate text-[var(--neu-text-muted)]" style={{ fontSize: 10 }}>{user.mutual ? `${user.mutual} 位共同好友` : "等待认识"}</span></div>
        <div className="flex items-center justify-between gap-2"><span className="text-[var(--neu-text-muted)]" style={{ fontSize: 10 }}>{followed ? "已关注，可以发消息" : "感兴趣就打个招呼"}</span><motion.button whileTap={{ scale: 0.9 }} onClick={(event) => { event.stopPropagation(); onFollow(); }} className="flex items-center justify-center rounded-full outline-none flex-shrink-0" aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`} title={followed ? "已关注" : "关注"} style={followed ? { width: 30, height: 30, background: "var(--neu-accent-soft)", border: "1px solid var(--neu-accent-soft-border)", color: "var(--neu-accent-primary)" } : { width: 30, height: 30, background: "var(--neu-accent-primary)", color: "white", boxShadow: "0 2px 8px var(--neu-glow)" }}>{followed ? <UserCheck size={15} /> : <UserPlus size={15} />}</motion.button></div>
      </div>
    </motion.article>
  );
}

export function NearbyPeoplePage() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const { reference_point } = useDefaultLocation();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState("距离");
  const [tagFilter, setTagFilter] = useState("全部");
  const [followedIds, setFollowedIds] = useState<Set<number>>(new Set());
  const [visibleCount, setVisibleCount] = useState(12);
  const [allUsers, setAllUsers] = useState<NearbyUser[]>([]);
  const [loading, setLoading] = useState(true);
  const isDesktop = platform === "desktop";
  const isTablet = platform === "tablet";

  useEffect(() => {
    // spec: feed/nearby_users returns nearby UserItem[]. reference_point from ClientConfig default_location.
    if (!reference_point) return;
    let cancelled = false;
    async function fetchUsers() {
      try {
        setLoading(true);
        const res = await getNearbyUsers({ reference_point }).catch(() => ({ code: -1, data: { items: [] } }));
        if (cancelled) return;
        const data = (res.code === 0 ? res.data : null) as any;
        if (data) {
          const items: NearbyUser[] = (data.items || []).map((u: any, i: number) => ({
            // spec UserItem: user_id, nickname, avatar_url, presence_status, distance, tags, bio, social_context
            id: u.user_id ?? i + 1,
            name: u.nickname ?? "",
            distance: u.distance ?? "",
            avatar: u.avatar_url ?? "",
            status: u.presence_status === "online" ? "在线" : "离线",
            tag: u.tags?.[0] ?? "",
            mutual: u.social_context?.mutual_friends_count ?? 0,
            note: u.bio ?? "",
          }));
          setAllUsers(items);
        }
      } catch {
        if (!cancelled) setAllUsers([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchUsers();
    return () => { cancelled = true; };
  }, [reference_point]);

  const list = useMemo(() => {
    const next = allUsers.filter((user) => (tagFilter === "全部" || user.tag === tagFilter) && (!search || `${user.name}${user.tag}${user.note}`.toLowerCase().includes(search.toLowerCase())));
    if (sort === "在线") return [...next].sort((a, b) => Number(b.status === "在线") - Number(a.status === "在线"));
    if (sort === "共同好友") return [...next].sort((a, b) => b.mutual - a.mutual);
    return next;
  }, [search, sort, tagFilter, allUsers]);

  const onlineUsers = allUsers.filter((user) => user.status === "在线");
  const onlineCount = list.filter((user) => user.status === "在线").length;
  const contentStyle = isDesktop ? { width: "100%" } : isTablet ? { maxWidth: 964, width: "100%" } : { maxWidth: 680, width: "100%" };
  const gridColumns = platform === "mobile" ? 2 : isDesktop ? 3 : 2;
  const toggleFollow = (id: number) => setFollowedIds((previous) => { const next = new Set(previous); next.has(id) ? next.delete(id) : next.add(id); return next; });

  return <>
    <header className="sticky top-0 z-10 bg-[var(--neu-bg)]" style={{ backdropFilter: "blur(16px)", borderBottom: "1px solid var(--neu-border)" }}>
      <div className={isDesktop ? "px-7 py-3" : "px-5 py-3 mx-auto"} style={contentStyle}>
        <div className="flex items-center gap-3">
          <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} size={34} />
          <div className="min-w-0 flex-1"><h1 className="text-[var(--neu-text)]" style={{ fontSize: 18 }}>附近的人</h1><p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>徐州 · 5km 内 · 每 3 分钟刷新</p></div>
          <div className="hidden sm:flex items-center gap-2 flex-1 max-w-sm px-3 py-2 rounded-[14px] neu-input-surface"><Search size={14} color="var(--neu-text-muted)" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索昵称、兴趣或此刻状态" className="min-w-0 flex-1 bg-transparent outline-none text-[var(--neu-text)]" style={{ fontSize: 13 }} /></div>
          <ActionIconButton icon={Map} label="在地图中查看" onClick={() => navigate(ROUTES.map)} variant="ghost" tone="accent" size={34} />
        </div>
        <div className="sm:hidden mt-3 flex items-center gap-2 px-3 py-2 rounded-[14px] neu-input-surface"><Search size={14} color="var(--neu-text-muted)" /><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索昵称、兴趣或状态" className="min-w-0 flex-1 bg-transparent outline-none text-[var(--neu-text)]" style={{ fontSize: 13 }} /></div>
      </div>
    </header>

    <div className={isDesktop ? "px-7 py-6" : "px-5 py-5 mx-auto"} style={contentStyle}>
      <div className={isTablet ? "grid gap-6" : ""}>
        <div className="min-w-0">
          <OnlineStrip users={onlineUsers} onNavigate={navigate} />
          <section className="neu-surface-hero rounded-[22px] p-4 mb-5 flex items-center gap-4">
            <div className="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-accent-primary)]" style={{ width: 46, height: 46, boxShadow: "0 4px 14px var(--neu-glow)" }}><Radar size={20} color="white" /></div>
            <div className="min-w-0 flex-1"><p className="text-[var(--neu-text)]" style={{ fontSize: 14, fontWeight: 600 }}>此刻的徐州，正有 {onlineCount} 位新朋友在附近</p><p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, marginTop: 3 }}>已按你的位置与兴趣筛出 {list.length} 人；打个招呼，从共同话题开始。</p></div>
            <button type="button" onClick={() => navigate(ROUTES.map)} className="hidden md:flex flex-shrink-0 items-center gap-1.5 px-3 py-2 rounded-full neu-surface-ghost neu-focus-ring text-[var(--neu-accent-primary)]" style={{ fontSize: 11 }}><Map size={13} />查看地图</button>
          </section>
          <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-3 mb-5">
            <div className="flex items-center gap-1 p-1 rounded-full neu-input-surface w-fit">{SORT_OPTIONS.map((option) => <button key={option} type="button" onClick={() => setSort(option)} aria-pressed={sort === option} className={`px-3.5 py-1 rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${sort === option ? "bg-[var(--neu-accent-primary)] text-white" : "neu-surface-ghost text-[var(--neu-text-muted)]"}`} style={sort === option ? { fontSize: 12, fontWeight: 600 } : { fontSize: 12 }}>{option}</button>)}</div>
            <div className="flex gap-1.5 overflow-x-auto no-scrollbar">{FILTER_TAGS.map((tag) => <button key={tag} type="button" onClick={() => setTagFilter(tag)} aria-pressed={tagFilter === tag} className={`flex-shrink-0 px-3 py-1 rounded-full outline-none neu-surface-ghost focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${tagFilter === tag ? "bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"}`} style={tagFilter === tag ? { border: "1px solid var(--neu-accent-soft-border)", fontSize: 12, fontWeight: 600 } : { fontSize: 12 }}>{tag}</button>)}</div>
          </div>
          <div className="flex items-center justify-between mb-3"><span className="neu-eyebrow">为你排序</span><span className="flex items-center gap-1 text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}><SlidersHorizontal size={12} />{list.length} 位可认识</span></div>
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
          </motion.div> : <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex flex-col items-center gap-3 py-20"><Users size={40} color="var(--neu-text-muted)" opacity={0.4} /><p className="text-[var(--neu-text-muted)]" style={{ fontSize: 14 }}>没有找到匹配的附近用户</p></motion.div>}</AnimatePresence>
        </div>
      </div>
    </div>
  </>;
}
