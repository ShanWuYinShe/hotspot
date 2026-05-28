import { useState, useEffect, useCallback } from "react";
import { usersService } from "@/lib/api/services/users";
import { isSuccess, TOKEN_INVALID_CODE } from "@/lib/api/client";
import type { UserSettings } from "@/lib/types/api";

interface SettingsData {
  settings: UserSettings | null;
  loading: boolean;
  error: string | null;
  updateSetting: (partial: Partial<UserSettings>) => Promise<void>;
  refresh: () => void;
}

export function useSettingsData(): SettingsData {
  const [settings, setSettings] = useState<UserSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await usersService.getMySettings();
      if (isSuccess(res) && res.data) {
        setSettings(res.data);
      } else if (res.code !== TOKEN_INVALID_CODE) {
        setError("加载失败，请重试");
      }
    } catch {
      setError("网络异常");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const updateSetting = useCallback(
    async (partial: Partial<UserSettings>) => {
      const prev = settings;
      const updated = settings
        ? { ...settings, ...partial }
        : (partial as UserSettings);
      setSettings(updated);
      try {
        const res = await usersService.updateMySettings(updated);
        if (!isSuccess(res)) {
          setSettings(prev);
        }
      } catch {
        setSettings(prev);
      }
    },
    [settings],
  );

  return { settings, loading, error, updateSetting, refresh: load };
}
