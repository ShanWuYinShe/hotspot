import type { Map, Marker, MarkerOptions } from 'leaflet';

// Default center: Xuzhou, Jiangsu
export const DEFAULT_CENTER: [number, number] = [34.2618, 117.1859];
export const DEFAULT_ZOOM = 14;

let L: typeof import('leaflet') | null = null;

async function getLeaflet() {
  if (L) return L;
  L = await import('leaflet');
  return L;
}

export async function createMap(
  container: HTMLElement,
  center?: [number, number],
  zoom?: number
): Promise<Map> {
  const leaflet = await getLeaflet();

  // Fix default icon path issue
  delete (leaflet.Icon.Default.prototype as any)._getIconUrl;
  leaflet.Icon.Default.mergeOptions({
    iconRetinaUrl: '',
    iconUrl: '',
    shadowUrl: ''
  });

  const map = leaflet.map(container, {
    center: center || DEFAULT_CENTER,
    zoom: zoom || DEFAULT_ZOOM,
    zoomControl: true,
    attributionControl: true
  });

  leaflet
    .tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>',
      maxZoom: 19
    })
    .addTo(map);

  return map;
}

export async function createCustomMarker(
  iconUrl: string,
  lat: number,
  lng: number,
  options?: MarkerOptions
): Promise<{ marker: Marker; leaflet: typeof import('leaflet') }> {
  const leaflet = await getLeaflet();
  const icon = leaflet.divIcon({
    html: `<img src="${iconUrl}" style="width:32px;height:32px;" />`,
    className: '',
    iconSize: [32, 32],
    iconAnchor: [16, 32],
    popupAnchor: [0, -32]
  });

  const marker = leaflet.marker([lat, lng], { icon, ...options });
  return { marker, leaflet };
}

export async function createClusterMarker(
  iconUrl: string,
  count: number,
  lat: number,
  lng: number
): Promise<{ marker: Marker; leaflet: typeof import('leaflet') }> {
  const leaflet = await getLeaflet();
  const icon = leaflet.divIcon({
    html: `<div style="position:relative;width:40px;height:40px;">
      <img src="${iconUrl}" style="width:40px;height:40px;" />
      <span style="position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);color:white;font-weight:700;font-size:12px;text-shadow:0 1px 2px rgba(0,0,0,0.5);">${count}</span>
    </div>`,
    className: '',
    iconSize: [40, 40],
    iconAnchor: [20, 40],
    popupAnchor: [0, -40]
  });

  const marker = leaflet.marker([lat, lng], { icon });
  return { marker, leaflet };
}

export async function destroyMap(map: Map) {
  map.remove();
}
