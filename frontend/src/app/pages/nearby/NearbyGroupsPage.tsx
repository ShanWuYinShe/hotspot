'use client'
import React, { useEffect, useMemo, useState } from "react";
import { CalendarDays, Check, Globe, Lock, MapPin, Plus, Sparkles, Users } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useRouter } from "next/navigation";
import { CreateGroupSheet } from "@/app/pages/groups/CreateGroupSheet";
import { usePlatform } from "@/contexts/PlatformContext";
import { useHoverConfig } from "@/hooks/useHoverConfig";
import { useDismissibleLayer } from "@/hooks/useDismissibleLayer";
import { useDefaultLocation } from "@/hooks/useDefaultLocation";
import { BspCardCanvas } from "@/components/shared/BspCardCanvas";
import { PageHeader } from "@/components/shared/PageHeader";
import { SearchInput } from "@/components/shared/SearchInput";
import { TagPill } from "@/components/shared/TagPill";
import { EmptyState } from "@/components/shared/EmptyState";
import { ROUTES } from "@/lib/routes";
import { getNearbyGroups } from "@/api";

const CATEGORIES = ["全部", "摄影", "音乐", "户外", "阅读", "美食", "健身", "游戏"];

// Group shape consumed by this page's cards. Populated from searchGroups
// (auth-optional, so anonymous users still see nearby public groups).
interface Group {
  id: string;
  name: string;
  banner: string;
  category: string;
  members: number;
  online: number;
  distance: string;
  privacy: "public" | "private";
  joined: boolean;
  desc: string;
  activity: string;
  avatars: string[];
}

/** Map a searchGroups result item to the card shape. */
function mapGroup(item: any): Group {
  const g = item?.group ?? item ?? {};
  return {
    id: String(g.group_id ?? g.id ?? ""),
    name: g.name ?? "",
    banner: g.avatar_url ?? g.banner ?? g.cover_url ?? "",
    category: (item?.tags?.[0] ?? g.category ?? "") || "同城",
    members: g.members_count ?? g.members ?? 0,
    online: g.online_count ?? g.online ?? 0,
    distance: item?.distance ?? g.distance ?? "",
    privacy: g.privacy_level === "private" ? "private" : "public",
    joined: item?.my_state?.is_joined ?? g.is_joined ?? false,
    desc: item?.description ?? g.description ?? g.bio ?? "",
    activity: item?.next_activity ?? g.activity ?? "",
    avatars: [],
  };
}

function JoinButton({ group, onToggle }: { group: Group; onToggle: () => void }) {
  return <motion.button whileTap={{ scale: 0.94 }} onClick={(event) => { event.stopPropagation(); onToggle(); }} className={`rounded-full outline-none text-[11px] px-[13px] py-1.5 ${group.joined ? "border border-[var(--neu-border-pressed)] text-[var(--neu-text-muted)]" : "bg-[var(--neu-accent-primary)] text-white font-semibold"}`} style={group.joined ? undefined : { boxShadow: "0 2px 8px var(--neu-glow)" }}>{group.joined ? <span className="flex items-center gap-1"><Check size={12} />已加入</span> : "加入"}</motion.button>;
}

function FeaturedGroupCard({ group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void }) {
  const hover = useHoverConfig();
  return <motion.article whileHover={hover.card as any} whileTap={hover.tap as any} onClick={onOpen} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") onOpen(); }} role="button" tabIndex={0} className="neu-surface-hero neu-card-hover rounded-[24px] overflow-hidden cursor-pointer h-full neu-focus-ring">
    <div className="relative h-48"><img src={group.banner} alt={group.name} className="w-full h-full object-cover" /><div className="absolute inset-0" style={{ background: "linear-gradient(to top, rgba(0,0,0,.72), transparent 70%)" }} /><div className="absolute top-3 left-3 flex items-center gap-1.5 px-2 py-1 rounded-full" style={{ background: "var(--overlay-strong)", backdropFilter: "blur(8px)" }}><Sparkles size={11} color="white" /><span className="text-white text-[10px]">本周精选</span></div><div className="absolute bottom-3 left-4 right-4"><p className="text-white text-lg font-semibold">{group.name}</p><p className="truncate text-white/80 text-[11px] mt-[3px]">{group.desc}</p></div></div>
    <div className="p-4 flex items-center gap-3"><div className="flex-1 min-w-0"><div className="flex items-center gap-3 text-[var(--neu-text-muted)] text-[11px]"><span className="flex items-center gap-1"><Users size={12} />{group.members}</span><span className="flex items-center gap-1 text-[var(--semantic-online)]"><span className="rounded-full" style={{ width: 6, height: 6, background: "var(--semantic-online)" }} />{group.online} 在线</span><span>{group.distance}</span></div><p className="truncate flex items-center gap-1 mt-2 text-[var(--neu-accent-primary)] text-[11px]"><CalendarDays size={12} />{group.activity}</p></div><JoinButton group={group} onToggle={onToggle} /></div>
  </motion.article>;
}

