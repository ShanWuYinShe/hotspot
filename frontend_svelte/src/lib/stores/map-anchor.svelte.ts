/**
 * 地图锚点状态管理
 * 用于存储用户选择的浏览位置锚点
 *
 * 默认值优先从 ClientConfig.defaults.default_location 取得，
 * 若配置未加载则回退到硬编码值（与后端默认值一致）。
 */

import { browser } from '$app/environment';
import { configStore } from './config.svelte';

const STORAGE_KEY = 'hotspot_map_anchor';

/** 默认锚点名称（未自定义时的占位），与后端一致 */
export const DEFAULT_ANCHOR_NAME = '南天门';

// 硬编码回退值（与后端 system_config_service 中的默认值一致）
const FALLBACK_DEFAULTS = {
	latitude: 35.154922,
	longitude: 112.682524,
	name: DEFAULT_ANCHOR_NAME,
	radius: 5 // Default 5km
};

/** 运行时默认值：优先取 ClientConfig，回退到硬编码 */
function getDefaults() {
	const loc = configStore.defaultLocation;
	if (loc) {
		return {
			latitude: loc.point.latitude,
			longitude: loc.point.longitude,
			name: loc.location_name || DEFAULT_ANCHOR_NAME,
			radius: FALLBACK_DEFAULTS.radius
		};
	}
	return FALLBACK_DEFAULTS;
}

export interface MapAnchor {
	latitude: number;
	longitude: number;
	name?: string;
	radius?: number;
	updatedAt: number;
}

function loadFromStorage(): MapAnchor {
	if (!browser) {
		return { ...getDefaults(), updatedAt: Date.now() };
	}

	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored);
		}
	} catch {
		// 忽略解析错误
	}

	return { ...getDefaults(), updatedAt: Date.now() };
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
	#anchor = $state.raw<MapAnchor>({ ...FALLBACK_DEFAULTS, updatedAt: Date.now() });
	#initialized = $state(false);

	constructor() {
		if (browser) {
			this.#anchor = loadFromStorage();
			this.#initialized = true;
		}
	}

	// ==================== Derived State（所有公开属性）====================

	/** 地图锚点对象 */
	anchor = $derived(this.#anchor);

	/** 是否已初始化 */
	initialized = $derived(this.#initialized);

	/** 纬度 */
	latitude = $derived(this.#anchor.latitude);

	/** 经度 */
	longitude = $derived(this.#anchor.longitude);

	/** 锚点名称 */
	name = $derived(this.#anchor.name || '');

	/** 搜索半径（公里） */
	radius = $derived(this.#anchor.radius || 5);

	/** 显示名称（如果没有名称则显示坐标） */
	displayName = $derived(
		this.#anchor.name || 
		`${this.#anchor.latitude.toFixed(4)}, ${this.#anchor.longitude.toFixed(4)}`
	);

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

	/** 重置为默认位置（优先使用 ClientConfig） */
	reset() {
		this.#anchor = { ...getDefaults(), updatedAt: Date.now() };
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
