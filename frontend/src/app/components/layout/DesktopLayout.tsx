import React, { useState, useEffect } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import {
  Home,
  Users,
  Rss,
  MessageCircle,
  Bell,
  ImageIcon,
  Heart,
  UserPlus,
  UserCheck,
  Search,
  Moon,
  Sun,
  Plus,
  Navigation2,
  ChevronRight,
  UserRound,
  Settings,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useTheme } from "../../contexts/ThemeContext";
import { CreateSheet } from "./CreateSheet";
import { useFocusTrap } from "../../hooks/useFocusTrap";
import imgHotSpot from "figma:asset/9254a1e0e11b516a6c1aecb80d92b66f2caa243e.png";

/* ── Constants ──────────────────────────────────────────────────── */

const USER_AVATAR =
  "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=200";

interface NavItem {
  id: string;
  icon: React.ElementType;
  label: string;
  path: string;
}

const TOP_NAV: NavItem[] = [
  { id: "home", icon: Home, label: "首页", path: "/" },
  { id: "nearby-people", icon: Users, label: "附近的人", path: "/nearby" },
  {
    id: "nearby-groups",
    icon: UserRound,
    label: "附近群组",
    path: "/nearby-groups",
  },
  { id: "nearby-activity", icon: Rss, label: "附近动态", path: "/discover" },
];

const PERSONAL_NAV: NavItem[] = [
  { id: "messages", icon: MessageCircle, label: "消息", path: "/messages" },
  { id: "notifications", icon: Bell, label: "通知", path: "/notifications" },
  {
    id: "moments",
    icon: ImageIcon,
    label: "瞬间",
    path: "/profile?section=moments",
  },
  {
    id: "my-groups",
    icon: Users,
    label: "群组",
    path: "/profile?section=groups",
  },
  {
    id: "friends",
    icon: Heart,
    label: "好友",
    path: "/profile?section=friends",
  },
  {
    id: "following",
    icon: UserPlus,
    label: "关注",
    path: "/profile?section=following",
  },
  { id: "fans", icon: UserCheck, label: "粉丝", path: "/profile?section=fans" },
];

function getNavIdFromPath(pathname: string, search: string): string {
  if (pathname.startsWith("/messages")) return "messages";
  if (pathname === "/notifications") return "notifications";
  if (pathname === "/nearby-groups") return "nearby-groups";
  if (pathname === "/nearby") return "nearby-people";
  if (pathname === "/profile" || pathname === "/profile/edit") {
    const params = new URLSearchParams(search);
    const section = params.get("section");
    if (section === "groups") return "my-groups";
    if (section === "friends") return "friends";
    if (section === "following") return "following";
    if (section === "fans") return "fans";
    return "moments";
  }
  if (pathname === "/settings") return "moments";
  if (pathname === "/discover") return "nearby-activity";
  if (pathname === "/") return "home";
  return "home";
}

/* ── HeaderBtn ──────────────────────────────────────────────────── */
function HeaderBtn({
  icon: Icon,
  onClick,
  label,
  active = false,
}: {
  icon: React.ElementType;
  onClick?: () => void;
  label: string;
  active?: boolean;
}) {
  return (
    <motion.button
      whileTap={{
        scale: 0.88,
        transition: { type: "spring", stiffness: 600, damping: 28 },
      }}
      whileHover={{
        scale: 1.18,
        transition: { type: "spring", stiffness: 400, damping: 16, mass: 0.6 },
      }}
      onClick={onClick}
      aria-label={label}
      className="flex items-center justify-center rounded-full outline-none"
      style={{ width: 36, height: 36 }}
    >
      <Icon
        style={{
          width: 17,
          height: 17,
          color: active ? "var(--neu-accent-primary)" : "var(--neu-text-muted)",
          transition: "color 0.2s",
        }}
        strokeWidth={active ? 2 : 1.5}
      />
    </motion.button>
  );
}

