/**
 * 地图锚点状态管理
 * 用于存储用户选择的浏览位置锚点
 */

import { browser } from '$app/environment';

const STORAGE_KEY = 'hotspot_map_anchor';

/** 默认锚点名称（未自定义时的占位），与后端一致 */
export const DEFAULT_ANCHOR_NAME = '南天门';

// 默认位置（与后端默认值一致）
const DEFAULTS = {
	latitude: 35.154921,
	longitude: 112.68253,
	name: DEFAULT_ANCHOR_NAME,
	radius: 5 // Default 5km
};

export interface MapAnchor {
	latitude: number;
	longitude: number;
	name?: string;
	radius?: number;
	updatedAt: number;
}

function loadFromStorage(): MapAnchor {
	if (!browser) {
		return { ...DEFAULTS, updatedAt: Date.now() };
	}

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored);
		}
	} catch {
		// 忽略解析错误
	}

	return { ...DEFAULTS, updatedAt: Date.now() };
}

function saveToStorage(anchor: MapAnchor) {
	if (!browser) return;

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(anchor));
	} catch {
		// 忽略存储错误
	}
}

class MapAnchorStore {
	#anchor = $state<MapAnchor>({ ...DEFAULTS, updatedAt: Date.now() });
	#initialized = $state(false);

	constructor() {
		if (browser) {
			this.#anchor = loadFromStorage();
			this.#initialized = true;
		}
	}

	get anchor() {
		return this.#anchor;
	}

	get initialized() {
		return this.#initialized;
	}

	get latitude() {
		return this.#anchor.latitude;
	}

	get longitude() {
		return this.#anchor.longitude;
	}

	get name() {
		return this.#anchor.name || '';
	}

	get radius() {
		return this.#anchor.radius || 5;
	}

	get displayName() {
		return this.#anchor.name || `${this.#anchor.latitude.toFixed(4)}, ${this.#anchor.longitude.toFixed(4)}`;
	}

	/** 设置锚点 */
	setAnchor(latitude: number, longitude: number, name?: string, radius?: number) {
		this.#anchor = {
			latitude,
			longitude,
			name,
			radius: radius || this.#anchor.radius || 5,
			updatedAt: Date.now()
		};
		saveToStorage(this.#anchor);
	}

	/** 设置搜索半径 */
	setRadius(radius: number) {
		this.#anchor = {
			...this.#anchor,
			radius,
			updatedAt: Date.now()
		};
		saveToStorage(this.#anchor);
	}

	/** 设置锚点名称 */
	setName(name: string) {
		this.#anchor = {
			...this.#anchor,
			name,
			updatedAt: Date.now()
		};
		saveToStorage(this.#anchor);
	}

	/** 重置为默认位置 */
	reset() {
		this.#anchor = { ...DEFAULTS, updatedAt: Date.now() };
		saveToStorage(this.#anchor);
	}

	/** 使用当前 GPS 位置 */
	useCurrentLocation(): Promise<boolean> {
		return new Promise((resolve) => {
			if (!browser || !navigator.geolocation) {
				resolve(false);
				return;
			}

			navigator.geolocation.getCurrentPosition(
				(position) => {
					this.setAnchor(
						position.coords.latitude,
						position.coords.longitude,
						'当前位置'
					);
					resolve(true);
				},
				() => {
					resolve(false);
				},
				{
					enableHighAccuracy: true,
					timeout: 10000,
					maximumAge: 60000
				}
			);
		});
	}
}

export const mapAnchorStore = new MapAnchorStore();
