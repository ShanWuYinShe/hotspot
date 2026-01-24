/**
 * 位置状态管理
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { LocationState } from '@/types/client';
import type { GeoPoint, Location } from '@/types/api';

interface LocationStore extends LocationState {
  // Actions
  setLocation: (location: Location) => void;
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
  clearLocation: () => void;
  getReferencePoint: () => GeoPoint | null;
}

export const useLocationStore = create<LocationStore>()(
  persist(
    (set, get) => ({
      // State
      latitude: null,
      longitude: null,
      location_name: null,
      isLoading: false,
      error: null,
      lastUpdated: null,

      // Actions
      setLocation: (location) => {
        set({
          latitude: location.latitude,
          longitude: location.longitude,
          location_name: location.location_name,
          lastUpdated: new Date().toISOString(),
          error: null,
        });
      },

      setLoading: (isLoading) => set({ isLoading }),

      setError: (error) => set({ error, isLoading: false }),

      clearLocation: () => {
        set({
          latitude: null,
          longitude: null,
          location_name: null,
          lastUpdated: null,
          error: null,
        });
      },

      getReferencePoint: () => {
        const { latitude, longitude } = get();
        if (latitude !== null && longitude !== null) {
          return { latitude, longitude };
        }
        return null;
      },
    }),
    {
      name: 'hotspot-location',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        latitude: state.latitude,
        longitude: state.longitude,
        location_name: state.location_name,
        lastUpdated: state.lastUpdated,
      }),
    }
  )
);
