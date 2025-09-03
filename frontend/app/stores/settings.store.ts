import { defineStore } from "pinia";
import {
  useSettingsService,
  type UserSettings,
} from "~/services/settings.service";

// 默认设置
const getDefaultSettings = (): UserSettings => ({
  notifications: {
    push_enabled: true,
    email_enabled: true,
    sms_enabled: false,
    group_notifications: true,
    message_notifications: true,
    friend_requests: true,
    direct_message_toast: true,
    group_message_toast: true,
    system_notification_toast: true,
    error_notifications: true,
    success_notifications: true,
  },
  privacy: {
    profile_visibility: "public",
    location_sharing: true,
    online_status: true,
    show_in_search: true,
  },
  preferences: {
    language: "zh-CN",
    theme: "auto",
    distance_unit: "km",
    time_format: "24h",
  },
  account: {
    nickname: "用户",
    bio: "",
    email: "",
    phone: "",
    avatar_url: "",
  },
});

/**
 * 设置 Store - 简化版
 * 主要负责主题管理和基础设置缓存
 */
export const useSettingsStore = defineStore(
  "settings",
  () => {
    // 核心状态
    const settings = ref<UserSettings | null>(null);
    const theme = ref<"light" | "dark" | "auto">("auto");
    const preferredLocale = ref<string>("zh-CN");
    const isLoading = ref(false);

    // 计算属性
    const effectiveTheme = computed(() => {
      if (theme.value === "auto") {
        return "light"; // 服务端渲染时默认为light主题
      }
      return theme.value;
    });

    // 设置相关方法
    async function fetchSettings() {
      if (isLoading.value) return settings.value;

      try {
        isLoading.value = true;
        const settingsService = useSettingsService();
        const userSettings = await settingsService.getUserSettings();
        settings.value = userSettings;

        if (userSettings.preferences?.theme) {
          theme.value = userSettings.preferences.theme;
        }
        if (userSettings.preferences?.language) {
          preferredLocale.value = userSettings.preferences.language;
        }
        return userSettings;
      } catch (error) {
        console.error("获取设置失败:", error);
        return null;
      } finally {
        isLoading.value = false;
      }
    }

    async function updateSettings(newSettings: Partial<UserSettings>) {
      try {
        isLoading.value = true;
        const settingsService = useSettingsService();
        await settingsService.updateUserSettings(newSettings);

        // 重新获取设置
        await fetchSettings();
        return settings.value;
      } catch (error) {
        console.error("更新设置失败:", error);
        throw error;
      } finally {
        isLoading.value = false;
      }
    }

    // 主题管理
    function setTheme(newTheme: "light" | "dark" | "auto") {
      theme.value = newTheme;
      applyTheme();

      // 如果有设置，更新到服务器
      if (settings.value) {
        updateSettings({
          preferences: {
            ...settings.value.preferences,
            theme: newTheme,
          },
        });
      }
    }

    function applyTheme() {
      if (typeof document === "undefined") return;

      const root = document.documentElement;
      const prefersDark =
        window.matchMedia &&
        window.matchMedia("(prefers-color-scheme: dark)").matches;
      const wantDark =
        theme.value === "dark" || (theme.value === "auto" && prefersDark);
      root.classList.toggle("dark", wantDark);
    }

    function initThemeWatcher() {
      // 监听 store 中 theme 的变化
      const stopTheme = watch(theme, () => applyTheme());

      // 监听系统主题变化（当 theme=auto 时生效）
      let media: MediaQueryList | null = null;
      const onMediaChange = () => {
        if (theme.value === "auto") applyTheme();
      };

      if (typeof window !== "undefined" && window.matchMedia) {
        media = window.matchMedia("(prefers-color-scheme: dark)");
        media.addEventListener?.("change", onMediaChange);
      }

      // 初次应用一次
      applyTheme();

      // 返回清理函数
      return () => {
        stopTheme();
        media?.removeEventListener?.("change", onMediaChange);
      };
    }

    // 语言偏好管理
    function setPreferredLocale(locale: string) {
      preferredLocale.value = locale;
    }

    function clearSettings() {
      settings.value = null;
      theme.value = "auto";
      preferredLocale.value = "zh-CN";
      applyTheme();
    }

    return {
      // 状态
      settings: computed(() => settings.value || getDefaultSettings()),
      theme: computed(() => theme.value),
      preferredLocale: computed(() => preferredLocale.value),
      isLoading: computed(() => isLoading.value),
      effectiveTheme,

      // 方法
      fetchSettings,
      updateSettings,
      setTheme,
      setPreferredLocale,
      applyTheme,
      initThemeWatcher,
      clearSettings,
    };
  },
  {
    persist: {
      pick: ["theme", "preferredLocale"],
    },
  },
);
