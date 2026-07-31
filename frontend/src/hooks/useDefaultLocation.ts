'use client'
import { useEffect, useState } from "react";
import { getClientConfig } from "@/api";

/**
 * GeoPoint — canonical geographic coordinate (spec: api_spec_types.md).
 * `reference_point` on feed/search/map endpoints and `Location.point` both use this shape.
 */
export interface GeoPoint {
  latitude: number;
  longitude: number;
}

/**
 * Module-level cache for the client-config default location.
 *
 * `reference_point` is REQUIRED on every feed/search endpoint
 * (feed/nearby, feed/following, feed/nearby_groups, feed/nearby_users, moments/search,
 * users/search, groups/search, search/suggest) but the frontend has no
 * browser-geolocation source yet. Per the cross-end boundary (root AGENTS.md:
 * distance/location computed by backend), we fall back to the server-provided
 * `ClientConfig.defaults.default_location.point` — the same value the spec
 * recommends ("use ClientConfig.defaults.default_location.point when location
 * denied"). Cached once for the whole app; ClientConfig is near-static.
 */
interface DefaultLocation {
  point: GeoPoint;
  location_name: string;
}

let cached: DefaultLocation | null = null;
let inflight: Promise<DefaultLocation | null> | null = null;

async function loadDefaultLocation(): Promise<DefaultLocation | null> {
  if (cached) return cached;
  if (inflight) return inflight;
  inflight = (async () => {
    try {
      const res = await getClientConfig();
      if (res.code === 0) {
        const loc = (res.data as any)?.defaults?.default_location;
        if (loc?.point?.latitude != null && loc?.point?.longitude != null) {
          cached = {
            point: { latitude: loc.point.latitude, longitude: loc.point.longitude },
            location_name: loc.location_name ?? "",
          };
          return cached;
        }
      }
    } catch {
      /* network/mock unavailable — caller handles null */
    } finally {
      inflight = null;
    }
    return null;
  })();
  return inflight;
}

export interface UseDefaultLocation {
  /** GeoPoint for `reference_point` on feed/search endpoints; null until loaded. */
  reference_point: GeoPoint | null;
  /** Full default location (point + name), useful for moment `location` field. */
  defaultLocation: DefaultLocation | null;
  loading: boolean;
}

/**
 * Provides the server-configured default location as the `reference_point`
 * source for nearby/search calls. Every consumer should guard on `reference_point`
 * being non-null before issuing a location-dependent request.
 */
export function useDefaultLocation(): UseDefaultLocation {
  const [defaultLocation, setDefaultLocation] = useState<DefaultLocation | null>(cached);
  const [loading, setLoading] = useState(!cached);

  useEffect(() => {
    if (cached) { setDefaultLocation(cached); setLoading(false); return; }
    let cancelled = false;
    loadDefaultLocation().then(loc => {
      if (cancelled) return;
      setDefaultLocation(loc);
      setLoading(false);
    });
    return () => { cancelled = true; };
  }, []);

  return {
    reference_point: defaultLocation?.point ?? null,
    defaultLocation,
    loading,
  };
}