function GroupCard({ group, onToggle, onOpen }: { group: Group; onToggle: () => void; onOpen: () => void }) {
  const hover = useHoverConfig(); const [hovered, setHovered] = useState(false);
  return <motion.article whileHover={hover.card as any} whileTap={hover.tap as any} onClick={onOpen} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") onOpen(); }} role="button" tabIndex={0} onHoverStart={() => setHovered(true)} onHoverEnd={() => setHovered(false)} className="flex h-full flex-col rounded-[18px] overflow-hidden cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring" style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background .18s ease" }}>
    <div className="relative min-h-[96px] flex-1 overflow-hidden"><img src={group.banner} alt={group.name} className="w-full h-full object-cover" /><div className="absolute inset-0" style={{ background: "linear-gradient(to top, rgba(0,0,0,.68), transparent 60%)" }} /><span className="absolute top-2 left-2 px-1.5 py-0.5 rounded-full text-white bg-[var(--neu-accent-primary)] text-[9px]">{group.category}</span><span className="absolute top-2 right-2 flex items-center justify-center rounded-full" style={{ width: 20, height: 20, background: "var(--overlay-strong)" }}>{group.privacy === "private" ? <Lock size={10} color="white" /> : <Globe size={10} color="white" />}</span><p className="absolute bottom-2.5 left-3 right-3 line-clamp-2 text-white text-[13px] font-semibold">{group.name}</p></div>
    <div className="p-3"><div className="flex items-center justify-between gap-2 mb-2"><span className="flex items-center gap-1 truncate text-[var(--neu-text-muted)] text-[10px]"><Users size={10} />{group.members} 人</span><span className="flex items-center gap-1 flex-shrink-0 text-[var(--neu-accent-primary)] text-[10px]"><MapPin size={10} />{group.distance}</span></div><p className="truncate flex items-center gap-1 mb-3 text-[var(--semantic-online)] text-[10px]"><CalendarDays size={10} />{group.activity}</p><div className="flex items-center justify-between gap-2"><div className="flex -space-x-2">{group.avatars.map((avatar, index) => <img key={avatar} src={avatar} alt="群成员" className="w-5 h-5 rounded-full object-cover" style={{ border: "1px solid var(--neu-bg)", zIndex: 3 - index }} />)}</div><JoinButton group={group} onToggle={onToggle} /></div></div>
  </motion.article>;
}

