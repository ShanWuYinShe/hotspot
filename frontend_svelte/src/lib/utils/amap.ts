const AMAP_KEY = import.meta.env['VITE_AMAP_KEY'];
const AMAP_SECURITY_CODE = import.meta.env['VITE_AMAP_SECURITY_CODE'];
const AMAP_SERVICE_HOST = import.meta.env['VITE_AMAP_SERVICE_HOST'] || '/_AMapService';
const SDK_SCRIPT_ID = 'amap-sdk-script';
const SDK_LOAD_TIMEOUT_MS = 15000;

let loadPromise: Promise<void> | null = null;
let securityConfigured = false;

export interface AMapCoordinate {
  latitude: number;
  longitude: number;
}

export interface AMapVisibleRegion {
  northeast: AMapCoordinate;
  northwest: AMapCoordinate;
  southeast: AMapCoordinate;
  southwest: AMapCoordinate;
  bounds: {
    north: number;
    south: number;
    east: number;
    west: number;
  };
  zoom: number;
}

function configureSecurity() {
  if (securityConfigured || typeof window === 'undefined') return;
  window._AMapSecurityConfig = {
    serviceHost: AMAP_SERVICE_HOST,
    ...(AMAP_SECURITY_CODE && { securityJsCode: AMAP_SECURITY_CODE })
  };
  securityConfigured = true;
}

export function loadAMap(): Promise<void> {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('AMap only available in browser'));
  }
  if (window.AMap) return Promise.resolve();
  if (loadPromise) return loadPromise;

  if (!AMAP_KEY) {
    return Promise.reject(new Error('AMap key is not configured'));
  }

  configureSecurity();

  loadPromise = new Promise((resolve, reject) => {
    let settled = false;
    let timeoutId: number | null = null;

    const clearTimer = () => {
      if (timeoutId !== null) {
        window.clearTimeout(timeoutId);
        timeoutId = null;
      }
    };

    const fail = (error: Error, removeScript: boolean) => {
      if (settled) return;
      settled = true;
      clearTimer();
      loadPromise = null;
      if (removeScript) {
        document.getElementById(SDK_SCRIPT_ID)?.remove();
      }
      reject(error);
    };

    const succeed = () => {
      if (settled) return;
      if (!window.AMap) {
        fail(new Error('AMap SDK loaded but AMap is unavailable'), true);
        return;
      }
      settled = true;
      clearTimer();
      resolve();
    };

    timeoutId = window.setTimeout(() => {
      fail(new Error('AMap SDK load timeout'), true);
    }, SDK_LOAD_TIMEOUT_MS);

    const existing = document.getElementById(SDK_SCRIPT_ID) as HTMLScriptElement | null;
    if (existing) {
      if (window.AMap) {
        succeed();
        return;
      }
      existing.addEventListener('load', succeed, { once: true });
      existing.addEventListener('error', () => fail(new Error('AMap SDK load failed'), true), {
        once: true
      });
      if (existing.dataset['hotspotLoaded'] === '1') {
        succeed();
      }
      return;
    }

    const script = document.createElement('script');
    script.id = SDK_SCRIPT_ID;
    script.src = `https://webapi.amap.com/maps?v=2.0&key=${AMAP_KEY}`;
    script.async = true;
    script.dataset['hotspotLoaded'] = '0';
    script.onload = () => {
      script.dataset['hotspotLoaded'] = '1';
      succeed();
    };
    script.onerror = () => fail(new Error('AMap SDK load failed'), true);
    document.head.appendChild(script);
  });

  return loadPromise;
}

export function loadAMapPlugin(plugins: string[]): Promise<void> {
  return new Promise((resolve, reject) => {
    loadAMap()
      .then(() => {
        if (!window.AMap) {
          reject(new Error('AMap SDK not available'));
          return;
        }
        window.AMap.plugin(plugins, () => resolve());
      })
      .catch(reject);
  });
}

export function getAMapVisibleRegion(map: AMap.Map): AMapVisibleRegion | null {
  const bounds = map.getBounds();
  const northEast = bounds?.getNorthEast();
  const southWest = bounds?.getSouthWest();
  if (!northEast || !southWest) {
    return null;
  }

  const north = northEast.getLat();
  const east = northEast.getLng();
  const south = southWest.getLat();
  const west = southWest.getLng();

  return {
    northeast: { latitude: north, longitude: east },
    northwest: { latitude: north, longitude: west },
    southeast: { latitude: south, longitude: east },
    southwest: { latitude: south, longitude: west },
    bounds: {
      north,
      south,
      east,
      west
    },
    zoom: map.getZoom()
  };
}
