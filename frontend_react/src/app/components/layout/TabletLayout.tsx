import React, { useState, useMemo } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import {
  MapPin,
  Compass,
  Plus,
  X,
  MessageCircle,
  CircleUser,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { readThemeCssVar } from "@/lib/theme";
import { useTheme } from "../../contexts/ThemeContext";
import { CreateSheet } from "./CreateSheet";
import { useFocusTrap } from "../../hooks/useFocusTrap";
import { StatusBar } from "./StatusBar";
import { useHoverConfig } from "../../hooks/useHoverConfig";

const NAV_ITEMS = [
  { path: "/", icon: MapPin, label: "附近" },
  { path: "/discover", icon: Compass, label: "发现" },
  { path: "/messages", icon: MessageCircle, label: "消息" },
  { path: "/profile", icon: CircleUser, label: "个人" },
];

const SUB_PATHS = [
  "/settings",
  "/location",
  "/notifications",
  "/profile/edit",
  "/search",
  "/nearby",
  "/nearby-groups",
];

function useIsSubPage() {
  const location = useLocation();
  return (
    SUB_PATHS.some((p) => location.pathname === p) ||
    location.pathname.startsWith("/messages/") ||
    location.pathname.startsWith("/user/") ||
    location.pathname.startsWith("/post/") ||
    location.pathname.startsWith("/events/")
  );
}

function isPathActive(itemPath: string, pathname: string) {
  if (itemPath === "/") return pathname === "/";
  return (
    pathname === itemPath ||
    pathname.startsWith(itemPath + "/") ||
    (itemPath === "/profile" && pathname === "/settings")
  );
}

/* ════════════════════════════════════════════════════════════════════
   Tablet layout — mobile-like UX
   Full-width content + floating glass pill bottom nav.
   Same layout for both portrait and landscape orientations.
   No centering / no max-width constraints — content fills 100% width.
═══════════════════════════════════════════════════════════════════════ */
function TabletMobileLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const [showCreate, setShowCreate] = useState(false);
  const isSubPage = useIsSubPage();

  return (
    <div
      className="flex flex-col h-[100dvh] w-full overflow-hidden transition-colors duration-500 font-sans"
      style={{ backgroundColor: "var(--neu-bg)", color: "var(--neu-text)" }}
    >
      <StatusBar />

      {/* ── Scrollable area: full width, individual elements keep their own proportions ── */}
      <main
        className="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar w-full"
        style={{ backgroundColor: "var(--neu-bg)" }}
      >
        <Outlet />
      </main>

      {/* ── Floating glass pill bottom nav ─────────────────────────── */}
      {!isSubPage && (
        <div
          className="flex-shrink-0 w-full flex justify-center"
          style={{
            padding: "8px 24px",
            paddingBottom: "max(env(safe-area-inset-bottom, 16px), 16px)",
            backgroundColor: "transparent",
          }}
        >
          <div
            className="flex items-center"
            style={{
              width: "100%",
              maxWidth: 520,
              borderRadius: 40,
              background: "var(--bottom-nav-glass-bg)",
              backdropFilter: "blur(28px) saturate(1.8)",
              WebkitBackdropFilter: "blur(28px) saturate(1.8)",
              boxShadow: "var(--bottom-nav-glass-shadow)",
              border: "1px solid var(--bottom-nav-glass-border)",
              padding: "8px 16px",
              gap: 6,
            }}
          >
            {NAV_ITEMS.slice(0, 2).map((item) => (
              <NavPill
                key={item.path}
                icon={item.icon}
                label={item.label}
                isActive={isPathActive(item.path, location.pathname)}
                onClick={() => navigate(item.path)}
              />
            ))}

            {/* Centre FAB */}
            <div
              className="flex items-center justify-center flex-shrink-0"
              style={{ width: 64 }}
            >
              <FabButton
                showCreate={showCreate}
                onClick={() => setShowCreate((v) => !v)}
              />
            </div>

            {NAV_ITEMS.slice(2, 4).map((item) => (
              <NavPill
                key={item.path}
                icon={item.icon}
                label={item.label}
                isActive={isPathActive(item.path, location.pathname)}
                onClick={() => navigate(item.path)}
              />
            ))}
          </div>
        </div>
      )}

      <CreateOverlay show={showCreate} onClose={() => setShowCreate(false)} />
    </div>
  );
}

