export type AppTheme = "neu" | "forest" | "vibrant";

export interface ThemePreview {
  bg: string;
  accent: string;
}

export interface ThemeOption {
  id: AppTheme;
  name: string;
  description: string;
  preview: {
    light: ThemePreview;
    dark: ThemePreview;
  };
}

export const DEFAULT_THEME: AppTheme = "vibrant";
export const DEFAULT_IS_DARK = false;

export const THEME_STORAGE_KEYS = {
  theme: "appTheme",
  isDark: "isDark",
} as const;

export const THEME_OPTIONS: readonly ThemeOption[] = [
  {
    id: "vibrant",
    name: "星际",
    description: "电光立体",
    preview: {
      light: { bg: "#FAFAFE", accent: "#6366F1" },
      dark: { bg: "#130F26", accent: "#818CF8" },
    },
  },
  {
    id: "neu",
    name: "暖砂",
    description: "质感立体",
    preview: {
      light: { bg: "#EAE5DE", accent: "#D4845A" },
      dark: { bg: "#1A1C23", accent: "#E8A87C" },
    },
  },
  {
    id: "forest",
    name: "清野",
    description: "自然扁平",
    preview: {
      light: { bg: "#F9F9F5", accent: "#3D7A55" },
      dark: { bg: "#161A16", accent: "#52B878" },
    },
  },
] as const;

export function isAppTheme(
  value: string | null | undefined,
): value is AppTheme {
  return value === "neu" || value === "forest" || value === "vibrant";
}

export function getStoredTheme(): AppTheme {
  try {
    const saved = localStorage.getItem(THEME_STORAGE_KEYS.theme);
    return isAppTheme(saved) ? saved : DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

export function getStoredDarkMode(): boolean {
  try {
    return localStorage.getItem(THEME_STORAGE_KEYS.isDark) === "true";
  } catch {
    return DEFAULT_IS_DARK;
  }
}

export function applyThemeToDocument(
  root: HTMLElement,
  isDark: boolean,
  theme: AppTheme,
) {
  root.classList.toggle("dark", isDark);
  root.classList.remove("forest", "vibrant");

  if (theme !== "neu") {
    root.classList.add(theme);
  }

  root.dataset.theme = theme;
  root.dataset.colorMode = isDark ? "dark" : "light";
  root.style.colorScheme = isDark ? "dark" : "light";
}

export function readThemeCssVar(name: string, fallback: string): string {
  if (typeof document === "undefined") {
    return fallback;
  }

  return (
    getComputedStyle(document.documentElement).getPropertyValue(name).trim() ||
    fallback
  );
}
