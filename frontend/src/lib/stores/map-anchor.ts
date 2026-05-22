/**
 * 地图锚点状态管理 (Zustand)
 * 用于存储用户选择的浏览位置锚点
 *
 * 默认值优先从 ClientConfig.defaults.default_location 取得，
 * 若配置未加载则回退到硬编码值（与后端默认值一致）。
 */

import { create } from 'zustand';
import { useConfigStore } from './config';

const STORAGE_KEY = 'hotspot_map_anchor';

export const DEFAULT_ANCHOR_NAME = '南天门';

const FALLBACK_DEFAULTS = {
  latitude: 35.154922,
  longitude: 112.682524,
  name: DEFAULT_ANCHOR_NAME,
  radius: 5,
};

function getDefaults() {
  const loc = useConfigStore.getState().defaultLocation;
  if (loc) {
    return {
      latitude: loc.point.latitude,
      longitude: loc.point.longitude,
      name: loc.location_name || DEFAULT_ANCHOR_NAME,
      radius: FALLBACK_DEFAULTS.radius,
    };
  }
  return FALLBACK_DEFAULTS;
}

export interface MapAnchor {
  latitude: number;
  longitude: number;
  name?: string;
  radius?: number;
  updatedAt: number;
}

function loadFromStorage(): MapAnchor {
  if (typeof window === 'undefined') {
    return { ...getDefaults(), updatedAt: Date.now() };
  }

  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return { ...getDefaults(), updatedAt: Date.now() };
    }
    const parsed = JSON.parse(stored) as Partial<MapAnchor>;
    if (!parsed || typeof parsed.latitude !== 'number' || typeof parsed.longitude !== 'number') {
      return { ...getDefaults(), updatedAt: Date.now() };
    }
    return {
      latitude: parsed.latitude,
      longitude: parsed.longitude,
      name: typeof parsed.name === 'string' ? parsed.name : undefined,
      radius: typeof parsed.radius === 'number' && parsed.radius > 0 ? parsed.radius : undefined,
      updatedAt: typeof parsed.updatedAt === 'number' ? parsed.updatedAt : Date.now(),
    };
  } catch {
    // 忽略解析错误
  }

  return { ...getDefaults(), updatedAt: Date.now() };
}

function saveToStorage(anchor: MapAnchor) {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(anchor));
  } catch {
    // 忽略存储错误
  }
}

interface MapAnchorState {
  anchor: MapAnchor;
  initialized: boolean;

  setAnchor: (latitude: number, longitude: number, name?: string, radius?: number) => void;
  setRadius: (radius: number) => void;
  setName: (name: string) => void;
  reset: () => void;
  useCurrentLocation: () => Promise<boolean>;
}

export const useMapAnchorStore = create<MapAnchorState>((set, get) => ({
  anchor: typeof window !== 'undefined' ? loadFromStorage() : { ...FALLBACK_DEFAULTS, updatedAt: Date.now() },
  initialized: typeof window !== 'undefined',

  setAnchor(latitude, longitude, name, radius) {
    const baseRadius = radius ?? get().anchor.radius ?? FALLBACK_DEFAULTS.radius;
    const maxRadiusKm = getMaxRadiusKm();
    const clampedRadius = Math.min(baseRadius, maxRadiusKm);
    const anchor: MapAnchor = {
      latitude,
      longitude,
      name,
      radius: clampedRadius,
      updatedAt: Date.now(),
    };
    saveToStorage(anchor);
    set({ anchor });
  },

  setRadius(radius) {
    const baseRadius = radius > 0 ? radius : (get().anchor.radius ?? FALLBACK_DEFAULTS.radius);
    const maxRadiusKm = getMaxRadiusKm();
    const clampedRadius = Math.min(baseRadius, maxRadiusKm);
    const anchor: MapAnchor = {
      ...get().anchor,
      radius: clampedRadius,
      updatedAt: Date.now(),
    };
    saveToStorage(anchor);
    set({ anchor });
  },

  setName(name) {
    const anchor: MapAnchor = {
      ...get().anchor,
      name,
      updatedAt: Date.now(),
    };
    saveToStorage(anchor);
    set({ anchor });
  },

  reset() {
    const anchor: MapAnchor = { ...getDefaults(), updatedAt: Date.now() };
    saveToStorage(anchor);
    set({ anchor });
  },

  async useCurrentLocation() {
    return new Promise((resolve) => {
      if (typeof window === 'undefined' || !navigator.geolocation) {
        resolve(false);
        return;
      }

      navigator.geolocation.getCurrentPosition(
        (position) => {
          get().setAnchor(position.coords.latitude, position.coords.longitude, '');
          resolve(true);
        },
        () => {
          resolve(false);
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 60000,
        },
      );
    });
  },
}));

function getMaxRadiusKm(): number {
  const maxMeters = useConfigStore.getState().searchRadiusMax;
  const maxKm = maxMeters / 1000;
  return maxKm > 0 ? maxKm : FALLBACK_DEFAULTS.radius;
}
