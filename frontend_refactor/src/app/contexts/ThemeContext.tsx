import React, { createContext, useContext, useEffect, useState } from "react";

export type AppTheme = "forest" | "vibrant";

interface ThemeContextValue {
  isDark: boolean;
  toggleDark: () => void;
  theme: AppTheme;
  setTheme: (t: AppTheme) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  isDark: false,
  toggleDark: () => {},
  theme: "vibrant",
  setTheme: () => {},
});

/* Apply / remove classes on <html> based on current state */
function applyClasses(isDark: boolean, theme: AppTheme) {
  const root = document.documentElement;

  // Dark
  isDark ? root.classList.add("dark") : root.classList.remove("dark");

  // Both surviving themes are explicit classes; this prevents a stale warm-sand class from lingering.
  root.classList.remove("forest", "vibrant", "neu");
  root.classList.add(theme);
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [isDark, setIsDark] = useState<boolean>(() => {
    try {
      return localStorage.getItem("isDark") === "true";
    } catch {
      return false;
    }
  });

  const [theme, setThemeState] = useState<AppTheme>(() => {
    try {
      const saved = localStorage.getItem("appTheme");
      if (saved === "forest" || saved === "vibrant") return saved;
      // Migrate retired 暖砂 preference to 清野 so old local storage cannot revive it.
      return saved === "neu" ? "forest" : "vibrant";
    } catch {
      return "vibrant";
    }
  });

  // Apply on every change
  useEffect(() => {
    applyClasses(isDark, theme);
    try {
      localStorage.setItem("isDark", String(isDark));
    } catch {}
  }, [isDark, theme]);

  const toggleDark = () => setIsDark((d) => !d);

  const setTheme = (t: AppTheme) => {
    setThemeState(t);
    try {
      localStorage.setItem("appTheme", t);
    } catch {}
    applyClasses(isDark, t);
  };

  return (
    <ThemeContext.Provider value={{ isDark, toggleDark, theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}