import { create } from 'zustand';

import { GlobalSettings, UserSettings } from '@/types/api';

interface SettingsState {
  userSettings: UserSettings | null;
  globalSettings: GlobalSettings | null;
  locale: 'zh' | 'en';
  setUserSettings: (settings: UserSettings) => void;
  setGlobalSettings: (settings: GlobalSettings) => void;
  setLocale: (locale: 'zh' | 'en') => void;
}

export const useSettingsStore = create<SettingsState>((set) => ({
  userSettings: null,
  globalSettings: null,
  locale: 'zh',
  setUserSettings: (userSettings) => set({ userSettings }),
  setGlobalSettings: (globalSettings) => set({ globalSettings }),
  setLocale: (locale) => set({ locale }),
}));
