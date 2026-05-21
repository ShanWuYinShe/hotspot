import { useState, useEffect, useCallback } from 'react';
import { usersService } from '@/lib/api/services/users';
import { isSuccess } from '@/lib/api/client';
import type { UserSettings } from '@/lib/types/api';

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
      } else {
        setError('加载失败，请重试');
      }
    } catch {
      setError('网络异常');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const updateSetting = useCallback(async (partial: Partial<UserSettings>) => {
    if (!settings) return;
    const prev = settings;
    const updated = { ...settings, ...partial };
    setSettings(updated);
    try {
      await usersService.updateMySettings(updated);
    } catch {
      setSettings(prev);
    }
  }, [settings]);

  return { settings, loading, error, updateSetting, refresh: load };
}
