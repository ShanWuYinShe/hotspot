/**
 * useHoverConfig — theme-aware Motion hover & tap props
 *
 * NEU (default):  Spring physics, tactile convex-shadow lift
 * Vibrant:        Smooth ease-out, directional accent-glow lift
 * Forest (flat):  Scale only, no y-lift, no shadow (shows CSS ring instead)
 */
import { useTheme } from "../contexts/ThemeContext";

type MotionValues = Record<string, unknown>;

export interface HoverConfig {
  /** Large content cards (PostCard, activity cards) */
  card: MotionValues;
  /** List-row items (user rows, chat rows, notif rows) */
  row: MotionValues;
  /** Circular icon buttons (back, settings, share…) */
  btn: MotionValues;
  /** Inline action icons (♥ 🔖 💬 ⬆) */
  action: MotionValues;
  /** whileTap for ALL interactive elements */
  tap: MotionValues;
}

export function useHoverConfig(): HoverConfig {
  const { theme } = useTheme();

  /* ── Forest: flat design — no y-lift, no shadow, gentle scale ── */
  if (theme === "forest") {
    const ease = [0.22, 1, 0.36, 1] as const;
    return {
      card:   { scale: 1.016, transition: { type: "tween", duration: 0.20, ease } },
      row:    { scale: 1.010, transition: { type: "tween", duration: 0.16, ease } },
      btn:    { scale: 1.08,  transition: { type: "tween", duration: 0.16, ease } },
      action: { scale: 1.18,  transition: { type: "tween", duration: 0.14, ease } },
      tap:    { scale: 0.97,  transition: { type: "tween", duration: 0.10, ease } },
    };
  }

  /* ── Vibrant: clean modern — smooth ease-out, accent-glow lift ── */
  if (theme === "vibrant") {
    const ease = [0.22, 1, 0.36, 1] as const;
    return {
      card:   { y: -5,  scale: 1.014, transition: { type: "tween", duration: 0.22, ease } },
      row:    { y: -3,  scale: 1.009, transition: { type: "tween", duration: 0.18, ease } },
      btn:    { scale: 1.10,           transition: { type: "tween", duration: 0.17, ease } },
      action: { scale: 1.20,           transition: { type: "tween", duration: 0.14, ease } },
      tap:    { scale: 0.94,           transition: { type: "tween", duration: 0.10, ease } },
    };
  }

  /* ── NEU (default): spring physics, tactile shadow lift ───────── */
  return {
    card:   { y: -7,  scale: 1.018, transition: { type: "spring", stiffness: 280, damping: 18, mass: 0.75 } },
    row:    { y: -5,  scale: 1.010, transition: { type: "spring", stiffness: 320, damping: 20, mass: 0.75 } },
    btn:    { scale: 1.12,           transition: { type: "spring", stiffness: 380, damping: 18, mass: 0.65 } },
    action: { scale: 1.22,           transition: { type: "spring", stiffness: 400, damping: 16, mass: 0.60 } },
    tap:    { scale: 0.94,           transition: { type: "spring", stiffness: 600, damping: 28 } },
  };
}
