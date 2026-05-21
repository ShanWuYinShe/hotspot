import React, {
  createContext,
  useContext,
  useLayoutEffect,
  useState,
} from "react";
import {
  applyThemeToDocument,
  DEFAULT_THEME,
  getStoredDarkMode,
  getStoredTheme,
  type AppTheme,
  THEME_STORAGE_KEYS,
} from "@/lib/theme";

interface ThemeContextValue {
  isDark: boolean;
  toggleDark: () => void;
  theme: AppTheme;
  setTheme: (t: AppTheme) => void;
}

const ThemeContext = createContext<ThemeContextValue>({
  isDark: false,
  toggleDark: () => {},
  theme: DEFAULT_THEME,
  setTheme: () => {},
});

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [isDark, setIsDark] = useState<boolean>(getStoredDarkMode);
  const [theme, setThemeState] = useState<AppTheme>(getStoredTheme);

  useLayoutEffect(() => {
    applyThemeToDocument(document.documentElement, isDark, theme);

    try {
      localStorage.setItem(THEME_STORAGE_KEYS.isDark, String(isDark));
      localStorage.setItem(THEME_STORAGE_KEYS.theme, theme);
    } catch {}
  }, [isDark, theme]);

  const toggleDark = () => setIsDark((value) => !value);
  const setTheme = (nextTheme: AppTheme) => setThemeState(nextTheme);

  return (
    <ThemeContext.Provider value={{ isDark, toggleDark, theme, setTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  return useContext(ThemeContext);
}
