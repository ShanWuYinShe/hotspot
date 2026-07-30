import React, { useEffect, useState } from "react";
import {
  MapPin, ChevronRight, Users,
  Coffee, Music, Star, Flame, Lock, Globe, Radar,
} from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../../contexts/PlatformContext";
import { useNavigate } from "react-router";
import { ROUTES } from "../../lib/routes";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDefaultLocation } from "../../hooks/useDefaultLocation";
import { getTrending, searchMoments, getClientConfig } from "../../../api";

/* ── Section heading ────────────────────────────────────────────── */
function SectionHead({ title, action, onAction }: { title: string; action?: string; onAction?: () => void }) {
  return (
    <div className="flex items-end justify-between mb-3">
      <span className="neu-eyebrow">{title}</span>
      {action && onAction && (
        <motion.button whileTap={{ scale: 0.94 }} onClick={onAction}
          className="flex items-center gap-0.5 outline-none neu-surface-ghost rounded-full px-2 py-0.5"
          style={{ color: "var(--neu-accent-primary)" }}>
          <span style={{ fontSize: 11 }}>{action}</span>
          <ChevronRight style={{ width: 12, height: 12 }} strokeWidth={1.5} />
        </motion.button>
      )}
    </div>
  );
}

/* ── Types ─────────────────────────────────────────────────────── */
interface NearbyUser { id: number; name: string; distance: string; avatar: string; status: string; tag: string; }
interface Activity { id: number; icon: React.ElementType; title: string; sub: string; color: string; bg: string; }
interface NearbyGroup { id: number; name: string; members: number; banner: string; distance: string; privacy: "public" | "private"; }

/* ── Hero stats banner — single focal point with ambient glow ───── */
function HeroStats({ onExplore, stats }: { onExplore: () => void; stats: { nearby: string; active: string; online: string } }) {
  const { platform } = usePlatform();
  const rangeLabel = platform === "mobile" ? "徐州 · 5km 内 · 持续更新" : "5km 内 · 持续更新";
  return (
    <div className="neu-ambient-glow">
      <motion.div
        whileTap={{ scale: 0.99 }}
        onClick={onExplore}
        className="neu-surface-hero rounded-[24px] cursor-pointer overflow-hidden"
        style={{ padding: "18px 20px" }}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div
              className="flex items-center justify-center rounded-full"
              style={{
                width: 38, height: 38,
                background: "var(--neu-accent-primary)",
                boxShadow: "0 4px 14px var(--neu-glow)",
              }}
            >
              <Radar style={{ width: 17, height: 17, color: "white" }} strokeWidth={2} />
            </div>
            <div>
              <p style={{ fontSize: 13, color: "var(--neu-text)", fontWeight: 500 }}>实时附近</p>
              <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }}>
                {rangeLabel}
              </p>
            </div>
          </div>
          <ChevronRight style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 mt-4" style={{ gap: 1, background: "var(--neu-border)", borderRadius: 14 }}>
          <div style={{ overflow: "hidden", borderRadius: 13 }}>
            {[
              { label: "人附近", v: stats.nearby, color: "var(--neu-accent-primary)" },
              { label: "活动中", v: stats.active,  color: "var(--semantic-warning)" },
              { label: "在线",   v: stats.online, color: "var(--semantic-online)" },
            ].map(s => (
              <div key={s.label} className="flex flex-col items-center justify-center py-2.5"
                style={{ background: "var(--neu-bg)" }}>
                <span style={{ fontSize: 19, color: s.color, fontWeight: 600, lineHeight: 1 }}>{s.v}</span>
                <span style={{ fontSize: 10, color: "var(--neu-text-muted)", marginTop: 4 }}>{s.label}</span>
              </div>
            ))}
          </div>
        </div>
      </motion.div>
    </div>
  );
}

