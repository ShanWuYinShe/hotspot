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

		class Map {
			constructor(container: string | HTMLElement, options?: MapOptions);
			destroy(): void;
			getCenter(): LngLat;
			setCenter(center: [number, number]): void;
			setZoom(zoom: number): void;
			getZoom(): number;
			setZoomAndCenter(zoom: number, center: [number, number]): void;
		on(event: string, handler: (event: { lnglat?: LngLat }) => void): void;
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

		class Marker {
			constructor(options?: MarkerOptions);
			setPosition(position: [number, number]): void;
			getPosition(): LngLat;
			setMap(map: Map | null): void;
			remove(): void;
		}

		interface MarkerOptions {
			position?: [number, number];
			anchor?: 'top-left' | 'top-center' | 'top-right' | 'middle-left' | 'center' | 'middle-right' | 'bottom-left' | 'bottom-center' | 'bottom-right';
			offset?: [number, number];
			icon?: string;
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
			search(
				keyword: string,
				callback: (status: string, result: PlaceSearchResult) => void
			): void;
			searchNearBy(
				keyword: string,
				center: [number, number],
				radius: number,
				callback: (status: string, result: PlaceSearchResult) => void
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
