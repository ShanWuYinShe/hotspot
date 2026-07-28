import React, { useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import { MapPin, Compass, Plus, X, MessageCircle, CircleUser } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { cn } from "../../../lib/utils";
import { useTheme } from "../../contexts/ThemeContext";
import { usePlatform } from "../../contexts/PlatformContext";
import { CreateSheet } from "./CreateSheet";
import { StatusBar } from "./StatusBar";
import { DesktopLayout } from "./DesktopLayout";
import { TabletLayout } from "./TabletLayout";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useDismissibleLayer } from "../../hooks/useDismissibleLayer";
import { useComposer } from "../../contexts/ComposerContext";
import { ROUTES, isPrimaryRouteActive, isSubPagePath } from "../../lib/routes";

const NAV_ITEMS = [
  { path: ROUTES.home,     icon: MapPin,         label: "附近" },
  { path: ROUTES.nearbyMoments, icon: Compass,        label: "发现" },
  { path: ROUTES.messages, icon: MessageCircle,  label: "消息" },
  { path: ROUTES.me,  icon: CircleUser,     label: "我的" },
];

export function MainLayout() {
  const { platform, orientation } = usePlatform();
  const { isDark, theme } = useTheme();
  const location = useLocation();
  const navigate = useNavigate();
  const { composerOpen: showCreate, closeComposer, toggleComposer } = useComposer();
  useDismissibleLayer({ open: showCreate, onDismiss: closeComposer });

  // Resolve actual CSS-variable colour values so Motion's layoutId engine
  // always receives concrete hex strings — it caches computed values and
  // won't re-read CSS variables when a class on <html> changes.
  const accentColor = useMemo(() => {
    return getComputedStyle(document.documentElement)
      .getPropertyValue("--neu-accent-primary").trim() || "#6366f1";
  }, [theme, isDark]);

  const mutedColor = useMemo(() => {
    return getComputedStyle(document.documentElement)
      .getPropertyValue("--neu-text-muted").trim() || "#8D94A0";
  }, [theme, isDark]);

  // Glass nav background — theme-aware semi-transparent tint
  const glassNavBg = useMemo(() => {
    if (isDark) {
      if (theme === "forest")  return "rgba(22,26,22,0.92)";   /* 清野 dark: #161A16 */
      if (theme === "vibrant") return "rgba(19,15,38,0.92)";   /* 星际 dark: #130F26 */
      return "rgba(19,15,38,0.92)";                             /* fallback: 星际 */
    }
    if (theme === "forest")  return "rgba(249,249,245,0.90)";  /* 清野 light: #F9F9F5 */
    if (theme === "vibrant") return "rgba(250,250,254,0.90)";  /* 星际 light: #FAFAFE */
    return "rgba(250,250,254,0.90)";                            /* fallback: 星际 */
  }, [theme, isDark]);

  const glassNavBorder = isDark
    ? theme === "forest"  ? "rgba(82,184,120,0.16)"
    : theme === "vibrant" ? "rgba(129,140,248,0.18)"
    : "rgba(255,255,255,0.08)"
    : theme === "forest"  ? "rgba(61,122,85,0.15)"
    : theme === "vibrant" ? "rgba(99,102,241,0.14)"
    : "rgba(255,255,255,0.7)";

  /* Platform routing */
  if (platform === "desktop") return <DesktopLayout />;
  if (platform === "tablet") return <TabletLayout />;
  const isMobileLandscape = orientation === "landscape";

  /* ── Mobile phone-frame layout ─────────────────────────────────── */
  if (isMobileLandscape) {
    return (
      <MobileLandscape
        showCreate={showCreate}
        onToggle={toggleComposer}
        onClose={closeComposer}
      />
    );
  }

  const isSubPage = isSubPagePath(location.pathname);

  const fabGlow = (() => {
    if (showCreate) return "none";
    if (theme === "forest") return isDark
      ? "0 6px 22px rgba(82,184,120,0.38), 0 2px 6px rgba(0,0,0,0.20)"
      : "0 6px 18px rgba(61,122,85,0.32), 0 2px 4px rgba(0,0,0,0.10)";
    return isDark
      ? "0 6px 22px rgba(129,140,248,0.42), 0 2px 6px rgba(0,0,0,0.20)"
      : "0 6px 18px rgba(99,102,241,0.38), 0 2px 4px rgba(0,0,0,0.10)";
  })();

  return (
    <div
      className="min-h-[100dvh] w-full flex justify-center items-start font-sans transition-colors duration-500"
      style={{ backgroundColor: "var(--page-bg)" }}
    >
      <div
        className={cn(
          "flex flex-col h-[100dvh] w-full",
          "transition-colors duration-500 overflow-hidden relative"
        )}
        style={{ backgroundColor: "var(--neu-bg)", color: "var(--neu-text)" }}
      >
        <StatusBar />

        <main className="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar scroll-smooth">
          <Outlet />
        </main>

        {!isSubPage && (
          <nav
            role="navigation"
            aria-label="Main navigation"
            className="flex-shrink-0 w-full"
            style={{
              background: glassNavBg,
              backdropFilter: "blur(24px) saturate(1.8)",
              WebkitBackdropFilter: "blur(24px) saturate(1.8)",
              borderTop: `1px solid ${glassNavBorder}`,
              boxShadow: `0 -1px 0 ${glassNavBorder}, 0 -8px 28px var(--neu-shadow-dark)`,
              transition: "background 0.4s ease, border-color 0.4s ease",
            }}
          >
            <div
              className="flex items-center justify-between px-2"
              style={{
                paddingTop: 8,
                paddingBottom: "max(20px, env(safe-area-inset-bottom, 20px))",
              }}
            >
              {NAV_ITEMS.slice(0, 2).map((item) => {
                const isActive = isPrimaryRouteActive(location.pathname, item.path);
                return (
                  <NavTab
                    key={item.path}
                    label={item.label}
                    isActive={isActive}
                    onClick={() => navigate(item.path)}
                    Icon={item.icon}
                    accentColor={accentColor}
                    mutedColor={mutedColor}
                  />
                );
              })}

              {/* Centre FAB */}
              <div className="flex flex-col items-center" style={{ width: 68 }}>
                <motion.button
                  whileTap={{ scale: 0.88, y: 1 }}
                  className="flex items-center justify-center outline-none select-none rounded-full"
                  style={{
                    width: 50,
                    height: 50,
                    background: showCreate
                      ? "var(--neu-text-muted)"
                      : "var(--neu-accent-primary)",
                    boxShadow: fabGlow,
                    transition: "background 0.22s ease, box-shadow 0.22s ease",
                  }}
                  onClick={() => toggleComposer()}
                  aria-label={showCreate ? "关闭" : "发布"}
                >
                  <AnimatePresence mode="wait" initial={false}>
                    {showCreate ? (
                      <motion.span
                        key="close"
                        initial={{ rotate: -45, opacity: 0 }}
                        animate={{ rotate: 0, opacity: 1 }}
                        exit={{ rotate: 45, opacity: 0 }}
                        transition={{ duration: 0.18, ease: "easeOut" }}
                        style={{ display: "flex" }}
                      >
                        <X style={{ width: 20, height: 20, color: "white" }} strokeWidth={2.2} />
                      </motion.span>
                    ) : (
                      <motion.span
                        key="plus"
                        initial={{ rotate: 45, opacity: 0 }}
                        animate={{ rotate: 0, opacity: 1 }}
                        exit={{ rotate: -45, opacity: 0 }}
                        transition={{ duration: 0.18, ease: "easeOut" }}
                        style={{ display: "flex" }}
                      >
                        <Plus style={{ width: 20, height: 20, color: "white" }} strokeWidth={2.2} />
                      </motion.span>
                    )}
                  </AnimatePresence>
                </motion.button>
              </div>

              {NAV_ITEMS.slice(2, 4).map((item) => {
                const isActive = isPrimaryRouteActive(location.pathname, item.path);
                return (
                  <NavTab
                    key={item.path}
                    label={item.label}
                    isActive={isActive}
                    onClick={() => navigate(item.path)}
                    Icon={item.icon}
                    accentColor={accentColor}
                    mutedColor={mutedColor}
                  />
                );
              })}
            </div>
          </nav>
        )}

        <AnimatePresence>
          {showCreate && (
            <>
              <motion.div
                key="backdrop"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.22 }}
                className="absolute inset-0 z-40"
                style={{ background: "rgba(0,0,0,0.35)", backdropFilter: "blur(3px)" }}
                onClick={() => closeComposer()}
              />
              <motion.div
                key="sheet"
                initial={{ y: "100%" }}
                animate={{ y: 0 }}
                exit={{ y: "100%" }}
                transition={{ type: "spring", damping: 30, stiffness: 260, mass: 0.9 }}
                className="absolute bottom-0 left-0 right-0 z-50 overflow-hidden"
                role="dialog"
                aria-modal="true"
                aria-labelledby="create-composer-title"
                style={{
                  height: "92%",
                  borderRadius: "28px 28px 0 0",
                  boxShadow: "0 -8px 40px rgba(0,0,0,0.18), 0 -2px 0 var(--neu-border)",
                  backgroundColor: "var(--neu-bg)",
                }}
              >
                <CreateSheet onClose={() => closeComposer()} />
              </motion.div>
            </>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

function NavTab({
  label, isActive, onClick, Icon, accentColor, mutedColor,
}: {
  label: string;
  isActive: boolean;
  onClick: () => void;
  Icon: React.ElementType;
  accentColor: string;
  mutedColor: string;
}) {
  const hover = useHoverConfig();
  const iconColor = isActive ? accentColor : mutedColor;
  return (
    <motion.button
      whileTap={hover.tap as any}
      whileHover={!isActive ? hover.btn as any : {}}
      onClick={onClick}
      className="relative flex flex-col items-center justify-center outline-none select-none"
      style={{ width: 68, paddingTop: 10, paddingBottom: 8, gap: 4 }}
      aria-label={label}
      aria-current={isActive ? "page" : undefined}
    >
      {/* Sliding pill — tall enough to wrap both icon AND label */}
      {isActive && (
        <motion.div
          layoutId="nav-tab-pill"
          style={{
            position: "absolute",
            top: 5,
            width: 60,
            height: 48,
            borderRadius: 24,
            background: accentColor,
            opacity: 0.13,
          }}
          transition={{ type: "spring", stiffness: 500, damping: 38, mass: 0.75 }}
        />
      )}

      <div
        style={{ display: "flex", alignItems: "center", justifyContent: "center", position: "relative", zIndex: 1 }}
      >
        <Icon
          style={{ width: 22, height: 22, color: iconColor, transition: "color 0.22s ease" }}
          strokeWidth={isActive ? 2.1 : 1.5}
        />
      </div>
      <span
        style={{
          fontSize: 11, lineHeight: 1,
          color: iconColor,
          fontWeight: isActive ? 600 : 400,
          transition: "color 0.22s ease",
          letterSpacing: "0.01em",
          position: "relative", zIndex: 1,
        }}
      >
        {label}
      </span>
    </motion.button>
  );
}

/* ── Mobile Landscape: compact left rail, no status bar, full-height content ── */
function MobileLandscape({
  showCreate,
  onToggle,
  onClose,
}: {
  showCreate: boolean;
  onToggle: () => void;
  onClose: () => void;
}) {
  const location = useLocation();
  const navigate = useNavigate();

  useDismissibleLayer({ open: showCreate, onDismiss: onClose });

  const isSubPage = isSubPagePath(location.pathname);

  const isActive = (path: string) => isPrimaryRouteActive(location.pathname, path);

  return (
    <div
      className="flex h-[100dvh] w-full overflow-hidden font-sans transition-colors duration-500"
      style={{ backgroundColor: "var(--page-bg)", color: "var(--neu-text)" }}
    >
      {!isSubPage && (
        <aside
          className="flex-shrink-0 flex flex-col items-center"
          style={{
            width: 64,
            paddingTop: "max(env(safe-area-inset-top, 8px), 8px)",
            paddingBottom: "max(env(safe-area-inset-bottom, 8px), 8px)",
            paddingLeft: "max(env(safe-area-inset-left, 0px), 0px)",
            gap: 8,
            borderRight: "1px solid var(--neu-border)",
            backgroundColor: "var(--neu-bg)",
          }}
        >
          {NAV_ITEMS.slice(0, 2).map((item) => (
            <RailIcon
              key={item.path}
              Icon={item.icon}
              label={item.label}
              isActive={isActive(item.path)}
              onClick={() => navigate(item.path)}
            />
          ))}

          <motion.button
            whileTap={{ scale: 0.88 }}
            onClick={onToggle}
            className="flex items-center justify-center rounded-full outline-none my-1"
            style={{
              width: 44,
              height: 44,
              background: showCreate ? "var(--neu-text-muted)" : "var(--neu-accent-primary)",
              boxShadow: showCreate ? "none" : "0 4px 14px var(--neu-glow)",
              transition: "background 0.22s",
            }}
            aria-label={showCreate ? "关闭" : "发布"}
          >
            <AnimatePresence mode="wait" initial={false}>
              {showCreate ? (
                <motion.span
                  key="x"
                  initial={{ rotate: -45, opacity: 0 }}
                  animate={{ rotate: 0, opacity: 1 }}
                  exit={{ rotate: 45, opacity: 0 }}
                  transition={{ duration: 0.18 }}
                  style={{ display: "flex" }}
                >
                  <X style={{ width: 18, height: 18, color: "white" }} strokeWidth={2.2} />
                </motion.span>
              ) : (
                <motion.span
                  key="plus"
                  initial={{ rotate: 45, opacity: 0 }}
                  animate={{ rotate: 0, opacity: 1 }}
                  exit={{ rotate: -45, opacity: 0 }}
                  transition={{ duration: 0.18 }}
                  style={{ display: "flex" }}
                >
                  <Plus style={{ width: 18, height: 18, color: "white" }} strokeWidth={2.2} />
                </motion.span>
              )}
            </AnimatePresence>
          </motion.button>

          {NAV_ITEMS.slice(2, 4).map((item) => (
            <RailIcon
              key={item.path}
              Icon={item.icon}
              label={item.label}
              isActive={isActive(item.path)}
              onClick={() => navigate(item.path)}
            />
          ))}
        </aside>
      )}

      <main
        className="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar"
        style={{ backgroundColor: "var(--neu-bg)" }}
      >
        <Outlet />
      </main>

      <AnimatePresence>
        {showCreate && (
          <>
            <motion.div
              key="backdrop"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="fixed inset-0 z-40"
              style={{ background: "rgba(0,0,0,0.4)", backdropFilter: "blur(4px)" }}
              onClick={onClose}
            />
            <motion.div
              key="sheet"
              initial={{ x: "100%" }}
              animate={{ x: 0 }}
              exit={{ x: "100%" }}
              transition={{ type: "spring", damping: 32, stiffness: 280 }}
              className="fixed top-0 bottom-0 right-0 z-50 overflow-hidden"
              role="dialog"
              aria-modal="true"
              aria-labelledby="create-composer-title"
              style={{
                width: "min(560px, 90%)",
                borderRadius: "28px 0 0 28px",
                backgroundColor: "var(--neu-bg)",
                boxShadow: "-8px 0 40px rgba(0,0,0,0.22)",
              }}
            >
              <CreateSheet onClose={onClose} />
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
}

function RailIcon({
  Icon,
  label,
  isActive,
  onClick,
}: {
  Icon: React.ElementType;
  label: string;
  isActive: boolean;
  onClick: () => void;
}) {
  const { theme, isDark } = useTheme();
  const [hovered, setHovered] = useState(false);
  const showHover = hovered && !isActive;

  const accentColor = useMemo(() =>
    getComputedStyle(document.documentElement).getPropertyValue("--neu-accent-primary").trim() || "#6366f1"
  , [theme, isDark]);

  const mutedColor = useMemo(() =>
    getComputedStyle(document.documentElement).getPropertyValue("--neu-text-muted").trim() || "#8D94A0"
  , [theme, isDark]);

  const iconColor = isActive ? accentColor : showHover ? accentColor : mutedColor;

  // Theme-aware hover state for RailIcon
  const hoverBg = showHover
    ? theme === "forest" || theme === "vibrant"
      ? "transparent"
      : "var(--neu-bg)"
    : "transparent";

  const hoverShadow = showHover
    ? theme === "forest"
      ? isDark ? "0 0 0 1px rgba(82,184,120,0.28)" : "0 0 0 1px rgba(61,122,85,0.22)"
      : theme === "vibrant"
      ? isDark
        ? "0 6px 16px rgba(129,140,248,0.22), 0 2px 6px rgba(129,140,248,0.12)"
        : "0 6px 16px rgba(99,102,241,0.18), 0 2px 6px rgba(99,102,241,0.10)"
      : "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)"
    : "none";

  const hoverScale = showHover
    ? theme === "forest" ? 1.08 : 1.14
    : 1;

  const transitionType = theme === "forest"
    ? { type: "tween", duration: 0.16, ease: [0.22, 1, 0.36, 1] }
    : { type: "spring", stiffness: 380, damping: 18, mass: 0.65 };

  return (
    <motion.button
      whileTap={theme === "forest"
        ? { scale: 0.97, transition: { type: "tween", duration: 0.1 } } as any
        : { scale: 0.86, transition: { type: "spring", stiffness: 600, damping: 26 } } as any
      }
      onHoverStart={() => setHovered(true)}
      onHoverEnd={() => setHovered(false)}
      onClick={onClick}
      className="relative flex flex-col items-center justify-center outline-none rounded-[16px]"
      style={{
        width: 52, height: 52, gap: 3,
        background: hoverBg,
        boxShadow: hoverShadow,
        transform: `scale(${hoverScale})`,
        transition: "background 0.16s ease, box-shadow 0.2s cubic-bezier(0.22,1,0.36,1), transform 0.2s cubic-bezier(0.34,1.56,0.64,1)",
      }}
      aria-label={label}
      aria-current={isActive ? "page" : undefined}
    >
      {isActive && (
        <motion.div
          layoutId="rail-active-bg"
          style={{
            position: "absolute",
            inset: 4,
            borderRadius: 12,
            background: accentColor,
            opacity: 0.14,
          }}
          transition={{ type: "spring", stiffness: 500, damping: 38 }}
        />
      )}
      <Icon
        style={{ width: 22, height: 22, color: iconColor, transition: "color 0.18s ease", position: "relative", zIndex: 1 }}
        strokeWidth={isActive ? 2.1 : showHover ? 1.75 : 1.5}
      />
      <span style={{ fontSize: 10, lineHeight: 1, color: iconColor, fontWeight: isActive ? 600 : 400, position: "relative", zIndex: 1, transition: "color 0.18s ease" }}>
        {label}
      </span>
      {isActive && (
        <motion.div
          layoutId="rail-indicator"
          style={{ position: "absolute", left: 0, top: 10, bottom: 10, width: 3, borderRadius: 2, background: accentColor }}
        />
      )}
    </motion.button>
  );
}