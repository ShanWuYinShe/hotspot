import React, { createContext, useContext, useEffect, useState } from "react";

export type AppTheme = "neu" | "forest" | "vibrant";

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

  // Remove all theme classes first
  root.classList.remove("forest", "vibrant");

  // Apply current theme class ("neu" uses :root defaults — no class needed)
  if (theme === "forest") root.classList.add("forest");
  else if (theme === "vibrant") root.classList.add("vibrant");
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
      if (saved === "forest" || saved === "neu" || saved === "vibrant") return saved;
      return "vibrant"; // default: vibrant (energetic)
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