/* ── SidebarNavItem ─────────────────────────────────────────────── */
function SidebarNavItem({
  item,
  isActive,
  onClick,
}: {
  item: NavItem;
  isActive: boolean;
  onClick: () => void;
}) {
  const Icon = item.icon;
  const { isDark } = useTheme();
  const [hovered, setHovered] = useState(false);
  const showHover = hovered && !isActive;

  // 极淡的背景填充 — 用 currentColor 触感的中性灰，避免和主题色冲突
  const hoverBg = showHover
    ? isDark
      ? "rgba(255,255,255,0.045)"
      : "rgba(0,0,0,0.035)"
    : "transparent";

  return (
    <motion.button
      whileTap={{ scale: 0.94 }}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      onClick={onClick}
      className="relative flex items-center gap-3 w-full rounded-[10px] outline-none text-left"
      style={{
        padding: "9px 12px",
        background: isActive ? "var(--neu-accent-primary)" : hoverBg,
        color: isActive
          ? "var(--text-on-accent, #fff)"
          : showHover
            ? "var(--neu-text)"
            : "var(--neu-text-muted)",
        boxShadow: isActive
          ? "0 2px 10px var(--neu-glow, rgba(0,0,0,0.10))"
          : "none",
        transition:
          "background 0.16s ease, color 0.16s ease, box-shadow 0.2s ease",
      }}
    >
      {/* 左侧指示条 — 仅在 hover 时显示，主题色，细 */}
      <span
        aria-hidden
        style={{
          position: "absolute",
          left: 4,
          top: "50%",
          width: 3,
          height: showHover ? 16 : 0,
          marginTop: showHover ? -8 : 0,
          borderRadius: 2,
          background: "var(--neu-accent-primary)",
          opacity: showHover ? 0.9 : 0,
          transition:
            "height 0.18s ease, opacity 0.18s ease, margin-top 0.18s ease",
          pointerEvents: "none",
        }}
      />
      <Icon
        style={{
          width: 17,
          height: 17,
          flexShrink: 0,
          color: isActive
            ? "var(--text-on-accent, #fff)"
            : showHover
              ? "var(--neu-accent-primary)"
              : "var(--neu-text-muted)",
          transition: "color 0.16s ease",
        }}
        strokeWidth={isActive ? 2 : 1.6}
      />
      <span
        style={{
          fontSize: 14,
          fontWeight: isActive ? 500 : 400,
          letterSpacing: "0.01em",
        }}
      >
        {item.label}
      </span>
    </motion.button>
  );
}

/* ── LocationCard ───────────────────────────────────────────────── */
function LocationCard({ onClick }: { onClick: () => void }) {
  return (
    <motion.button
      whileTap={{ scale: 0.97 }}
      whileHover={{
        y: -2,
        transition: { type: "spring", stiffness: 360, damping: 20 },
      }}
      onClick={onClick}
      className="flex items-center gap-3 w-full rounded-[16px] outline-none text-left neu-card-hover"
      style={{
        padding: "10px 12px",
        background: "var(--neu-bg)",
        border: "1px solid var(--neu-border)",
        boxShadow:
          "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
        transition: "box-shadow 0.22s cubic-bezier(0.22,1,0.36,1)",
      }}
    >
      {/* Map pin circle */}
      <div
        className="flex items-center justify-center rounded-full flex-shrink-0"
        style={{
          width: 34,
          height: 34,
          background: "var(--neu-accent-primary)",
          boxShadow: "0 3px 10px var(--neu-glow)",
        }}
      >
        <Navigation2
          style={{
            width: 14,
            height: 14,
            color: "var(--text-on-accent, white)",
          }}
          strokeWidth={2}
        />
      </div>

      <div className="flex-1 min-w-0 text-left">
        <p style={{ fontSize: 12, color: "var(--neu-text)", fontWeight: 500 }}>
          地图探索
        </p>
        <div className="flex items-center gap-1">
          <span
            className="inline-block rounded-full"
            style={{ width: 5, height: 5, background: "#22c55e" }}
          />
          <p
            style={{
              fontSize: 10,
              lineHeight: 1.4,
              color: "var(--neu-text-muted)",
            }}
          >
            徐州 · 云龙区
          </p>
        </div>
      </div>

      <ChevronRight
        style={{
          width: 13,
          height: 13,
          color: "var(--neu-text-muted)",
          flexShrink: 0,
        }}
        strokeWidth={1.5}
      />
    </motion.button>
  );
}

