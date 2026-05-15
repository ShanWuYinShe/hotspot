import React, { useEffect, useState } from "react";
import {
  MapPin,
  ChevronRight,
  Users,
  Coffee,
  Music,
  Star,
  Flame,
  Lock,
  Globe,
  Radar,
} from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { useNavigate } from "react-router";
import { useHoverConfig } from "../hooks/useHoverConfig";
import {
  TabletRail,
  NearbyPulseCard,
  StreakCard,
  DailyPickCard,
  SuggestedFollowsCard,
  VibeCard,
} from "../components/TabletRail";
import {
  feedService,
  type User,
  type Group,
} from "@/lib/api/services";
import { api } from "@/lib/api/client";
import { useAuth } from "../contexts/AuthContext";

/* ── UI display types ────────────────────────────────────────────── */
interface HomeNearbyUser {
  id: number;
  name: string;
  avatar: string;
  distance: string;
  status: string;
  tag: string;
}

interface HomeNearbyGroup {
  id: number;
  name: string;
  members: number;
  banner: string;
  distance: string;
  privacy: "public" | "private";
}

function formatDistance(m?: number): string {
  if (m == null) return "未知";
  if (m < 1000) return `${Math.round(m)}m`;
  return `${(m / 1000).toFixed(1)}km`;
}

function mapUser(u: User): HomeNearbyUser {
  return {
    id: u.id,
    name: u.nickname,
    avatar: u.avatar_url || "",
    distance: formatDistance(u.distance),
    status: u.presence_status === "online" ? "Online" : "Offline",
    tag: u.tags?.[0] ?? "",
  };
}

function mapGroup(g: Group): HomeNearbyGroup {
  return {
    id: g.id,
    name: g.name,
    members: g.member_count,
    banner: g.banner_url || "",
    distance: formatDistance(g.distance),
    privacy: g.privacy,
  };
}

/* ── Themed decoration (UI chrome, not real data) ────────────────── */
const ACTIVITIES = [
  {
    id: 1,
    icon: Coffee,
    title: "晨间咖啡会",
    sub: "12人 · 0.3km",
    color: "#F59E0B",
    bg: "rgba(245,158,11,0.10)",
  },
  {
    id: 2,
    icon: Music,
    title: "街头音乐节",
    sub: "38人 · 0.7km",
    color: "#8B5CF6",
    bg: "rgba(139,92,246,0.10)",
  },
  {
    id: 3,
    icon: Users,
    title: "读书小组",
    sub: "9人 · 1.1km",
    color: "#06B6D4",
    bg: "rgba(6,182,212,0.10)",
  },
  {
    id: 4,
    icon: Star,
    title: "摄影走街",
    sub: "21人 · 1.5km",
    color: "#EC4899",
    bg: "rgba(236,72,153,0.10)",
  },
];

/* ── Section heading ────────────────────────────────────────────── */
function SectionHead({
  title,
  action,
  onAction,
}: {
  title: string;
  action?: string;
  onAction?: () => void;
}) {
  return (
    <div className="flex items-end justify-between mb-3">
      <span className="neu-eyebrow">{title}</span>
      {action && onAction && (
        <motion.button
          whileTap={{ scale: 0.94 }}
          onClick={onAction}
          className="flex items-center gap-0.5 outline-none neu-surface-ghost rounded-full px-2 py-0.5"
          style={{ color: "var(--neu-accent-primary)" }}
        >
          <span style={{ fontSize: 11 }}>{action}</span>
          <ChevronRight style={{ width: 12, height: 12 }} strokeWidth={1.5} />
        </motion.button>
      )}
    </div>
  );
}

