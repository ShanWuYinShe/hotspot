/* Location mock data — extracted from the legacy page file to keep the page lean. */

export type PinType = "user" | "group" | "post";

export interface Pin {
  id: number; type: PinType; lat: number; lng: number;
  count?: number;
  name?: string; avatar?: string; distance?: string; online?: boolean;
  members?: number; tag?: string; title?: string; likes?: number; image?: string;
}

export interface LocationResult {
  displayName: string; lat: number; lng: number;
  emoji?: string; desc?: string; users?: number;
}

export const XUZHOU_CENTER = { lat: 34.2604, lng: 117.1855 };

// All pin/place data is now fetched from POST /api/v1/map/canvas at runtime.
// These are zero-state defaults shown before the API returns.
export const MOCK_PINS: Pin[] = [];
export const RECENT_LOCATIONS: LocationResult[] = [];
export const HOT_LOCATIONS: LocationResult[] = [];
export const SEARCH_DB: LocationResult[] = [];