/* ── People radar strip — flat avatars on ambient ───────────────── */
function PeopleRadarStrip({ users, onSeeAll, onSelectUser }: { users: NearbyUser[]; onSeeAll: () => void; onSelectUser: (userId: number) => void }) {
  return (
    <div>
      <SectionHead title="附近的人 · 实时" action="查看全部" onAction={onSeeAll} />
      <div className="flex gap-4 overflow-x-auto no-scrollbar pb-1">
        {users.map(u => (
          <motion.button key={u.id} type="button" whileTap={{ scale: 0.94 }}
            onClick={() => onSelectUser(u.id)}
            aria-label={`查看${u.name}的个人主页`}
            className="flex flex-col items-center gap-2 flex-shrink-0 rounded-[14px] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]">
            <div className="relative">
              <div className="rounded-full overflow-hidden"
                style={{ width: 56, height: 56, border: "1.5px solid var(--neu-border)" }}>
                <img src={u.avatar} alt="" className="w-full h-full object-cover" />
              </div>
              {u.status === "在线" && (
                <span className="absolute bottom-0 right-0 rounded-full"
                  style={{ width: 11, height: 11, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }} />
              )}
            </div>
            <span style={{ fontSize: 11, color: "var(--neu-text)" }}>{u.name}</span>
            <span style={{ fontSize: 10, color: "var(--neu-accent-primary)", marginTop: -4 }}>{u.distance}</span>
          </motion.button>
        ))}
        <motion.button type="button" whileTap={{ scale: 0.94 }} onClick={onSeeAll}
          className="flex flex-col items-center gap-2 flex-shrink-0 rounded-[14px] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]">
          <div className="flex items-center justify-center rounded-full"
            style={{ width: 56, height: 56, border: "1.5px dashed var(--neu-border-pressed)" }}>
            <ChevronRight style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </div>
          <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>更多</span>
        </motion.button>
      </div>
    </div>
  );
}

/* ── Activity rows (flat list) ──────────────────────────────────── */
function ActivityRow({ item, onClick }: { item: Activity; onClick: () => void }) {
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  const Icon = item.icon;
  return (
    <motion.button
      type="button"
      whileTap={hover.tap as any}
      onClick={onClick}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      className="neu-row-hover flex w-full items-center gap-3 px-4 py-3 text-left outline-none"
      style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <div className="rounded-[10px] flex items-center justify-center flex-shrink-0"
        style={{ width: 34, height: 34, background: item.bg }}>
        <Icon style={{ width: 16, height: 16, color: item.color }} strokeWidth={1.75} />
      </div>
      <div className="flex-1 min-w-0">
        <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{item.title}</p>
        <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }}>{item.sub}</p>
      </div>
      <ChevronRight style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
    </motion.button>
  );
}

function ActivityList({ items, onClick }: { items: Activity[]; onClick: (id: number) => void }) {
  return (
    <div className="flex flex-col" style={{ gap: 1, background: "var(--neu-border)", borderRadius: 16, boxShadow: "0 0 0 1px var(--neu-border)" }}>
      <div style={{ overflow: "hidden", borderRadius: 15 }}>
        {items.map(a => <ActivityRow key={a.id} item={a} onClick={() => onClick(a.id)} />)}
      </div>
    </div>
  );
}

