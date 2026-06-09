export {};

declare global {
  interface Window {
    AMap?: typeof AMap;
    _AMapSecurityConfig?: {
      serviceHost?: string;
    };
  }

  namespace AMap {
    function plugin(plugins: string[], callback: () => void): void;

    type MarkerAnchor =
      | 'top-left'
      | 'top-center'
      | 'top-right'
      | 'middle-left'
      | 'center'
      | 'middle-right'
      | 'bottom-left'
      | 'bottom-center'
      | 'bottom-right';

    interface MapClickEvent {
      lnglat?: LngLat;
      stopPropagation?(): void;
    }

    class Map {
      constructor(container: string | HTMLElement, options?: MapOptions);
      destroy(): void;
      getCenter(): LngLat;
      getBounds(): Bounds;
      setCenter(center: [number, number]): void;
      setZoom(zoom: number): void;
      getZoom(): number;
      setZoomAndCenter(zoom: number, center: [number, number]): void;
      setMapStyle?(style: string): void;
      containerToLngLat(pixel: PixelLike): LngLat;
      on(event: string, handler: (event: MapClickEvent) => void): void;
    }

    interface MapOptions {
      center?: [number, number];
      zoom?: number;
      viewMode?: '2D' | '3D';
      zooms?: [number, number];
      mapStyle?: string;
      doubleClickZoom?: boolean;
      resizeEnable?: boolean;
      touchZoom?: boolean;
      dragEnable?: boolean;
    }

    class LngLat {
      getLng(): number;
      getLat(): number;
    }

    class Bounds {
      getNorthEast(): LngLat;
      getSouthWest(): LngLat;
    }

    interface PixelLike {
      x: number;
      y: number;
    }

    class Pixel {
      constructor(x: number, y: number);
    }

    class Size {
      constructor(width: number, height: number);
    }

    class Icon {
      constructor(options?: IconOptions);
    }

    interface IconOptions {
      size?: Size;
      image?: string;
      imageSize?: Size;
      imageOffset?: Pixel;
    }

    class Marker {
      constructor(options?: MarkerOptions);
      setPosition(position: [number, number]): void;
      getPosition(): LngLat;
      setMap(map: Map | null): void;
      remove(): void;
      setLabel?(opts: { content: string; direction?: string }): void;
      setExtData?(data: unknown): void;
      getExtData?(): unknown;
      setContent?(content: string | HTMLElement): void;
      on(event: string, handler: (event: MapClickEvent) => void): void;
    }

    interface MarkerOptions {
      position?: [number, number];
      anchor?: MarkerAnchor;
      offset?: Pixel | [number, number];
      icon?: string | Icon;
      content?: string | HTMLElement;
      title?: string;
      clickable?: boolean;
      map?: Map;
    }

    class Geocoder {
      constructor(options?: { city?: string });
      getAddress(
        location: [number, number],
        callback: (status: string, result: GeocodeResult) => void
      ): void;
      getLocation(
        address: string,
        callback: (status: string, result: ForwardGeocodeResult) => void
      ): void;
    }

    class PlaceSearch {
      constructor(options?: PlaceSearchOptions);
      search(keyword: string, callback: (status: string, result: PlaceSearchResult) => void): void;
      searchNearBy(
        keyword: string,
        center: [number, number],
        radius: number,
        callback: (status: string, result: PlaceSearchResult) => void
      ): void;
    }

    class DistrictSearch {
      constructor(options?: DistrictSearchOptions);
      search(
        keyword: string,
        callback: (status: string, result: DistrictSearchResult) => void
      ): void;
    }

    interface PlaceSearchOptions {
      city?: string;
      citylimit?: boolean;
      type?: string;
      pageSize?: number;
      pageIndex?: number;
      extensions?: 'base' | 'all';
    }

    interface PlaceSearchResult {
      info: string;
      poiList?: {
        pois: Array<PlaceSearchPoi>;
        count: number;
        pageIndex: number;
        pageSize: number;
      };
    }

    interface DistrictSearchOptions {
      subdistrict?: number;
      extensions?: 'base' | 'all';
    }

    interface DistrictSearchDistrict {
      center?: LngLat;
    }

    interface DistrictSearchResult {
      districts?: DistrictSearchDistrict[];
    }

    interface PlaceSearchPoi {
      id: string;
      name: string;
      type: string;
      address: string;
      location: LngLat;
      tel?: string;
      distance?: number;
      pname?: string;
      cityname?: string;
      adname?: string;
    }

    class Geolocation {
      constructor(options?: { enableHighAccuracy?: boolean; timeout?: number });
      getCurrentPosition(callback: (status: string, result: GeolocationResult) => void): void;
    }

    interface GeocodeResult {
      info: string;
      regeocode?: {
        formattedAddress: string;
        addressComponent?: {
          province?: string;
          city?: string | string[];
          citycode?: string;
          adcode?: string;
          district?: string;
          street?: string;
          township?: string;
          streetNumber?: string;
          neighborhood?: string;
          building?: string;
        };
        pois?: Array<{
          name: string;
          distance: number;
        }>;
        aois?: Array<{
          name: string;
          area: string;
        }>;
      };
    }

    interface ForwardGeocodeResult {
      info: string;
      geocodes: Array<{
        location: LngLat;
        formattedAddress: string;
        level: string;
      }>;
    }

    interface GeolocationResult {
      info?: string;
      position?: LngLat;
      formattedAddress?: string;
      addressComponent?: {
        province?: string;
        city?: string | string[];
        district?: string;
      };
    }
  }
}
