import React, { useState, useMemo, useEffect } from "react";
import {
  Heart,
  MapPin,
  Grid3X3,
  Settings,
  Share2,
  Bookmark,
  ChevronRight,
  Camera,
  Users,
  MessageCircle,
  Search,
  Plus,
  Check,
  Clock,
  Filter,
  Pencil,
  UserPlus,
  TrendingUp,
  Shield,
  X,
  ArrowLeft,
} from "lucide-react";
import { EmptyState } from "../components/ui/EmptyState";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate, useSearchParams } from "react-router";
import { usePlatform } from "../contexts/PlatformContext";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { CreateGroupSheet } from "../components/ui/CreateGroupSheet";
import { useFocusTrap } from "../hooks/useFocusTrap";
import {
  TabletRail,
  StreakCard,
  VisitorsCard,
  MoodCheckinCard,
  AchievementCard,
  SuggestedFollowsCard,
  VibeCard,
} from "../components/TabletRail";
import { useProfileData } from "@/lib/hooks/use-profile-data";
import { useAuthStore } from "@/lib/stores/auth";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { normalizeAvatar, formatRelativeTime } from "@/lib/utils";
import type { MomentItem } from "@/lib/types/api";

/* ── Shared: Tab bar ──────────────────────────────────────────────── */
type Tab = "posts" | "liked" | "saved";
function ProfileTabs({
  activeTab,
  setActiveTab,
}: {
  activeTab: Tab;
  setActiveTab: (t: Tab) => void;
}) {
  const tabs: { id: Tab; Icon: React.ElementType; label: string }[] = [
    { id: "posts", Icon: Grid3X3, label: "帖子" },
    { id: "liked", Icon: Heart, label: "喜欢" },
    { id: "saved", Icon: Bookmark, label: "收藏" },
  ];
  return (
    <div
      className="flex"
      style={{
        borderBottom: "1px solid var(--neu-border)",
        paddingLeft: 16,
        paddingRight: 16,
      }}
    >
      {tabs.map(({ id, Icon, label }) => {
        const isActive = activeTab === id;
        return (
          <motion.button
            key={id}
            whileTap={{ scale: 0.94 }}
            onClick={() => setActiveTab(id)}
            className="relative flex items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none"
            style={{
              flex: 1,
              color: isActive
                ? "var(--neu-accent-primary)"
                : "var(--neu-text-muted)",
              transition: "color 0.2s",
              fontSize: 13,
              fontWeight: isActive ? 500 : 400,
            }}
          >
            <Icon
              style={{ width: 14, height: 14 }}
              strokeWidth={isActive ? 2 : 1.5}
            />
            <span>{label}</span>
            {isActive && (
              <motion.div
                layoutId="profile-tab-indicator"
                style={{
                  position: "absolute",
                  bottom: -1,
                  left: "18%",
                  right: "18%",
                  height: 2,
                  borderRadius: 1,
                  background: "var(--neu-accent-primary)",
                }}
                transition={{ type: "spring", stiffness: 420, damping: 34 }}
              />
            )}
          </motion.button>
        );
      })}
    </div>
  );
}

/* ── Shared: Gallery grid ────────────────────────────────────────── */
function GalleryGrid({
  moments,
  activeTab,
}: {
  moments: MomentItem[];
  activeTab: Tab;
}) {
  const { platform } = usePlatform();
  const numCols = platform === "tablet" ? 3 : 2;
  const gap = 10;
  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={activeTab}
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -4 }}
        transition={{ duration: 0.2 }}
        style={{
          display: "grid",
          gridTemplateColumns: `repeat(${numCols}, 1fr)`,
          gap,
          width: "100%",
        }}
      >
        {moments.map((moment) => (
          <motion.div
            key={moment.moment_id}
            whileTap={{ scale: 0.97 }}
            className="relative rounded-[18px] overflow-hidden cursor-pointer group"
            style={{
              aspectRatio: "1/1",
              boxShadow:
                "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            {moment.images?.[0] ? (
              <img
                src={normalizeAvatar(moment.images[0].url)}
                alt=""
                className="w-full h-full object-cover"
                loading="lazy"
              />
            ) : (
              <div
                className="w-full h-full flex items-center justify-center"
                style={{ background: "var(--neu-border)" }}
              >
                <Grid3X3
                  style={{
                    width: 24,
                    height: 24,
                    color: "var(--neu-text-muted)",
                    opacity: 0.3,
                  }}
                />
              </div>
            )}
            <div
              className="absolute inset-0 flex items-end p-2"
              style={{
                background:
                  "linear-gradient(to top, rgba(0,0,0,0.55) 0%, transparent 55%)",
              }}
            >
              <div className="flex items-center gap-2">
                <div className="flex items-center gap-0.5">
                  <Heart
                    style={{ width: 10, height: 10, color: "white" }}
                    fill="white"
                    strokeWidth={0}
                  />
                  <span style={{ fontSize: 10, color: "white" }}>
                    {moment.likes_count ?? 0}
                  </span>
                </div>
                <div className="flex items-center gap-0.5">
                  <MessageCircle
                    style={{ width: 10, height: 10, color: "white" }}
                    strokeWidth={1.5}
                  />
                  <span style={{ fontSize: 10, color: "white" }}>
                    {moment.comments_count ?? 0}
                  </span>
                </div>
              </div>
            </div>
          </motion.div>
        ))}
      </motion.div>
    </AnimatePresence>
  );
}

/* ── Skeleton for loading ─────────────────────────────────────────── */
function GallerySkeleton({ count = 6 }: { count?: number }) {
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(3, 1fr)",
        gap: 10,
        width: "100%",
      }}
    >
      {Array.from({ length: count }).map((_, i) => (
        <div
          key={i}
          className="rounded-[16px]"
          style={{ aspectRatio: "1/1", background: "var(--neu-border)" }}
        />
      ))}
    </div>
  );
}

const PROFILE_SECTION_WIDE_CLASS = "w-full max-w-[920px] mx-auto";
const PROFILE_SECTION_LIST_CLASS = "w-full max-w-[760px] mx-auto";

