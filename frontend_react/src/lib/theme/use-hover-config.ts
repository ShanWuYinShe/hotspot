import { useTheme } from "@/lib/theme/theme-context";

type MotionValues = Record<string, unknown>;

export interface HoverConfig {
  card: MotionValues;
  row: MotionValues;
  btn: MotionValues;
  action: MotionValues;
  tap: MotionValues;
}

export function useHoverConfig(): HoverConfig {
  const { theme } = useTheme();

  if (theme === "forest") {
    const ease = [0.22, 1, 0.36, 1] as const;
    return {
      card: {
        scale: 1.016,
        transition: { type: "tween", duration: 0.2, ease },
      },
      row: {
        scale: 1.01,
        transition: { type: "tween", duration: 0.16, ease },
      },
      btn: {
        scale: 1.08,
        transition: { type: "tween", duration: 0.16, ease },
      },
      action: {
        scale: 1.18,
        transition: { type: "tween", duration: 0.14, ease },
      },
      tap: {
        scale: 0.97,
        transition: { type: "tween", duration: 0.1, ease },
      },
    };
  }

  if (theme === "vibrant") {
    const ease = [0.22, 1, 0.36, 1] as const;
    return {
      card: {
        y: -5,
        scale: 1.014,
        transition: { type: "tween", duration: 0.22, ease },
      },
      row: {
        y: -3,
        scale: 1.009,
        transition: { type: "tween", duration: 0.18, ease },
      },
      btn: {
        scale: 1.1,
        transition: { type: "tween", duration: 0.17, ease },
      },
      action: {
        scale: 1.2,
        transition: { type: "tween", duration: 0.14, ease },
      },
      tap: {
        scale: 0.94,
        transition: { type: "tween", duration: 0.1, ease },
      },
    };
  }

  return {
    card: {
      y: -7,
      scale: 1.018,
      transition: { type: "spring", stiffness: 280, damping: 18, mass: 0.75 },
    },
    row: {
      y: -5,
      scale: 1.01,
      transition: { type: "spring", stiffness: 320, damping: 20, mass: 0.75 },
    },
    btn: {
      scale: 1.12,
      transition: { type: "spring", stiffness: 380, damping: 18, mass: 0.65 },
    },
    action: {
      scale: 1.22,
      transition: { type: "spring", stiffness: 400, damping: 16, mass: 0.6 },
    },
    tap: {
      scale: 0.94,
      transition: { type: "spring", stiffness: 600, damping: 28 },
    },
  };
}