/* ══════════════════════════════════════════════════════════════════
   DesktopLayout — full-screen sidebar layout for ≥1024px
══════════════════════════════════════════════════════════════════ */
export function DesktopLayout() {
  const { isDark, toggleDark } = useTheme();
  const location = useLocation();
  const navigate = useNavigate();
  const [showCreate, setShowCreate] = useState(false);
  const createTrapRef = useFocusTrap(showCreate, () => setShowCreate(false));

  /* Track which sidebar item is visually active */
  const [activeNavId, setActiveNavId] = useState(() =>
    getNavIdFromPath(location.pathname, location.search),
  );

  /* Sync with browser back/forward navigation */
  useEffect(() => {
    setActiveNavId(getNavIdFromPath(location.pathname, location.search));
  }, [location.pathname, location.search]);

  const handleNavClick = (item: NavItem) => {
    setActiveNavId(item.id);
    navigate(item.path);
  };

  return (
    <div
      className="flex flex-col h-screen w-screen overflow-hidden transition-colors duration-500 font-sans"
      style={{ backgroundColor: "var(--page-bg)", color: "var(--neu-text)" }}
    >
      {/* ═══════════════════════════════════════════════════════
          HEADER — 52px, full width
      ═══════════════════════════════════════════════════════ */}
      <header
        className="flex-shrink-0 flex items-center px-5 gap-2"
        style={{
          height: 52,
          backgroundColor: "var(--page-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        {/* Logo */}
        <motion.div
          whileTap={{ scale: 0.97 }}
          className="flex items-center gap-2 cursor-pointer mr-3 select-none"
          onClick={() => {
            setActiveNavId("home");
            navigate("/");
          }}
        >
          <img
            src={imgHotSpot}
            alt="HotSpot 图标"
            className="rounded-full object-cover flex-shrink-0"
            style={{ width: 28, height: 28 }}
          />
          <span
            className="tracking-tight"
            style={{ fontSize: 17, color: "var(--neu-text)" }}
          >
            HotSpot
          </span>
        </motion.div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* ── Right Actions ───────────────────────────────── */}
        <div className="flex items-center gap-0.5">
          <HeaderBtn
            icon={Search}
            label="搜索"
            onClick={() => navigate("/search")}
          />
          <HeaderBtn
            icon={isDark ? Sun : Moon}
            label={isDark ? "切换浅色模式" : "切换深色模式"}
            onClick={toggleDark}
          />

          {/* Create (+) button — accent circle */}
          <motion.button
            whileTap={{ scale: 0.88 }}
            whileHover={{ scale: 1.05 }}
            onClick={() => setShowCreate(true)}
            aria-label="发布"
            className="flex items-center justify-center rounded-full outline-none mx-1.5 transition-shadow duration-200"
            style={{
              width: 32,
              height: 32,
              background: "var(--neu-accent-primary)",
              boxShadow: "0 4px 12px rgba(0,0,0,0.16)",
            }}
          >
            <Plus
              style={{
                width: 15,
                height: 15,
                color: "var(--text-on-accent, white)",
              }}
              strokeWidth={2.5}
            />
          </motion.button>

          <HeaderBtn
            icon={Bell}
            label="通知"
            onClick={() => {
              setActiveNavId("notifications");
              navigate("/notifications");
            }}
          />

          {/* Settings shortcut */}
          <HeaderBtn
            icon={Settings}
            label="设置"
            onClick={() => {
              setActiveNavId("moments");
              navigate("/settings");
            }}
          />

          {/* Avatar with notification badge */}
          <motion.div
            whileTap={{ scale: 0.94 }}
            onClick={() => {
              setActiveNavId("moments");
              navigate("/profile");
            }}
            className="relative cursor-pointer ml-1.5 flex-shrink-0"
            style={{ width: 32, height: 32 }}
          >
            <img
              src={USER_AVATAR}
              alt="我的头像"
              className="w-full h-full rounded-full object-cover"
              style={{
                border: "1.5px solid var(--neu-border)",
              }}
            />
            {/* Badge */}
            <div
              className="absolute flex items-center justify-center rounded-full pointer-events-none"
              style={{
                color: "var(--text-on-accent, white)",
                width: 15,
                height: 15,
                fontSize: 8,
                fontWeight: 700,
                top: -3,
                right: -3,
                background: "var(--neu-accent-primary)",
                boxShadow: "0 1px 4px rgba(0,0,0,0.25)",
                border: "1.5px solid var(--page-bg)",
              }}
            >
              12
            </div>
          </motion.div>
        </div>
      </header>

      {/* ═══════════════════════════════════════════════════════
          BODY — sidebar + main content
      ═══════════════════════════════════════════════════════ */}
      <div className="flex flex-1 overflow-hidden p-4 gap-4">
        {/* ── Sidebar (192px, fixed width) ─────────────────── */}
        <aside className="flex flex-col flex-shrink-0" style={{ width: 192 }}>
          {/* Scrollable nav region — takes all available space */}
          <div
            className="flex-1 flex flex-col overflow-y-auto no-scrollbar min-h-0"
            style={{ overscrollBehavior: "contain" }}
          >
            {/* Section 1: Discovery nav */}
            <div className="flex flex-col gap-0.5">
              {TOP_NAV.map((item) => (
                <SidebarNavItem
                  key={item.id}
                  item={item}
                  isActive={activeNavId === item.id}
                  onClick={() => handleNavClick(item)}
                />
              ))}
            </div>

            {/* Divider */}
            <div
              className="my-3"
              style={{ height: 1, background: "var(--neu-border)" }}
            />

            {/* Section 2 label */}
            <p
              className="px-3 mb-2"
              style={{
                fontSize: 10,
                fontWeight: 600,
                letterSpacing: "0.12em",
                textTransform: "uppercase",
                color: "var(--neu-text-muted)",
                opacity: 0.9,
              }}
            >
              个人中心
            </p>

            {/* Section 2: Personal nav */}
            <div className="flex flex-col gap-0.5">
              {PERSONAL_NAV.map((item) => (
                <SidebarNavItem
                  key={item.id}
                  item={item}
                  isActive={activeNavId === item.id}
                  onClick={() => handleNavClick(item)}
                />
              ))}
            </div>
          </div>

          {/* ── Location Anchor — ALWAYS pinned to bottom ─── */}
          {/* flex-shrink-0 ensures this never gets squeezed up */}
          <div className="flex-shrink-0 pt-3">
            <LocationCard
              onClick={() => {
                navigate("/location");
              }}
            />
          </div>
        </aside>

        {/* ── Main Content Card ─────────────────────────────── */}
        <main
          className="flex-1 overflow-hidden rounded-[24px] transition-colors duration-500"
          style={{
            background: "var(--neu-bg)",
            border: "1px solid var(--neu-border)",
            boxShadow:
              "8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light), 0 2px 6px rgba(0,0,0,0.06)",
          }}
        >
          {/* Inner scroll container — sticky headers inside will stick to this */}
          <div
            className="h-full overflow-y-auto no-scrollbar"
            style={{ overscrollBehavior: "contain" }}
          >
            <Outlet />
          </div>
        </main>
      </div>

      {/* ═══════════════════════════════════════════════════════
          CREATE MODAL — centered dialog on desktop
      ══════════════════════════════════════════════════════ */}
      <AnimatePresence>
        {showCreate && (
          <>
            {/* Backdrop */}
            <motion.div
              key="backdrop"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.18 }}
              className="fixed inset-0 z-50"
              style={{
                background: "rgba(0,0,0,0.42)",
                backdropFilter: "blur(5px)",
              }}
              onClick={() => setShowCreate(false)}
            />

            {/* Centering wrapper — doesn't participate in animation */}
            <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
              {/* Animated modal */}
              <motion.div
                key="modal"
                ref={createTrapRef}
                role="dialog"
                aria-modal="true"
                aria-label="发布新动态"
                initial={{ opacity: 0, scale: 0.93, y: 14 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.93, y: 14 }}
                transition={{ type: "spring", damping: 26, stiffness: 300 }}
                className="overflow-hidden pointer-events-auto"
                style={{
                  width: 480,
                  height: "78vh",
                  maxHeight: 700,
                  borderRadius: 24,
                  background: "var(--neu-bg)",
                  boxShadow:
                    "0 24px 80px rgba(0,0,0,0.24), 0 0 0 1px var(--neu-border), 8px 8px 24px var(--neu-shadow-dark), -8px -8px 24px var(--neu-shadow-light)",
                }}
              >
                <CreateSheet onClose={() => setShowCreate(false)} />
              </motion.div>
            </div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}
