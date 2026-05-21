import { create } from 'zustand';
import type { GeoPoint, Location } from '@/lib/types/api';
import { useConfigStore } from './config';

const STORAGE_KEY = 'hotspot_location';

interface LocationState {
  current: Location | null;
  lastUpdated: string | null;
  permissionDenied: boolean;
  isLoading: boolean;
  geoPoint: GeoPoint | null;
  setLocation: (location: Location) => void;
  setPermissionDenied: () => void;
  clearLocation: () => void;
  requestLocation: () => Promise<GeoPoint | null>;
}

function loadFromStorage(): { current: Location | null; lastUpdated: string | null; permissionDenied: boolean } {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) return { current: null, lastUpdated: null, permissionDenied: false };
    const parsed = JSON.parse(stored);
    return {
      current: parsed.current ?? null,
      lastUpdated: parsed.lastUpdated ?? null,
      permissionDenied: parsed.permissionDenied ?? false
    };
  } catch {
    return { current: null, lastUpdated: null, permissionDenied: false };
  }
}

function saveToStorage(state: { current: Location | null; lastUpdated: string | null; permissionDenied: boolean }) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // ignore
  }
}

const stored = loadFromStorage();

let pendingLocationRequest: Promise<GeoPoint | null> | null = null;

export const useLocationStore = create<LocationState>((set, get) => ({
  current: stored.current,
  lastUpdated: stored.lastUpdated,
  permissionDenied: stored.permissionDenied,
  geoPoint: stored.current
    ? { latitude: stored.current.point.latitude, longitude: stored.current.point.longitude }
    : null,
  isLoading: false,

  setLocation: (location: Location) => {
    const geoPoint = { latitude: location.point.latitude, longitude: location.point.longitude };
    const state = { current: location, lastUpdated: new Date().toISOString(), permissionDenied: false };
    saveToStorage(state);
    set({ ...state, geoPoint });
  },

  setPermissionDenied: () => {
    const { current, lastUpdated } = get();
    const state = { current, lastUpdated, permissionDenied: true };
    saveToStorage(state);
    set({ permissionDenied: true });
  },

  clearLocation: () => {
    const state = { current: null, lastUpdated: null, permissionDenied: false };
    saveToStorage(state);
    set({ current: null, lastUpdated: null });
  },

  requestLocation: async () => {
    if (typeof navigator === 'undefined' || !navigator.geolocation) return null;
    if (pendingLocationRequest) return pendingLocationRequest;

    set({ isLoading: true });
    const request = (async () => {
      try {
        const position = await new Promise<GeolocationPosition>((resolve, reject) => {
          navigator.geolocation.getCurrentPosition(resolve, reject, {
            enableHighAccuracy: true,
            timeout: 10000,
            maximumAge: 60000
          });
        });

        const geoPoint: GeoPoint = {
          latitude: position.coords.latitude,
          longitude: position.coords.longitude
        };
        const location: Location = { point: geoPoint, location_name: '' };
        const state = { current: location, lastUpdated: new Date().toISOString(), permissionDenied: false };
        saveToStorage(state);
        set({ ...state, geoPoint, isLoading: false });
        return geoPoint;
      } catch (error) {
        if (error instanceof GeolocationPositionError && error.code === error.PERMISSION_DENIED) {
          set({ permissionDenied: true });
        }
        // Fallback to default location from config
        const { current } = get();
        if (!current) {
          const defaultLoc = useConfigStore.getState().defaultLocation;
          if (defaultLoc) {
            const loc: Location = {
              point: { latitude: defaultLoc.point.latitude, longitude: defaultLoc.point.longitude },
              location_name: defaultLoc.location_name
            };
            const fallbackGeo = { latitude: defaultLoc.point.latitude, longitude: defaultLoc.point.longitude };
            const state = { current: loc, lastUpdated: new Date().toISOString(), permissionDenied: false };
            saveToStorage(state);
            set({ ...state, geoPoint: fallbackGeo, isLoading: false });
          }
        }
        set({ isLoading: false });
        return null;
      } finally {
        pendingLocationRequest = null;
      }
    })();

    pendingLocationRequest = request;
    return request;
  }
}));
