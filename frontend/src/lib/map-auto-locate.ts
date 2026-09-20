// 地图首屏定位 helper（桌面/移动地图页共用）。
//
// 两件事：
//   1. 上次锚点同步初值 —— 确认页写入 localStorage 的浏览锚点（hotspot_last_location
//      [lat, lng]），页面初始化即用它渲染底图，不再干等 client_config 回来；
//   2. 进入自动定位一次 —— Geolocation 单次快照（getCurrentPosition 语义：调一次
//      给一次，不会自动更新；权限按域名持久记忆，允许过之后调用不再弹窗）。
//      成功由调用方决定是否采纳；拒绝/不支持则记住不再自动打扰（手动定位按钮不受影响）；
//      超时等临时失败不记，下次进入再试。

import { GeoError, requestGeoPosition, type GeoFix } from './geolocation.svelte';

export interface MapPoint {
	lat: number;
	lng: number;
}

const LAST_KEY = 'hotspot_last_location';
const DENIED_KEY = 'hotspot:locate-denied';

function storage(): Storage | null {
	try {
		return typeof localStorage === 'undefined' ? null : localStorage;
	} catch {
		return null;
	}
}

/** 上次确认的浏览锚点；缺失/非法/越界一律返回 null（调用方回退默认位置）。 */
export function readLastLocation(): MapPoint | null {
	const store = storage();
	if (!store) return null;
	try {
		const parsed: unknown = JSON.parse(store.getItem(LAST_KEY) ?? '');
		if (!Array.isArray(parsed)) return null;
		const [lat, lng] = parsed;
		if (typeof lat !== 'number' || typeof lng !== 'number') return null;
		if (!Number.isFinite(lat) || !Number.isFinite(lng)) return null;
		if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return null;
		return { lat, lng };
	} catch {
		return null;
	}
}

/** 用户是否拒绝过自动定位（拒绝后不再自动打扰）。 */
export function autoLocateDeclined(): boolean {
	const store = storage();
	if (!store) return true;
	try {
		return store.getItem(DENIED_KEY) === '1';
	} catch {
		return true;
	}
}

/**
 * 进入地图时自动定位一次。成功返回 fix（调用方决定是否采纳）；
 * 拿不到时返回 null（拒绝/不支持会顺手记住，下次不再自动问）。
 */
export async function autoLocateOnce(): Promise<GeoFix | null> {
	if (autoLocateDeclined()) return null;
	try {
		return await requestGeoPosition();
	} catch (err) {
		if (err instanceof GeoError && (err.code === 'denied' || err.code === 'unsupported')) {
		try {
		storage()?.setItem(DENIED_KEY, '1');
		} catch {
		/* 存储不可用则下次再问，不抛错 */
		}
	}
	return null;
	}
}
