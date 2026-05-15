import React, { useState } from "react";
import {
  Heart, MapPin, Grid3X3, Settings, Share2, Bookmark,
  ChevronRight, Camera, Users, MessageCircle, Search,
  Plus, Check, Bell, Clock, Filter, Pencil, UserPlus,
  MoreHorizontal, TrendingUp, Shield, X, ArrowLeft, LogOut,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate, useSearchParams } from "react-router";
import { usePlatform } from "../contexts/PlatformContext";
import { useAuth } from "../contexts/AuthContext";
import { useHoverConfig } from "../hooks/useHoverConfig";
import { CreateGroupSheet } from "../components/ui/CreateGroupSheet";
import {
  TabletRail, StreakCard, VisitorsCard, MoodCheckinCard,
  AchievementCard, SuggestedFollowsCard, VibeCard,
} from "../components/TabletRail";

/* ── Constants ──────────────────────────────────────────────────── */
const GALLERY_POSTS = [
  { id: 1, img: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 124, comments: 18 },
  { id: 2, img: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 89, comments: 7 },
  { id: 3, img: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 312, comments: 44 },
  { id: 4, img: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 56, comments: 3 },
  { id: 5, img: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 201, comments: 28 },
  { id: 6, img: "https://images.unsplash.com/photo-1758613172074-7275c3455034?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 77, comments: 9 },
  { id: 7, img: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 143, comments: 21 },
  { id: 8, img: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 98, comments: 12 },
  { id: 9, img: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 65, comments: 5 },
];

const LIKED_POSTS = [...GALLERY_POSTS].reverse();
const SAVED_POSTS = GALLERY_POSTS.slice(0, 6);

const STATS = [
  { label: "帖子", value: "48", section: "moments" },
  { label: "关注者", value: "312", section: "fans" },
  { label: "足迹", value: "27", section: null },
];

const MOCK_GROUPS = [
  { id: 1, name: "摄影爱好者", members: 128, online: 14, avatar: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=100", banner: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=400", desc: "记录生活中每一个美好瞬间", role: "管理员", lastActive: "2分钟前" },
  { id: 2, name: "徐州骑行圈", members: 256, online: 31, avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=100", banner: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=400", desc: "用骑行丈量城市的每一条街道", role: "成员", lastActive: "1小时前" },
  { id: 3, name: "城市探索组", members: 89, online: 8, avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=100", banner: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=400", desc: "发现城市中隐藏的惊喜角落", role: "成员", lastActive: "3小时前" },
  { id: 4, name: "美食打卡群", members: 412, online: 67, avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", banner: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400", desc: "跟着味蕾去旅行，探索每一道美食", role: "管理员", lastActive: "刚刚" },
];

const MOCK_FRIENDS = [
  { id: 1, name: "Liam Chen", handle: "@liam.c", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", online: true, mutual: 12, tag: "摄影" },
  { id: 2, name: "Sophia Wang", handle: "@sophia.w", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", online: true, mutual: 8, tag: "旅行" },
  { id: 3, name: "Marco Li", handle: "@marco.li", avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100", online: false, mutual: 5, tag: "音乐" },
  { id: 4, name: "Aiko Tanaka", handle: "@aiko.t", avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=100", online: true, mutual: 3, tag: "美食" },
  { id: 5, name: "Noah Park", handle: "@noah.p", avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=100", online: false, mutual: 15, tag: "户外" },
  { id: 6, name: "Elena Zhao", handle: "@elena.z", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", online: true, mutual: 6, tag: "艺术" },
];

const MOCK_FOLLOWING = [
  { id: 1, name: "Studio Ghibli Art", handle: "@ghibli_art", avatar: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=100", desc: "独立艺术工作室 · 动画爱好者", posts: 234, latestImg: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=80" },
  { id: 2, name: "Urban Explorer", handle: "@urban_exp", avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=100", desc: "城市探索达人 · 街头摄影师", posts: 189, latestImg: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=80" },
  { id: 3, name: "Foodie Diary", handle: "@foodie_d", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", desc: "美食博主 · 环球美食探索", posts: 412, latestImg: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
  { id: 4, name: "Photo Master", handle: "@photo_m", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", desc: "风光摄影师 · 全球旅行", posts: 876, latestImg: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
  { id: 5, name: "City Nights", handle: "@city_nights", avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100", desc: "夜拍爱好者 · 城市光影", posts: 156, latestImg: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" },
];

const MOCK_FANS = [
  { id: 1, name: "Luna Star", handle: "@luna.s", avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=100", followBack: false, joinedDays: 2 },
  { id: 2, name: "Kai Zhang", handle: "@kai.z", avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100", followBack: true, joinedDays: 5 },
  { id: 3, name: "Mia Song", handle: "@mia.song", avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=100", followBack: false, joinedDays: 14 },
  { id: 4, name: "Yuki Sato", handle: "@yuki.s", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", followBack: true, joinedDays: 30 },
  { id: 5, name: "Daniel Wu", handle: "@daniel.w", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", followBack: false, joinedDays: 7 },
  { id: 6, name: "Emily Roe", handle: "@emily.r", avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=100", followBack: false, joinedDays: 1 },
  { id: 7, name: "James Lin", handle: "@james.l", avatar: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=100", followBack: true, joinedDays: 60 },
];

/* ── Shared: Tab bar ──────────────────────────────────────────────── */
type Tab = "posts" | "liked" | "saved";
function ProfileTabs({ activeTab, setActiveTab }: { activeTab: Tab; setActiveTab: (t: Tab) => void }) {
  const tabs: { id: Tab; Icon: React.ElementType; label: string }[] = [
    { id: "posts", Icon: Grid3X3, label: "帖子" },
    { id: "liked", Icon: Heart, label: "喜欢" },
    { id: "saved", Icon: Bookmark, label: "收藏" },
  ];
  return (
    <div className="flex" style={{ borderBottom: "1px solid var(--neu-border)", paddingLeft: 16, paddingRight: 16 }}>
      {tabs.map(({ id, Icon, label }) => {
        const isActive = activeTab === id;
        return (
          <motion.button key={id} whileTap={{ scale: 0.94 }} onClick={() => setActiveTab(id)}
            className="relative flex items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none"
            style={{ flex: 1, color: isActive ? "var(--neu-accent-primary)" : "var(--neu-text-muted)", transition: "color 0.2s", fontSize: 13, fontWeight: isActive ? 500 : 400 }}>
            <Icon style={{ width: 14, height: 14 }} strokeWidth={isActive ? 2 : 1.5} />
            <span>{label}</span>
            {isActive && (
              <motion.div layoutId="profile-tab-indicator"
                style={{ position: "absolute", bottom: -1, left: "18%", right: "18%", height: 2, borderRadius: 1, background: "var(--neu-accent-primary)" }}
                transition={{ type: "spring", stiffness: 420, damping: 34 }} />
            )}
          </motion.button>
        );
      })}
    </div>
  );
}

/* ── Shared: Gallery grid (mobile/harmony) ────────────────────────── */
function GalleryGrid({ posts, activeTab, cols = 2 }: { posts: typeof GALLERY_POSTS; activeTab: Tab; cols?: number }) {
  const { platform } = usePlatform();
  const numCols = platform === "tablet" ? 3 : 2;
  const gap = 10;
  return (
    <AnimatePresence mode="wait">
      <motion.div key={activeTab}
        initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -4 }}
        transition={{ duration: 0.2 }}
        style={{ display: "grid", gridTemplateColumns: `repeat(${numCols}, 1fr)`, gap, width: "100%" }}>
        {posts.map((post) => (
          <motion.div key={post.id} whileTap={{ scale: 0.97 }}
            className="relative rounded-[18px] overflow-hidden cursor-pointer group"
            style={{ aspectRatio: "1/1", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <img src={post.img} alt="" className="w-full h-full object-cover" />
            <div className="absolute inset-0 flex items-end p-2"
              style={{ background: "linear-gradient(to top, rgba(0,0,0,0.55) 0%, transparent 55%)" }}>
              <div className="flex items-center gap-2">
                <div className="flex items-center gap-0.5">
                  <Heart style={{ width: 10, height: 10, color: "white" }} fill="white" strokeWidth={0} />
                  <span style={{ fontSize: 10, color: "white" }}>{post.likes}</span>
                </div>
                <div className="flex items-center gap-0.5">
                  <MessageCircle style={{ width: 10, height: 10, color: "white" }} strokeWidth={1.5} />
                  <span style={{ fontSize: 10, color: "white" }}>{post.comments}</span>
                </div>
              </div>
            </div>
          </motion.div>
        ))}
      </motion.div>
    </AnimatePresence>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 我的瞬间
══════════════════════════════════════════════════════════════════ */
function SectionMoments() {
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  const postsMap: Record<Tab, typeof GALLERY_POSTS> = { posts: GALLERY_POSTS, liked: LIKED_POSTS, saved: SAVED_POSTS };

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      {/* Section header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>我的瞬间</span>
          <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "var(--neu-accent-primary)", color: "white" }}>
            {GALLERY_POSTS.length} 帖子
          </span>
        </div>
        <motion.button whileTap={{ scale: 0.93 }}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none"
          style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 4px 12px var(--neu-glow)" }}>
          <Plus style={{ width: 13, height: 13 }} strokeWidth={2.5} />
          发布新瞬间
        </motion.button>
      </div>

      {/* Tab bar */}
      <div className="flex-shrink-0">
        <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      {/* Gallery */}
      <div style={{ padding: "14px 16px 24px" }}>
        <AnimatePresence mode="wait">
          <motion.div key={activeTab}
            initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.2 }}
            style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 10, width: "100%" }}>
            {postsMap[activeTab].map((post) => (
              <motion.div key={post.id} whileTap={{ scale: 0.97 }}
                className="relative rounded-[16px] overflow-hidden cursor-pointer"
                style={{ aspectRatio: "1/1", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                <img src={post.img} alt="" className="w-full h-full object-cover" />
                <div className="absolute inset-0 flex items-end p-2"
                  style={{ background: "linear-gradient(to top, rgba(0,0,0,0.6) 0%, transparent 55%)" }}>
                  <div className="flex items-center gap-2">
                    <div className="flex items-center gap-0.5">
                      <Heart style={{ width: 10, height: 10, color: "white" }} fill="white" strokeWidth={0} />
                      <span style={{ fontSize: 10, color: "white" }}>{post.likes}</span>
                    </div>
                    <div className="flex items-center gap-0.5">
                      <MessageCircle style={{ width: 10, height: 10, color: "white" }} strokeWidth={1.5} />
                      <span style={{ fontSize: 10, color: "white" }}>{post.comments}</span>
                    </div>
                  </div>
                </div>
              </motion.div>
            ))}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 我的群组
══════════════════════════════════════════════════════════════════ */
function SectionGroups() {
  const [searchQ, setSearchQ] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const hover = useHoverConfig();
  const filtered = MOCK_GROUPS.filter(g =>
    searchQ === "" || g.name.toLowerCase().includes(searchQ.toLowerCase())
  );

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      {/* Section header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>我的群组</span>
          <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "var(--neu-accent-primary)", color: "white" }}>
            {MOCK_GROUPS.length} 个
          </span>
        </div>
        <motion.button whileTap={hover.tap as any} whileHover={hover.btn as any}
          onClick={() => setShowCreate(true)}
          className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none"
          style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 4px 12px var(--neu-glow)" }}>
          <Plus style={{ width: 13, height: 13 }} strokeWidth={2.5} />
          创建群组
        </motion.button>
      </div>

      {/* Search */}
      <div className="px-5 py-3 flex-shrink-0">
        <div className="flex items-center gap-2.5 px-4 rounded-[14px] neu-input-surface"
          style={{ height: 40 }}>
          <Search style={{ width: 13, height: 13, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input value={searchQ} onChange={e => setSearchQ(e.target.value)}
            placeholder="搜索群组…" className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }} />
          {searchQ && <motion.button whileTap={{ scale: 0.85 }} onClick={() => setSearchQ("")} className="outline-none">
            <X style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={2} />
          </motion.button>}
        </div>
      </div>

      {/* Group cards */}
      <div className="flex flex-col gap-3 px-5 pb-6">
        {filtered.map(g => (
          <motion.div key={g.id}
            whileTap={hover.tap as any}
            whileHover={hover.card as any}
            className="rounded-[20px] overflow-hidden cursor-pointer neu-card-hover"
            style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            {/* Banner */}
            <div className="relative" style={{ height: "clamp(60px, 10vw, 80px)" }}>
              <img src={g.banner} alt={g.name} className="w-full h-full object-cover" />
              <div className="absolute inset-0" style={{ background: "linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.6) 100%)" }} />
              <div className="absolute bottom-2 left-3 flex items-center gap-1.5">
                <span className="px-2 py-0.5 rounded-full"
                  style={{ fontSize: 10, background: g.role === "管理员" ? "rgba(99,102,241,0.9)" : "rgba(0,0,0,0.5)", color: "white", backdropFilter: "blur(8px)" }}>
                  {g.role === "管理员" && <Shield style={{ width: 8, height: 8, display: "inline", marginRight: 2 }} strokeWidth={2} />}
                  {g.role}
                </span>
              </div>
              <div className="absolute bottom-2 right-3 flex items-center gap-1">
                <div style={{ width: 7, height: 7, borderRadius: "50%", background: "#22c55e" }} />
                <span style={{ fontSize: 10, color: "white" }}>{g.online} 在线</span>
              </div>
            </div>
            {/* Info */}
            <div className="flex items-center gap-3 p-3.5">
              <img src={g.avatar} alt={g.name} className="rounded-[12px] object-cover flex-shrink-0"
                style={{ width: 44, height: 44, border: "2px solid var(--neu-bg)", marginTop: -22, boxShadow: "0 4px 12px rgba(0,0,0,0.15)" }} />
              <div className="flex-1 min-w-0">
                <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{g.name}</p>
                <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 2 }}>{g.desc}</p>
              </div>
              <div className="flex flex-col items-end gap-1.5 flex-shrink-0">
                <div className="flex items-center gap-1">
                  <Users style={{ width: 11, height: 11, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                  <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{g.members}</span>
                </div>
                <motion.button whileTap={{ scale: 0.93 }}
                  className="flex items-center gap-1 px-2.5 py-1 rounded-full outline-none"
                  style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 11, boxShadow: "0 3px 8px var(--neu-glow)" }}>
                  进入
                  <ChevronRight style={{ width: 10, height: 10 }} strokeWidth={2} />
                </motion.button>
              </div>
            </div>
            {/* Last active */}
            <div className="px-3.5 pb-3 flex items-center gap-1.5" style={{ borderTop: "1px solid var(--neu-border)", paddingTop: 8, marginTop: -4 }}>
              <Clock style={{ width: 11, height: 11, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>最近活跃 {g.lastActive}</span>
            </div>
          </motion.div>
        ))}
      </div>

      <AnimatePresence>
        {showCreate && (
          <>
            <motion.div key="grp-backdrop" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-40"
              style={{ background: "rgba(0,0,0,0.35)", backdropFilter: "blur(3px)" }}
              onClick={() => setShowCreate(false)} />
            <motion.div key="grp-sheet"
              initial={{ y: "100%" }} animate={{ y: 0 }} exit={{ y: "100%" }}
              transition={{ type: "spring", damping: 30, stiffness: 260, mass: 0.9 }}
              className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
              style={{ height: "92%", borderRadius: "28px 28px 0 0", backgroundColor: "var(--neu-bg)", boxShadow: "0 -8px 40px rgba(0,0,0,0.18)" }}>
              <CreateGroupSheet onClose={() => setShowCreate(false)} />
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 好友
══════════════════════════════════════════════════════════════════ */
function SectionFriends() {
  const [searchQ, setSearchQ] = useState("");
  const [activeFilter, setActiveFilter] = useState<"all" | "online" | "offline">("all");

  const filtered = MOCK_FRIENDS
    .filter(f => activeFilter === "all" || (activeFilter === "online" ? f.online : !f.online))
    .filter(f => searchQ === "" || f.name.toLowerCase().includes(searchQ.toLowerCase()) || f.handle.includes(searchQ));

  const onlineCount = MOCK_FRIENDS.filter(f => f.online).length;

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>好友</span>
          <div className="flex items-center gap-1 px-2 py-0.5 rounded-full"
            style={{ fontSize: 11, background: "#22c55e20", border: "1px solid #22c55e30" }}>
            <div style={{ width: 6, height: 6, borderRadius: "50%", background: "#22c55e" }} />
            <span style={{ color: "#22c55e" }}>{onlineCount} 在线</span>
          </div>
        </div>
        <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{MOCK_FRIENDS.length} 位好友</span>
      </div>

      {/* Search */}
      <div className="px-5 pt-3 pb-2 flex-shrink-0">
        <div className="flex items-center gap-2.5 px-4 rounded-[14px] neu-input-surface"
          style={{ height: 40 }}>
          <Search style={{ width: 13, height: 13, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
          <input value={searchQ} onChange={e => setSearchQ(e.target.value)}
            placeholder="搜索好友…" className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }} />
          {searchQ && <motion.button whileTap={{ scale: 0.85 }} onClick={() => setSearchQ("")} className="outline-none">
            <X style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={2} />
          </motion.button>}
        </div>
      </div>

      {/* Filter pills */}
      <div className="flex gap-2 px-5 pb-3 flex-shrink-0">
        {([["all", "全部"], ["online", "在线"], ["offline", "离线"]] as const).map(([id, label]) => (
          <motion.button key={id} whileTap={{ scale: 0.93 }} onClick={() => setActiveFilter(id)}
            className="px-3 py-1.5 rounded-full outline-none"
            style={activeFilter === id
              ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 3px 8px var(--neu-glow)" }
              : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)" }
            }>{label}</motion.button>
        ))}
      </div>

      {/* Friend list */}
      <div className="flex flex-col gap-3 px-5 pb-6">
        {filtered.map(f => (
          <motion.div key={f.id} className="flex items-center gap-3 p-3.5 rounded-[18px]"
            style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <div className="relative flex-shrink-0">
              <img src={f.avatar} alt={f.name} className="rounded-full object-cover" style={{ width: 48, height: 48 }} />
              <span className="absolute bottom-0 right-0 rounded-full"
                style={{ width: 12, height: 12, background: f.online ? "#22c55e" : "var(--neu-text-muted)", border: "2.5px solid var(--neu-bg)", opacity: f.online ? 1 : 0.5 }} />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span style={{ fontSize: 14, color: "var(--neu-text)" }}>{f.name}</span>
                <span className="px-1.5 py-0.5 rounded-[6px]"
                  style={{ fontSize: 10, background: "var(--neu-accent-primary)18", color: "var(--neu-accent-primary)" }}>
                  {f.tag}
                </span>
              </div>
              <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>
                {f.handle} · {f.mutual} 共同好友
              </p>
            </div>
            <div className="flex flex-col gap-1.5 flex-shrink-0">
              <motion.button whileTap={{ scale: 0.93 }}
                className="flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none"
                style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 11, boxShadow: "0 3px 8px var(--neu-glow)" }}>
                <MessageCircle style={{ width: 11, height: 11 }} strokeWidth={2} />
                发消息
              </motion.button>
              <motion.button whileTap={{ scale: 0.93 }}
                className="flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none"
                style={{ background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 11, border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)" }}>
                查看主页
              </motion.button>
            </div>
          </motion.div>
        ))}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 关注
══════════════════════════════════════════════════════════════════ */
function SectionFollowing() {
  const [sortBy, setSortBy] = useState<"recent" | "active">("recent");
  const [unfollowedIds, setUnfollowedIds] = useState<Set<number>>(new Set());

  const toggleUnfollow = (id: number) => {
    setUnfollowedIds(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const sorted = [...MOCK_FOLLOWING].sort((a, b) =>
    sortBy === "active" ? b.posts - a.posts : 0
  );

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>关注</span>
          <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "var(--neu-accent-primary)", color: "white" }}>
            {MOCK_FOLLOWING.length} 人
          </span>
        </div>
        <motion.button whileTap={{ scale: 0.93 }}
          className="flex items-center gap-1 px-3 py-1 rounded-full outline-none neu-surface-ghost"
          style={{ color: "var(--neu-text-muted)", fontSize: 12, borderColor: "var(--neu-border-pressed)" }}>
          <Filter style={{ width: 11, height: 11 }} strokeWidth={1.5} />
          管理
        </motion.button>
      </div>

      {/* Sort tabs — ghost chips */}
      <div className="flex gap-1.5 px-5 py-3 flex-shrink-0">
        {([["recent", "最近关注"], ["active", "最活跃"]] as const).map(([id, label]) => {
          const active = sortBy === id;
          return (
            <motion.button key={id} whileTap={{ scale: 0.94 }} onClick={() => setSortBy(id)}
              className="flex items-center gap-1.5 px-3 py-1 rounded-full outline-none neu-surface-ghost"
              style={active
                ? {
                    background: "var(--neu-accent-soft)",
                    color: "var(--neu-accent-primary)",
                    fontSize: 12,
                    fontWeight: 500,
                    borderColor: "var(--neu-accent-soft-border)",
                  }
                : { fontSize: 12, color: "var(--neu-text-muted)" }}>
              {id === "active" && <TrendingUp style={{ width: 11, height: 11 }} strokeWidth={active ? 2 : 1.5} />}
              {label}
            </motion.button>
          );
        })}
      </div>

      {/* Following list — flat joined rows */}
      <div className="px-5 pb-6">
        <div className="rounded-[16px] overflow-hidden flex flex-col"
          style={{ gap: 1, background: "var(--neu-border)", border: "1px solid var(--neu-border)" }}>
          {sorted.map(f => {
            const isUnfollowed = unfollowedIds.has(f.id);
            return (
              <div key={f.id}
                className="flex items-center gap-3 p-3 transition-colors"
                style={{ background: "var(--neu-bg)", opacity: isUnfollowed ? 0.55 : 1 }}
                onMouseEnter={e => (e.currentTarget.style.background = "var(--neu-border)")}
                onMouseLeave={e => (e.currentTarget.style.background = "var(--neu-bg)")}>
                <img src={f.avatar} alt={f.name} className="rounded-full object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
                <div className="flex-1 min-w-0">
                  <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{f.name}</p>
                  <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }} className="truncate">{f.desc}</p>
                  <p style={{ fontSize: 10, color: "var(--neu-accent-primary)", marginTop: 1 }}>{f.posts} 帖子</p>
                </div>
                <img src={f.latestImg} alt="" className="rounded-[8px] object-cover flex-shrink-0"
                  style={{ width: 36, height: 36, opacity: 0.85 }} />
                <motion.button whileTap={{ scale: 0.93 }} onClick={() => toggleUnfollow(f.id)}
                  className="flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0"
                  style={isUnfollowed
                    ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 11, border: "1px solid transparent" }
                    : { background: "transparent", color: "var(--neu-text-muted)", fontSize: 11, border: "1px solid var(--neu-border-pressed)" }
                  }>
                  {isUnfollowed ? (
                    <><UserPlus style={{ width: 10, height: 10 }} strokeWidth={2} />关注</>
                  ) : (
                    <><Check style={{ width: 10, height: 10 }} strokeWidth={2.5} />已关注</>
                  )}
                </motion.button>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 粉丝
══════════════════════════════════════════════════════════════════ */
function SectionFans() {
  const [fanFilter, setFanFilter] = useState<"all" | "mutual" | "pending">("all");
  const [followedBack, setFollowedBack] = useState<Set<number>>(
    new Set(MOCK_FANS.filter(f => f.followBack).map(f => f.id))
  );

  const toggleFollow = (id: number) => {
    setFollowedBack(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const filtered = MOCK_FANS.filter(f => {
    const isMutual = followedBack.has(f.id);
    if (fanFilter === "mutual") return isMutual;
    if (fanFilter === "pending") return !isMutual;
    return true;
  });

  const mutualCount = [...followedBack].length;
  const pendingCount = MOCK_FANS.length - mutualCount;

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div className="flex items-center gap-2.5">
          <span style={{ fontSize: 15, color: "var(--neu-text)" }}>粉丝</span>
          <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "var(--neu-accent-primary)", color: "white" }}>
            {MOCK_FANS.length} 人
          </span>
        </div>
        <div className="flex items-center gap-2">
          {pendingCount > 0 && (
            <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "#f59e0b20", color: "#f59e0b", border: "1px solid #f59e0b30" }}>
              {pendingCount} 待回关
            </span>
          )}
        </div>
      </div>

      {/* Stats row — flat joined cells, no isolated shadows */}
      <div className="px-5 py-3 flex-shrink-0">
        <div className="grid grid-cols-3 rounded-[14px] overflow-hidden"
          style={{ background: "var(--neu-border)", gap: 1, border: "1px solid var(--neu-border)" }}>
          {[
            { v: MOCK_FANS.length, label: "总粉丝", color: "var(--neu-text)" },
            { v: mutualCount,      label: "互相关注", color: "#22c55e" },
            { v: pendingCount,     label: "待回关", color: "#f59e0b" },
          ].map(s => (
            <div key={s.label} className="flex flex-col items-center py-2.5"
              style={{ background: "var(--neu-bg)" }}>
              <span style={{ fontSize: 19, color: s.color, fontWeight: 600, lineHeight: 1 }}>{s.v}</span>
              <span style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 4 }}>{s.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Filter pills — ghost */}
      <div className="flex gap-1.5 px-5 pb-3 flex-shrink-0">
        {([["all", "全部"], ["mutual", "互相关注"], ["pending", "待回关"]] as const).map(([id, label]) => {
          const active = fanFilter === id;
          return (
            <motion.button key={id} whileTap={{ scale: 0.94 }} onClick={() => setFanFilter(id)}
              className="px-3 py-1 rounded-full outline-none neu-surface-ghost"
              style={active
                ? {
                    background: "var(--neu-accent-soft)",
                    color: "var(--neu-accent-primary)",
                    fontSize: 12,
                    fontWeight: 500,
                    borderColor: "var(--neu-accent-soft-border)",
                  }
                : { fontSize: 12, color: "var(--neu-text-muted)" }}>
              {label}
            </motion.button>
          );
        })}
      </div>

      {/* Fan list — flat joined rows */}
      <div className="px-5 pb-6">
        <div className="rounded-[16px] overflow-hidden flex flex-col"
          style={{ gap: 1, background: "var(--neu-border)", border: "1px solid var(--neu-border)" }}>
          {filtered.map(f => {
            const isMutual = followedBack.has(f.id);
            return (
              <div key={f.id} className="flex items-center gap-3 p-3 transition-colors"
                style={{ background: "var(--neu-bg)" }}
                onMouseEnter={e => (e.currentTarget.style.background = "var(--neu-border)")}
                onMouseLeave={e => (e.currentTarget.style.background = "var(--neu-bg)")}>
                <img src={f.avatar} alt={f.name} className="rounded-full object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span style={{ fontSize: 13, color: "var(--neu-text)" }}>{f.name}</span>
                    {isMutual && (
                      <span className="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full"
                        style={{ fontSize: 10, background: "rgba(34,197,94,0.12)", color: "#22c55e" }}>
                        <Check style={{ width: 9, height: 9 }} strokeWidth={2.5} />互关
                      </span>
                    )}
                  </div>
                  <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 2 }}>
                    {f.handle} · {f.joinedDays <= 1 ? "今天" : f.joinedDays <= 7 ? `${f.joinedDays}天前` : `${Math.floor(f.joinedDays / 7)}周前`}关注了你
                  </p>
                </div>
                <motion.button whileTap={{ scale: 0.93 }} onClick={() => toggleFollow(f.id)}
                  className="flex items-center gap-1 px-3 py-1 rounded-full outline-none flex-shrink-0"
                  style={isMutual
                    ? { background: "transparent", color: "var(--neu-text-muted)", fontSize: 11, border: "1px solid var(--neu-border-pressed)" }
                    : { background: "var(--neu-accent-primary)", color: "white", fontSize: 11, border: "1px solid transparent" }
                  }>
                  {isMutual ? (
                    <><Check style={{ width: 10, height: 10 }} strokeWidth={2.5} />互关</>
                  ) : (
                    <><UserPlus style={{ width: 10, height: 10 }} strokeWidth={2} />回关</>
                  )}
                </motion.button>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Desktop Profile — 个人主页总览 + 分区全宽两种模式
══════════════════════════════════════════════════════════════════ */
function ProfileDesktop() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { user, logout } = useAuth();
  // null = 个人主页总览；有值 = 进入对应分区全宽页面
  const section = searchParams.get("section");
  const nickname = user?.nickname || "";
  const avatarUrl = user?.avatar_url || "";
  const bio = user?.bio || "";
  const handle = nickname ? `@${nickname.replace(/\s+/g, ".").toLowerCase()}` : "";

  const SECTION_META: Record<string, { title: string; render: () => React.ReactNode }> = {
    moments:  { title: "我的瞬间",  render: () => <SectionMoments /> },
    groups:   { title: "我的群组",  render: () => <SectionGroups /> },
    friends:  { title: "好友",      render: () => <SectionFriends /> },
    following:{ title: "关注",      render: () => <SectionFollowing /> },
    fans:     { title: "粉丝",      render: () => <SectionFans /> },
  };

  /* ── 分区全宽模式（有 section 参数）────────────────────────── */
  if (section && SECTION_META[section]) {
    const meta = SECTION_META[section];
    return (
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex-shrink-0 flex items-center gap-3 px-6 py-4"
          style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
          <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }} onClick={() => navigate("/profile")}
            className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
            style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
          <p style={{ fontSize: 18, color: "var(--neu-text)", flex: 1 }}>{meta.title}</p>
          <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }} onClick={() => navigate("/settings")}
            className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
            style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <Settings style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
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
      <div className="flex-shrink-0 flex items-center justify-between px-6 py-4"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div>
          <p style={{ fontSize: 20, color: "var(--neu-text)" }}>{nickname || "用户"}</p>
          {handle && <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 1 }}>{handle}</p>}
        </div>
        <div className="flex gap-2">
          <motion.button whileTap={{ scale: 0.9 }} className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <Share2 style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
          <motion.button whileTap={{ scale: 0.9 }} onClick={() => navigate("/settings")}
            className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <Settings style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
          <motion.button whileTap={{ scale: 0.9 }} onClick={async () => { await logout(); navigate("/login"); }}
            className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <LogOut style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
        </div>
      </div>

      <div className="flex flex-1 min-h-0">
        {/* ── Left profile panel ─────────────────────────── */}
        <div className="flex-shrink-0 overflow-y-auto no-scrollbar px-6 py-6 flex flex-col items-center"
          style={{ width: 260, borderRight: "1px solid var(--neu-border)" }}>
          {/* Avatar */}
          <div className="relative mb-4">
            <div className="absolute inset-0 rounded-full pointer-events-none"
              style={{ background: "var(--neu-glow)", filter: "blur(20px)", transform: "scale(1.2)", opacity: 0.5 }} />
            <div className="relative rounded-full"
              style={{ width: 96, height: 96, padding: 2, background: "var(--neu-bg)", boxShadow: "8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light)", border: "1.5px solid var(--neu-border)" }}>
              <img src={avatarUrl} alt={nickname} className="w-full h-full object-cover rounded-full" />
            </div>
            <motion.button whileTap={{ scale: 0.9 }}
              className="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
              style={{ width: 26, height: 26, background: "var(--neu-accent-primary)", border: "2.5px solid var(--neu-bg)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
              <Camera style={{ width: 12, height: 12, color: "white" }} strokeWidth={2} />
            </motion.button>
          </div>

          <h2 style={{ fontSize: 18, color: "var(--neu-text)", textAlign: "center" }}>{nickname || "用户"}</h2>
          {handle && <span style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>{handle}</span>}
          <p style={{ fontSize: 12, color: "var(--neu-text-muted)", textAlign: "center", marginTop: 8, lineHeight: 1.6, paddingLeft: 8, paddingRight: 8 }}>
            {bio || "这个人很懒，什么都没写..."}
          </p>
          <div className="flex items-center gap-1 mt-2 mb-4">
            <MapPin style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>江苏·徐州</span>
          </div>

          {/* Edit profile button */}
          <motion.button whileTap={{ scale: 0.96 }} onClick={() => navigate("/profile/edit")}
            className="w-full py-2.5 rounded-full outline-none mb-5 flex items-center justify-center gap-2"
            style={{ background: "var(--neu-bg)", color: "var(--neu-text-muted)", border: "1px solid var(--neu-border)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", fontSize: 13 }}>
            <Pencil style={{ width: 13, height: 13 }} strokeWidth={1.5} />
            编辑资料
          </motion.button>

          {/* Stats */}
          <div className="w-full rounded-[18px] overflow-hidden mb-4"
            style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            {STATS.map((stat, i) => (
              <React.Fragment key={stat.label}>
                {i > 0 && <div style={{ height: 1, background: "var(--neu-border)" }} />}
                <motion.button whileTap={{ scale: 0.97 }}
                  onClick={() => stat.section && navigate(`/profile?section=${stat.section}`)}
                  className="w-full flex items-center justify-between px-4 py-3 outline-none"
                  style={{ cursor: stat.section ? "pointer" : "default" }}>
                  <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>{stat.label}</span>
                  <div className="flex items-center gap-1.5">
                    <span style={{ fontSize: 17, color: "var(--neu-text)" }}>{stat.value}</span>
                    {stat.section && <ChevronRight style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />}
                  </div>
                </motion.button>
              </React.Fragment>
            ))}
          </div>

          {/* Section quick links */}
          <div className="w-full rounded-[18px] overflow-hidden"
            style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            {[
              { label: "我的瞬间",  section: "moments",   icon: Grid3X3,      count: GALLERY_POSTS.length },
              { label: "我的群组",  section: "groups",    icon: Users,        count: MOCK_GROUPS.length },
              { label: "好友",      section: "friends",   icon: MessageCircle,count: MOCK_FRIENDS.length },
              { label: "关注",      section: "following", icon: Heart,        count: MOCK_FOLLOWING.length },
              { label: "粉丝",      section: "fans",      icon: UserPlus,     count: MOCK_FANS.length },
            ].map((item, i) => (
              <React.Fragment key={item.label}>
                {i > 0 && <div style={{ height: 1, background: "var(--neu-border)" }} />}
                <motion.button whileTap={{ scale: 0.97 }}
                  onClick={() => navigate(`/profile?section=${item.section}`)}
                  className="w-full flex items-center justify-between px-4 py-2.5 outline-none">
                  <div className="flex items-center gap-2.5">
                    <item.icon style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                    <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>{item.label}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{item.count}</span>
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
            <p style={{ fontSize: 15, color: "var(--neu-text)" }}>最近瞬间</p>
            <motion.button whileTap={{ scale: 0.93 }} onClick={() => navigate("/profile?section=moments")}
              className="flex items-center gap-1 outline-none"
              style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}>
              查看全部
              <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={2} />
            </motion.button>
          </div>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 10, maxWidth: 3 * 240 + 2 * 10 }}>
            {GALLERY_POSTS.slice(0, 9).map(post => (
              <motion.div key={post.id} whileTap={{ scale: 0.97 }}
                className="relative rounded-[16px] overflow-hidden cursor-pointer"
                style={{ aspectRatio: "1/1", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                <img src={post.img} alt="" className="w-full h-full object-cover" />
                <div className="absolute inset-0 flex items-end p-2"
                  style={{ background: "linear-gradient(to top, rgba(0,0,0,0.6) 0%, transparent 55%)" }}>
                  <div className="flex items-center gap-2">
                    <div className="flex items-center gap-0.5">
                      <Heart style={{ width: 10, height: 10, color: "white" }} fill="white" strokeWidth={0} />
                      <span style={{ fontSize: 10, color: "white" }}>{post.likes}</span>
                    </div>
                    <div className="flex items-center gap-0.5">
                      <MessageCircle style={{ width: 10, height: 10, color: "white" }} strokeWidth={1.5} />
                      <span style={{ fontSize: 10, color: "white" }}>{post.comments}</span>
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
function ProfileHarmony() {
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const nickname = user?.nickname || "";
  const avatarUrl = user?.avatar_url || "";
  const bio = user?.bio || "";
  const handle = nickname ? `@${nickname.replace(/\s+/g, ".").toLowerCase()}` : "";
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  const postsMap: Record<Tab, typeof GALLERY_POSTS> = { posts: GALLERY_POSTS, liked: LIKED_POSTS, saved: SAVED_POSTS };

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex flex-col pb-6">
      {/* Gradient banner */}
      <div className="relative overflow-hidden mx-4 mt-3 rounded-[28px]" style={{ height: "clamp(120px, 20vw, 170px)" }}>
        <img src={GALLERY_POSTS[0].img} alt="横幅" className="absolute inset-0 w-full h-full object-cover" />
        <div className="absolute inset-0" style={{ background: "linear-gradient(135deg, rgba(99,102,241,0.75) 0%, rgba(139,92,246,0.5) 60%, rgba(0,0,0,0.2) 100%)" }} />
        <div className="absolute top-3 right-3 flex gap-2">
          <motion.button whileTap={{ scale: 0.9 }} className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 34, height: 34, background: "rgba(255,255,255,0.2)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.3)" }}>
            <Share2 style={{ width: 14, height: 14, color: "white" }} strokeWidth={1.5} />
          </motion.button>
          <motion.button whileTap={{ scale: 0.9 }} onClick={() => navigate("/settings")}
            className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 34, height: 34, background: "rgba(255,255,255,0.2)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.3)" }}>
            <Settings style={{ width: 14, height: 14, color: "white" }} strokeWidth={1.5} />
          </motion.button>
          <motion.button whileTap={{ scale: 0.9 }} onClick={async () => { await logout(); navigate("/login"); }}
            className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 34, height: 34, background: "rgba(255,255,255,0.2)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.3)" }}>
            <LogOut style={{ width: 14, height: 14, color: "white" }} strokeWidth={1.5} />
          </motion.button>
        </div>
      </div>

      {/* Avatar overlay */}
      <div className="flex items-end gap-4 px-6 -mt-10 mb-4">
        <div className="relative flex-shrink-0">
          <div className="rounded-[24px] overflow-hidden"
            style={{ width: 80, height: 80, border: "3px solid var(--neu-bg)", boxShadow: "0 8px 24px rgba(0,0,0,0.2)" }}>
            <img src={avatarUrl} alt={nickname} className="w-full h-full object-cover" />
          </div>
          <motion.button whileTap={{ scale: 0.9 }}
            className="absolute -bottom-1 -right-1 flex items-center justify-center rounded-full outline-none"
            style={{ width: 24, height: 24, background: "var(--neu-accent-primary)", border: "2px solid var(--neu-bg)" }}>
            <Camera style={{ width: 11, height: 11, color: "white" }} strokeWidth={2} />
          </motion.button>
        </div>
        <div className="flex-1 pb-1">
          <p style={{ fontSize: 20, color: "var(--neu-text)" }}>{nickname || "用户"}</p>
          {handle && <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>{handle}</p>}
        </div>
        <motion.button whileTap={{ scale: 0.96 }} onClick={() => navigate("/profile/edit")}
          className="flex-shrink-0 px-4 py-2 rounded-[16px] outline-none flex items-center gap-1.5"
          style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 13, boxShadow: "0 4px 12px var(--neu-glow)", border: "none" }}>
          <Pencil style={{ width: 12, height: 12 }} strokeWidth={2} />
          编辑
        </motion.button>
      </div>

      {/* Bio + location */}
      <div className="px-6 mb-4">
        <p style={{ fontSize: 13, color: "var(--neu-text-muted)", lineHeight: 1.7 }}>{bio || "这个人很懒，什么都没写..."}</p>
        <div className="flex items-center gap-1 mt-2">
          <MapPin style={{ width: 12, height: 12, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
          <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>江苏·徐州</span>
        </div>
      </div>

      {/* Stats row */}
      <div className="flex gap-3 px-6 mb-5">
        {[
          { label: "帖子", value: "48" },
          { label: "关��者", value: "312" },
          { label: "足迹", value: "27" },
        ].map(stat => (
          <div key={stat.label} className="flex-1 flex flex-col items-center py-3 rounded-[18px]"
            style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <span style={{ fontSize: 22, color: "var(--neu-text)" }}>{stat.value}</span>
            <span style={{ fontSize: 11, color: "var(--neu-accent-primary)", marginTop: 2 }}>{stat.label}</span>
          </div>
        ))}
      </div>

      {/* Action buttons */}
      <div className="flex gap-3 px-6 mb-5">
        <motion.button whileTap={{ scale: 0.95 }} className="flex-1 flex items-center justify-center gap-2 py-3 rounded-[18px] outline-none"
          style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 13, boxShadow: "0 4px 14px var(--neu-glow)", border: "none" }}>
          <Plus style={{ width: 15, height: 15 }} strokeWidth={2.5} />
          发布瞬间
        </motion.button>
        <motion.button whileTap={{ scale: 0.95 }} className="flex-1 flex items-center justify-center gap-2 py-3 rounded-[18px] outline-none"
          style={{ background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 13, boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <Share2 style={{ width: 15, height: 15 }} strokeWidth={1.5} />
          分享主页
        </motion.button>
      </div>

      <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
      <div className="px-4 mt-3">
        <GalleryGrid posts={postsMap[activeTab]} activeTab={activeTab} cols={2} />
      </div>
    </motion.div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Mobile / Tablet Profile
   Hero section is constrained to maxWidth so it doesn't sprawl on tablets.
══════════════════════════════════════════════════════════════════ */
function ProfileMobile() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const { user, logout } = useAuth();
  const nickname = user?.nickname || "";
  const avatarUrl = user?.avatar_url || "";
  const bio = user?.bio || "";
  const handle = nickname ? `@${nickname.replace(/\s+/g, ".").toLowerCase()}` : "";
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  const postsMap: Record<Tab, typeof GALLERY_POSTS> = { posts: GALLERY_POSTS, liked: LIKED_POSTS, saved: SAVED_POSTS };

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
      <div className="sticky top-0 z-20 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderBottom: "1px solid var(--neu-border)" }}>
        <div className={isTablet ? `${tabletShellClass} px-0` : "px-5"} style={tabletShellStyle}>
          <div className={isTablet ? `${mainColClass} px-5` : "w-full px-0"}
            style={mainColStyle}>
            <div className="flex items-center justify-between">
              <div>
                <p style={{ fontSize: 18, color: "var(--neu-text)" }}>{nickname || "用户"}</p>
                {handle && <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }}>{handle}</p>}
              </div>
              <div className="flex gap-2">
                <motion.button whileTap={{ scale: 0.92 }}
                  className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                  style={{ width: 36, height: 36 }}>
                  <Share2 style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                </motion.button>
                <motion.button whileTap={{ scale: 0.92 }} onClick={() => navigate("/settings")}
                  className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                  style={{ width: 36, height: 36 }}>
                  <Settings style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                </motion.button>
                <motion.button whileTap={{ scale: 0.92 }} onClick={async () => { await logout(); navigate("/login"); }}
                  className="flex items-center justify-center rounded-full outline-none neu-surface-flat"
                  style={{ width: 36, height: 36 }}>
                  <LogOut style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                </motion.button>
              </div>
            </div>
          </div>
          {/* Spacer matching rail width on wide tablet so header stretches edge-to-edge */}
          {isTablet && <div className="hidden lg:block flex-shrink-0" style={{ width: 300 + 20 }} />}
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
                  <img src={avatarUrl} alt={nickname} className="w-full h-full object-cover" />
                </div>
                <motion.button whileTap={{ scale: 0.9 }}
                  className="absolute bottom-0 right-0 flex items-center justify-center rounded-full outline-none"
                  style={{ width: 26, height: 26, background: "var(--neu-accent-primary)", border: "2.5px solid var(--neu-bg)" }}>
                  <Camera style={{ width: 12, height: 12, color: "white" }} strokeWidth={2} />
                </motion.button>
              </div>

              {/* Identity */}
              <h2 style={{ fontSize: 22, color: "var(--neu-text)" }}>{nickname || "用户"}</h2>
              {handle && <span style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>{handle}</span>}
              <p style={{ fontSize: 12, color: "var(--neu-text-muted)", textAlign: "center", padding: "10px 20px 4px", lineHeight: 1.6 }}>
                {bio || "这个人很懒，什么都没写..."}
              </p>
              <div className="flex items-center gap-1 mb-4">
                <MapPin style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>江苏·徐州</span>
              </div>

              {/* Dual action — equal weight, ghost on neutral surface */}
              <div className="flex w-full gap-2 mb-5">
                <motion.button whileTap={{ scale: 0.97 }} onClick={() => navigate("/profile/edit")}
                  className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none"
                  style={{
                    background: "var(--neu-accent-primary)",
                    color: "white",
                    fontSize: 13,
                    fontWeight: 500,
                    boxShadow: "0 3px 10px var(--neu-glow)",
                  }}>
                  <Pencil style={{ width: 13, height: 13 }} strokeWidth={2} />
                  编辑资料
                </motion.button>
                <motion.button whileTap={{ scale: 0.97 }}
                  className="flex-1 flex items-center justify-center gap-1.5 py-2.5 rounded-[14px] outline-none transition-colors"
                  style={{
                    background: "transparent",
                    color: "var(--neu-text)",
                    border: "1px solid var(--neu-border-pressed)",
                    fontSize: 13,
                  }}
                  onMouseEnter={e => (e.currentTarget.style.background = "var(--neu-border)")}
                  onMouseLeave={e => (e.currentTarget.style.background = "transparent")}>
                  <Share2 style={{ width: 13, height: 13 }} strokeWidth={1.75} />
                  分享主页
                </motion.button>
              </div>

              {/* Stats — flat, integrated INTO the hero card (not a separate elevated bar) */}
              <div className="grid grid-cols-3 w-full rounded-[14px] overflow-hidden"
                style={{ background: "var(--neu-border)", gap: 1, border: "1px solid var(--neu-border)" }}>
                {[
                  { label: "帖子", value: "48" },
                  { label: "关注者", value: "312" },
                  { label: "足迹", value: "27" },
                ].map(stat => (
                  <motion.button key={stat.label} whileTap={{ scale: 0.97 }}
                    className="flex flex-col items-center justify-center py-2.5 outline-none transition-colors"
                    style={{ background: "var(--neu-bg)" }}
                    onMouseEnter={e => (e.currentTarget.style.background = "var(--neu-border)")}
                    onMouseLeave={e => (e.currentTarget.style.background = "var(--neu-bg)")}>
                    <span style={{ fontSize: 20, color: "var(--neu-text)", fontWeight: 600, lineHeight: 1 }}>{stat.value}</span>
                    <span style={{ fontSize: 10, color: "var(--neu-text-muted)", marginTop: 5 }}>{stat.label}</span>
                  </motion.button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Tabs + content — keep aligned with hero on tablet */}
        <div className="mt-6">
          <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
          <div className="px-4 mt-3">
            <GalleryGrid posts={postsMap[activeTab]} activeTab={activeTab} cols={isTablet ? 3 : 2} />
          </div>
        </div>
        </div>

        {/* ───── Right rail — tablet landscape only ───── */}
        {isTablet && (
          <aside className="hidden lg:block flex-shrink-0 pr-5" style={{ width: 300 + 20 /* incl pr-5 */ }}>
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

/* ══════════════════════════════════════════════════════════════════
   Entry
══════════════════════════════════════════════════════════════════ */
export function Profile() {
  const { platform } = usePlatform();
  if (platform === "desktop") return <ProfileDesktop />;
  return <ProfileMobile />;
}