/* ── Group cards horizontal (flat) ──────────────────────────────── */
function GroupCarousel({ groups, onSeeAll }: { groups: NearbyGroup[]; onSeeAll: () => void }) {
  return (
    <div>
      <SectionHead title="附近群组" action="更多" onAction={onSeeAll} />
      <div className="flex gap-3 overflow-x-auto no-scrollbar pb-1">
        {groups.map(g => (
          <motion.div key={g.id} whileTap={{ scale: 0.98 }}
            className="flex-shrink-0 rounded-[16px] overflow-hidden cursor-pointer neu-surface-flat"
            style={{ width: 140 }}>
            <div className="relative" style={{ height: 72 }}>
              <img src={g.banner} alt={g.name} className="w-full h-full object-cover" />
              <div className="absolute inset-0" style={{ background: "linear-gradient(to top, rgba(0,0,0,0.5) 0%, transparent 55%)" }} />
              <div className="absolute top-1.5 right-1.5">
                {g.privacy === "private"
                  ? <Lock style={{ width: 10, height: 10, color: "rgba(255,255,255,0.85)" }} strokeWidth={2} />
                  : <Globe style={{ width: 10, height: 10, color: "rgba(255,255,255,0.85)" }} strokeWidth={2} />
                }
              </div>
              <div className="absolute bottom-1.5 left-2 flex items-center gap-0.5">
                <MapPin style={{ width: 9, height: 9, color: "rgba(255,255,255,0.9)" }} strokeWidth={2} />
                <span style={{ fontSize: 10, color: "rgba(255,255,255,0.95)" }}>{g.distance}</span>
              </div>
            </div>
            <div className="px-2.5 pt-2 pb-2.5">
              <p style={{ fontSize: 12, color: "var(--neu-text)", fontWeight: 500 }} className="truncate">{g.name}</p>
              <div className="flex items-center gap-1 mt-1">
                <Users style={{ width: 9, height: 9, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{g.members} 成员</span>
              </div>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  );
}

/* ── Compact nearby user rows (flat, joined list) ──────────────── */
function NearbyUserRow({ user, followed, onToggle }: { user: NearbyUser; followed: boolean; onToggle: () => void }) {
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  return (
    <motion.div
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      className="flex items-center gap-3 px-4 py-3 cursor-pointer"
      style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <div className="relative flex-shrink-0">
        <div className="rounded-full overflow-hidden" style={{ width: 38, height: 38 }}>
          <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
        </div>
        {user.status === "在线" && (
          <span className="absolute bottom-0 right-0 rounded-full"
            style={{ width: 9, height: 9, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }} />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{user.name}</p>
        <div className="flex items-center gap-1.5 mt-0.5">
          <span style={{ fontSize: 10, color: "var(--neu-accent-primary)" }}>#{user.tag}</span>
          <span style={{ fontSize: 10, color: "var(--neu-text-muted)", opacity: 0.5 }}>·</span>
          <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>{user.distance}</span>
        </div>
      </div>
      <motion.button
        whileTap={{ scale: 0.93 }}
        onClick={e => { e.stopPropagation(); onToggle(); }}
        className="px-3 py-1 rounded-full outline-none transition-all"
        style={followed
          ? { background: "transparent", color: "var(--neu-text-muted)", border: "1px solid var(--neu-border-pressed)", fontSize: 11 }
          : { background: "var(--neu-accent-primary)", color: "white", border: "1px solid transparent", fontSize: 11 }
        }>
        {followed ? "已关注" : "关注"}
      </motion.button>
    </motion.div>
  );
}

function NearbyUsersFlatList({
  users, followingIds, onToggle,
}: { users: NearbyUser[]; followingIds: Set<number>; onToggle: (id: number) => void }) {
  return (
    <div className="flex flex-col" style={{ gap: 1, background: "var(--neu-border)", borderRadius: 16, boxShadow: "0 0 0 1px var(--neu-border)" }}>
      <div style={{ overflow: "hidden", borderRadius: 15 }}>
        {users.map(u => (
          <NearbyUserRow key={u.id} user={u} followed={followingIds.has(u.id)} onToggle={() => onToggle(u.id)} />
        ))}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile/Tablet Home
══════════════════════════════════════════════════════════════════ */
function HomeMobile({ nearbyUsers, activities, nearbyGroups, heroStats, loading }: { nearbyUsers: NearbyUser[]; activities: Activity[]; nearbyGroups: NearbyGroup[]; heroStats: { nearby: string; active: string; online: string }; loading: boolean }) {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const [followingIds, setFollowingIds] = useState<Set<number>>(new Set());
  const toggle = (id: number) => setFollowingIds(p => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const onlineCount = nearbyUsers.filter(u => u.status === "在线").length;

  const isTablet = platform === "tablet";
  const shellStyle: React.CSSProperties = isTablet
    ? { maxWidth: 924, marginLeft: "auto", marginRight: "auto", width: "100%" }
    : {};
  const mainColClass = isTablet ? "mx-auto w-full lg:mx-0 min-w-0" : "w-full";
  const mainColStyle: React.CSSProperties = isTablet ? { maxWidth: 600 } : {};

  return (
    <>
      <div className="sticky top-0 z-10 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className={isTablet ? "flex gap-6 px-0" : "flex px-5"} style={shellStyle}>
          <div className={isTablet ? `${mainColClass} px-5 flex items-center justify-between` : "w-full flex items-center justify-between"}
            style={mainColStyle}>
            <div>
              <h1 style={{ fontSize: 22, color: "var(--neu-text)" }}>附近</h1>
              <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>发现周边有趣的人和事</p>
            </div>
             {onlineCount > 0 && <motion.div whileTap={{ scale: 0.94 }}
                className="flex items-center gap-1.5 py-1.5 px-3 rounded-full cursor-pointer neu-surface-flat"
                onClick={() => navigate(ROUTES.map)}>
                <span className="rounded-full" style={{ width: 6, height: 6, background: "var(--semantic-online)", flexShrink: 0 }} />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{onlineCount} 在线</span>
              </motion.div>}
          </div>
          {isTablet && <div className="hidden lg:block flex-shrink-0" style={{ width: "var(--context-rail-width)" }} />}
        </div>
      </div>

      <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className={isTablet ? "flex gap-6 pt-5 pb-8 px-0 mx-auto" : "flex flex-col px-5 pt-5 pb-8 gap-7"}
        style={shellStyle}>

        <div className={isTablet ? `${mainColClass} flex flex-col gap-7 px-5` : "flex flex-col gap-7 w-full"}
          style={mainColStyle}>
          <HeroStats onExplore={() => navigate(ROUTES.nearbyPeople)} stats={heroStats} />
          <PeopleRadarStrip users={nearbyUsers} onSeeAll={() => navigate(ROUTES.nearbyPeople)} onSelectUser={(userId) => navigate(ROUTES.user(userId))} />
          <div>
            <SectionHead title="附近话题" action="更多" onAction={() => navigate(ROUTES.nearbyMoments)} />
            <ActivityList items={activities} onClick={() => navigate(ROUTES.nearbyMoments)} />
          </div>
          <GroupCarousel groups={nearbyGroups} onSeeAll={() => navigate(ROUTES.nearbyGroups)} />
          <div>
            <SectionHead title="人物推荐" action="查看全部" onAction={() => navigate(ROUTES.nearbyPeople)} />
            <NearbyUsersFlatList users={nearbyUsers.slice(0, 4)} followingIds={followingIds} onToggle={toggle} />
          </div>
        </div>
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Home — 2-column with clear hierarchy
══════════════════════════════════════════════════════════════════ */
function HomeDesktop({ nearbyUsers, activities, nearbyGroups, heroStats, loading }: { nearbyUsers: NearbyUser[]; activities: Activity[]; nearbyGroups: NearbyGroup[]; heroStats: { nearby: string; active: string; online: string }; loading: boolean }) {
  const navigate = useNavigate();
  const [followingIds, setFollowingIds] = useState<Set<number>>(new Set());
  const toggle = (id: number) => setFollowingIds(p => { const n = new Set(p); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const onlineCount = nearbyUsers.filter(u => u.status === "在线").length;

  return (
    <div className="content-mode-wide flex flex-col h-full">
      <div className="flex-shrink-0 flex items-center justify-between px-7 py-4"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div>
          <h1 style={{ fontSize: 24, color: "var(--neu-text)" }}>首页</h1>
          <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>发现周边有趣的人和事</p>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full neu-surface-flat">
            <span className="rounded-full" style={{ width: 6, height: 6, background: "var(--semantic-online)" }} />
            <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{onlineCount} 在线</span>
          </div>
        </div>
      </div>

      <div className="flex flex-1 min-h-0 gap-0 overflow-hidden">

        {/* ── Left: hero + activity + groups (320px) ── */}
        <div className="flex flex-col overflow-y-auto no-scrollbar p-6 gap-6"
          style={{ width: 340, flexShrink: 0, borderRight: "1px solid var(--neu-border)" }}>

          <HeroStats onExplore={() => navigate(ROUTES.nearbyPeople)} stats={heroStats} />

          <div>
            <SectionHead title="附近话题" action="更多" onAction={() => navigate(ROUTES.nearbyMoments)} />
            <ActivityList items={activities} onClick={() => navigate(ROUTES.nearbyMoments)} />
          </div>

          <GroupCarousel groups={nearbyGroups} onSeeAll={() => navigate(ROUTES.nearbyGroups)} />
        </div>

        {/* ── Right: people radar + full list ── */}
        <div className="flex-1 overflow-y-auto no-scrollbar p-6">
          <div className="flex items-end justify-between mb-5">
            <div>
              <span className="neu-eyebrow">附近的人</span>
              <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 4 }}>
                5km 范围内 · 实时更新
              </p>
            </div>
            <motion.button whileTap={{ scale: 0.93 }} onClick={() => navigate(ROUTES.nearbyPeople)}
              className="flex items-center gap-0.5 outline-none neu-surface-ghost rounded-full px-2 py-0.5"
              style={{ color: "var(--neu-accent-primary)" }}>
              <span style={{ fontSize: 12 }}>查看全部</span>
              <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={1.5} />
            </motion.button>
          </div>

          <PeopleRadarStrip users={nearbyUsers} onSeeAll={() => navigate(ROUTES.nearbyPeople)} onSelectUser={(userId) => navigate(ROUTES.user(userId))} />

          <div className="mt-6">
            <NearbyUsersFlatList users={nearbyUsers} followingIds={followingIds} onToggle={toggle} />
          </div>
        </div>
      </div>
    </div>
  );
}

export function NearbyFeedPage() {
  const { platform } = usePlatform();
  const { reference_point } = useDefaultLocation();
  const [nearbyUsers, setNearbyUsers] = useState<NearbyUser[]>([]);
  const [activities, setActivities] = useState<Activity[]>([]);
  const [nearbyGroups, setNearbyGroups] = useState<NearbyGroup[]>([]);
  const [heroStats, setHeroStats] = useState({ nearby: "0", active: "0", online: "0" });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // spec: moments/search requires reference_point + ≥1 of keyword/tag/radius.
    // reference_point comes from ClientConfig.default_location (location denied fallback).
    if (!reference_point) return;
    let cancelled = false;
    async function fetchData() {
      try {
        setLoading(true);
        setError(null);
        const [trendingRes, nearbyRes, configRes] = await Promise.all([
          getTrending().catch(() => ({ code: -1, data: null })),
          // moments/search is auth-optional, so anonymous users still see nearby public moments.
          searchMoments({ reference_point, radius: 5000, sort_by: "distance", pagination: { limit: 12 } }).catch(() => ({ code: -1, data: { items: [] } })),
          getClientConfig().catch(() => ({ code: -1, data: null })),
        ]);
        if (cancelled) return;

        const trendingData = (trendingRes.code === 0 ? trendingRes.data : null) as any;
        const nearbyData = (nearbyRes.code === 0 ? nearbyRes.data : null) as any;
        const configData = (configRes.code === 0 ? configRes.data : null) as any;

        if (trendingData) {
          const moments = trendingData.hot_moments || trendingData.activities || trendingData.items || [];
          const acts: Activity[] = moments.map((a: any, i: number) => ({
            id: a.id ?? a.moment_id ?? i + 1,
            icon: [Coffee, Music, Users, Star][i % 4],
            title: a.title ?? a.description ?? a.name ?? "",
            sub: a.sub ?? a.location?.location_name ?? a.subtitle ?? "",
            color: ["var(--semantic-warning)", "var(--neu-accent-primary)", "var(--map-group)", "var(--semantic-hot)"][i % 4],
            bg: [`color-mix(in srgb, var(--semantic-warning) 10%, transparent)`, "var(--neu-accent-soft)", `color-mix(in srgb, var(--map-group) 10%, transparent)`, `color-mix(in srgb, var(--semantic-hot) 10%, transparent)`][i % 4],
          }));
          setActivities(acts);

          // Groups from trending (legacy fallback: groups come from separate group search now)
          if (trendingData.groups) {
            const groups: NearbyGroup[] = trendingData.groups.map((g: any, i: number) => ({
              id: g.id ?? i + 1, name: g.name ?? "", members: g.members ?? g.member_count ?? 0,
              banner: g.banner ?? g.avatar ?? "", distance: g.distance ?? "", privacy: g.privacy ?? "public",
            }));
            setNearbyGroups(groups);
          }
        }

        if (nearbyData) {
          const users: NearbyUser[] = (nearbyData.items || []).map((u: any, i: number) => ({
            id: u.id ?? u.user_id ?? u.moment_id ?? i + 1,
            name: u.name ?? u.nickname ?? u.user?.nickname ?? "",
            distance: u.distance ?? u.location?.distance ?? "",
            avatar: u.avatar ?? u.user?.avatar_url ?? "",
            status: u.online ? "在线" : "离线",
            tag: u.tag ?? u.tags?.[0] ?? "",
          }));
          setNearbyUsers(users);
        }

        if (configData) {
          setHeroStats({
            nearby: String(configData.nearby_count ?? configData.nearby ?? 0),
            active: String(configData.active_count ?? configData.active ?? 0),
            online: String(configData.online_count ?? configData.online ?? 0),
          });
        }
      } catch (e: any) {
        if (!cancelled) setError(e.message ?? "加载失败");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchData();
    return () => { cancelled = true; };
  }, [reference_point]);

  if (platform === "desktop") return <HomeDesktop nearbyUsers={nearbyUsers} activities={activities} nearbyGroups={nearbyGroups} heroStats={heroStats} loading={loading} />;
  return <HomeMobile nearbyUsers={nearbyUsers} activities={activities} nearbyGroups={nearbyGroups} heroStats={heroStats} loading={loading} />;
}