/* ── Hero stats banner — single focal point with ambient glow ───── */
function HeroStats({ onExplore }: { onExplore: () => void }) {
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
                width: 38,
                height: 38,
                background: "var(--neu-accent-primary)",
                boxShadow: "0 4px 14px var(--neu-glow)",
              }}
            >
              <Radar
                style={{ width: 17, height: 17, color: "white" }}
                strokeWidth={2}
              />
            </div>
            <div>
              <p
                style={{
                  fontSize: 13,
                  color: "var(--neu-text)",
                  fontWeight: 500,
                }}
              >
                实时附近
              </p>
              <p
                style={{
                  fontSize: 11,
                  color: "var(--neu-text-muted)",
                  marginTop: 1,
                }}
              >
                徐州 · 5km 内 · 持续更新
              </p>
            </div>
          </div>
          <ChevronRight
            style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </div>

        {/* Stats row */}
        <div
          className="grid grid-cols-3 mt-4"
          style={{
            gap: 1,
            background: "var(--neu-border)",
            borderRadius: 14,
            overflow: "hidden",
          }}
        >
          {[
            { label: "人附近", v: "23", color: "var(--neu-accent-primary)" },
            { label: "活动中", v: "6", color: "#F59E0B" },
            { label: "在线", v: "14", color: "#22c55e" },
          ].map((s) => (
            <div
              key={s.label}
              className="flex flex-col items-center justify-center py-2.5"
              style={{ background: "var(--neu-bg)" }}
            >
              <span
                style={{
                  fontSize: 19,
                  color: s.color,
                  fontWeight: 600,
                  lineHeight: 1,
                }}
              >
                {s.v}
              </span>
              <span
                style={{
                  fontSize: 10,
                  color: "var(--neu-text-muted)",
                  marginTop: 4,
                }}
              >
                {s.label}
              </span>
            </div>
          ))}
        </div>
      </motion.div>
    </div>
  );
}

/* ── People radar strip — flat avatars on ambient ───────────────── */
function PeopleRadarStrip({
  users,
  loading,
  onSeeAll,
}: {
  users: HomeNearbyUser[];
  loading?: boolean;
  onSeeAll: () => void;
}) {
  return (
    <div>
      <SectionHead
        title="附近的人 · 实时"
        action="查看全部"
        onAction={onSeeAll}
      />
      <div className="flex gap-4 overflow-x-auto no-scrollbar pb-1">
        {loading
          ? Array.from({ length: 5 }).map((_, i) => (
              <div
                key={i}
                className="flex flex-col items-center gap-2 flex-shrink-0"
              >
                <div
                  className="rounded-full neu-skeleton-pulse"
                  style={{ width: 56, height: 56 }}
                />
                <div
                  className="rounded neu-skeleton-pulse"
                  style={{ width: 40, height: 10 }}
                />
                <div
                  className="rounded neu-skeleton-pulse"
                  style={{ width: 32, height: 9 }}
                />
              </div>
            ))
          : users.map((u) => (
              <motion.div
                key={u.id}
                whileTap={{ scale: 0.94 }}
                className="flex flex-col items-center gap-2 flex-shrink-0 cursor-pointer"
              >
                <div className="relative">
                  <div
                    className="rounded-full overflow-hidden"
                    style={{
                      width: 56,
                      height: 56,
                      border: "1.5px solid var(--neu-border)",
                    }}
                  >
                    <img
                      src={u.avatar}
                      alt={u.name}
                      className="w-full h-full object-cover"
                    />
                  </div>
                  {u.status === "Online" && (
                    <span
                      className="absolute bottom-0 right-0 rounded-full"
                      style={{
                        width: 11,
                        height: 11,
                        background: "#22c55e",
                        border: "2px solid var(--neu-bg)",
                      }}
                    />
                  )}
                </div>
                <span style={{ fontSize: 11, color: "var(--neu-text)" }}>
                  {u.name}
                </span>
                <span
                  style={{
                    fontSize: 10,
                    color: "var(--neu-accent-primary)",
                    marginTop: -4,
                  }}
                >
                  {u.distance}
                </span>
              </motion.div>
            ))}
        <motion.div
          whileTap={{ scale: 0.94 }}
          onClick={onSeeAll}
          className="flex flex-col items-center gap-2 flex-shrink-0 cursor-pointer"
        >
          <div
            className="flex items-center justify-center rounded-full"
            style={{
              width: 56,
              height: 56,
              border: "1.5px dashed var(--neu-border-pressed)",
            }}
          >
            <ChevronRight
              style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          </div>
          <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
            更多
          </span>
        </motion.div>
      </div>
    </div>
  );
}

