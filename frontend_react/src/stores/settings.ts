/**
 * 本地设置状态管理
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { SettingsState, ThemeMode, Locale } from '@/types/client';

interface SettingsStore extends SettingsState {
  // Actions
  setTheme: (theme: ThemeMode) => void;
  setLocale: (locale: Locale) => void;
}

export const useSettingsStore = create<SettingsStore>()(
  persist(
    (set) => ({
      // State
      theme: 'system',
      locale: 'zh',

      // Actions
      setTheme: (theme) => set({ theme }),
      setLocale: (locale) => {
        set({ locale });
        // 设置 cookie 以便服务端读取
        if (typeof document !== 'undefined') {
          document.cookie = `locale=${locale};path=/;max-age=31536000`;
        }
      },
    }),
    {
      name: 'hotspot-settings',
      storage: createJSONStorage(() => localStorage),
    }
  )
);