export function NearbyGroupsPage() {
  const router = useRouter(); const { platform } = usePlatform();
  const { reference_point } = useDefaultLocation();
  const [activeCategory, setActiveCategory] = useState("全部"); const [searchQuery, setSearchQuery] = useState(""); const [showCreate, setShowCreate] = useState(false); const [groups, setGroups] = useState<Group[]>([]); const [visibleCount, setVisibleCount] = useState(12); const [loading, setLoading] = useState(true);

  useEffect(() => {
    // spec: feed/nearby_groups returns nearby groups (reference_point from ClientConfig).
    if (!reference_point) return;
    let cancelled = false;
    async function fetchGroups() {
      setLoading(true);
      const res = await getNearbyGroups({ reference_point, pagination: { limit: 24 } }).catch(() => ({ code: -1, data: { items: [] } }));
      if (cancelled) return;
      const items = (res.code === 0 && res.data ? (res.data as any).items : []) as any[];
      setGroups(items.map(mapGroup));
      setLoading(false);
    }
    fetchGroups();
    return () => { cancelled = true; };
  }, [reference_point]);
  useDismissibleLayer({ open: showCreate, onDismiss: () => setShowCreate(false) });
  const isDesktop = platform === "desktop"; const isTablet = platform === "tablet"; const showRail = isTablet;
  const filtered = useMemo(() => groups.filter((group) => (activeCategory === "全部" || group.category === activeCategory) && (!searchQuery || `${group.name}${group.desc}${group.category}`.includes(searchQuery))), [activeCategory, groups, searchQuery]);
  const visibleGroups = filtered.slice(0, visibleCount);
  const [featured, ...rest] = filtered; const contentStyle = isDesktop ? { width: "100%" } : isTablet ? { maxWidth: 964, width: "100%" } : { maxWidth: 680, width: "100%" };
  // TODO: connect to groups/join & groups/leave once client wrappers exist;
  // for now toggle locally so the button reflects intent immediately.
  const toggleJoin = (id: string) => setGroups((previous) => previous.map((group) => group.id === id ? { ...group, joined: !group.joined } : group));

  return <>
    <div className="sticky top-0 z-10 bg-[var(--neu-bg)]" style={{ backdropFilter: "blur(16px)", borderBottom: "1px solid var(--neu-border)" }}>
      <PageHeader
        title="附近群组"
        subtitle={`徐州 · 5km 内 · ${filtered.length} 个活跃群组`}
        onBack={() => router.back()}
        className={isDesktop ? "px-7" : ""}
        actions={
          <motion.button type="button" whileTap={{ scale: .94 }} onClick={() => setShowCreate(true)} className="flex items-center gap-1 px-3 py-1.5 rounded-full neu-focus-ring bg-[var(--neu-accent-primary)] text-white text-xs font-semibold" style={{ boxShadow: "0 3px 10px var(--neu-glow)" }}><Plus size={14} />创建</motion.button>
        }
      />
      <div className={isDesktop ? "px-7 pb-3" : "px-5 pb-3 mx-auto"} style={contentStyle}>
        <SearchInput value={searchQuery} onChange={setSearchQuery} placeholder="搜索群组、活动或兴趣" compact />
        <div className="flex gap-1.5 overflow-x-auto no-scrollbar mt-3">
          {CATEGORIES.map((category) => (
            <TagPill key={category} active={activeCategory === category} onClick={() => setActiveCategory(category)}>{category}</TagPill>
          ))}
        </div>
      </div>
    </div>
    <main className={isDesktop ? "px-7 py-6" : "px-5 py-5 mx-auto"} style={contentStyle}><div className="min-w-0">
      {loading ? <div className="flex flex-col items-center gap-3 py-20"><div className="rounded-full border-2 border-[var(--neu-border)] border-t-[var(--neu-accent-primary)] animate-spin" style={{ width: 28, height: 28 }} /><p className="text-[var(--neu-text-muted)] text-[13px]">加载附近群组…</p></div>
      : filtered.length ? <>
        <div className="flex items-center justify-between mb-3"><span className="neu-eyebrow">正在附近的群组</span><span className="text-[var(--semantic-online)] text-[11px]">{filtered.reduce((sum, group) => sum + group.online, 0)} 人正在群聊</span></div>
        {isDesktop ? <BspCardCanvas items={visibleGroups} gap={24} minRatio={0.68} maxRatio={1.48} minCardWidth={160} minCardHeight={210} minCanvasHeight={520} ariaLabel="附近群组的不规则满铺布局" hasMore={visibleCount < filtered.length} onLoadMore={() => setVisibleCount((c) => Math.min(c + 8, filtered.length))} renderItem={(group) => <GroupCard group={group} onToggle={() => toggleJoin(group.id)} onOpen={() => router.push(ROUTES.group(group.id))} />} /> : <>{featured && <div className="mb-5"><FeaturedGroupCard group={featured} onToggle={() => toggleJoin(featured.id)} onOpen={() => router.push(ROUTES.group(featured.id))} /></div>}<div className="grid grid-cols-2 gap-3">{rest.map((group) => <GroupCard key={group.id} group={group} onToggle={() => toggleJoin(group.id)} onOpen={() => router.push(ROUTES.group(group.id))} />)}</div></>}
      </> : <EmptyState icon={<Users size={32} color="var(--neu-text-muted)" />} title="没有找到相关群组" />}
    </div></main>
    <AnimatePresence>{showCreate && <><motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="fixed inset-0 z-40" style={{ background: "rgba(0,0,0,.35)", backdropFilter: "blur(3px)" }} onClick={() => setShowCreate(false)} /><motion.div initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }} transition={{ type: "spring", damping: 30, stiffness: 260 }} className="fixed inset-x-0 bottom-0 z-50 h-[92%] overflow-hidden bg-[var(--neu-bg)]" role="dialog" aria-modal="true" aria-label="创建群组" style={{ borderRadius: "28px 28px 0 0", boxShadow: "0 -8px 40px rgba(0,0,0,.18)" }}><CreateGroupSheet onClose={() => setShowCreate(false)} /></motion.div></>}</AnimatePresence>
  </>;
}
