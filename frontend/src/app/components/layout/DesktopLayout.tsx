import React, { useState, useEffect } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import {
  Home, Users, Rss, MessageCircle, Bell, ImageIcon,
  Heart, Search, Settings,
  Plus, Navigation2, ChevronRight, UserRound, PanelLeftClose, PanelLeftOpen,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useTheme } from "../../contexts/ThemeContext";
import { AppearanceMenu } from "./AppearanceMenu";
import { ActionIconButton } from "../shared/ActionIconButton";
import { ROUTES, getDesktopNavId, type DesktopNavId } from "../../lib/routes";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDismissibleLayer } from "../../hooks/useDismissibleLayer";
import { useComposer } from "../../contexts/ComposerContext";
import { CreateSheet } from "./CreateSheet";
import { cn } from "../../../lib/utils";
import imgHotSpot from "../../../assets/9254a1e0e11b516a6c1aecb80d92b66f2caa243e.png";
import { useAuth } from "../../contexts/AuthContext";

interface NavItem {
  id: DesktopNavId;
  icon: React.ElementType;
  label: string;
  path: string;
}

const TOP_NAV: NavItem[] = [
  { id: "home",            icon: Home,        label: "首页",     path: ROUTES.home },
  { id: "nearby-people",   icon: Users,       label: "附近的人", path: ROUTES.nearbyPeople },
  { id: "nearby-groups",   icon: UserRound,   label: "附近群组", path: ROUTES.nearbyGroups },
  { id: "nearby-activity", icon: Rss,   label: "附近瞬间", path: ROUTES.nearbyMoments },
];

const PERSONAL_NAV: NavItem[] = [
  { id: "messages",      icon: MessageCircle, label: "消息",     path: ROUTES.messages },
  { id: "moments",       icon: ImageIcon,     label: "我的瞬间", path: ROUTES.meSection("moments") },
  { id: "my-groups",     icon: Users,         label: "我的群组", path: ROUTES.meSection("groups") },
  { id: "relationships", icon: Heart,         label: "关系",     path: ROUTES.meSection("relationships") },
];

const LAST_DESKTOP_NAV_KEY = "hotspot:last-desktop-nav";
const DESKTOP_NAV_IDS: DesktopNavId[] = [
  "home", "nearby-people", "nearby-groups", "nearby-activity",
  "messages", "notifications", "moments", "my-groups", "relationships",
];

function readLastDesktopNav(): DesktopNavId | null {
  try {
    const value = window.sessionStorage.getItem(LAST_DESKTOP_NAV_KEY);
    return DESKTOP_NAV_IDS.includes(value as DesktopNavId) ? value as DesktopNavId : null;
  } catch {
    return null;
  }
}

function persistLastDesktopNav(id: DesktopNavId) {
  try {
    window.sessionStorage.setItem(LAST_DESKTOP_NAV_KEY, id);
  } catch {
    // Storage can be unavailable in private or embedded browser contexts.
  }
}

/* ── HeaderBtn ──────────────────────────────────────────────────── */
function HeaderBtn({
  icon,
  onClick,
  label,
  active = false,
}: {
  icon: React.ElementType;
  onClick?: () => void;
  label: string;
  active?: boolean;
}) {
  return <ActionIconButton icon={icon} label={label} onClick={onClick} variant="ghost" active={active} size={36} />;
}

