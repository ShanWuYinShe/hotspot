import { useMemo } from "react";
import { useTheme } from "../contexts/ThemeContext";

/**
 * Read a CSS custom property from <html> at runtime.
 * Returns the resolved value (trimmed) so Framer Motion and other
 * JS-driven animations can interpolate theme-aware colors.
 *
 * Falls back to `fallback` when the variable is not set or the
 * browser doesn't support getComputedStyle.
 */
export function useCssVar(varName: `--${string}`, fallback = ""): string {
  const { theme, isDark } = useTheme();

  return useMemo(() => {
    if (typeof window === "undefined") return fallback;
    const raw = getComputedStyle(document.documentElement)
      .getPropertyValue(varName)
      .trim();
    return raw || fallback;
    // Recompute whenever the theme or dark mode changes
  }, [varName, fallback, theme, isDark]);
}

/**
 * Convenience hook that returns the two most commonly read
 * variables: --neu-accent-primary and --neu-text-muted.
 */
export function useThemeColors() {
  const accent = useCssVar("--neu-accent-primary", "#6366f1");
  const muted = useCssVar("--neu-text-muted", "#8D94A0");
  return { accent, muted } as const;
}
