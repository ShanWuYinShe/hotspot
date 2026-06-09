import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export {
  applyThemeToDocument,
  DEFAULT_IS_DARK,
  DEFAULT_THEME,
  getStoredDarkMode,
  getStoredTheme,
  isAppTheme,
  THEME_OPTIONS,
  THEME_STORAGE_KEYS,
  type AppTheme,
  type ThemeOption,
  type ThemePreview,
} from "./theme";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function normalizeAvatar(url?: string | null): string {
  if (!url) {
    return (
      "data:image/svg+xml," +
      encodeURIComponent(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect fill="%23d1ccc4" width="100" height="100"/><text x="50" y="56" text-anchor="middle" fill="%23999" font-size="36">?</text></svg>',
      )
    );
  }
  if (url.startsWith("http://") || url.startsWith("https://")) return url;
  if (url.startsWith("/s3/")) return `${window.__API_ORIGIN__ || ""}${url}`;
  if (url.startsWith("/")) return url;
  return url;
}

export function formatDistance(meters?: number | null): string {
  if (meters == null) return "";
  if (meters < 1000) return `${Math.round(meters)}m`;
  return `${(meters / 1000).toFixed(1)}km`;
}

export function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = now - then;

  if (diff < 60000) return "刚刚";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`;
  return new Date(dateStr).toLocaleDateString("zh-CN", {
    month: "short",
    day: "numeric",
  });
}
