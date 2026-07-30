import React, { useState } from "react";
import {
  Heart, MapPin, Grid3X3, Share2, Bookmark,
  ChevronRight, Camera, Users, MessageCircle, Search,
  Plus, Check, Bell, Clock, Filter, Pencil, UserPlus,
  MoreHorizontal, TrendingUp, Shield, X, ArrowLeft,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useLocation, useNavigate } from "react-router";
import { usePlatform } from "../../contexts/PlatformContext";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { AppearanceMenu } from "../../components/layout/AppearanceMenu";
import { CreateGroupSheet } from "../groups/CreateGroupSheet";
import {
  TabletRail, StreakCard, VisitorsCard, MoodCheckinCard,
  AchievementCard, SuggestedFollowsCard, VibeCard,
} from "../../components/layout/TabletRail";
import { ContextRail } from "../../components/layout/ContextRail";
import { ActionIconButton } from "../../components/shared/ActionIconButton";
import { ROUTES, type ProfileSection } from "../../lib/routes";

/* ── Data hooks ──────────────────────────── */
import {
  useAccountData, useGalleryPosts, useReactedPosts, useFavoritePosts, useMyGroups,
  useRelationships,
} from "./profile.data";

import {
  type Tab, ProfileTabs, GalleryGrid, SectionMoments, SectionGroups, SectionFriends, SectionFollowing, SectionFans,
} from "./ProfileSections";

type RelationshipTab = "friends" | "following" | "followers";

function RelationshipsWorkspace() {
  const location = useLocation();
  const navigate = useNavigate();
  const { friends, following, followers } = useRelationships();
  const tabParam = new URLSearchParams(location.search).get("tab");
  const activeTab: RelationshipTab = tabParam === "following" || tabParam === "followers" ? tabParam : "friends";

  const RELATIONSHIP_TABS: Array<{ id: RelationshipTab; label: string; detail: string; count: number; Icon: React.ElementType }> = [
    { id: "friends", label: "好友", detail: "可以直接对话的双向关系", count: friends.length, Icon: MessageCircle },
    { id: "following", label: "关注", detail: "你主动订阅的动态与创作者", count: following.length, Icon: Heart },
    { id: "followers", label: "粉丝", detail: "关注你的人与待回关关系", count: followers.length, Icon: UserPlus },
  ];
  const activeMeta = RELATIONSHIP_TABS.find((tab) => tab.id === activeTab)!;

  const setTab = (tab: RelationshipTab) => {
    navigate(`${ROUTES.meSection("relationships")}?tab=${tab}`);
  };

  return (
    <div className="flex h-full min-h-0 w-full flex-1 flex-col">
      <div className="flex-shrink-0 border-b border-[var(--neu-border)] bg-[var(--neu-bg)] px-6 pt-5 lg:px-8">
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-6">
          <div>
            <div className="flex items-center gap-2">
              <p className="text-[var(--neu-text)]" style={{ fontSize: 16 }}>关系网络</p>
              <span className="rounded-full px-2 py-0.5 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]" style={{ fontSize: 10, border: "1px solid var(--neu-accent-soft-border)" }}>
                {friends.length + following.length + followers.length} 条关系
              </span>
            </div>
            <p className="mt-1 text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>{activeMeta.detail}</p>
          </div>
          <span className="mt-0.5 whitespace-nowrap text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>{activeMeta.count} 人</span>
        </div>

        <div className="mt-5 flex gap-1" role="tablist" aria-label="关系类型">
          {RELATIONSHIP_TABS.map(({ id, label, count, Icon }) => {
            const selected = activeTab === id;
            return (
              <motion.button
                key={id}
                type="button"
                role="tab"
                aria-selected={selected}
                whileTap={{ scale: 0.96 }}
                onClick={() => setTab(id)}
                className={`relative flex items-center gap-1.5 px-3 pb-3 pt-1 outline-none ${selected ? "text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"}`}
                style={{ fontSize: 13, fontWeight: selected ? 600 : 400 }}
              >
                <Icon style={{ width: 13, height: 13 }} strokeWidth={selected ? 2 : 1.5} />
                {label}
                <span style={{ fontSize: 10, opacity: 0.75 }}>{count}</span>
                {selected && <motion.span layoutId="relationship-active-tab" className="absolute bottom-0 left-2 right-2 h-0.5 rounded-full bg-[var(--neu-accent-primary)]" />}
              </motion.button>
            );
          })}
        </div>
      </div>

      <div className="min-h-0 w-full flex-1 overflow-hidden">
        {activeTab === "friends" && <SectionFriends embedded />}
        {activeTab === "following" && <SectionFollowing embedded />}
        {activeTab === "followers" && <SectionFans embedded />}
      </div>
    </div>
  );
}