/* ── Activity rows (flat list) ──────────────────────────────────── */
function ActivityList({
  items,
  onClick,
}: {
  items: typeof ACTIVITIES;
  onClick: (id: number) => void;
}) {
  const hover = useHoverConfig();
  return (
    <div
      className="flex flex-col"
      style={{
        gap: 1,
        background: "var(--neu-border)",
        borderRadius: 16,
        overflow: "hidden",
        border: "1px solid var(--neu-border)",
      }}
    >
      {items.map((a) => {
        const Icon = a.icon;
        return (
          <motion.div
            key={a.id}
            whileTap={hover.tap as any}
            onClick={() => onClick(a.id)}
            className="flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors"
            style={{ background: "var(--neu-bg)" }}
            onMouseEnter={(e) =>
              (e.currentTarget.style.background = "var(--neu-border)")
            }
            onMouseLeave={(e) =>
              (e.currentTarget.style.background = "var(--neu-bg)")
            }
          >
            <div
              className="rounded-[10px] flex items-center justify-center flex-shrink-0"
              style={{ width: 34, height: 34, background: a.bg }}
            >
              <Icon
                style={{ width: 16, height: 16, color: a.color }}
                strokeWidth={1.75}
              />
            </div>
            <div className="flex-1 min-w-0">
              <p style={{ fontSize: 13, color: "var(--neu-text)" }}>
                {a.title}
              </p>
              <p
                style={{
                  fontSize: 11,
                  color: "var(--neu-text-muted)",
                  marginTop: 1,
                }}
              >
                {a.sub}
              </p>
            </div>
            <ChevronRight
              style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          </motion.div>
        );
      })}
    </div>
  );
}

/* ── Group cards horizontal (flat) ──────────────────────────────── */
function GroupCarousel({
  groups,
  loading,
  onSeeAll,
}: {
  groups: HomeNearbyGroup[];
  loading?: boolean;
  onSeeAll: () => void;
}) {
  return (
    <div>
      <SectionHead title="附近群组" action="更多" onAction={onSeeAll} />
      <div className="flex gap-3 overflow-x-auto no-scrollbar pb-1">
        {loading
          ? Array.from({ length: 3 }).map((_, i) => (
              <div
                key={i}
                className="flex-shrink-0 rounded-[16px] overflow-hidden neu-surface-flat"
                style={{ width: 140 }}
              >
                <div className="neu-skeleton-pulse" style={{ height: 72 }} />
                <div className="px-2.5 pt-2 pb-2.5">
                  <div
                    className="rounded neu-skeleton-pulse"
                    style={{ width: 80, height: 12 }}
                  />
                  <div
                    className="rounded neu-skeleton-pulse mt-1.5"
                    style={{ width: 50, height: 10 }}
                  />
                </div>
              </div>
            ))
          : groups.map((g) => (
              <motion.div
                key={g.id}
                whileTap={{ scale: 0.98 }}
                className="flex-shrink-0 rounded-[16px] overflow-hidden cursor-pointer neu-surface-flat"
                style={{ width: 140 }}
              >
                <div className="relative" style={{ height: 72 }}>
                  <img
                    src={g.banner}
                    alt={g.name}
                    className="w-full h-full object-cover"
                  />
                  <div
                    className="absolute inset-0"
                    style={{
                      background:
                        "linear-gradient(to top, rgba(0,0,0,0.5) 0%, transparent 55%)",
                    }}
                  />
                  <div className="absolute top-1.5 right-1.5">
                    {g.privacy === "private" ? (
                      <Lock
                        style={{
                          width: 10,
                          height: 10,
                          color: "rgba(255,255,255,0.85)",
                        }}
                        strokeWidth={2}
                      />
                    ) : (
                      <Globe
                        style={{
                          width: 10,
                          height: 10,
                          color: "rgba(255,255,255,0.85)",
                        }}
                        strokeWidth={2}
                      />
                    )}
                  </div>
                  <div className="absolute bottom-1.5 left-2 flex items-center gap-0.5">
                    <MapPin
                      style={{
                        width: 9,
                        height: 9,
                        color: "rgba(255,255,255,0.9)",
                      }}
                      strokeWidth={2}
                    />
                    <span
                      style={{ fontSize: 10, color: "rgba(255,255,255,0.95)" }}
                    >
                      {g.distance}
                    </span>
                  </div>
                </div>
                <div className="px-2.5 pt-2 pb-2.5">
                  <p
                    style={{
                      fontSize: 12,
                      color: "var(--neu-text)",
                      fontWeight: 500,
                    }}
                    className="truncate"
                  >
                    {g.name}
                  </p>
                  <div className="flex items-center gap-1 mt-1">
                    <Users
                      style={{
                        width: 9,
                        height: 9,
                        color: "var(--neu-text-muted)",
                      }}
                      strokeWidth={1.5}
                    />
                    <span
                      style={{ fontSize: 10, color: "var(--neu-text-muted)" }}
                    >
                      {g.members} 成员
                    </span>
                  </div>
                </div>
              </motion.div>
            ))}
      </div>
    </div>
  );
}