/* ── NavPill ──────────────────────────────────────────────────────── */
function NavPill({
  icon: Icon,
  label,
  isActive,
  onClick,
}: {
  icon: React.ElementType;
  label: string;
  isActive: boolean;
  onClick: () => void;
}) {
  // Read computed CSS variable values so Motion's layoutId engine gets
  // concrete hex strings — same fix as MainLayout's NavTab.
  const { theme, isDark } = useTheme();

  const accentColor = useMemo(
    () => readThemeCssVar("--neu-accent-primary", "#6366f1"),
    [theme, isDark],
  );

  const mutedColor = useMemo(
    () => readThemeCssVar("--neu-text-muted", "#8D94A0"),
    [theme, isDark],
  );

  const iconColor = isActive ? accentColor : mutedColor;

  const hover = useHoverConfig();

  return (
    <motion.button
      whileTap={hover.tap as any}
      whileHover={!isActive ? (hover.btn as any) : {}}
      onClick={onClick}
      className="relative flex flex-col items-center justify-center outline-none select-none neu-focus-ring"
      style={{ flex: 1, minWidth: 0, paddingTop: 9, paddingBottom: 9, gap: 5 }}
      aria-label={label}
      aria-current={isActive ? "page" : undefined}
    >
      {/* Sliding pill — concrete hex so Motion tracks colour changes */}
      {isActive && (
        <motion.div
          layoutId="tablet-nav-pill"
          style={{
            position: "absolute",
            inset: "4px 6px",
            borderRadius: 20,
            background: accentColor,
            opacity: 0.14,
          }}
          transition={{
            type: "spring",
            stiffness: 500,
            damping: 38,
            mass: 0.75,
          }}
        />
      )}
      <motion.div
        animate={{ y: isActive ? -1 : 0 }}
        transition={{ type: "spring", stiffness: 440, damping: 30 }}
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          position: "relative",
          zIndex: 1,
        }}
      >
        <Icon
          style={{
            width: 24,
            height: 24,
            color: iconColor,
            transition: "color 0.22s ease",
          }}
          strokeWidth={isActive ? 2.1 : 1.5}
        />
      </motion.div>
      <span
        style={{
          fontSize: 11,
          lineHeight: 1,
          color: iconColor,
          fontWeight: isActive ? 600 : 400,
          transition: "color 0.22s ease",
          letterSpacing: "0.02em",
          position: "relative",
          zIndex: 1,
        }}
      >
        {label}
      </span>
    </motion.button>
  );
}

/* ── FabButton ────────────────────────────────────────────────────── */
function FabButton({
  showCreate,
  onClick,
}: {
  showCreate: boolean;
  onClick: () => void;
}) {
  return (
    <motion.button
      whileTap={{ scale: 0.88, y: 1 }}
      onClick={onClick}
      className="flex items-center justify-center rounded-full outline-none select-none neu-focus-ring"
      style={{
        width: 48,
        height: 48,
        background: showCreate
          ? "var(--neu-text-muted)"
          : "var(--neu-accent-primary)",
        boxShadow: showCreate ? "none" : "var(--fab-shadow)",
        transition: "background 0.22s, box-shadow 0.22s",
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
            <X
              style={{ width: 19, height: 19, color: "white" }}
              strokeWidth={2.2}
            />
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
            <Plus
              style={{ width: 19, height: 19, color: "white" }}
              strokeWidth={2.2}
            />
          </motion.span>
        )}
      </AnimatePresence>
    </motion.button>
  );
}

/* ── CreateOverlay ────────────────────────────────────────────────── */
function CreateOverlay({
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
            key="backdrop"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-40"
            style={{
              background: "rgba(0,0,0,0.40)",
              backdropFilter: "blur(6px)",
            }}
            onClick={onClose}
          />
          <motion.div
            key="sheet"
            ref={trapRef}
            role="dialog"
            aria-modal="true"
            aria-label="发布新动态"
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{
              type: "spring",
              damping: 32,
              stiffness: 280,
              mass: 0.85,
            }}
            className="fixed bottom-0 left-0 right-0 z-50 overflow-hidden"
            style={{
              height: "90%",
              borderRadius: "28px 28px 0 0",
              backgroundColor: "var(--neu-bg)",
              boxShadow:
                "0 -8px 48px rgba(0,0,0,0.22), 0 -2px 0 var(--neu-border)",
            }}
          >
            <CreateSheet onClose={onClose} />
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}

/* ════════════════════════════════════════════════════════════════════
   Export — same mobile-like layout for both orientations
═══════════════════════════════════════════════════════════════════════ */
export function TabletLayout() {
  return <TabletMobileLayout />;
}