/* ══════════════════════════════════════════════════════════════════
   Section: 我的瞬间
══════════════════════════════════════════════════════════════════ */
function SectionMoments() {
  const {
    moments,
    likedMoments,
    loadingMoments,
    loadingLiked,
    loadMoreMoments,
    hasMoreMoments,
  } = useProfileData();
  const [activeTab, setActiveTab] = useState<Tab>("posts");

  const postsMap: Record<Tab, MomentItem[]> = {
    posts: moments,
    liked: likedMoments,
    saved: [],
  };

  const activePosts = postsMap[activeTab];
  const isLoading = activeTab === "liked" ? loadingLiked : loadingMoments;

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      <div
        className="flex items-center justify-between px-5 py-4 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>瞬间</span>
          <span
            className="px-2 py-0.5 rounded-full"
            style={{
              fontSize: 11,
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
            }}
          >
            {moments.length} 帖子
          </span>
        </div>
        <motion.button
          whileTap={{ scale: 0.93 }}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none"
          style={{
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 12,
            boxShadow: "0 4px 12px var(--neu-glow)",
          }}
        >
          <Plus style={{ width: 13, height: 13 }} strokeWidth={2.5} />
          发布新瞬间
        </motion.button>
      </div>

      <div className="flex-shrink-0">
        <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      <div
        className={PROFILE_SECTION_WIDE_CLASS}
        style={{ padding: "14px 20px 24px" }}
      >
        {isLoading ? (
          <GallerySkeleton />
        ) : activePosts.length === 0 ? (
          <EmptyState
            icon={Grid3X3}
            title={
              activeTab === "posts"
                ? "暂无瞬间"
                : activeTab === "liked"
                  ? "暂无喜欢"
                  : "暂无收藏"
            }
          />
        ) : (
          <>
            <AnimatePresence mode="wait">
              <motion.div
                key={activeTab}
                initial={{ opacity: 0, y: 6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -4 }}
                transition={{ duration: 0.2 }}
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fit, minmax(168px, 1fr))",
                  gap: 12,
                  width: "100%",
                }}
              >
                {activePosts.map((moment) => (
                  <motion.div
                    key={moment.moment_id}
                    whileTap={{ scale: 0.97 }}
                    className="relative rounded-[16px] overflow-hidden cursor-pointer"
                    style={{
                      aspectRatio: "1/1",
                      boxShadow:
                        "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                      border: "1px solid var(--neu-border)",
                    }}
                  >
                    {moment.images?.[0] ? (
                      <img
                        src={normalizeAvatar(moment.images[0].url)}
                        alt=""
                        className="w-full h-full object-cover"
                        loading="lazy"
                      />
                    ) : (
                      <div
                        className="w-full h-full flex items-center justify-center"
                        style={{ background: "var(--neu-border)" }}
                      >
                        <Grid3X3
                          style={{
                            width: 24,
                            height: 24,
                            color: "var(--neu-text-muted)",
                            opacity: 0.3,
                          }}
                        />
                      </div>
                    )}
                    <div
                      className="absolute inset-0 flex items-end p-2"
                      style={{
                        background:
                          "linear-gradient(to top, rgba(0,0,0,0.6) 0%, transparent 55%)",
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <div className="flex items-center gap-0.5">
                          <Heart
                            style={{ width: 10, height: 10, color: "white" }}
                            fill="white"
                            strokeWidth={0}
                          />
                          <span style={{ fontSize: 10, color: "white" }}>
                            {moment.likes_count ?? 0}
                          </span>
                        </div>
                        <div className="flex items-center gap-0.5">
                          <MessageCircle
                            style={{ width: 10, height: 10, color: "white" }}
                            strokeWidth={1.5}
                          />
                          <span style={{ fontSize: 10, color: "white" }}>
                            {moment.comments_count ?? 0}
                          </span>
                        </div>
                      </div>
                    </div>
                  </motion.div>
                ))}
              </motion.div>
            </AnimatePresence>
            {hasMoreMoments && activeTab === "posts" && (
              <div className="flex justify-center mt-4">
                <motion.button
                  whileTap={{ scale: 0.93 }}
                  onClick={loadMoreMoments}
                  className="px-4 py-2 rounded-full outline-none"
                  style={{
                    fontSize: 12,
                    color: "var(--neu-accent-primary)",
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  加载更多
                </motion.button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 我的群组
══════════════════════════════════════════════════════════════════ */
function SectionGroups() {
  const { groups, loadingGroups } = useProfileData();
  const [searchQ, setSearchQ] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const hover = useHoverConfig();

  const filtered = useMemo(
    () =>
      searchQ === ""
        ? groups
        : groups.filter((g) =>
            g.group.name.toLowerCase().includes(searchQ.toLowerCase()),
          ),
    [searchQ, groups],
  );

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      <div
        className="flex items-center justify-between px-5 py-4 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>群组</span>
          <span
            className="px-2 py-0.5 rounded-full"
            style={{
              fontSize: 11,
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
            }}
          >
            {groups.length} 个
          </span>
        </div>
        <motion.button
          whileTap={hover.tap}
          whileHover={hover.btn}
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none"
          style={{
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 12,
            boxShadow: "0 4px 12px var(--neu-glow)",
          }}
        >
          <Plus style={{ width: 13, height: 13 }} strokeWidth={2.5} />
          创建群组
        </motion.button>
      </div>

      <div className={`${PROFILE_SECTION_WIDE_CLASS} px-5 py-3 flex-shrink-0`}>
        <div
          className="flex items-center gap-2.5 px-4 rounded-[14px] neu-input-surface"
          style={{ height: 40 }}
        >
          <Search
            style={{
              width: 13,
              height: 13,
              color: "var(--neu-text-muted)",
              flexShrink: 0,
            }}
            strokeWidth={1.5}
          />
          <input
            value={searchQ}
            onChange={(e) => setSearchQ(e.target.value)}
            aria-label="搜索关注"
            placeholder="搜索群组…"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
          {searchQ && (
            <motion.button
              whileTap={{ scale: 0.85 }}
              onClick={() => setSearchQ("")}
              aria-label="清除搜索"
              className="outline-none"
            >
              <X
                style={{
                  width: 12,
                  height: 12,
                  color: "var(--neu-text-muted)",
                }}
                strokeWidth={2}
              />
            </motion.button>
          )}
        </div>
      </div>

      <div
        className={`${PROFILE_SECTION_WIDE_CLASS} grid gap-4 px-5 pb-6 xl:grid-cols-2`}
      >
        {loadingGroups ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3 xl:col-span-2">
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
              加载中…
            </p>
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Users}
            title="暂无群组"
            className="xl:col-span-2"
          />
        ) : (
          filtered.map((g) => (
            <motion.div
              key={g.group.group_id}
              whileTap={hover.tap}
              whileHover={hover.card}
              className="rounded-[20px] overflow-hidden cursor-pointer neu-card-hover"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <div
                className="relative"
                style={{ height: "clamp(60px, 10vw, 80px)" }}
              >
                {g.group.avatar_url ? (
                  <img
                    src={normalizeAvatar(g.group.avatar_url)}
                    alt={g.group.name}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div
                    className="w-full h-full"
                    style={{ background: "var(--neu-border)" }}
                  />
                )}
                <div
                  className="absolute inset-0"
                  style={{
                    background:
                      "linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.6) 100%)",
                  }}
                />
                <div className="absolute bottom-2 left-3 flex items-center gap-1.5">
                  <span
                    className="px-2 py-0.5 rounded-full"
                    style={{
                      fontSize: 10,
                      background:
                        g.my_state?.role === "admin"
                          ? "rgba(99,102,241,0.9)"
                          : "rgba(0,0,0,0.5)",
                      color: "white",
                      backdropFilter: "blur(8px)",
                    }}
                  >
                    {g.my_state?.role === "admin" && (
                      <Shield
                        style={{
                          width: 8,
                          height: 8,
                          display: "inline",
                          marginRight: 2,
                        }}
                        strokeWidth={2}
                      />
                    )}
                    {g.my_state?.role === "admin" ? "管理员" : "成员"}
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-3 p-3.5">
                {g.group.avatar_url ? (
                  <img
                    src={normalizeAvatar(g.group.avatar_url)}
                    alt={g.group.name}
                    className="rounded-[12px] object-cover flex-shrink-0"
                    style={{
                      width: 44,
                      height: 44,
                      border: "2px solid var(--neu-bg)",
                      marginTop: -22,
                      boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
                    }}
                  />
                ) : (
                  <div
                    className="rounded-[12px] flex items-center justify-center flex-shrink-0"
                    style={{
                      width: 44,
                      height: 44,
                      border: "2px solid var(--neu-bg)",
                      marginTop: -22,
                      boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
                      background: "var(--neu-border)",
                    }}
                  >
                    <Users
                      style={{
                        width: 16,
                        height: 16,
                        color: "var(--neu-text-muted)",
                      }}
                      strokeWidth={1.5}
                    />
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <p style={{ fontSize: 14, color: "var(--neu-text)" }}>
                    {g.group.name}
                  </p>
                  <p
                    style={{
                      fontSize: 11,
                      color: "var(--neu-text-muted)",
                      marginTop: 2,
                    }}
                  >
                    {g.description || ""}
                  </p>
                </div>
                <div className="flex flex-col items-end gap-1.5 flex-shrink-0">
                  <div className="flex items-center gap-1">
                    <Users
                      style={{
                        width: 11,
                        height: 11,
                        color: "var(--neu-text-muted)",
                      }}
                      strokeWidth={1.5}
                    />
                    <span
                      style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                    >
                      {g.group.members_count}
                    </span>
                  </div>
                  <motion.button
                    whileTap={{ scale: 0.93 }}
                    className="flex items-center gap-1 px-2.5 py-1 rounded-full outline-none"
                    style={{
                      background: "var(--neu-accent-primary)",
                      color: "var(--text-on-accent, white)",
                      fontSize: 11,
                      boxShadow: "0 3px 8px var(--neu-glow)",
                    }}
                  >
                    进入
                    <ChevronRight
                      style={{ width: 10, height: 10 }}
                      strokeWidth={2}
                    />
                  </motion.button>
                </div>
              </div>
              <div
                className="px-3.5 pb-3 flex items-center gap-1.5"
                style={{
                  borderTop: "1px solid var(--neu-border)",
                  paddingTop: 8,
                  marginTop: -4,
                }}
              >
                <Clock
                  style={{
                    width: 11,
                    height: 11,
                    color: "var(--neu-text-muted)",
                  }}
                  strokeWidth={1.5}
                />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
                  最近活跃{" "}
                  {formatRelativeTime(
                    g.last_activity_at ?? g.my_state?.joined_at ?? "",
                  )}
                </span>
              </div>
            </motion.div>
          ))
        )}
      </div>

      <ProfileGroupSheet
        show={showCreate}
        onClose={() => setShowCreate(false)}
      />
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 好友
══════════════════════════════════════════════════════════════════ */
function SectionFriends() {
  const { friends, loadingFriends } = useProfileData();
  const [searchQ, setSearchQ] = useState("");
  const [activeFilter, setActiveFilter] = useState<
    "all" | "online" | "offline"
  >("all");

  const filtered = useMemo(() => {
    let list = friends;
    if (activeFilter === "online")
      list = list.filter((f) => f.presence_status === "online");
    if (activeFilter === "offline")
      list = list.filter((f) => f.presence_status !== "online");
    if (searchQ)
      list = list.filter((f) =>
        f.user.nickname.toLowerCase().includes(searchQ.toLowerCase()),
      );
    return list;
  }, [friends, activeFilter, searchQ]);

  const onlineCount = friends.filter(
    (f) => f.presence_status === "online",
  ).length;

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      <div
        className="flex items-center justify-between px-5 py-4 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>好友</span>
          <div
            className="flex items-center gap-1 px-2 py-0.5 rounded-full"
            style={{
              fontSize: 11,
              background: "var(--neu-success-soft)",
              border: "1px solid var(--neu-success-border)",
            }}
          >
            <div
              style={{
                width: 6,
                height: 6,
                borderRadius: "50%",
                background: "var(--neu-success)",
              }}
            />
            <span style={{ color: "var(--neu-success)" }}>{onlineCount} 在线</span>
          </div>
        </div>
        <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
          {friends.length} 位好友
        </span>
      </div>

      <div
        className={`${PROFILE_SECTION_WIDE_CLASS} px-5 pt-3 pb-2 flex-shrink-0`}
      >
        <div
          className="flex items-center gap-2.5 px-4 rounded-[14px] neu-input-surface"
          style={{ height: 40 }}
        >
          <Search
            style={{
              width: 13,
              height: 13,
              color: "var(--neu-text-muted)",
              flexShrink: 0,
            }}
            strokeWidth={1.5}
          />
          <input
            value={searchQ}
            onChange={(e) => setSearchQ(e.target.value)}
            aria-label="搜索好友"
            placeholder="搜索好友…"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
          {searchQ && (
            <motion.button
              whileTap={{ scale: 0.85 }}
              onClick={() => setSearchQ("")}
              aria-label="清除搜索"
              className="outline-none"
            >
              <X
                style={{
                  width: 12,
                  height: 12,
                  color: "var(--neu-text-muted)",
                }}
                strokeWidth={2}
              />
            </motion.button>
          )}
        </div>
      </div>

      <div
        className={`${PROFILE_SECTION_WIDE_CLASS} flex gap-2 px-5 pb-3 flex-shrink-0`}
      >
        {(
          [
            ["all", "全部"],
            ["online", "在线"],
            ["offline", "离线"],
          ] as const
        ).map(([id, label]) => (
          <motion.button
            key={id}
            whileTap={{ scale: 0.93 }}
            onClick={() => setActiveFilter(id)}
            className="px-3 py-1.5 rounded-full outline-none"
            style={
              activeFilter === id
                ? {
                    background: "var(--neu-accent-primary)",
                    color: "var(--text-on-accent, white)",
                    fontSize: 12,
                    boxShadow: "0 3px 8px var(--neu-glow)",
                  }
                : {
                    background: "var(--neu-bg)",
                    color: "var(--neu-text-muted)",
                    fontSize: 12,
                    border: "1px solid var(--neu-border)",
                  }
            }
          >
            {label}
          </motion.button>
        ))}
      </div>

      <div
        className={`${PROFILE_SECTION_WIDE_CLASS} grid gap-4 px-5 pb-6 xl:grid-cols-2`}
      >
        {loadingFriends ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3 xl:col-span-2">
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
              加载中…
            </p>
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={Users}
            title="暂无好友"
            className="xl:col-span-2"
          />
        ) : (
          filtered.map((f) => {
            const isOnline = f.presence_status === "online";
            return (
              <motion.div
                key={f.user.user_id}
                className="flex items-center gap-3 p-3.5 rounded-[18px] neu-card-hover"
                style={{
                  background: "var(--neu-bg)",
                  boxShadow:
                    "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                  border: "1px solid var(--neu-border)",
                }}
              >
                <div className="relative flex-shrink-0">
                  <img
                    src={normalizeAvatar(f.user.avatar_url)}
                    alt={f.user.nickname}
                    className="rounded-full object-cover"
                    style={{ width: 48, height: 48 }}
                  />
                  <span
                    className="absolute bottom-0 right-0 rounded-full"
                    style={{
                      width: 12,
                      height: 12,
                      background: isOnline
                        ? "var(--neu-success)"
                        : "var(--neu-text-muted)",
                      border: "2.5px solid var(--neu-bg)",
                      opacity: isOnline ? 1 : 0.5,
                    }}
                  />
                </div>
                <div className="flex-1 min-w-0">
                  <span
                    className="block truncate"
                    style={{ fontSize: 14, color: "var(--neu-text)" }}
                  >
                    {f.user.nickname}
                  </span>
                  <p
                    style={{
                      fontSize: 12,
                      color: "var(--neu-text-muted)",
                      marginTop: 2,
                    }}
                  >
                    {f.remark ? `${f.remark} · ` : ""}
                    {formatRelativeTime(f.followed_at)}开始关注
                  </p>
                </div>
                <div className="flex flex-col gap-1.5 flex-shrink-0">
                  <motion.button
                    whileTap={{ scale: 0.93 }}
                    className="flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none"
                    style={{
                      background: "var(--neu-accent-primary)",
                      color: "var(--text-on-accent, white)",
                      fontSize: 11,
                      boxShadow: "0 3px 8px var(--neu-glow)",
                    }}
                  >
                    <MessageCircle
                      style={{ width: 11, height: 11 }}
                      strokeWidth={2}
                    />
                    发消息
                  </motion.button>
                  <motion.button
                    whileTap={{ scale: 0.93 }}
                    className="flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none"
                    style={{
                      background: "var(--neu-bg)",
                      color: "var(--neu-text-muted)",
                      fontSize: 11,
                      border: "1px solid var(--neu-border)",
                      boxShadow:
                        "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)",
                    }}
                  >
                    查看主页
                  </motion.button>
                </div>
              </motion.div>
            );
          })
        )}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 关注
══════════════════════════════════════════════════════════════════ */
function SectionFollowing() {
  const { following, loadingFollowing } = useProfileData();
  const [sortBy, setSortBy] = useState<"recent" | "active">("recent");
  const [unfollowedIds, setUnfollowedIds] = useState<Set<string>>(new Set());

  const toggleUnfollow = (id: string) => {
    setUnfollowedIds((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const sorted = [...following].sort((a, b) =>
    sortBy === "active"
      ? new Date(b.followed_at).getTime() - new Date(a.followed_at).getTime()
      : 0,
  );

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      <div
        className="flex items-center justify-between px-5 py-4 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>关注</span>
          <span
            className="px-2 py-0.5 rounded-full"
            style={{
              fontSize: 11,
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
            }}
          >
            {following.length} 人
          </span>
        </div>
        <motion.button
          whileTap={{ scale: 0.93 }}
          className="flex items-center gap-1 px-3 py-1 rounded-full outline-none neu-surface-ghost"
          style={{
            color: "var(--neu-text-muted)",
            fontSize: 12,
            borderColor: "var(--neu-border-pressed)",
          }}
        >
          <Filter style={{ width: 11, height: 11 }} strokeWidth={1.5} />
          管理
        </motion.button>
      </div>

      <div
        className={`${PROFILE_SECTION_LIST_CLASS} flex gap-1.5 px-5 py-3 flex-shrink-0`}
      >
        {(
          [
            ["recent", "最近关注"],
            ["active", "最活跃"],
          ] as const
        ).map(([id, label]) => {
          const active = sortBy === id;
          return (
            <motion.button
              key={id}
              whileTap={{ scale: 0.94 }}
              onClick={() => setSortBy(id)}
              className="flex items-center gap-1.5 px-3 py-1 rounded-full outline-none neu-surface-ghost"
              style={
                active
                  ? {
                      background: "var(--neu-accent-soft)",
                      color: "var(--neu-accent-primary)",
                      fontSize: 12,
                      fontWeight: 500,
                      borderColor: "var(--neu-accent-soft-border)",
                    }
                  : { fontSize: 12, color: "var(--neu-text-muted)" }
              }
            >
              {id === "active" && (
                <TrendingUp
                  style={{ width: 11, height: 11 }}
                  strokeWidth={active ? 2 : 1.5}
                />
              )}
              {label}
            </motion.button>
          );
        })}
      </div>

      <div className={`${PROFILE_SECTION_LIST_CLASS} px-5 pb-6`}>
        {loadingFollowing ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3">
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
              加载中…
            </p>
          </div>
        ) : sorted.length === 0 ? (
          <EmptyState icon={Heart} title="暂无关注" />
        ) : (
          <div className="flex flex-col gap-3">
            {sorted.map((f) => {
              const isUnfollowed = unfollowedIds.has(f.user.user_id);
              return (
                <div
                  key={f.user.user_id}
                  className="flex items-center gap-3 p-3 rounded-[18px] transition-colors neu-card-hover"
                  style={{
                    background: "var(--neu-bg)",
                    opacity: isUnfollowed ? 0.55 : 1,
                    border: "1px solid var(--neu-border)",
                    boxShadow:
                      "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                  }}
                  onMouseEnter={(e) =>
                    (e.currentTarget.style.background = "var(--neu-border)")
                  }
                  onMouseLeave={(e) =>
                    (e.currentTarget.style.background = "var(--neu-bg)")
                  }
                >
                  <img
                    src={normalizeAvatar(f.user.avatar_url)}
                    alt={f.user.nickname}
                    className="rounded-full object-cover flex-shrink-0"
                    style={{ width: 42, height: 42 }}
                  />
                  <div className="flex-1 min-w-0">
                    <p
                      className="truncate"
                      style={{ fontSize: 13, color: "var(--neu-text)" }}
                    >
                      {f.user.nickname}
                    </p>
                    <p
                      style={{
                        fontSize: 11,
                        color: "var(--neu-text-muted)",
                        marginTop: 1,
                      }}
                      className="truncate"
                    >
                      {formatRelativeTime(f.followed_at)}开始关注
                    </p>
                  </div>
                  <motion.button
                    whileTap={{ scale: 0.93 }}
                    onClick={() => toggleUnfollow(f.user.user_id)}
                    className="flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0"
                    style={
                      isUnfollowed
                        ? {
                            background: "var(--neu-accent-primary)",
                            color: "var(--text-on-accent, white)",
                            fontSize: 11,
                            border: "1px solid transparent",
                          }
                        : {
                            background: "transparent",
                            color: "var(--neu-text-muted)",
                            fontSize: 11,
                            border: "1px solid var(--neu-border-pressed)",
                          }
                    }
                  >
                    {isUnfollowed ? (
                      <>
                        <UserPlus
                          style={{ width: 10, height: 10 }}
                          strokeWidth={2}
                        />
                        关注
                      </>
                    ) : (
                      <>
                        <Check
                          style={{ width: 10, height: 10 }}
                          strokeWidth={2.5}
                        />
                        已关注
                      </>
                    )}
                  </motion.button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 粉丝
══════════════════════════════════════════════════════════════════ */
function SectionFans() {
  const { followers, loadingFollowers } = useProfileData();
  const [fanFilter, setFanFilter] = useState<"all" | "mutual" | "pending">(
    "all",
  );
  const [followedBack, setFollowedBack] = useState<Set<string>>(
    new Set(
      followers.filter((f) => f.is_following_back).map((f) => f.user.user_id),
    ),
  );

  const toggleFollow = (id: string) => {
    setFollowedBack((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const filtered = followers.filter((f) => {
    const isMutual = followedBack.has(f.user.user_id);
    if (fanFilter === "mutual") return isMutual;
    if (fanFilter === "pending") return !isMutual;
    return true;
  });

  const mutualCount = followedBack.size;
  const pendingCount = followers.length - mutualCount;

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      <div
        className="flex items-center justify-between px-5 py-4 flex-shrink-0"
        style={{ borderBottom: "1px solid var(--neu-border)" }}
      >
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>粉丝</span>
          <span
            className="px-2 py-0.5 rounded-full"
            style={{
              fontSize: 11,
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
            }}
          >
            {followers.length} 人
          </span>
        </div>
        <div className="flex items-center gap-2">
          {pendingCount > 0 && (
            <span
              className="px-2 py-0.5 rounded-full"
              style={{
                fontSize: 11,
                background: "var(--neu-warn-soft)",
                color: "var(--neu-warn)",
                border: "1px solid var(--neu-warn-border)",
              }}
            >
              {pendingCount} 待回关
            </span>
          )}
        </div>
      </div>

      <div className={`${PROFILE_SECTION_LIST_CLASS} px-5 py-3 flex-shrink-0`}>
        <div
          className="grid grid-cols-3 rounded-[14px] overflow-hidden"
          style={{
            background: "var(--neu-border)",
            gap: 1,
            border: "1px solid var(--neu-border)",
          }}
        >
          {[
            { v: followers.length, label: "总粉丝", color: "var(--neu-text)" },
            { v: mutualCount, label: "互相关注", color: "var(--neu-success)" },
            { v: pendingCount, label: "待回关", color: "var(--neu-warn)" },
          ].map((s) => (
            <div
              key={s.label}
              className="flex flex-col items-center py-2.5"
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
                  fontSize: 11,
                  color: "var(--neu-text-muted)",
                  marginTop: 4,
                }}
              >
                {s.label}
              </span>
            </div>
          ))}
        </div>
      </div>

      <div
        className={`${PROFILE_SECTION_LIST_CLASS} flex gap-1.5 px-5 pb-3 flex-shrink-0`}
      >
        {(
          [
            ["all", "全部"],
            ["mutual", "互相关注"],
            ["pending", "待回关"],
          ] as const
        ).map(([id, label]) => {
          const active = fanFilter === id;
          return (
            <motion.button
              key={id}
              whileTap={{ scale: 0.94 }}
              onClick={() => setFanFilter(id)}
              className="px-3 py-1 rounded-full outline-none neu-surface-ghost"
              style={
                active
                  ? {
                      background: "var(--neu-accent-soft)",
                      color: "var(--neu-accent-primary)",
                      fontSize: 12,
                      fontWeight: 500,
                      borderColor: "var(--neu-accent-soft-border)",
                    }
                  : { fontSize: 12, color: "var(--neu-text-muted)" }
              }
            >
              {label}
            </motion.button>
          );
        })}
      </div>

      <div className={`${PROFILE_SECTION_LIST_CLASS} px-5 pb-6`}>
        {loadingFollowers ? (
          <div className="flex flex-col items-center justify-center py-12 gap-3">
            <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
              加载中…
            </p>
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState icon={UserPlus} title="暂无粉丝" />
        ) : (
          <div className="flex flex-col gap-3">
            {filtered.map((f) => {
              const isMutual = followedBack.has(f.user.user_id);
              return (
                <div
                  key={f.user.user_id}
                  className="flex items-center gap-3 p-3 rounded-[18px] transition-colors neu-card-hover"
                  style={{
                    background: "var(--neu-bg)",
                    border: "1px solid var(--neu-border)",
                    boxShadow:
                      "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                  }}
                  onMouseEnter={(e) =>
                    (e.currentTarget.style.background = "var(--neu-border)")
                  }
                  onMouseLeave={(e) =>
                    (e.currentTarget.style.background = "var(--neu-bg)")
                  }
                >
                  <img
                    src={normalizeAvatar(f.user.avatar_url)}
                    alt={f.user.nickname}
                    className="rounded-full object-cover flex-shrink-0"
                    style={{ width: 42, height: 42 }}
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 min-w-0">
                      <span
                        className="block truncate"
                        style={{ fontSize: 13, color: "var(--neu-text)" }}
                      >
                        {f.user.nickname}
                      </span>
                      {isMutual && (
                        <span
                          className="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full"
                          style={{
                            fontSize: 10,
                            background: "var(--neu-success-soft)",
                            color: "var(--neu-success)",
                          }}
                        >
                          <Check
                            style={{ width: 9, height: 9 }}
                            strokeWidth={2.5}
                          />
                          互关
                        </span>
                      )}
                    </div>
                    <p
                      style={{
                        fontSize: 11,
                        color: "var(--neu-text-muted)",
                        marginTop: 2,
                      }}
                    >
                      {formatRelativeTime(f.followed_at)}关注了你
                    </p>
                  </div>
                  <motion.button
                    whileTap={{ scale: 0.93 }}
                    onClick={() => toggleFollow(f.user.user_id)}
                    className="flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0"
                    style={
                      isMutual
                        ? {
                            background: "transparent",
                            color: "var(--neu-text-muted)",
                            fontSize: 11,
                            border: "1px solid var(--neu-border-pressed)",
                          }
                        : {
                            background: "var(--neu-accent-primary)",
                            color: "var(--text-on-accent, white)",
                            fontSize: 11,
                            border: "1px solid transparent",
                          }
                    }
                  >
                    {isMutual ? (
                      <>
                        <Check
                          style={{ width: 10, height: 10 }}
                          strokeWidth={2.5}
                        />
                        互关
                      </>
                    ) : (
                      <>
                        <UserPlus
                          style={{ width: 10, height: 10 }}
                          strokeWidth={2}
                        />
                        回关
                      </>
                    )}
                  </motion.button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Profile
══════════════════════════════════════════════════════════════════ */
function ProfileDesktop() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const {
    profile,
    moments,
    groups,
    friends,
    following,
    followers,
    loadingProfile,
    profileError,
    refreshProfile,
  } = useProfileData();
  const section = searchParams.get("section");

  const SECTION_META: Record<
    string,
    { title: string; render: () => React.ReactNode }
  > = {
    moments: { title: "瞬间", render: () => <SectionMoments /> },
    groups: { title: "群组", render: () => <SectionGroups /> },
    friends: { title: "好友", render: () => <SectionFriends /> },
    following: { title: "关注", render: () => <SectionFollowing /> },
    fans: { title: "粉丝", render: () => <SectionFans /> },
  };

  if (section && SECTION_META[section]) {
    const meta = SECTION_META[section];
    return (
      <div className="flex flex-col h-full">
        <div
          className="flex-shrink-0 px-6 py-4"
          style={{
            backgroundColor: "var(--neu-bg)",
            borderBottom: "1px solid var(--neu-border)",
          }}
        >
          <div className="flex items-center gap-3 w-full max-w-[920px] mx-auto">
            <motion.button
              whileTap={{
                scale: 0.88,
                transition: { type: "spring", stiffness: 600, damping: 28 },
              }}
              whileHover={{
                scale: 1.14,
                transition: { type: "spring", stiffness: 400, damping: 16 },
              }}
              onClick={() => navigate("/profile")}
              aria-label="返回"
              className="flex items-center justify-center rounded-full outline-none neu-circle-hover neu-focus-ring"
              style={{
                width: 44,
                height: 44,
                background: "var(--neu-bg)",
                boxShadow:
                  "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <ArrowLeft
                style={{
                  width: 15,
                  height: 15,
                  color: "var(--neu-text-muted)",
                }}
                strokeWidth={1.5}
              />
            </motion.button>
            <p style={{ fontSize: 18, color: "var(--neu-text)", flex: 1 }}>
              {meta.title}
            </p>
          </div>
        </div>
        <div className="flex-1 min-h-0 overflow-hidden flex">
          {meta.render()}
        </div>
      </div>
    );
  }

  const displayName = profile?.nickname ?? "加载中…";
  const displayHandle = profile?.user_id
    ? `@${profile.user_id.slice(0, 8)}`
    : "";
  const stats = profile?.stats;

  return (
    <div className="flex flex-col h-full">
      <div
        className="flex-shrink-0 flex items-center justify-between px-6 py-4"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div>
          <p style={{ fontSize: 20, color: "var(--neu-text)" }}>
            {displayName}
          </p>
          <p
            style={{
              fontSize: 12,
              color: "var(--neu-text-muted)",
              marginTop: 1,
            }}
          >
            {displayHandle}
          </p>
        </div>
        <div className="flex gap-2">
          <motion.button
            whileTap={{ scale: 0.94 }}
            aria-label="分享主页"
            className="flex items-center justify-center rounded-full outline-none"
            style={{
              width: 36,
              height: 36,
              background: "var(--neu-bg)",
              boxShadow:
                "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <Share2
              style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          </motion.button>
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={() => navigate("/settings")}
            aria-label="设置"
            className="flex items-center justify-center rounded-full outline-none"
            style={{
              width: 36,
              height: 36,
              background: "var(--neu-bg)",
              boxShadow:
                "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <Settings
              style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          </motion.button>
        </div>
      </div>

      <div className="flex flex-1 min-h-0">
        <div
          className="flex-shrink-0 overflow-y-auto no-scrollbar px-6 py-6 flex flex-col items-center"
          style={{ width: 260, borderRight: "1px solid var(--neu-border)" }}
        >
          {profileError && (
            <div
              className="w-full flex items-center justify-between px-3 py-2 rounded-[12px] mb-4"
              style={{ background: "var(--neu-accent-soft)" }}
            >
              <span
                style={{ fontSize: 11, color: "var(--neu-accent-primary)" }}
              >
                {profileError}
              </span>
              <motion.button
                whileTap={{ scale: 0.94 }}
                onClick={refreshProfile}
                className="px-2 py-0.5 rounded-full outline-none"
                style={{
                  background: "var(--neu-accent-primary)",
                  color: "var(--text-on-accent, white)",
                  fontSize: 10,
                }}
              >
                重试
              </motion.button>
            </div>
          )}
          <div className="relative mb-4">
            <div
              className="absolute inset-0 rounded-full pointer-events-none"
              style={{
                background: "var(--neu-glow)",
                filter: "blur(20px)",
                transform: "scale(1.2)",
                opacity: 0.5,
              }}
            />
            <div
              className="relative rounded-full"
              style={{
                width: 96,
                height: 96,
                padding: 2,
                background: "var(--neu-bg)",
                boxShadow:
                  "8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light)",
                border: "1.5px solid var(--neu-border)",
              }}
            >
              <img
                src={normalizeAvatar(profile?.avatar_url)}
                alt={displayName}
                className="w-full h-full object-cover rounded-full"
              />
            </div>
            <motion.button
              whileTap={{ scale: 0.94 }}
              aria-label="更换头像"
              className="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
              style={{
                width: 26,
                height: 26,
                background: "var(--neu-accent-primary)",
                border: "2.5px solid var(--neu-bg)",
                boxShadow: "0 2px 8px var(--neu-glow)",
              }}
            >
              <Camera
                style={{
                  width: 12,
                  height: 12,
                  color: "var(--text-on-accent, white)",
                }}
                strokeWidth={2}
              />
            </motion.button>
          </div>

          <h2
            style={{
              fontSize: 18,
              color: "var(--neu-text)",
              textAlign: "center",
            }}
          >
            {displayName}
          </h2>
          <span
            style={{
              fontSize: 12,
              color: "var(--neu-text-muted)",
              marginTop: 2,
            }}
          >
            {displayHandle}
          </span>
          <p
            style={{
              fontSize: 12,
              color: "var(--neu-text-muted)",
              textAlign: "center",
              marginTop: 8,
              lineHeight: 1.6,
              paddingLeft: 8,
              paddingRight: 8,
            }}
          >
            {profile?.bio || ""}
          </p>
          <div className="flex items-center gap-1 mt-2 mb-4">
            <MapPin
              style={{
                width: 11,
                height: 11,
                color: "var(--neu-accent-primary)",
              }}
              strokeWidth={1.5}
            />
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
              {profile?.location?.location_name ?? ""}
            </span>
          </div>

          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={() => navigate("/profile/edit")}
            className="w-full py-2.5 rounded-full outline-none mb-5 flex items-center justify-center gap-2"
            style={{
              background: "var(--neu-bg)",
              color: "var(--neu-text-muted)",
              border: "1px solid var(--neu-border)",
              boxShadow:
                "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
              fontSize: 13,
            }}
          >
            <Pencil style={{ width: 13, height: 13 }} strokeWidth={1.5} />
            编辑资料
          </motion.button>

          <div
            className="w-full rounded-[18px] overflow-hidden mb-4"
            style={{
              background: "var(--neu-bg)",
              boxShadow:
                "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            {[
              {
                label: "帖子",
                value: stats?.moments_count ?? 0,
                section: "moments",
              },
              {
                label: "关注者",
                value: stats?.followers_count ?? 0,
                section: "fans",
              },
              { label: "足迹", value: 0, section: null },
            ].map((stat, i) => (
              <React.Fragment key={stat.label}>
                {i > 0 && (
                  <div style={{ height: 1, background: "var(--neu-border)" }} />
                )}
                <motion.button
                  whileTap={{ scale: 0.97 }}
                  onClick={() =>
                    stat.section && navigate(`/profile?section=${stat.section}`)
                  }
                  className="w-full flex items-center justify-between px-4 py-3 outline-none"
                  style={{ cursor: stat.section ? "pointer" : "default" }}
                >
                  <span
                    style={{ fontSize: 13, color: "var(--neu-text-muted)" }}
                  >
                    {stat.label}
                  </span>
                  <div className="flex items-center gap-1.5">
                    <span style={{ fontSize: 17, color: "var(--neu-text)" }}>
                      {stat.value}
                    </span>
                    {stat.section && (
                      <ChevronRight
                        style={{
                          width: 12,
                          height: 12,
                          color: "var(--neu-text-muted)",
                        }}
                        strokeWidth={1.5}
                      />
                    )}
                  </div>
                </motion.button>
              </React.Fragment>
            ))}
          </div>

          <div
            className="w-full rounded-[18px] overflow-hidden"
            style={{
              background: "var(--neu-bg)",
              boxShadow:
                "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            {[
              {
                label: "瞬间",
                section: "moments",
                icon: Grid3X3,
                count: moments.length,
              },
              {
                label: "群组",
                section: "groups",
                icon: Users,
                count: groups.length,
              },
              {
                label: "好友",
                section: "friends",
                icon: MessageCircle,
                count: friends.length,
              },
              {
                label: "关注",
                section: "following",
                icon: Heart,
                count: following.length,
              },
              {
                label: "粉丝",
                section: "fans",
                icon: UserPlus,
                count: followers.length,
              },
            ].map((item, i) => (
              <React.Fragment key={item.label}>
                {i > 0 && (
                  <div style={{ height: 1, background: "var(--neu-border)" }} />
                )}
                <motion.button
                  whileTap={{ scale: 0.97 }}
                  onClick={() => navigate(`/profile?section=${item.section}`)}
                  className="w-full flex items-center justify-between px-4 py-2.5 outline-none neu-focus-ring"
                >
                  <div className="flex items-center gap-2.5">
                    <item.icon
                      style={{
                        width: 14,
                        height: 14,
                        color: "var(--neu-text-muted)",
                      }}
                      strokeWidth={1.5}
                    />
                    <span
                      style={{ fontSize: 13, color: "var(--neu-text-muted)" }}
                    >
                      {item.label}
                    </span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span
                      style={{ fontSize: 12, color: "var(--neu-text-muted)" }}
                    >
                      {item.count}
                    </span>
                    <ChevronRight
                      style={{
                        width: 12,
                        height: 12,
                        color: "var(--neu-text-muted)",
                      }}
                      strokeWidth={1.5}
                    />
                  </div>
                </motion.button>
              </React.Fragment>
            ))}
          </div>
        </div>

        <div className="flex-1 overflow-y-auto no-scrollbar px-6 py-5">
          <div className="flex items-center justify-between mb-4">
            <p style={{ fontSize: 15, color: "var(--neu-text)" }}>最近瞬间</p>
            <motion.button
              whileTap={{ scale: 0.93 }}
              onClick={() => navigate("/profile?section=moments")}
              className="flex items-center gap-1 outline-none"
              style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}
            >
              查看全部
              <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={2} />
            </motion.button>
          </div>
          {loadingProfile || moments.length === 0 ? (
            <EmptyState icon={Grid3X3} title="暂无瞬间" />
          ) : (
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(172px, 1fr))",
                gap: 12,
                width: "100%",
                maxWidth: 760,
              }}
            >
              {moments.slice(0, 9).map((moment) => (
                <motion.div
                  key={moment.moment_id}
                  whileTap={{ scale: 0.97 }}
                  className="relative rounded-[16px] overflow-hidden cursor-pointer"
                  style={{
                    aspectRatio: "1/1",
                    boxShadow:
                      "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  {moment.images?.[0] ? (
                    <img
                      src={normalizeAvatar(moment.images[0].url)}
                      alt=""
                      className="w-full h-full object-cover"
                      loading="lazy"
                    />
                  ) : (
                    <div
                      className="w-full h-full flex items-center justify-center"
                      style={{ background: "var(--neu-border)" }}
                    >
                      <Grid3X3
                        style={{
                          width: 24,
                          height: 24,
                          color: "var(--neu-text-muted)",
                          opacity: 0.3,
                        }}
                      />
                    </div>
                  )}
                  <div
                    className="absolute inset-0 flex items-end p-2"
                    style={{
                      background:
                        "linear-gradient(to top, rgba(0,0,0,0.6) 0%, transparent 55%)",
                    }}
                  >
                    <div className="flex items-center gap-2">
                      <div className="flex items-center gap-0.5">
                        <Heart
                          style={{ width: 10, height: 10, color: "white" }}
                          fill="white"
                          strokeWidth={0}
                        />
                        <span style={{ fontSize: 10, color: "white" }}>
                          {moment.likes_count ?? 0}
                        </span>
                      </div>
                      <div className="flex items-center gap-0.5">
                        <MessageCircle
                          style={{ width: 10, height: 10, color: "white" }}
                          strokeWidth={1.5}
                        />
                        <span style={{ fontSize: 10, color: "white" }}>
                          {moment.comments_count ?? 0}
                        </span>
                      </div>
                    </div>
                  </div>
                </motion.div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile / Tablet Profile
══════════════════════════════════════════════════════════════════ */
function ProfileMobile() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const {
    profile,
    moments,
    likedMoments,
    loadingMoments,
    loadingLiked,
    hasMoreMoments,
    loadMoreMoments,
    profileError,
    refreshProfile: refreshMobile,
  } = useProfileData();
  const [activeTab, setActiveTab] = useState<Tab>("posts");

  const isTablet = platform === "tablet";
  const wideMax = isTablet ? 884 : undefined;
  const tabletShellClass = isTablet ? "mx-auto px-0 flex gap-6" : "";
  const tabletShellStyle: React.CSSProperties = wideMax
    ? { maxWidth: wideMax, width: "100%" }
    : {};
  const mainColClass = isTablet ? "mx-auto w-full lg:mx-0 min-w-0" : "w-full";
  const mainColStyle: React.CSSProperties = isTablet ? { maxWidth: 560 } : {};

  const displayName = profile?.nickname ?? "加载中…";
  const displayHandle = profile?.user_id
    ? `@${profile.user_id.slice(0, 8)}`
    : "";
  const stats = profile?.stats;

  const postsMap: Record<Tab, MomentItem[]> = {
    posts: moments,
    liked: likedMoments,
    saved: [],
  };

  return (
    <>
      <div
        className="sticky top-0 z-20 py-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        <div
          className={isTablet ? `${tabletShellClass} px-0` : "px-5"}
          style={tabletShellStyle}
        >
          <div
            className={isTablet ? `${mainColClass} px-5` : "w-full px-0"}
            style={mainColStyle}
          >
            <div className="flex items-center justify-between">
              <div>
                <p style={{ fontSize: 18, color: "var(--neu-text)" }}>
                  {displayName}
                </p>
                <p
                  style={{
                    fontSize: 11,
                    color: "var(--neu-text-muted)",
                    marginTop: 1,
                  }}
                >
                  {displayHandle}
                </p>
              </div>
              <div className="flex gap-2">
                <motion.button
                  whileTap={{ scale: 0.94 }}
                  aria-label="分享主页"
                  className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                  style={{ width: 36, height: 36 }}
                >
                  <Share2
                    style={{
                      width: 15,
                      height: 15,
                      color: "var(--neu-text-muted)",
                    }}
                    strokeWidth={1.5}
                  />
                </motion.button>
                <motion.button
                  whileTap={{ scale: 0.94 }}
                  onClick={() => navigate("/settings")}
                  aria-label="设置"
                  className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                  style={{ width: 36, height: 36 }}
                >
                  <Settings
                    style={{
                      width: 15,
                      height: 15,
                      color: "var(--neu-text-muted)",
                    }}
                    strokeWidth={1.5}
                  />
                </motion.button>
              </div>
            </div>
          </div>
          {isTablet && (
            <div
              className="hidden lg:block flex-shrink-0"
              style={{ width: 300 + 20 }}
            />
          )}
        </div>
      </div>

      {profileError && (
        <div
          className="flex items-center justify-between px-5 py-2"
          style={{ background: "var(--neu-accent-soft)" }}
        >
          <span style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}>
            {profileError}
          </span>
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={refreshMobile}
            className="px-3 py-1 rounded-full outline-none"
            style={{
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
              fontSize: 11,
            }}
          >
            重试
          </motion.button>
        </div>
      )}

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className={
          isTablet
            ? `${tabletShellClass} pb-6 pt-5`
            : "flex flex-col w-full pb-6"
        }
        style={tabletShellStyle}
      >
        <div
          className={
            isTablet ? `${mainColClass} flex flex-col` : "flex flex-col w-full"
          }
          style={mainColStyle}
        >
          <div className={isTablet ? "px-5" : "px-5 pt-5"}>
            <div className="neu-ambient-glow">
              <div
                className="neu-surface-hero rounded-[24px] flex flex-col items-center"
                style={{ padding: "24px 20px 18px" }}
              >
                <div className="relative mb-3">
                  <div
                    className="rounded-full overflow-hidden"
                    style={{
                      width: 92,
                      height: 92,
                      border: "3px solid var(--neu-bg)",
                      boxShadow: "0 6px 18px var(--neu-glow)",
                    }}
                  >
                    <img
                      src={normalizeAvatar(profile?.avatar_url)}
                      alt={displayName}
                      className="w-full h-full object-cover"
                    />
                  </div>
                  <motion.button
                    whileTap={{ scale: 0.94 }}
                    aria-label="更换头像"
                    className="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
                    style={{
                      width: 26,
                      height: 26,
                      background: "var(--neu-accent-primary)",
                      border: "2.5px solid var(--neu-bg)",
                    }}
                  >
                    <Camera
                      style={{
                        width: 12,
                        height: 12,
                        color: "var(--text-on-accent, white)",
                      }}
                      strokeWidth={2}
                    />
                  </motion.button>
                </div>

                <h2 style={{ fontSize: 22, color: "var(--neu-text)" }}>
                  {displayName}
                </h2>
                <span
                  style={{
                    fontSize: 12,
                    color: "var(--neu-text-muted)",
                    marginTop: 2,
                  }}
                >
                  {displayHandle}
                </span>
                <p
                  style={{
                    fontSize: 12,
                    color: "var(--neu-text-muted)",
                    textAlign: "center",
                    padding: "10px 20px 4px",
                    lineHeight: 1.6,
                  }}
                >
                  {profile?.bio || ""}
                </p>
                <div className="flex items-center gap-1 mb-4">
                  <MapPin
                    style={{
                      width: 11,
                      height: 11,
                      color: "var(--neu-accent-primary)",
                    }}
                    strokeWidth={1.5}
                  />
                  <span
                    style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                  >
                    {profile?.location?.location_name ?? ""}
                  </span>
                </div>

                <div className="flex w-full gap-2 mb-5">
                  <motion.button
                    whileTap={{ scale: 0.97 }}
                    onClick={() => navigate("/profile/edit")}
                    className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none"
                    style={{
                      background: "var(--neu-accent-primary)",
                      color: "var(--text-on-accent, white)",
                      fontSize: 13,
                      fontWeight: 500,
                      boxShadow: "0 3px 10px var(--neu-glow)",
                    }}
                  >
                    <Pencil style={{ width: 13, height: 13 }} strokeWidth={2} />
                    编辑资料
                  </motion.button>
                  <motion.button
                    whileTap={{ scale: 0.97 }}
                    className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none transition-colors"
                    style={{
                      background: "transparent",
                      color: "var(--neu-text)",
                      border: "1px solid var(--neu-border-pressed)",
                      fontSize: 13,
                    }}
                    onMouseEnter={(e) =>
                      (e.currentTarget.style.background = "var(--neu-border)")
                    }
                    onMouseLeave={(e) =>
                      (e.currentTarget.style.background = "transparent")
                    }
                  >
                    <Share2
                      style={{ width: 13, height: 13 }}
                      strokeWidth={1.75}
                    />
                    分享主页
                  </motion.button>
                </div>

                <div
                  className="grid grid-cols-3 w-full rounded-[14px] overflow-hidden"
                  style={{
                    background: "var(--neu-border)",
                    gap: 1,
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  {[
                    { label: "帖子", value: stats?.moments_count ?? 0 },
                    { label: "关注者", value: stats?.followers_count ?? 0 },
                    { label: "足迹", value: 0 },
                  ].map((stat) => (
                    <motion.button
                      key={stat.label}
                      whileTap={{ scale: 0.97 }}
                      className="flex flex-col items-center justify-center py-2.5 outline-none transition-colors"
                      style={{ background: "var(--neu-bg)" }}
                      onMouseEnter={(e) =>
                        (e.currentTarget.style.background = "var(--neu-border)")
                      }
                      onMouseLeave={(e) =>
                        (e.currentTarget.style.background = "var(--neu-bg)")
                      }
                    >
                      <span
                        style={{
                          fontSize: 20,
                          color: "var(--neu-text)",
                          fontWeight: 600,
                          lineHeight: 1,
                        }}
                      >
                        {stat.value}
                      </span>
                      <span
                        style={{
                          fontSize: 10,
                          color: "var(--neu-text-muted)",
                          marginTop: 5,
                        }}
                      >
                        {stat.label}
                      </span>
                    </motion.button>
                  ))}
                </div>
              </div>
            </div>
          </div>

          <div className="mt-6">
            <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
            <div className="px-4 mt-3">
              {(activeTab === "liked" ? loadingLiked : loadingMoments) ? (
                <GallerySkeleton />
              ) : postsMap[activeTab].length === 0 ? (
                <EmptyState
                  icon={Grid3X3}
                  title={
                    activeTab === "posts"
                      ? "暂无瞬间"
                      : activeTab === "liked"
                        ? "暂无喜欢"
                        : "暂无收藏"
                  }
                />
              ) : (
                <>
                  <GalleryGrid
                    moments={postsMap[activeTab]}
                    activeTab={activeTab}
                  />
                  {hasMoreMoments && activeTab === "posts" && (
                    <div className="flex justify-center mt-4">
                      <motion.button
                        whileTap={{ scale: 0.93 }}
                        onClick={loadMoreMoments}
                        className="px-4 py-2 rounded-full outline-none"
                        style={{
                          fontSize: 12,
                          color: "var(--neu-accent-primary)",
                          border: "1px solid var(--neu-border)",
                        }}
                      >
                        加载更多
                      </motion.button>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </div>

        {isTablet && (
          <aside
            className="hidden lg:block flex-shrink-0 pr-5"
            style={{ width: 300 + 20 }}
          >
            <div style={{ width: 300 }}>
              <TabletRail>
                <StreakCard />
                <VisitorsCard />
                <MoodCheckinCard />
                <AchievementCard />
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

export function Profile() {
  const { platform } = usePlatform();
  useDocumentTitle("个人中心");
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const navigate = useNavigate();
  useEffect(() => {
    if (!isAuthenticated) {
      navigate("/login", { replace: true });
    }
  }, [isAuthenticated, navigate]);
  if (!isAuthenticated) return null;
  if (platform === "desktop") return <ProfileDesktop />;
  return <ProfileMobile />;
}

function ProfileGroupSheet({
  show,
  onClose,
}: {
  show: boolean;
  onClose: () => void;
}) {
  const trapRef = useFocusTrap(show, onClose);
  return (
    <AnimatePresence>
      {show && (
        <>
          <motion.div
            key="grp-backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-40"
            style={{
              background: "rgba(0,0,0,0.35)",
              backdropFilter: "blur(3px)",
            }}
            onClick={onClose}
          />
          <motion.div
            key="grp-sheet"
            ref={trapRef}
            role="dialog"
            aria-modal="true"
            aria-label="创建群组"
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{
              type: "spring",
              damping: 30,
              stiffness: 260,
              mass: 0.9,
            }}
            className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
            style={{
              height: "92%",
              borderRadius: "28px 28px 0 0",
              backgroundColor: "var(--neu-bg)",
              boxShadow: "0 -8px 40px rgba(0,0,0,0.18)",
            }}
          >
            <CreateGroupSheet onClose={onClose} />
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