/* ── Compact nearby user rows (flat, joined list) ──────────────── */
function NearbyUsersFlatList({
  users,
  followingIds,
  onToggle,
}: {
  users: HomeNearbyUser[];
  followingIds: Set<number>;
  onToggle: (id: number) => void;
}) {
  return (
    <div
      className="flex flex-col"
      style={{
        gap: 1,
        background: "var(--neu-border)",
        borderRadius: 16,
        overflow: "hidden",
        border: "1px solid var(--neu-border)",
      }}
    >
      {users.map((u) => {
        const followed = followingIds.has(u.id);
        return (
          <div
            key={u.id}
            className="flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors"
            style={{ background: "var(--neu-bg)" }}
            onMouseEnter={(e) =>
              (e.currentTarget.style.background = "var(--neu-border)")
            }
            onMouseLeave={(e) =>
              (e.currentTarget.style.background = "var(--neu-bg)")
            }
          >
            <div className="relative flex-shrink-0">
              <div
                className="rounded-full overflow-hidden"
                style={{ width: 38, height: 38 }}
              >
                <img
                  src={u.avatar}
                  alt={u.name}
                  className="w-full h-full object-cover"
                />
              </div>
              {u.status === "Online" && (
                <span
                  className="absolute bottom-0 right-0 rounded-full"
                  style={{
                    width: 9,
                    height: 9,
                    background: "#22c55e",
                    border: "2px solid var(--neu-bg)",
                  }}
                />
              )}
            </div>
            <div className="flex-1 min-w-0">
              <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{u.name}</p>
              <div className="flex items-center gap-1.5 mt-0.5">
                <span
                  style={{ fontSize: 10, color: "var(--neu-accent-primary)" }}
                >
                  #{u.tag}
                </span>
                <span
                  style={{
                    fontSize: 10,
                    color: "var(--neu-text-muted)",
                    opacity: 0.5,
                  }}
                >
                  ·
                </span>
                <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
                  {u.distance}
                </span>
              </div>
            </div>
            <motion.button
              whileTap={{ scale: 0.93 }}
              onClick={(e) => {
                e.stopPropagation();
                onToggle(u.id);
              }}
              className="px-3 py-1 rounded-full outline-none transition-all"
              style={
                followed
                  ? {
                      background: "transparent",
                      color: "var(--neu-text-muted)",
                      border: "1px solid var(--neu-border-pressed)",
                      fontSize: 11,
                    }
                  : {
                      background: "var(--neu-accent-primary)",
                      color: "white",
                      border: "1px solid transparent",
                      fontSize: 11,
                    }
              }
            >
              {followed ? "已关注" : "关注"}
            </motion.button>
          </div>
        );
      })}
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile/Tablet Home
══════════════════════════════════════════════════════════════════ */
function HomeMobile() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const { user } = useAuth();
  const [followingIds, setFollowingIds] = useState<Set<number>>(new Set());
  const toggle = (id: number) =>
    setFollowingIds((p) => {
      const n = new Set(p);
      n.has(id) ? n.delete(id) : n.add(id);
      return n;
    });

  const [users, setUsers] = useState<HomeNearbyUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  const [groups, setGroups] = useState<HomeNearbyGroup[]>([]);
  const [groupsLoading, setGroupsLoading] = useState(true);

  useEffect(() => {
    feedService.listUsers().then((res) => {
      if (api.isSuccess(res) && res.data?.items) {
        setUsers(res.data.items.map(mapUser));
      }
      setUsersLoading(false);
    });
    feedService.listGroups().then((res) => {
      if (api.isSuccess(res) && res.data?.items) {
        setGroups(res.data.items.map(mapGroup));
      }
      setGroupsLoading(false);
    });
  }, []);

  const isTablet = platform === "tablet";
  const shellStyle: React.CSSProperties = isTablet
    ? { maxWidth: 924, marginLeft: "auto", marginRight: "auto", width: "100%" }
    : {};
  const mainColClass = isTablet ? "mx-auto w-full lg:mx-0 min-w-0" : "w-full";
  const mainColStyle: React.CSSProperties = isTablet ? { maxWidth: 600 } : {};

  return (
    <>
      <div
        className="sticky top-0 z-10 py-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div
          className={isTablet ? "flex gap-6 px-0" : "flex px-5"}
          style={shellStyle}
        >
          <div
            className={
              isTablet
                ? `${mainColClass} px-5 flex items-center justify-between`
                : "w-full flex items-center justify-between"
            }
            style={mainColStyle}
          >
            <div>
              <h1 style={{ fontSize: 22, color: "var(--neu-text)" }}>附近</h1>
              <p
                style={{
                  fontSize: 12,
                  color: "var(--neu-text-muted)",
                  marginTop: 2,
                }}
              >
                发现周边有趣的人和事
              </p>
            </div>
            <motion.div
              whileTap={{ scale: 0.94 }}
              className="flex items-center gap-1.5 py-1.5 px-3 rounded-full cursor-pointer neu-surface-flat"
              onClick={() => navigate("/location")}
            >
              <span
                className="rounded-full"
                style={{
                  width: 6,
                  height: 6,
                  background: "#22c55e",
                  flexShrink: 0,
                }}
              />
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                {user?.nickname ?? "徐州"}
              </span>
            </motion.div>
          </div>
          {isTablet && (
            <div
              className="hidden lg:block flex-shrink-0"
              style={{ width: 300 + 20 }}
            />
          )}
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className={
          isTablet
            ? "flex gap-6 pt-5 pb-8 px-0 mx-auto"
            : "flex flex-col px-5 pt-5 pb-8 gap-7"
        }
        style={shellStyle}
      >
        <div
          className={
            isTablet
              ? `${mainColClass} flex flex-col gap-7 px-5`
              : "flex flex-col gap-7 w-full"
          }
          style={mainColStyle}
        >
          <HeroStats onExplore={() => navigate("/nearby")} />
          <PeopleRadarStrip
            users={users}
            loading={usersLoading}
            onSeeAll={() => navigate("/nearby")}
          />
          <div>
            <SectionHead title="附近活动" action="更多" onAction={() => {}} />
            <ActivityList
              items={ACTIVITIES}
              onClick={(id) => navigate(`/events/${id}`)}
            />
          </div>
          <GroupCarousel
            groups={groups}
            loading={groupsLoading}
            onSeeAll={() => navigate("/nearby-groups")}
          />
          <div>
            <SectionHead
              title="人物推荐"
              action="查看全部"
              onAction={() => navigate("/nearby")}
            />
            <NearbyUsersFlatList
              users={users.slice(0, 4)}
              followingIds={followingIds}
              onToggle={toggle}
            />
          </div>
        </div>

        {isTablet && (
          <aside
            className="hidden lg:block flex-shrink-0 pr-5"
            style={{ width: 300 + 20 }}
          >
            <div style={{ width: 300 }}>
              <TabletRail>
                <NearbyPulseCard />
                <StreakCard />
                <DailyPickCard />
                <SuggestedFollowsCard />
                <VibeCard />
              </TabletRail>
            </div>
          </aside>
        )}
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Home — 2-column with clear hierarchy
══════════════════════════════════════════════════════════════════ */
function HomeDesktop() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const [followingIds, setFollowingIds] = useState<Set<number>>(new Set());
  const toggle = (id: number) =>
    setFollowingIds((p) => {
      const n = new Set(p);
      n.has(id) ? n.delete(id) : n.add(id);
      return n;
    });

  const [users, setUsers] = useState<HomeNearbyUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(true);
  const [groups, setGroups] = useState<HomeNearbyGroup[]>([]);
  const [groupsLoading, setGroupsLoading] = useState(true);

  useEffect(() => {
    feedService.listUsers().then((res) => {
      if (api.isSuccess(res) && res.data?.items) {
        setUsers(res.data.items.map(mapUser));
      }
      setUsersLoading(false);
    });
    feedService.listGroups().then((res) => {
      if (api.isSuccess(res) && res.data?.items) {
        setGroups(res.data.items.map(mapGroup));
      }
      setGroupsLoading(false);
    });
  }, []);

  const onlineCount = users.filter((u) => u.status === "Online").length;

  return (
    <>
      <div
        className="sticky top-0 z-10 flex items-center justify-between px-7 py-4"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div>
          <h1 style={{ fontSize: 24, color: "var(--neu-text)" }}>首页</h1>
          <p
            style={{
              fontSize: 12,
              color: "var(--neu-text-muted)",
              marginTop: 2,
            }}
          >
            发现周边有趣的人和事
          </p>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full neu-surface-flat">
            <span
              className="rounded-full"
              style={{ width: 6, height: 6, background: "#22c55e" }}
            />
            <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
              {onlineCount} 在线
            </span>
          </div>
        </div>
      </div>

      <div
        className="flex gap-0 overflow-hidden"
        style={{ height: "calc(100% - 65px)" }}
      >
        {/* ── Left: hero + activity + groups (320px) ── */}
        <div
          className="flex flex-col overflow-y-auto no-scrollbar p-6 gap-6"
          style={{
            width: 340,
            flexShrink: 0,
            borderRight: "1px solid var(--neu-border)",
          }}
        >
          <HeroStats onExplore={() => navigate("/nearby")} />

          <div>
            <SectionHead title="附近活动" action="更多" onAction={() => {}} />
            <ActivityList
              items={ACTIVITIES}
              onClick={(id) => navigate(`/events/${id}`)}
            />
          </div>

          <GroupCarousel
            groups={groups}
            loading={groupsLoading}
            onSeeAll={() => navigate("/nearby-groups")}
          />
        </div>

        {/* ── Right: people radar + full list ── */}
        <div className="flex-1 overflow-y-auto no-scrollbar p-6">
          <div className="flex items-end justify-between mb-5">
            <div>
              <span className="neu-eyebrow">附近的人</span>
              <p
                style={{
                  fontSize: 12,
                  color: "var(--neu-text-muted)",
                  marginTop: 4,
                }}
              >
                5km 范围内 · 实时更新
              </p>
            </div>
            <motion.button
              whileTap={{ scale: 0.93 }}
              onClick={() => navigate("/nearby")}
              className="flex items-center gap-0.5 outline-none neu-surface-ghost rounded-full px-2 py-0.5"
              style={{ color: "var(--neu-accent-primary)" }}
            >
              <span style={{ fontSize: 12 }}>查看全部</span>
              <ChevronRight
                style={{ width: 13, height: 13 }}
                strokeWidth={1.5}
              />
            </motion.button>
          </div>

          <PeopleRadarStrip
            users={users}
            loading={usersLoading}
            onSeeAll={() => navigate("/nearby")}
          />

          <div className="mt-6">
            <NearbyUsersFlatList
              users={users}
              followingIds={followingIds}
              onToggle={toggle}
            />
          </div>

          <div className="mt-7">
            <SectionHead title="周边热度" />
            <ActivityList
              items={ACTIVITIES}
              onClick={(id) => navigate(`/events/${id}`)}
            />
          </div>
        </div>
      </div>
    </>
  );
}

export function Home() {
  const { platform } = usePlatform();
  if (platform === "desktop") return <HomeDesktop />;
  return <HomeMobile />;
}
