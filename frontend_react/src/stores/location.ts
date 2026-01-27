"use client";

import { create } from 'zustand';

import { GeoPoint, Location } from '@/types/api';

interface LocationState {
  currentLocation: Location | null;
  referencePoint: GeoPoint | null;
  isLoading: boolean;
  error: string | null;
  setLocation: (location: Location) => void;
  setReferencePoint: (point: GeoPoint) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useLocationStore = create<LocationState>((set) => ({
  currentLocation: null,
  referencePoint: null,
  isLoading: false,
  error: null,
  setLocation: (location) => set({ currentLocation: location }),
  setReferencePoint: (point) => set({ referencePoint: point }),
  setLoading: (isLoading) => set({ isLoading }),
  setError: (error) => set({ error }),
}));
