import React from "react";
import { MapPin, ChevronRight, Users, Lock, Globe, Radar } from "lucide-react";
import { motion } from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";
import { useNavigate } from "react-router";
import {
  TabletRail,
  NearbyPulseCard,
  StreakCard,
  DailyPickCard,
  SuggestedFollowsCard,
  VibeCard,
} from "../components/TabletRail";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { useHomeData } from "@/lib/hooks/use-home-data";
import { useFollowToggle } from "@/lib/hooks/use-follow-toggle";
import type { UserItem, GroupItem } from "@/lib/types/api";

import { formatDistance } from "@/lib/utils";

/* ── Home skeleton ────────────────────────────────────────────── */
function HomeSkeleton() {
  return (
    <div className="flex flex-col gap-7 animate-pulse">
      <div
        className="rounded-[24px]"
        style={{ height: 90, background: "var(--neu-shadow-light)" }}
      />
      <div>
        <div
          className="rounded"
          style={{
            height: 12,
            width: 100,
            background: "var(--neu-shadow-light)",
            marginBottom: 12,
          }}
        />
        <div className="flex gap-4">
          {[0, 1, 2, 3, 4].map((i) => (
            <div key={i} className="flex flex-col items-center gap-2">
              <div
                className="rounded-full"
                style={{
                  width: 56,
                  height: 56,
                  background: "var(--neu-shadow-light)",
                }}
              />
              <div
                className="rounded"
                style={{
                  height: 10,
                  width: 36,
                  background: "var(--neu-shadow-light)",
                }}
              />
            </div>
          ))}
        </div>
      </div>
      <div>
        <div
          className="rounded"
          style={{
            height: 12,
            width: 80,
            background: "var(--neu-shadow-light)",
            marginBottom: 12,
          }}
        />
        <div className="flex gap-3">
          {[0, 1, 2].map((i) => (
            <div
              key={i}
              className="rounded-[16px] flex-shrink-0"
              style={{
                width: 140,
                height: 120,
                background: "var(--neu-shadow-light)",
              }}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

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

/* ── Hero stats banner ───── */
function HeroStats({
  onExplore,
  usersCount,
  groupsCount,
  onlineCount,
}: {
  onExplore: () => void;
  usersCount: number;
  groupsCount: number;
  onlineCount: number;
}) {
  return (
    <div className="neu-ambient-glow">
      <motion.div
        whileTap={{ scale: 0.98 }}
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
                style={{
                  width: 17,
                  height: 17,
                  color: "var(--text-on-accent, white)",
                }}
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
                5km 内 · 持续更新
              </p>
            </div>
          </div>
          <ChevronRight
            style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </div>

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
            {
              label: "人附近",
              v: String(usersCount),
              color: "var(--neu-accent-primary)",
            },
            { label: "群组", v: String(groupsCount), color: "var(--neu-warn)" },
            { label: "在线", v: String(onlineCount), color: "var(--neu-success)" },
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

/* ── People radar strip ───────────────── */
function PeopleRadarStrip({
  users,
  onSeeAll,
  onUserClick,
}: {
  users: UserItem[];
  onSeeAll: () => void;
  onUserClick: (userId: string) => void;
}) {
  return (
    <div>
      <SectionHead
        title="附近的人 · 实时"
        action="查看全部"
        onAction={onSeeAll}
      />
      <div className="flex gap-4 overflow-x-auto no-scrollbar pb-1">
        {users.slice(0, 8).map((u) => (
          <motion.div
            key={u.user.user_id}
            whileTap={{ scale: 0.98 }}
            className="flex flex-col items-center gap-2 flex-shrink-0 cursor-pointer"
            onClick={() => onUserClick(u.user.user_id)}
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
                  src={u.user.avatar_url || ""}
                  alt={u.user.nickname}
                  className="w-full h-full object-cover bg-[var(--neu-border)]"
                />
              </div>
              {u.presence_status === "online" && (
                <span
                  className="absolute bottom-0 right-0 rounded-full"
                  style={{
                    width: 11,
                    height: 11,
                    background: "var(--neu-success)",
                    border: "2px solid var(--neu-bg)",
                  }}
                />
              )}
            </div>
            <span
              className="truncate max-w-[64px] text-center"
              style={{ fontSize: 11, color: "var(--neu-text)" }}
            >
              {u.user.nickname}
            </span>
            <span
              style={{
                fontSize: 10,
                color: "var(--neu-accent-primary)",
                marginTop: -4,
              }}
            >
              {formatDistance(u.location?.distance)}
            </span>
          </motion.div>
        ))}
        <motion.div
          whileTap={{ scale: 0.94 }}
          onClick={onSeeAll}
          role="button"
          tabIndex={0}
          aria-label="查看全部"
          onKeyDown={(e) => e.key === "Enter" && onSeeAll()}
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

/* ── Trending tags ──────────────────────────────── */
function TrendingTags({
  tags,
}: {
  tags: { name: string; usage_count: number; trend?: string }[];
}) {
  if (!tags.length) return null;
  return (
    <div>
      <SectionHead title="热门标签" />
      <div className="flex flex-wrap gap-2">
        {tags.slice(0, 10).map((tag) => (
          <span
            key={tag.name}
            className="px-2.5 py-1 rounded-full"
            style={{
              fontSize: 11,
              background: "var(--neu-border)",
              color: "var(--neu-text)",
            }}
          >
            #{tag.name}
          </span>
        ))}
      </div>
    </div>
  );
}

/* ── Group cards horizontal ──────────────────────────────── */
function GroupCarousel({
  groups,
  onSeeAll,
  onGroupClick,
}: {
  groups: GroupItem[];
  onSeeAll: () => void;
  onGroupClick: (groupId: string) => void;
}) {
  return (
    <div>
      <SectionHead title="附近群组" action="更多" onAction={onSeeAll} />
      <div className="flex gap-3 overflow-x-auto no-scrollbar pb-1">
        {groups.slice(0, 6).map((g) => (
          <motion.div
            key={g.group.group_id}
            whileTap={{ scale: 0.98 }}
            className="flex-shrink-0 rounded-[16px] overflow-hidden cursor-pointer neu-surface-flat"
            style={{ width: 140 }}
            onClick={() => onGroupClick(g.group.group_id)}
          >
            <div
              className="relative"
              style={{ height: 72, background: "var(--neu-border)" }}
            >
              {g.group.avatar_url && (
                <img
                  src={g.group.avatar_url}
                  alt={g.group.name}
                  className="w-full h-full object-cover"
                />
              )}
              <div
                className="absolute inset-0"
                style={{
                  background:
                    "linear-gradient(to top, rgba(0,0,0,0.5) 0%, transparent 55%)",
                }}
              />
              <div className="absolute top-1.5 right-1.5">
                {g.join_method === "invite_only" ? (
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
                <span style={{ fontSize: 10, color: "rgba(255,255,255,0.95)" }}>
                  {formatDistance(g.location?.distance)}
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
                {g.group.name}
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
                <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>
                  {g.group.members_count} 成员
                </span>
              </div>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  );
}

/* ── Compact nearby user rows ──────────────────────── */
function NearbyUsersFlatList({
  users,
  followingIds,
  onToggle,
  onUserClick,
}: {
  users: UserItem[];
  followingIds: Set<string>;
  onToggle: (userId: string) => void;
  onUserClick: (userId: string) => void;
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
        const followed = followingIds.has(u.user.user_id);
        return (
          <div
            key={u.user.user_id}
            role="button"
            tabIndex={0}
            aria-label={u.user.nickname}
            onKeyDown={(e) => e.key === "Enter" && onUserClick(u.user.user_id)}
            className="flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors"
            style={{ background: "var(--neu-bg)" }}
            onMouseEnter={(e) =>
              (e.currentTarget.style.background = "var(--neu-border)")
            }
            onMouseLeave={(e) =>
              (e.currentTarget.style.background = "var(--neu-bg)")
            }
            onClick={() => onUserClick(u.user.user_id)}
          >
            <div className="relative flex-shrink-0">
              <div
                className="rounded-full overflow-hidden"
                style={{ width: 38, height: 38 }}
              >
                <img
                  src={u.user.avatar_url || ""}
                  alt={u.user.nickname}
                  className="w-full h-full object-cover bg-[var(--neu-border)]"
                />
              </div>
              {u.presence_status === "online" && (
                <span
                  className="absolute bottom-0 right-0 rounded-full"
                  style={{
                    width: 9,
                    height: 9,
                    background: "var(--neu-success)",
                    border: "2px solid var(--neu-bg)",
                  }}
                />
              )}
            </div>
            <div className="flex-1 min-w-0">
              <p
                className="truncate"
                style={{ fontSize: 13, color: "var(--neu-text)" }}
              >
                {u.user.nickname}
              </p>
              <div className="flex items-center gap-1.5 mt-0.5">
                {u.tags?.[0] && (
                  <span
                    style={{ fontSize: 10, color: "var(--neu-accent-primary)" }}
                  >
                    #{u.tags[0]}
                  </span>
                )}
                {u.location?.distance != null && (
                  <>
                    <span
                      style={{
                        fontSize: 10,
                        color: "var(--neu-text-muted)",
                        opacity: 0.5,
                      }}
                    >
                      ·
                    </span>
                    <span
                      style={{ fontSize: 10, color: "var(--neu-text-muted)" }}
                    >
                      {formatDistance(u.location.distance)}
                    </span>
                  </>
                )}
              </div>
            </div>
            <motion.button
              whileTap={{ scale: 0.93 }}
              onClick={(e) => {
                e.stopPropagation();
                onToggle(u.user.user_id);
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
                      color: "var(--text-on-accent, white)",
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
  const { nearbyUsers, nearbyGroups, trending, loading, error, refresh } =
    useHomeData();
  const { toggledIds: followingIds, toggle: toggleFollow } = useFollowToggle();

  const isTablet = platform === "tablet";
  const shellStyle: React.CSSProperties = isTablet
    ? { maxWidth: 924, marginLeft: "auto", marginRight: "auto", width: "100%" }
    : {};
  const mainColClass = isTablet ? "mx-auto w-full lg:mx-0 min-w-0" : "w-full";
  const mainColStyle: React.CSSProperties = isTablet ? { maxWidth: 600 } : {};

  const onlineCount = nearbyUsers.filter(
    (u) => u.presence_status === "online",
  ).length;

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
                  background: "var(--neu-success)",
                  flexShrink: 0,
                }}
              />
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                定位中
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
          {loading && nearbyUsers.length === 0 && nearbyGroups.length === 0 ? (
            <HomeSkeleton />
          ) : error && nearbyUsers.length === 0 && nearbyGroups.length === 0 ? (
            <div className="flex flex-col items-center gap-3 py-16">
              <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
                {error}
              </p>
              <motion.button
                whileTap={{ scale: 0.94 }}
                onClick={refresh}
                className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none"
                style={{
                  background: "var(--neu-accent-primary)",
                  color: "var(--text-on-accent, white)",
                  fontSize: 13,
                }}
              >
                重试
              </motion.button>
            </div>
          ) : (
            <>
              <HeroStats
                onExplore={() => navigate("/nearby")}
                usersCount={nearbyUsers.length}
                groupsCount={nearbyGroups.length}
                onlineCount={onlineCount}
              />
              {nearbyUsers.length > 0 ? (
                <PeopleRadarStrip
                  users={nearbyUsers}
                  onSeeAll={() => navigate("/nearby")}
                  onUserClick={(id) => navigate(`/user/${id}`)}
                />
              ) : (
                <div className="flex flex-col items-center gap-2 py-6">
                  <Radar
                    style={{
                      width: 28,
                      height: 28,
                      color: "var(--neu-text-muted)",
                      opacity: 0.3,
                    }}
                    strokeWidth={1.25}
                  />
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
                    附近暂无用户
                  </p>
                  <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                    开启定位后发现更多附近的人
                  </p>
                </div>
              )}
              {trending && trending.trending_tags.length > 0 && (
                <TrendingTags tags={trending.trending_tags} />
              )}
              {nearbyGroups.length > 0 ? (
                <GroupCarousel
                  groups={nearbyGroups}
                  onSeeAll={() => navigate("/nearby-groups")}
                  onGroupClick={(id) => navigate(`/events/${id}`)}
                />
              ) : (
                <div>
                  <SectionHead
                    title="附近群组"
                    action="更多"
                    onAction={() => navigate("/nearby-groups")}
                  />
                  <div className="flex flex-col items-center gap-2 py-5">
                    <Users
                      style={{
                        width: 24,
                        height: 24,
                        color: "var(--neu-text-muted)",
                        opacity: 0.3,
                      }}
                      strokeWidth={1.25}
                    />
                    <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
                      附近暂无群组
                    </p>
                    <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                      换个位置或创建一个新群组
                    </p>
                  </div>
                </div>
              )}
              <div>
                <SectionHead
                  title="人物推荐"
                  action="查看全部"
                  onAction={() => navigate("/nearby")}
                />
                <NearbyUsersFlatList
                  users={nearbyUsers.slice(0, 4)}
                  followingIds={followingIds}
                  onToggle={toggleFollow}
                  onUserClick={(id) => navigate(`/user/${id}`)}
                />
              </div>
            </>
          )}
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
  const { nearbyUsers, nearbyGroups, trending, loading, error, refresh } =
    useHomeData();
  const { toggledIds: followingIds, toggle: toggleFollow } = useFollowToggle();

  const onlineCount = nearbyUsers.filter(
    (u) => u.presence_status === "online",
  ).length;

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
              style={{ width: 6, height: 6, background: "var(--neu-success)" }}
            />
            <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
              {onlineCount} 在线
            </span>
          </div>
        </div>
      </div>

      {loading && nearbyUsers.length === 0 && nearbyGroups.length === 0 ? (
        <div className="flex items-center justify-center p-12">
          <HomeSkeleton />
        </div>
      ) : error && nearbyUsers.length === 0 && nearbyGroups.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-20">
          <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
            {error}
          </p>
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={refresh}
            className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none"
            style={{
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
              fontSize: 13,
            }}
          >
            重试
          </motion.button>
        </div>
      ) : (
        <div
          className="flex gap-0 overflow-hidden"
          style={{ height: "calc(100% - 65px)" }}
        >
          <div
            className="flex flex-col overflow-y-auto no-scrollbar p-6 gap-6"
            style={{
              width: 340,
              flexShrink: 0,
              borderRight: "1px solid var(--neu-border)",
            }}
          >
            <HeroStats
              onExplore={() => navigate("/nearby")}
              usersCount={nearbyUsers.length}
              groupsCount={nearbyGroups.length}
              onlineCount={onlineCount}
            />

            {trending && trending.trending_tags.length > 0 && (
              <TrendingTags tags={trending.trending_tags} />
            )}

            {nearbyGroups.length > 0 ? (
              <GroupCarousel
                groups={nearbyGroups}
                onSeeAll={() => navigate("/nearby-groups")}
                onGroupClick={(id) => navigate(`/events/${id}`)}
              />
            ) : (
              <div className="flex flex-col items-center gap-2 py-4">
                <p style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
                  附近暂无群组
                </p>
              </div>
            )}
          </div>

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

            {nearbyUsers.length > 0 ? (
              <PeopleRadarStrip
                users={nearbyUsers}
                onSeeAll={() => navigate("/nearby")}
                onUserClick={(id) => navigate(`/user/${id}`)}
              />
            ) : (
              <div className="flex flex-col items-center gap-2 py-6">
                <Radar
                  style={{
                    width: 28,
                    height: 28,
                    color: "var(--neu-text-muted)",
                    opacity: 0.3,
                  }}
                  strokeWidth={1.25}
                />
                <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
                  附近暂无用户
                </p>
                <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                  开启定位后发现更多附近的人
                </p>
              </div>
            )}

            <div className="mt-6">
              <NearbyUsersFlatList
                users={nearbyUsers}
                followingIds={followingIds}
                onToggle={toggleFollow}
                onUserClick={(id) => navigate(`/user/${id}`)}
              />
            </div>
          </div>
        </div>
      )}
    </>
  );
}

export function Home() {
  const { platform } = usePlatform();
  useDocumentTitle("附近");
  if (platform === "desktop") return <HomeDesktop />;
  return <HomeMobile />;
}
