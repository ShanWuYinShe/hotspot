/**
 * 位置状态管理 (Svelte 5 Runes)
 */

import { browser } from '$app/environment';
import type { GeoPoint, Location } from '$lib/types/api';

const STORAGE_KEY = 'hotspot_location';

interface LocationState {
	current: Location | null;
	lastUpdated: string | null;
	permissionDenied: boolean;
}

function loadFromStorage(): LocationState {
	if (!browser) {
		return { current: null, lastUpdated: null, permissionDenied: false };
	}

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored);
		}
	} catch {
		// 忽略解析错误
	}

	return { current: null, lastUpdated: null, permissionDenied: false };
}

function saveToStorage(state: LocationState) {
	if (!browser) return;

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
	} catch {
		// 忽略存储错误
	}
}

class LocationStore {
	#current = $state<Location | null>(null);
	#lastUpdated = $state<string | null>(null);
	#permissionDenied = $state(false);
	#isLoading = $state(false);

	constructor() {
		if (browser) {
			const stored = loadFromStorage();
			this.#current = stored.current;
			this.#lastUpdated = stored.lastUpdated;
			this.#permissionDenied = stored.permissionDenied;
		}
	}

	// ==================== Derived State（所有公开属性）====================

	/** 当前位置 */
	current = $derived(this.#current);

	/** 最后更新时间 */
	lastUpdated = $derived(this.#lastUpdated);

	/** 权限被拒绝 */
	permissionDenied = $derived(this.#permissionDenied);

	/** 正在加载中 */
	isLoading = $derived(this.#isLoading);

	/** 提取经纬度坐标 */
	geoPoint = $derived(
		this.#current ? {
			latitude: this.#current.latitude,
			longitude: this.#current.longitude
		} : null
	);

	#persist() {
		saveToStorage({
			current: this.#current,
			lastUpdated: this.#lastUpdated,
			permissionDenied: this.#permissionDenied
		});
	}

	setLocation(location: Location) {
		this.#current = location;
		this.#lastUpdated = new Date().toISOString();
		this.#permissionDenied = false;
		this.#persist();
	}

	setPermissionDenied() {
		this.#permissionDenied = true;
		this.#persist();
	}

	clearLocation() {
		this.#current = null;
		this.#lastUpdated = null;
		this.#persist();
	}

	/**
	 * 请求当前位置
	 */
	async requestLocation(): Promise<GeoPoint | null> {
		if (!browser || !navigator.geolocation) {
			return null;
		}

		this.#isLoading = true;

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

			// 注意：这里只返回坐标，location_name 使用默认占位
			this.#current = {
				...geoPoint,
				location_name: '当前位置'
			};
			this.#lastUpdated = new Date().toISOString();
			this.#permissionDenied = false;
			this.#persist();

			return geoPoint;
		} catch (error) {
			if (error instanceof GeolocationPositionError && error.code === error.PERMISSION_DENIED) {
				this.#permissionDenied = true;
				this.#persist();
			}
			return null;
		} finally {
			this.#isLoading = false;
		}
	}
}

export const locationStore = new LocationStore();