/* ── SidebarNavItem ─────────────────────────────────────────────── */
function SidebarNavItem({
  item,
  isActive,
  onClick,
  collapsed = false,
}: {
  item: NavItem;
  isActive: boolean;
  onClick: () => void;
  collapsed?: boolean;
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
      whileTap={{ scale: 0.985 }}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      onClick={onClick}
      aria-label={item.label}
      title={collapsed ? item.label : undefined}
      className={`relative flex outline-none text-left ${collapsed ? "h-10 w-10 self-center items-center justify-center rounded-xl" : "w-full items-center gap-3 rounded-[10px]"}`}
      style={{
        padding: collapsed ? 0 : "9px 12px",
        background: isActive ? (hovered ? "var(--neu-active-hover)" : "var(--neu-accent-soft)") : hoverBg,
        border: isActive ? `1px solid ${hovered ? "var(--neu-accent-primary)" : "var(--neu-accent-soft-border)"}` : "1px solid transparent",
        color: isActive || showHover ? "var(--neu-accent-primary)" : "var(--neu-text-muted)",
        boxShadow: "none",
        transition: "background 0.16s ease, color 0.16s ease, border-color 0.16s ease",
      }}
    >
      {/* 左侧指示条 — 仅在 hover 时显示，主题色，细 */}
      <span
        aria-hidden
        style={{
          position: "absolute",
          left: collapsed ? 2 : 4,
          top: "50%",
          width: 3,
          // Same hover/active rule in both states: the bar appears on hover OR
          // when active, so the collapsed rail does not lose its hover affordance.
          height: isActive ? (collapsed ? 18 : 16) : showHover ? (collapsed ? 14 : 16) : 0,
          marginTop: isActive ? (collapsed ? -9 : -8) : showHover ? (collapsed ? -7 : -8) : 0,
          borderRadius: 2,
          background: "var(--neu-accent-primary)",
          opacity: isActive ? 1 : showHover ? 0.9 : 0,
          transition: "height 0.18s ease, opacity 0.18s ease, margin-top 0.18s ease",
          pointerEvents: "none",
        }}
      />
      <Icon
        style={{
          width: 18,
          height: 18,
          flexShrink: 0,
          color: isActive || showHover ? "var(--neu-accent-primary)" : "var(--neu-text-muted)",
          transition: "color 0.16s ease",
        }}
        strokeWidth={isActive ? 2 : 1.6}
      />
      <span
        className={collapsed ? "sr-only" : undefined}
        style={collapsed ? undefined : {
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
function LocationCard({ onClick, collapsed = false }: { onClick: () => void; collapsed?: boolean }) {
  const { isDark } = useTheme();
  const [hovered, setHovered] = useState(false);
  const collapsedHover = isDark ? "rgba(255,255,255,0.045)" : "rgba(0,0,0,0.035)";

  return (
    <motion.button
      whileTap={{ scale: 0.985 }}
      whileHover={undefined}
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      onClick={onClick}
      title={collapsed ? "地图探索" : undefined}
      aria-label="地图探索"
      className={`flex items-center outline-none text-left ${collapsed ? "h-10 w-10 self-center justify-center rounded-xl" : "w-full gap-3 rounded-[16px]"}`}
      style={{
        padding: collapsed ? 0 : "10px 12px",
        // Collapsed state mirrors the sidebar nav item: transparent by default,
        // faint neutral fill on hover — no permanent accent-soft "always-active"
        // look, so the map anchor reads as a peer of the nav items rather than a
        // stuck-on control.
        background: hovered ? collapsedHover : collapsed ? "transparent" : "var(--neu-bg)",
        border: "1px solid transparent",
        boxShadow: "none",
        transition: "background 0.16s ease, color 0.16s ease",
      }}
    >
      <div
        className="flex items-center justify-center rounded-full flex-shrink-0"
        style={{
          // Solid accent circle in both states — the map entry keeps its visual
          // anchor weight whether the rail is open or collapsed.
          width: collapsed ? 30 : 34, height: collapsed ? 30 : 34,
          background: "var(--neu-accent-primary)",
          boxShadow: "none",
        }}
      >
        <Navigation2 style={{ width: 18, height: 18, color: "white" }} strokeWidth={2} />
      </div>

      <div className={collapsed ? "sr-only" : "flex-1 min-w-0 text-left"}>
        <p style={{ fontSize: 12, color: "var(--neu-text)", fontWeight: 500 }}>地图探索</p>
        <div className="flex items-center gap-1">
          <span className="inline-block rounded-full" style={{ width: 5, height: 5, background: "var(--semantic-online)" }} />
          <p style={{ fontSize: 10, lineHeight: 1.4, color: "var(--neu-text-muted)" }}>徐州 · 云龙湖畔</p>
        </div>
      </div>

      {!collapsed && <ChevronRight style={{ width: 13, height: 13, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />}
    </motion.button>
  );
}

/* ══════════════════════════════════════════════════════════════════
   DesktopLayout — full-screen sidebar layout for ≥1024px
══════════════════════════════════════════════════════════════════ */
export function DesktopLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { composerOpen: showCreate, openComposer, closeComposer } = useComposer();
  useDismissibleLayer({ open: showCreate, onDismiss: closeComposer });

  /* Keep the last meaningful rail context while viewing utility/detail routes. */
  const [activeNavId, setActiveNavId] = useState<DesktopNavId>(() =>
    getDesktopNavId(location.pathname, location.search) ?? readLastDesktopNav() ?? "home"
  );
  const isSplitWorkspace = location.pathname === ROUTES.messages;
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);
  // Reactive current user from AuthContext — reflects login/logout immediately.
  const { user, requireAuth, status } = useAuth();
  const authUser = user
    ? { avatar_url: user.avatar_url, nickname: user.nickname, user_id: user.user_id }
    : null;

  /** Navigate only when authenticated; otherwise prompt login (anonymous browsing strategy). */
  const guardNav = (label: string, path: string) => {
    if (status === "authenticated") { navigate(path); return; }
    requireAuth({ action: label, run: () => navigate(path) });
  };

  /* Only explicit rail routes replace the current rail context. */
  useEffect(() => {
    const resolvedNav = getDesktopNavId(location.pathname, location.search);
    if (resolvedNav) {
      setActiveNavId(resolvedNav);
      persistLastDesktopNav(resolvedNav);
    }
  }, [location.pathname, location.search]);

  const handleNavClick = (item: NavItem) => {
    // Personal/private destinations require auth; public discovery does not.
    const requiresAuth = item.id === "messages" || item.id === "moments"
      || item.id === "my-groups" || item.id === "relationships";
    if (requiresAuth && status !== "authenticated") {
      requireAuth({ action: item.label, run: () => { setActiveNavId(item.id); persistLastDesktopNav(item.id); navigate(item.path); } });
      return;
    }
    setActiveNavId(item.id);
    persistLastDesktopNav(item.id);
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
        className="relative z-30 flex-shrink-0 flex items-center px-5 gap-2"
        style={{
          height: 52,
          backgroundColor: "var(--page-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        {/* Logo */}
        <motion.button
          type="button"
          whileTap={{ scale: 0.97 }}
          className="flex items-center gap-2 mr-3 select-none rounded-full neu-focus-ring focus-visible:ring-offset-[var(--page-bg)]"
          onClick={() => {
            setActiveNavId("home");
            navigate(ROUTES.home);
          }}
        >
          <img
            src={imgHotSpot}
            alt="HotSpot Logo"
            className="rounded-full object-cover flex-shrink-0"
            style={{ width: 28, height: 28 }}
          />
          <span
            className="tracking-tight"
            style={{ fontSize: 17, color: "var(--neu-text)" }}
          >
            HotSpot
          </span>
        </motion.button>

        <HeaderBtn
          icon={isSidebarCollapsed ? PanelLeftOpen : PanelLeftClose}
          label={isSidebarCollapsed ? "展开侧边栏" : "收起侧边栏"}
          onClick={() => setIsSidebarCollapsed((collapsed) => !collapsed)}
        />

        {/* Spacer */}
        <div className="flex-1" />

        {/* ── Right Actions ───────────────────────────────── */}
        <div className="flex items-center gap-0.5">
          <HeaderBtn
            icon={Search}
            label="搜索"
            onClick={() => navigate(ROUTES.search)}
          />
          <HeaderBtn
            icon={Plus}
            label="发布"
            onClick={() => {
              if (status === "authenticated") { openComposer(); return; }
              requireAuth({ action: "发布动态", run: openComposer });
            }}
          />

          <HeaderBtn
            icon={Bell}
            label="通知"
            onClick={() => guardNav("查看通知", ROUTES.notify)}
          />

          <AppearanceMenu />

          <HeaderBtn
            icon={Settings}
            label="设置"
            onClick={() => navigate(ROUTES.settings)}
          />

          {authUser && (
            <motion.button
              type="button"
              whileTap={{ scale: 0.92 }}
              onClick={() => guardNav("打开个人主页", ROUTES.me)}
              aria-label="打开个人主页"
              className="relative ml-1.5 flex-shrink-0 rounded-full neu-focus-ring focus-visible:ring-offset-[var(--page-bg)]"
              style={{ width: 32, height: 32 }}
            >
              <img
                src={authUser.avatar_url || ""}
                alt={authUser.nickname || "用户"}
                className="w-full h-full rounded-full object-cover"
                style={{
                  border: "1.5px solid var(--neu-border)",
                }}
              />
            </motion.button>
          )}
          {!authUser && (
            <motion.button
              type="button"
              whileTap={{ scale: 0.96 }}
              onClick={() => navigate(ROUTES.login)}
              aria-label="登录"
              className="ml-1.5 flex-shrink-0 rounded-full px-3 neu-focus-ring"
              style={{
                height: 32,
                background: "var(--neu-accent-primary)",
                color: "white",
                fontSize: 12,
              }}
            >
              登录
            </motion.button>
          )}
        </div>
      </header>

      {/* ═══════════════════════════════════════════════════════
          BODY — sidebar + main content
      ═══════════════════════════════════════════════════════ */}
      <div className="desktop-workspace relative z-0 flex flex-1 overflow-hidden" style={{ gap: 0 }}>

        {/* ── Sidebar (192px, fixed width) ─────────────────── */}
        <aside
          className="flex flex-col flex-shrink-0 overflow-hidden box-border"
          style={{
            width: isSidebarCollapsed ? 56 : "calc(192px + var(--workspace-gutter) * 2)",
            padding: isSidebarCollapsed
              ? "calc(var(--workspace-gutter) / 2) 8px"
              : "var(--workspace-gutter)",
            background: "color-mix(in srgb, var(--neu-bg) 80%, transparent)",
            borderRight: "1px solid var(--neu-border)",
          }}
        >
          {/* Scrollable nav region — takes all available space */}
          <div className="flex-1 flex flex-col overflow-y-auto no-scrollbar min-h-0">

            {/* Section 1: Discovery nav */}
            <div className={`flex flex-col gap-0.5 ${isSidebarCollapsed ? "items-center" : ""}`}>
              {TOP_NAV.map((item) => (
                <SidebarNavItem
                  key={item.id}
                  item={item}
                  isActive={activeNavId === item.id}
                  onClick={() => handleNavClick(item)}
                  collapsed={isSidebarCollapsed}
                />
              ))}
            </div>

            {/* Divider */}
            <div
              className="my-3 self-center"
              style={{ height: 1, width: isSidebarCollapsed ? 24 : "100%", background: "var(--neu-border)" }}
            />

            {/* Section 2 label */}
            <p
              className={isSidebarCollapsed ? "sr-only" : "px-3 mb-2"}
              style={{
                fontSize: 10,
                fontWeight: 600,
                letterSpacing: "0.12em",
                textTransform: "uppercase",
                color: "var(--neu-text-muted)",
                opacity: 0.65,
              }}
            >
              个人中心
            </p>

            {/* Section 2: Personal nav */}
            <div className={`flex flex-col gap-0.5 ${isSidebarCollapsed ? "items-center" : ""}`}>
              {PERSONAL_NAV.map((item) => (
                <SidebarNavItem
                  key={item.id}
                  item={item}
                  isActive={activeNavId === item.id}
                  onClick={() => handleNavClick(item)}
                  collapsed={isSidebarCollapsed}
                />
              ))}
            </div>

          </div>

          {/* ── Location Anchor — ALWAYS pinned to bottom ─── */}
          {/* flex-shrink-0 ensures this never gets squeezed up */}
          <div className={`flex flex-shrink-0 pt-3 ${isSidebarCollapsed ? "justify-center" : ""}`}>
            <LocationCard
              collapsed={isSidebarCollapsed}
              onClick={() => {
                navigate(ROUTES.map);
              }}
            />
          </div>
        </aside>

        {/* ── Main Content Card ─────────────────────────────── */}
        <main
          className="desktop-main-panel flex-1 min-w-0 overflow-hidden transition-colors duration-500"
        >
          {/* Inner scroll container — sticky headers inside will stick to this */}
          <div className={cn("desktop-main-scroll h-full no-scrollbar", isSplitWorkspace ? "overflow-hidden" : "overflow-y-auto")}>
            <div className={cn("desktop-page-canvas", isSplitWorkspace && "h-full")}>
              <Outlet />
            </div>
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
              onClick={() => closeComposer()}
            />

            {/* Centering wrapper — doesn't participate in animation */}
            <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
              {/* Animated modal */}
              <motion.div
                key="modal"
                initial={{ opacity: 0, scale: 0.93, y: 14 }}
                animate={{ opacity: 1, scale: 1, y: 0 }}
                exit={{ opacity: 0, scale: 0.93, y: 14 }}
                transition={{ type: "spring", damping: 26, stiffness: 300 }}
                className="overflow-hidden pointer-events-auto"
                role="dialog"
                aria-modal="true"
                aria-labelledby="create-composer-title"
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
                <CreateSheet onClose={() => closeComposer()} />
              </motion.div>
            </div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}