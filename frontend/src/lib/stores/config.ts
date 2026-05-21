import { create } from 'zustand';
import type { ClientConfig } from '@/lib/types/api';
import { systemService } from '@/lib/api/services/system';

interface ConfigState {
  config: ClientConfig | null;
  loaded: boolean;
  error: string | null;
  searchRadiusMax: number;
  maxMomentImages: number;
  maxMomentTags: number;
  momentDescriptionMaxLength: number;
  commentMaxLength: number;
  features: ClientConfig['features'] | null;
  limits: ClientConfig['limits'] | null;
  contact: ClientConfig['contact'] | null;
  defaults: ClientConfig['defaults'] | null;
  defaultLocation: ClientConfig['defaults']['default_location'] | null;
  fetch: () => Promise<void>;
}

export const useConfigStore = create<ConfigState>((set, get) => ({
  config: null,
  loaded: false,
  error: null,
  get searchRadiusMax() {
    return get().config?.limits?.search_radius_max ?? 50000;
  },
  get maxMomentImages() {
    return get().config?.limits?.max_moment_images ?? 9;
  },
  get maxMomentTags() {
    return get().config?.limits?.max_moment_tags ?? 10;
  },
  get momentDescriptionMaxLength() {
    return get().config?.limits?.moment_description_max_length ?? 2000;
  },
  get commentMaxLength() {
    return get().config?.limits?.comment_max_length ?? 500;
  },
  get features() {
    return get().config?.features ?? null;
  },
  get limits() {
    return get().config?.limits ?? null;
  },
  get contact() {
    return get().config?.contact ?? null;
  },
  get defaults() {
    return get().config?.defaults ?? null;
  },
  get defaultLocation() {
    return get().config?.defaults?.default_location ?? null;
  },

  fetch: async () => {
    try {
      const res = await systemService.getClientConfig();
      if (res.code === 0 && res.data) {
        set({ config: res.data, error: null });
      } else {
        set({ error: res.message || '获取配置失败' });
      }
    } catch (e) {
      set({ error: e instanceof Error ? e.message : '获取配置失败' });
    } finally {
      set({ loaded: true });
    }
  }
}));
