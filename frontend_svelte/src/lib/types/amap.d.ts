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

		class Geocoder {
			constructor(options?: { city?: string });
			getAddress(
				location: [number, number],
				callback: (status: string, result: GeocodeResult) => void
			): void;
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