function ProfileDesktop() {
  const navigate = useNavigate();
  const location = useLocation();
  const { avatar, nickname, stats } = useAccountData();
  const { posts: myGalleryPosts } = useGalleryPosts("my");
  const { friends, following, followers } = useRelationships();
  const { groups: myGroups } = useMyGroups();
  // /me is the overview; /me/:section resolves an independently addressable resource page.
  const section = location.pathname.startsWith(`${ROUTES.me}/`)
    ? location.pathname.slice(`${ROUTES.me}/`.length)
    : null;

  const SECTION_META: Record<string, { title: string; render: () => React.ReactNode }> = {
    moments:       { title: "我的瞬间", render: () => <SectionMoments /> },
    groups:        { title: "我的群组", render: () => <SectionGroups /> },
    relationships: { title: "关系网络", render: () => <RelationshipsWorkspace /> },
  };

  /* ── 分区全宽模式（有 section 参数）────────────────────────── */
  if (section && SECTION_META[section]) {
    const meta = SECTION_META[section];
    return (
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex-shrink-0 flex items-center gap-3 px-6 py-4 bg-[var(--neu-bg)]"
          style={{ borderBottom: "1px solid var(--neu-border)" }}>
          <ActionIconButton
            icon={ArrowLeft}
            label="返回个人主页"
            onClick={() => navigate(ROUTES.me)}
            size={36}
          />
          <p className="text-[var(--neu-text)]" style={{ fontSize: 18, flex: 1 }}>{meta.title}</p>
        </div>
        {/* 全宽分区内容 */}
        <div className="flex-1 min-h-0 overflow-hidden flex">
          {meta.render()}
        </div>
      </div>
    );
  }

  /* ── 个人主页总览（无 section 参数）────────────────────────── */
  return (
    <div className="flex flex-col h-full">
      {/* Sticky header */}
      <div className="flex-shrink-0 flex items-center justify-between px-6 py-4 bg-[var(--neu-bg)]"
        style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div>
          <p className="text-[var(--neu-text)]" style={{ fontSize: 20 }}>鹿鸣</p>
          <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, marginTop: 1 }}>@elara.v</p>
        </div>
        <div className="flex gap-2">
          <ActionIconButton icon={Share2} label="分享个人主页" size={36} />
        </div>
      </div>

      <div className="flex flex-1 min-h-0">
        {/* ── Left profile panel ─────────────────────────── */}
        <div className="flex-shrink-0 overflow-y-auto no-scrollbar px-6 py-6 flex flex-col items-center"
          style={{ width: 260, borderRight: "1px solid var(--neu-border)" }}>
          {/* Avatar */}
          <div className="relative mb-4">
            <div className="absolute inset-0 rounded-full pointer-events-none bg-[var(--neu-glow)]"
              style={{ filter: "blur(20px)", transform: "scale(1.2)", opacity: 0.5 }} />
            <div className="relative rounded-full bg-[var(--neu-bg)]"
              style={{ width: 96, height: 96, padding: 2, boxShadow: "8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light)", border: "1.5px solid var(--neu-border)" }}>
              <img src={avatar} alt={nickname || "用户"} className="w-full h-full object-cover rounded-full" />
            </div>
            <motion.button whileTap={{ scale: 0.9 }}
              className="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)]"
              style={{ width: 26, height: 26, border: "2.5px solid var(--neu-bg)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
              <Camera style={{ width: 12, height: 12, color: "white" }} strokeWidth={2} />
            </motion.button>
          </div>

          <h2 className="text-[var(--neu-text)]" style={{ fontSize: 18, textAlign: "center" }}>鹿鸣</h2>
          <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, marginTop: 2 }}>@elara.v</span>
          <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, textAlign: "center", marginTop: 8, lineHeight: 1.6, paddingLeft: 8, paddingRight: 8 }}>
            触感思考者 · 在流动中寻觅静止
          </p>
          <div className="flex items-center gap-1 mt-2 mb-4">
            <MapPin style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
            <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>江苏·徐州</span>
          </div>

          {/* Edit profile button */}
          <motion.button whileTap={{ scale: 0.96 }} onClick={() => navigate(ROUTES.meEdit)}
            className="w-full py-2.5 rounded-full outline-none mb-5 flex items-center justify-center gap-2 bg-[var(--neu-bg)] text-[var(--neu-text-muted)]"
            style={{ border: "1px solid var(--neu-border)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", fontSize: 13 }}>
            <Pencil style={{ width: 13, height: 13 }} strokeWidth={1.5} />
            编辑资料
          </motion.button>

          {/* Stats */}
          <div className="w-full rounded-[18px] mb-4 bg-[var(--neu-bg)]"
            style={{ boxShadow: "0 0 0 1px var(--neu-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)" }}>
            <div style={{ overflow: "hidden", borderRadius: 17 }}>
            {[
              { label: "瞬间", value: stats.moments_count ?? 0, section: "moments" as ProfileSection },
              { label: "关注者", value: stats.followers_count ?? 0, section: "followers" as const },
              { label: "关注", value: stats.following_count ?? 0, section: "following" as ProfileSection },
            ].map((stat, i) => (
              <React.Fragment key={stat.label}>
                {i > 0 && <div className="bg-[var(--neu-border)]" style={{ height: 1 }} />}
                <motion.button whileTap={{ scale: 0.97 }}
                  onClick={() => navigate(stat.section === "followers" ? `${ROUTES.meSection("relationships")}?tab=followers` : ROUTES.meSection(stat.section as ProfileSection))}
                  className="w-full flex items-center justify-between px-4 py-3 outline-none"
                  style={{ cursor: "pointer" }}>
                  <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 13 }}>{stat.label}</span>
                  <div className="flex items-center gap-1.5">
                    <span className="text-[var(--neu-text)]" style={{ fontSize: 17 }}>{stat.value}</span>
                    <ChevronRight style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                  </div>
                </motion.button>
              </React.Fragment>
            ))}
            </div>
          </div>
        </div>

          {/* Section quick links */}
          <div className="w-full rounded-[18px] bg-[var(--neu-bg)]"
            style={{ boxShadow: "0 0 0 1px var(--neu-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)" }}>
            <div style={{ overflow: "hidden", borderRadius: 17 }}>
            {[
              { label: "我的瞬间",  section: "moments" as ProfileSection, icon: Grid3X3, count: myGalleryPosts.length },
              { label: "我的群组",  section: "groups" as ProfileSection,    icon: Users,   count: myGroups.length },
              { label: "关系网络",  section: "relationships" as ProfileSection, icon: Heart, count: friends.length + following.length + followers.length },
            ].map((item, i) => (
              <React.Fragment key={item.label}>
                {i > 0 && <div className="bg-[var(--neu-border)]" style={{ height: 1 }} />}
                <motion.button whileTap={{ scale: 0.97 }}
                  onClick={() => navigate(ROUTES.meSection(item.section))}
                  className="w-full flex items-center justify-between px-4 py-2.5 outline-none">
                  <div className="flex items-center gap-2.5">
                    <item.icon style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                    <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 13 }}>{item.label}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>{item.count}</span>
                    <ChevronRight style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                  </div>
                </motion.button>
              </React.Fragment>
            ))}
          </div>
        </div>

        {/* ── Right: recent moments preview ──────────────────── */}
        <div className="flex-1 overflow-y-auto no-scrollbar px-6 py-5">
          <div className="flex items-center justify-between mb-4">
            <p className="text-[var(--neu-text)]" style={{ fontSize: 15 }}>最近瞬间</p>
            <motion.button whileTap={{ scale: 0.93 }} onClick={() => navigate(ROUTES.meSection("moments"))}
              className="flex items-center gap-1 outline-none text-[var(--neu-accent-primary)]"
              style={{ fontSize: 12 }}>
              查看全部
              <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={2} />
            </motion.button>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 10, maxWidth: 3 * 240 + 2 * 10 }}>
            {myGalleryPosts.slice(0, 9).map(post => (
              <motion.div key={post.id} whileTap={{ scale: 0.97 }}
                className="relative rounded-[16px] cursor-pointer"
                style={{ aspectRatio: "1/1", boxShadow: "0 0 0 1px var(--neu-border), 5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)" }}>
                <div className="relative h-full w-full" style={{ overflow: "hidden", borderRadius: 15 }}>
                  <img src={post.img} alt="" className="w-full h-full object-cover" />
                  <div className="absolute inset-0 flex items-end p-2"
                    style={{ background: "linear-gradient(to top, rgba(0,0,0,0.6) 0%, transparent 55%)" }}>
                    <div className="flex items-center gap-2">
                      <div className="flex items-center gap-0.5">
                        <Heart style={{ width: 10, height: 10, color: "white" }} fill="white" strokeWidth={0} />
                        <span className="text-white" style={{ fontSize: 10 }}>{post.likes}</span>
                      </div>
                      <div className="flex items-center gap-0.5">
                        <MessageCircle style={{ width: 10, height: 10, color: "white" }} strokeWidth={1.5} />
                        <span className="text-white" style={{ fontSize: 10 }}>{post.comments}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </motion.div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   HarmonyOS Profile
══════════════════════════════════════════════════════════════════ */
function ShareProfileBtn() {
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  return (
    <motion.button whileTap={{ scale: 0.97 }}
      onClick={() => {}}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none text-[var(--neu-text)]"
      style={{
        background: hovered ? hover.bgHover : "transparent",
        border: "1px solid var(--neu-border-pressed)",
        fontSize: 13,
        transition: "background 0.18s ease",
      }}>
      <Share2 style={{ width: 13, height: 13 }} strokeWidth={1.75} />
      分享主页
    </motion.button>
  );
}

function ProfileStatBtn({ label, value, onClick }: { label: string; value: string; onClick?: () => void }) {
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  return (
    <motion.button whileTap={{ scale: 0.97 }}
      onClick={onClick}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      className="flex flex-col items-center justify-center py-2.5 outline-none"
      style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <span className="text-[var(--neu-text)]" style={{ fontSize: 20, fontWeight: 600, lineHeight: 1 }}>{value}</span>
      <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 10, marginTop: 5 }}>{label}</span>
    </motion.button>
  );
}

function ProfileMobile() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const { avatar, nickname, stats } = useAccountData();
  const { posts: myPosts } = useGalleryPosts("my");
  const { posts: reacted } = useReactedPosts();
  const { posts: savedP } = useFavoritePosts();
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  const postsMap: Record<Tab, any> = { posts: myPosts, liked: reacted, saved: savedP };

  const isTablet = platform === "tablet";
  // On tablet:
  //   - narrow (< lg / 1024): main column max-w-560 centered, no rail
  //   - wide  (>= lg):        main 560 + gap 24 + rail 300 = 884, left-aligned within 884 container
  // On mobile: containerStyle is empty (full width).
  const wideMax = isTablet ? 884 : undefined;
  const tabletShellClass = isTablet ? "mx-auto px-0 flex gap-6" : "";
  const tabletShellStyle: React.CSSProperties = wideMax
    ? { maxWidth: wideMax, width: "100%" }
    : {};
  const mainColClass = isTablet ? "mx-auto w-full lg:mx-0 min-w-0" : "w-full";
  const mainColStyle: React.CSSProperties = isTablet
    ? { maxWidth: 560 }
    : {};
  // Sticky header content — same width contract as shell so header bar aligns
  const containerStyle: React.CSSProperties = isTablet
    ? { maxWidth: 560, marginLeft: "auto", marginRight: "auto", width: "100%" }
    : {};

  return (
    <>
      <div className="sticky top-0 z-20 py-3 bg-[var(--neu-bg)]"
        style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className={isTablet ? `${tabletShellClass} px-0` : "px-5"} style={tabletShellStyle}>
          <div className={isTablet ? `${mainColClass} px-5` : "w-full px-0"}
            style={mainColStyle}>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-[var(--neu-text)]" style={{ fontSize: 18 }}>鹿鸣</p>
                <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, marginTop: 1 }}>@elara.v</p>
              </div>
              <div className="flex gap-2">
                <ActionIconButton icon={Share2} label="分享个人主页" size={36} />
                <AppearanceMenu />
              </div>
            </div>
          </div>
          {/* Spacer matching rail width on wide tablet so header stretches edge-to-edge */}
          {isTablet && <div className="hidden lg:block flex-shrink-0" style={{ width: "var(--context-rail-width)" }} />}
        </div>
      </div>

      <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className={isTablet ? `${tabletShellClass} pb-6 pt-5` : "flex flex-col w-full pb-6"}
        style={tabletShellStyle}>

        {/* ───── Main column ───── */}
        <div className={isTablet ? `${mainColClass} flex flex-col` : "flex flex-col w-full"}
          style={mainColStyle}>

        {/* Hero identity card — single cohesive surface with ambient glow */}
        <div className={isTablet ? "px-5" : "px-5 pt-5"}>
          <div className="neu-ambient-glow">
            <div className="neu-surface-hero rounded-[24px] flex flex-col items-center"
              style={{ padding: "24px 20px 18px" }}>

              {/* Avatar */}
              <div className="relative mb-3">
                <div className="rounded-full overflow-hidden"
                  style={{
                    width: 92, height: 92,
                    border: "3px solid var(--neu-bg)",
                    boxShadow: "0 6px 18px var(--neu-glow)",
                  }}>
                  {avatar ? <img src={avatar} alt={nickname || "用户"} className="w-full h-full object-cover" /> : <span className="text-[var(--neu-text-muted)]" style={{fontSize:32}}>{(nickname||"?")[0]}</span>}
                </div>
                <motion.button whileTap={{ scale: 0.9 }}
                  className="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none bg-[var(--neu-accent-primary)]"
                  style={{ width: 26, height: 26, border: "2.5px solid var(--neu-bg)" }}>
                  <Camera style={{ width: 12, height: 12, color: "white" }} strokeWidth={2} />
                </motion.button>
              </div>

              {/* Identity */}
              <h2 className="text-[var(--neu-text)]" style={{ fontSize: 22 }}>鹿鸣</h2>
              <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, marginTop: 2 }}>@elara.v</span>
              <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, textAlign: "center", padding: "10px 20px 4px", lineHeight: 1.6 }}>
                触感思考者 · 在流动中寻觅静止
              </p>
              <div className="flex items-center gap-1 mb-4">
                <MapPin style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>江苏·徐州</span>
              </div>

              {/* Dual action — equal weight, ghost on neutral surface */}
              <div className="flex w-full gap-2 mb-5">
                <motion.button whileTap={{ scale: 0.97 }} onClick={() => navigate(ROUTES.meEdit)}
                  className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none bg-[var(--neu-accent-primary)] text-white"
                  style={{
                    fontSize: 13,
                    fontWeight: 500,
                    boxShadow: "0 3px 10px var(--neu-glow)",
                  }}>
                  <Pencil style={{ width: 13, height: 13 }} strokeWidth={2} />
                  编辑资料
                </motion.button>
                <ShareProfileBtn />
              </div>

              {/* Stats — flat, integrated INTO the hero card (not a separate elevated bar) */}
              <div className="grid grid-cols-3 w-full rounded-[14px] bg-[var(--neu-border)]"
                style={{ gap: 1, boxShadow: "0 0 0 1px var(--neu-border)" }}>
                <div style={{ overflow: "hidden", borderRadius: 13 }}>
                {[
                  { label: "帖子", value: "48" },
                  { label: "关注者", value: "312" },
                  { label: "足迹", value: "27" },
                ].map(stat => (
                  <ProfileStatBtn
                    key={stat.label}
                    label={stat.label}
                    value={stat.value}
                    onClick={stat.label === "关注者" ? () => navigate(`${ROUTES.meSection("relationships")}?tab=followers`) : undefined}
                  />
                ))}
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Tabs + content — keep aligned with hero on tablet */}
        <div className="mt-6">
          <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
          <div className="px-4 mt-3">
            <GalleryGrid posts={postsMap[activeTab]} activeTab={activeTab} />
          </div>
        </div>
        </div>

        {/* ───── Right rail — tablet landscape only ───── */}
        {isTablet && (
          <ContextRail className="hidden lg:block">
              <TabletRail>
                <StreakCard />
                <VisitorsCard />
                <MoodCheckinCard />
                <AchievementCard />
                <SuggestedFollowsCard />
                <VibeCard />
              </TabletRail>
          </ContextRail>
        )}
      </motion.div>
    </>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Entry
══════════════════════════════════════════════════════════════════ */
export function ProfilePage() {
  const { platform } = usePlatform();
  const location = useLocation();
  // Resource sections are independently addressable on every device. Reuse the
  // full section view rather than silently falling back to the profile overview.
  if (location.pathname.startsWith(`${ROUTES.me}/`)) return <ProfileDesktop />;
  if (platform === "desktop") return <ProfileDesktop />;
  return <ProfileMobile />;
}
