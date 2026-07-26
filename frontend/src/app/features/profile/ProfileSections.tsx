import React, { useState } from "react";
import { Heart, MapPin, Grid3X3, Share2, Bookmark, ChevronRight, Users, MessageCircle, Search, Plus, Check, Bell, Clock, UserPlus, TrendingUp, Shield, X } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useNavigate } from "react-router";
import { ROUTES } from "../../lib/routes";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDismissibleLayer } from "../../hooks/useDismissibleLayer";
import { useComposer } from "../../contexts/ComposerContext";
import { useSocial } from "../../contexts/SocialContext";
import { usePlatform } from "../../contexts/PlatformContext";
import { CreateGroupSheet } from "../../features/groups/CreateGroupSheet";
import { MetricStrip } from "../../components/shared/DataDisplay";
import { BspCardCanvas } from "../../components/shared/BspCardCanvas";
import { USER_AVATAR, GALLERY_POSTS, LIKED_POSTS, SAVED_POSTS, MOCK_GROUPS, MOCK_FRIENDS, MOCK_FOLLOWING, MOCK_FANS } from "./profile.data";

/* ── Shared: Tab bar ──────────────────────────────────────────────── */
export type Tab = "posts" | "liked" | "saved";
export function ProfileTabs({ activeTab, setActiveTab }: { activeTab: Tab; setActiveTab: (t: Tab) => void }) {
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
export function GalleryGrid({ posts, activeTab }: { posts: typeof GALLERY_POSTS; activeTab: Tab }) {
  const { platform } = usePlatform();
  const navigate = useNavigate();
  const numCols = platform === "mobile" ? 2 : 3;
  const gap = 10;
  return (
    <AnimatePresence mode="wait">
      <motion.div key={activeTab}
        initial={{ opacity: 0, y: 6 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -4 }}
        transition={{ duration: 0.2 }}
        style={{ display: "grid", gridTemplateColumns: `repeat(${numCols}, 1fr)`, gap, width: "100%" }}>
        {posts.map((post) => (
          <motion.div key={post.id} whileTap={{ scale: 0.97 }}
            onClick={() => navigate(`${ROUTES.moment(post.id)}?from=me`)}
            onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") navigate(`${ROUTES.moment(post.id)}?from=me`); }}
            role="button"
            tabIndex={0}
            className="relative rounded-[18px] overflow-hidden cursor-pointer group outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
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
export function SectionMoments() {
  const navigate = useNavigate();
  const { openComposer } = useComposer();
  const { profilePosts, commentsByPost } = useSocial();
  const { platform } = usePlatform();
  const isDesktop = platform === "desktop";
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  const [visibleCount, setVisibleCount] = useState(12);
  const publishedGalleryPosts = profilePosts.map((post) => ({
    id: post.id,
    img: post.image ?? "https://images.unsplash.com/photo-1762113396314-8e9e5417ed17?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800",
    likes: post.likes,
    comments: post.comments + (commentsByPost[post.id]?.length ?? 0),
  }));
  const ownPosts = [...publishedGalleryPosts, ...GALLERY_POSTS];
  const postsMap = { posts: ownPosts, liked: LIKED_POSTS, saved: SAVED_POSTS };
  const activePosts = postsMap[activeTab];
  const collectionMeta: Record<Tab, { eyebrow: string; title: string; detail: string }> = {
    posts: { eyebrow: "个人存档", title: "我的瞬间", detail: "你发布的每一次靠近与停留" },
    liked: { eyebrow: "留住灵感", title: "喜欢的瞬间", detail: "那些让你愿意再看一眼的片段" },
    saved: { eyebrow: "稍后再见", title: "收藏的瞬间", detail: "为下一次出发留下的线索" },
  };
  const current = collectionMeta[activeTab];
  const featuredPost = activePosts[0];
  const remainingPosts = activePosts.slice(1);

  return (
    <div className="flex flex-1 flex-col overflow-y-auto no-scrollbar">
      <header className="flex-none border-b border-[var(--neu-border)] px-6 pb-4 pt-5">
        <div className="flex items-start justify-between gap-5">
          <div className="min-w-0">
            <p className="neu-eyebrow">{current.eyebrow}</p>
            <div className="mt-1 flex items-baseline gap-2">
              <h1 className="text-[21px] font-medium tracking-[-0.01em] text-[var(--neu-text)]">{current.title}</h1>
              <span className="rounded-full border border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] px-2 py-0.5 text-[10px] font-medium text-[var(--neu-accent-primary)]">
                {activePosts.length} 条
              </span>
            </div>
            <p className="mt-1 text-[12px] text-[var(--neu-text-muted)]">{current.detail}</p>
          </div>
        </div>
      </header>

      <div className="flex-none px-5 pt-3">
        <ProfileTabs activeTab={activeTab} setActiveTab={setActiveTab} />
      </div>

      <div className="w-full px-6 pb-8 pt-5">
        {activePosts.length === 0 ? (
          <div className="flex flex-col items-center py-20 text-center">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)]"><Grid3X3 size={20} /></div>
            <p className="mt-4 text-[14px] text-[var(--neu-text)]">这里还没有瞬间</p>
            <p className="mt-1 text-[12px] text-[var(--neu-text-muted)]">从一次路过、一次相遇开始记录。</p>
            {activeTab === "posts" && <button type="button" onClick={openComposer} className="mt-4 rounded-full bg-[var(--neu-accent-primary)] px-4 py-2 text-[12px] font-medium text-white outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]">发布第一条</button>}
          </div>
        ) : isDesktop ? (
          // Desktop: BSP mosaic. Every moment is equal in the field — no featured
          // slot, no uniform grid — so the collection reads as one varied canvas.
          // Same image + likes/comments overlay as the non-desktop card below.
          <BspCardCanvas
            items={activePosts.slice(0, visibleCount)}
            gap={24}
            minRatio={0.68}
            maxRatio={1.48}
            minCardWidth={160}
            minCardHeight={232}
            minCanvasHeight={520}
            ariaLabel="我的瞬间的不规则满铺布局"
            hasMore={visibleCount < activePosts.length}
            onLoadMore={() => setVisibleCount((c) => Math.min(c + 8, activePosts.length))}
            renderItem={(post) => (
              <motion.button
                key={post.id}
                type="button"
                whileTap={{ scale: 0.98 }}
                onClick={() => navigate(`${ROUTES.moment(post.id)}?from=me`)}
                className="group relative h-full w-full overflow-hidden rounded-[16px] border border-[var(--neu-border)] bg-[var(--page-bg)] text-left outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)]"
                aria-label="查看瞬间详情"
              >
                <img src={post.img} alt="" className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.035]" />
                <div className="absolute inset-x-0 bottom-0 bg-[linear-gradient(to_top,rgba(0,0,0,.62),transparent)] p-2.5">
                  <div className="flex items-center gap-2 text-[10px] text-white">
                    <span className="flex items-center gap-0.5"><Heart size={10} fill="currentColor" strokeWidth={0} />{post.likes}</span>
                    <span className="flex items-center gap-0.5"><MessageCircle size={10} strokeWidth={1.6} />{post.comments}</span>
                  </div>
                </div>
              </motion.button>
            )}
          />
        ) : (
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -5 }}
              transition={{ duration: 0.22 }}
            >
              <button
                type="button"
                onClick={() => navigate(`${ROUTES.moment(featuredPost.id)}?from=me`)}
                className="group relative block w-full overflow-hidden rounded-[22px] border border-[var(--neu-border)] bg-[var(--page-bg)] text-left outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)]"
                style={{ aspectRatio: "2.1 / 1" }}
                aria-label="查看置顶瞬间详情"
              >
                <img src={featuredPost.img} alt="" className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.025]" />
                <div className="absolute inset-0 bg-[linear-gradient(90deg,rgba(0,0,0,.62),rgba(0,0,0,.08)_65%,transparent)]" />
                <div className="absolute inset-x-0 bottom-0 flex items-end justify-between gap-4 p-4">
                  <div>
                    <p className="text-[10px] font-medium tracking-[0.12em] text-white/75">{activeTab === "posts" ? "最近发布" : current.eyebrow}</p>
                    <p className="mt-1 text-[15px] font-medium text-white">这一刻，值得被好好留下</p>
                  </div>
                  <div className="flex items-center gap-3 rounded-full bg-black/25 px-3 py-1.5 text-[11px] text-white backdrop-blur-sm">
                    <span className="flex items-center gap-1"><Heart size={12} fill="currentColor" strokeWidth={0} />{featuredPost.likes}</span>
                    <span className="flex items-center gap-1"><MessageCircle size={12} strokeWidth={1.7} />{featuredPost.comments}</span>
                  </div>
                </div>
              </button>

              <div className="mb-3 mt-6 flex items-center justify-between">
                <p className="neu-eyebrow">更多片段</p>
                <span className="text-[11px] text-[var(--neu-text-muted)]">按最近记录</span>
              </div>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {remainingPosts.map((post) => (
                  <motion.button
                    key={post.id}
                    type="button"
                    whileTap={{ scale: 0.98 }}
                    onClick={() => navigate(`${ROUTES.moment(post.id)}?from=me`)}
                    className="group relative aspect-square overflow-hidden rounded-[16px] border border-[var(--neu-border)] bg-[var(--page-bg)] text-left outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)]"
                    aria-label="查看瞬间详情"
                  >
                    <img src={post.img} alt="" className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.035]" />
                    <div className="absolute inset-x-0 bottom-0 bg-[linear-gradient(to_top,rgba(0,0,0,.62),transparent)] p-2.5">
                      <div className="flex items-center gap-2 text-[10px] text-white">
                        <span className="flex items-center gap-0.5"><Heart size={10} fill="currentColor" strokeWidth={0} />{post.likes}</span>
                        <span className="flex items-center gap-0.5"><MessageCircle size={10} strokeWidth={1.6} />{post.comments}</span>
                      </div>
                    </div>
                  </motion.button>
                ))}
              </div>
            </motion.div>
          </AnimatePresence>
        )}
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 我的群组
══════════════════════════════════════════════════════════════════ */
export function SectionGroups() {
  const navigate = useNavigate();
  const { platform } = usePlatform();
  const isDesktop = platform === "desktop";
  const [searchQ, setSearchQ] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [visibleCount, setVisibleCount] = useState(12);
  useDismissibleLayer({ open: showCreate, onDismiss: () => setShowCreate(false) });
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

      {/* Group cards — desktop uses the shared BSP mosaic so "我的群组" reads as
          one varied canvas matching 附近群组; tablet/mobile keeps the readable
          vertical list. The card body is identical in both modes. */}
      {isDesktop ? (
        <div className="px-5 pb-6">
          <BspCardCanvas
            items={filtered.slice(0, visibleCount)}
            gap={24}
            minRatio={0.68}
            maxRatio={1.48}
            minCardWidth={160}
            minCardHeight={188}
            minCanvasHeight={520}
            ariaLabel="我的群组的不规则满铺布局"
            hasMore={visibleCount < filtered.length}
            onLoadMore={() => setVisibleCount((c) => Math.min(c + 8, filtered.length))}
            renderItem={(g) => (
              <MyGroupCard g={g} hover={hover} onOpen={() => navigate(`${ROUTES.group(g.id)}?from=me`)} onEnter={() => navigate(`${ROUTES.group(g.id)}?from=me`)} />
            )}
          />
        </div>
      ) : (
        <div className="flex flex-col gap-3 px-5 pb-6">
          {filtered.map(g => (
            <MyGroupCard key={g.id} g={g} hover={hover} onOpen={() => navigate(`${ROUTES.group(g.id)}?from=me`)} onEnter={() => navigate(`${ROUTES.group(g.id)}?from=me`)} />
          ))}
        </div>
      )}

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
              role="dialog"
              aria-modal="true"
              aria-label="创建群组"
              style={{ height: "92%", borderRadius: "28px 28px 0 0", backgroundColor: "var(--neu-bg)", boxShadow: "0 -8px 40px rgba(0,0,0,0.18)" }}>
              <CreateGroupSheet onClose={() => setShowCreate(false)} />
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}

/* Shared group card body for SectionGroups — rendered identically inside the
   desktop BSP mosaic and the tablet/mobile vertical list. `onEnter` is the
   inline "进入" action (stops propagation so it never re-triggers the card). */
function MyGroupCard({ g, hover, onOpen, onEnter }: {
  g: typeof MOCK_GROUPS[number];
  hover: ReturnType<typeof useHoverConfig>;
  onOpen: () => void;
  onEnter: () => void;
}) {
  return (
    <motion.div
      whileTap={hover.tap as any}
      whileHover={hover.card as any}
      onClick={onOpen}
      onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") onOpen(); }}
      role="button"
      tabIndex={0}
      className="flex h-full w-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-card-hover outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
      style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
      {/* Banner */}
      <div className="relative flex-1 min-h-0" style={{ minHeight: "clamp(60px, 10vw, 80px)" }}>
        <img src={g.banner} alt={g.name} className="w-full h-full object-cover" />
        <div className="absolute inset-0" style={{ background: "linear-gradient(to bottom, transparent 30%, rgba(0,0,0,0.6) 100%)" }} />
        <div className="absolute bottom-2 left-3 flex items-center gap-1.5">
          <span className="px-2 py-0.5 rounded-full"
            style={{ fontSize: 10, background: g.role === "管理员" ? "var(--neu-accent-primary)" : "rgba(0,0,0,0.5)", color: "white", backdropFilter: "blur(8px)" }}>
            {g.role === "管理员" && <Shield style={{ width: 8, height: 8, display: "inline", marginRight: 2 }} strokeWidth={2} />}
            {g.role}
          </span>
        </div>
        <div className="absolute bottom-2 right-3 flex items-center gap-1">
          <div style={{ width: 7, height: 7, borderRadius: "50%", background: "var(--semantic-online)" }} />
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
          <motion.button whileTap={{ scale: 0.93 }} onClick={(event) => { event.stopPropagation(); onEnter(); }}
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
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 好友
══════════════════════════════════════════════════════════════════ */
export function SectionFriends({ embedded = false }: { embedded?: boolean }) {
  const navigate = useNavigate();
  const hover = useHoverConfig();
  const [searchQ, setSearchQ] = useState("");
  const [activeFilter, setActiveFilter] = useState<"all" | "online" | "offline">("all");

  const filtered = MOCK_FRIENDS
    .filter(f => activeFilter === "all" || (activeFilter === "online" ? f.online : !f.online))
    .filter(f => searchQ === "" || f.name.toLowerCase().includes(searchQ.toLowerCase()) || f.handle.includes(searchQ));

  const onlineCount = MOCK_FRIENDS.filter(f => f.online).length;

  return (
    <div className="flex-1 overflow-y-auto no-scrollbar flex flex-col">
      {!embedded && <>
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div>
          <div className="flex items-center gap-2.5">
            <span style={{ fontSize: 15, color: "var(--neu-text)" }}>好友</span>
            <div className="flex items-center gap-1 px-2 py-0.5 rounded-full"
            style={{ fontSize: 11, background: "color-mix(in srgb, var(--semantic-online) 13%, transparent)", border: "1px solid color-mix(in srgb, var(--semantic-online) 24%, transparent)" }}>
            <div style={{ width: 6, height: 6, borderRadius: "50%", background: "var(--semantic-online)" }} />
              <span style={{ color: "var(--semantic-online)" }}>{onlineCount} 在线</span>
            </div>
          </div>
          <p className="mt-1" style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>双向关系，可直接开启对话</p>
        </div>
        <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{MOCK_FRIENDS.length} 位好友</span>
      </div>
      </>}

      {/* Search */}
      <div className={`${embedded ? "px-6 lg:px-8" : "px-5"} pt-4 pb-2 flex-shrink-0`}>
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
      <div className={`flex gap-2 ${embedded ? "px-6 lg:px-8" : "px-5"} pb-4 flex-shrink-0`}>
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
      <div className={embedded
        ? "grid w-full grid-cols-1 gap-4 px-6 pb-8 md:grid-cols-2 lg:px-8 xl:grid-cols-3"
        : "flex flex-col gap-3 px-5 pb-6"}>
        {filtered.map(f => (
          <motion.div key={f.id} whileHover={hover.card as any} whileTap={hover.tap as any} onClick={() => navigate(`${ROUTES.user(f.id)}?from=relationships`)} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") navigate(`${ROUTES.user(f.id)}?from=relationships`); }} role="button" tabIndex={0} className="neu-row-hover flex items-center gap-3 rounded-[18px] p-3.5 cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
            style={{ background: "var(--neu-bg)", boxShadow: "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
            <div className="relative flex-shrink-0">
              <img src={f.avatar} alt={f.name} className="rounded-full object-cover" style={{ width: 48, height: 48 }} />
              <span className="absolute bottom-0 right-0 rounded-full"
                style={{ width: 12, height: 12, background: f.online ? "var(--semantic-online)" : "var(--neu-text-muted)", border: "2.5px solid var(--neu-bg)", opacity: f.online ? 1 : 0.5 }} />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span style={{ fontSize: 14, color: "var(--neu-text)" }}>{f.name}</span>
                <span className="px-1.5 py-0.5 rounded-[6px]"
                  style={{ fontSize: 10, background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)", border: "1px solid var(--neu-accent-soft-border)" }}>
                  {f.tag}
                </span>
              </div>
              <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>
                {f.handle} · {f.mutual} 共同好友
              </p>
            </div>
            <div className="flex flex-col gap-1.5 flex-shrink-0">
              <motion.button whileTap={{ scale: 0.93 }} onClick={(event) => { event.stopPropagation(); navigate(ROUTES.message(f.id)); }}
                className="flex items-center justify-center gap-1 px-2.5 py-1.5 rounded-full outline-none"
                style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 11, boxShadow: "0 3px 8px var(--neu-glow)" }}>
                <MessageCircle style={{ width: 11, height: 11 }} strokeWidth={2} />
                发消息
              </motion.button>
              <motion.button whileTap={{ scale: 0.93 }} onClick={(event) => { event.stopPropagation(); navigate(`${ROUTES.user(f.id)}?from=relationships`); }}
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
export function FollowingRow({ f, isUnfollowed, onToggle, wide = false }: { f: typeof MOCK_FOLLOWING[0]; isUnfollowed: boolean; onToggle: () => void; wide?: boolean }) {
  const navigate = useNavigate();
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  return (
    <motion.div
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      onClick={() => navigate(`${ROUTES.user(f.id)}?from=relationships`)}
      onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") navigate(`${ROUTES.user(f.id)}?from=relationships`); }}
      role="button"
      tabIndex={0}
      className={`neu-row-hover flex cursor-pointer items-center gap-3 p-3 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--neu-accent-primary)] ${wide ? "min-h-[84px] rounded-[18px] border border-[var(--neu-border)] shadow-[5px_5px_12px_var(--neu-shadow-dark),-5px_-5px_12px_var(--neu-shadow-light)]" : ""}`}
      style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", opacity: isUnfollowed ? 0.55 : 1, transition: "background 0.18s ease" }}>
      <img src={f.avatar} alt={f.name} className="rounded-full object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
      <div className="flex-1 min-w-0">
        <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{f.name}</p>
        <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }} className="truncate">{f.desc}</p>
        <p style={{ fontSize: 10, color: "var(--neu-accent-primary)", marginTop: 1 }}>{f.posts} 帖子</p>
      </div>
      <img src={f.latestImg} alt="" className="rounded-[8px] object-cover flex-shrink-0"
        style={{ width: 36, height: 36, opacity: 0.85 }} />
      <motion.button whileTap={{ scale: 0.93 }} onClick={(event) => { event.stopPropagation(); onToggle(); }}
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
    </motion.div>
  );
}

export function SectionFollowing({ embedded = false }: { embedded?: boolean }) {
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
      {!embedded && <>
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div>
          <div className="flex items-center gap-2.5">
            <span style={{ fontSize: 15, color: "var(--neu-text)" }}>关注</span>
            <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "var(--neu-accent-primary)", color: "white" }}>
              {MOCK_FOLLOWING.length} 人
            </span>
          </div>
          <p className="mt-1" style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>你主动订阅的动态与创作者</p>
        </div>
        <span className="rounded-full px-3 py-1" style={{ fontSize: 11, color: "var(--neu-accent-primary)", background: "var(--neu-accent-soft)", border: "1px solid var(--neu-accent-soft-border)" }}>
          点击卡片查看主页
        </span>
      </div>
      </>}

      {/* Sort tabs — ghost chips */}
      <div className={`flex gap-1.5 ${embedded ? "px-6 lg:px-8" : "px-5"} py-4 flex-shrink-0`}>
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

      {/* Following list */}
      <div className={embedded ? "px-6 pb-8 lg:px-8" : "px-5 pb-6"}>
        <div className={embedded ? "grid w-full grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3" : "rounded-[16px] overflow-hidden flex flex-col"}
          style={embedded ? undefined : { gap: 1, background: "var(--neu-border)", border: "1px solid var(--neu-border)" }}>
          {sorted.map(f => (
            <FollowingRow key={f.id} f={f} wide={embedded} isUnfollowed={unfollowedIds.has(f.id)} onToggle={() => toggleUnfollow(f.id)} />
          ))}
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Section: 粉丝
══════════════════════════════════════════════════════════════════ */
export function FanRow({ f, isMutual, onToggle, wide = false }: { f: typeof MOCK_FANS[0]; isMutual: boolean; onToggle: () => void; wide?: boolean }) {
  const navigate = useNavigate();
  const [hovered, setHovered] = useState(false);
  const hover = useHoverConfig();
  return (
    <motion.div
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      onClick={() => navigate(`${ROUTES.user(f.id)}?from=relationships`)}
      onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") navigate(`${ROUTES.user(f.id)}?from=relationships`); }}
      role="button"
      tabIndex={0}
      className={`neu-row-hover flex cursor-pointer items-center gap-3 p-3 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--neu-accent-primary)] ${wide ? "min-h-[76px] rounded-[18px] border border-[var(--neu-border)] shadow-[5px_5px_12px_var(--neu-shadow-dark),-5px_-5px_12px_var(--neu-shadow-light)]" : ""}`}
      style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <img src={f.avatar} alt={f.name} className="rounded-full object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span style={{ fontSize: 13, color: "var(--neu-text)" }}>{f.name}</span>
          {isMutual && (
            <span className="flex items-center gap-0.5 px-1.5 py-0.5 rounded-full"
              style={{ fontSize: 10, background: "color-mix(in srgb, var(--semantic-online) 12%, transparent)", color: "var(--semantic-online)" }}>
              <Check style={{ width: 9, height: 9 }} strokeWidth={2.5} />互关
            </span>
          )}
        </div>
        <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 2 }}>
          {f.handle} · {f.joinedDays <= 1 ? "今天" : f.joinedDays <= 7 ? `${f.joinedDays}天前` : `${Math.floor(f.joinedDays / 7)}周前`}关注了你
        </p>
      </div>
      <motion.button whileTap={{ scale: 0.93 }} onClick={(event) => { event.stopPropagation(); onToggle(); }}
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
    </motion.div>
  );
}

export function SectionFans({ embedded = false }: { embedded?: boolean }) {
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
      {!embedded && <>
      {/* Header */}
      <div className="flex items-center justify-between px-5 py-4 flex-shrink-0" style={{ borderBottom: "1px solid var(--neu-border)" }}>
        <div>
          <div className="flex items-center gap-2.5">
            <span style={{ fontSize: 15, color: "var(--neu-text)" }}>粉丝</span>
            <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "var(--neu-accent-primary)", color: "white" }}>
              {MOCK_FANS.length} 人
            </span>
          </div>
          <p className="mt-1" style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>关注你的人，优先处理待回关关系</p>
        </div>
        <div className="flex items-center gap-2">
          {pendingCount > 0 && (
            <span className="px-2 py-0.5 rounded-full" style={{ fontSize: 11, background: "color-mix(in srgb, var(--semantic-warning) 13%, transparent)", color: "var(--semantic-warning)", border: "1px solid color-mix(in srgb, var(--semantic-warning) 24%, transparent)" }}>
              {pendingCount} 待回关
            </span>
          )}
        </div>
      </div>
      </>}

      {/* Stats row — flat joined cells, no isolated shadows */}
      <div className={`${embedded ? "px-6 lg:px-8" : "px-5"} py-4 flex-shrink-0`}>
        <MetricStrip items={[
          { value: MOCK_FANS.length, label: "总粉丝" },
          { value: mutualCount, label: "互相关注", tone: "positive" },
          { value: pendingCount, label: "待回关", tone: "warning" },
        ]} />
      </div>

      {/* Filter pills — ghost */}
      <div className={`flex gap-1.5 ${embedded ? "px-6 lg:px-8" : "px-5"} pb-4 flex-shrink-0`}>
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

      {/* Fan list */}
      <div className={embedded ? "px-6 pb-8 lg:px-8" : "px-5 pb-6"}>
        <div className={embedded ? "grid w-full grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3" : "rounded-[16px] overflow-hidden flex flex-col"}
          style={embedded ? undefined : { gap: 1, background: "var(--neu-border)", border: "1px solid var(--neu-border)" }}>
          {filtered.map(f => {
            const isMutual = followedBack.has(f.id);
            return (
              <FanRow key={f.id} f={f} wide={embedded} isMutual={isMutual} onToggle={() => toggleFollow(f.id)} />
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
