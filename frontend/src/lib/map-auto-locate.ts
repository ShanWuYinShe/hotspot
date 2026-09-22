// 地图首屏定位 helper（桌面/移动地图页共用）。
//
// 三件事：
//   1. 上次锚点同步初值 —— 确认页写入 localStorage 的浏览锚点（hotspot_last_location），
//      页面初始化即用它渲染底图，不再干等 client_config 回来；
//   2. 进入自动定位一次 —— Geolocation 单次快照（getCurrentPosition 语义：调一次
//      给一次，不会自动更新；权限按域名持久记忆，允许过之后调用不再弹窗）。
//      成功由调用方决定是否采纳；拒绝/不支持则记住不再自动打扰（手动定位按钮不受影响）；
//      超时等临时失败不记，下次进入再试；
//   3. 锚点读写 —— 存 { lat, lng, name }；兼容历史 [lat, lng] 数组格式。
//      名称随锚点一起持久化，附近页与侧栏才能显示与锚点一致的位置名
//      （只存坐标会让「XX · 5km 内」的语义与真实距离基准脱节）。

import { GeoError, requestGeoPosition, type GeoFix } from './geolocation.svelte';

export interface MapPoint {
	lat: number;
	lng: number;
	/** 位置名（可选）：来源于选中地点的 displayName；纯拖拽设锚点时为 undefined。 */
	name?: string;
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

/** 坐标合法性与范围校验（越界坐标会被后端 S2 索引拒绝，不入库）。 */
function validPoint(lat: unknown, lng: unknown): boolean {
	return (
		typeof lat === 'number' &&
		typeof lng === 'number' &&
		Number.isFinite(lat) &&
		Number.isFinite(lng) &&
		lat >= -90 &&
		lat <= 90 &&
		lng >= -180 &&
		lng <= 180
	);
}

/**
 * 上次确认的浏览锚点；缺失/非法/越界一律返回 null（调用方回退默认位置）。
 * 兼容历史数组格式 [lat, lng]（无名称）。
 */
export function readLastLocation(): MapPoint | null {
	const store = storage();
	if (!store) return null;
	try {
		const parsed: unknown = JSON.parse(store.getItem(LAST_KEY) ?? '');
		if (Array.isArray(parsed)) {
			const [lat, lng] = parsed;
			return validPoint(lat, lng) ? { lat: lat as number, lng: lng as number } : null;
		}
		if (parsed && typeof parsed === 'object') {
			const { lat, lng, name } = parsed as { lat?: unknown; lng?: unknown; name?: unknown };
			if (!validPoint(lat, lng)) return null;
			const point: MapPoint = { lat: lat as number, lng: lng as number };
			if (typeof name === 'string' && name.trim()) point.name = name.trim();
			return point;
		}
		return null;
	} catch {
		return null;
	}
}

/** 写入浏览锚点（坐标必填；名称可选）。存储不可用时静默忽略，不打断交互。 */
export function writeLastLocation(point: MapPoint): void {
	const store = storage();
	if (!store || !validPoint(point.lat, point.lng)) return;
	try {
		const payload: MapPoint = { lat: point.lat, lng: point.lng };
		const name = point.name?.trim();
		if (name) payload.name = name;
		store.setItem(LAST_KEY, JSON.stringify(payload));
	} catch {
		/* 隐私模式等存储不可用场景：不打断交互 */
	}
}

/**
 * 清除自动定位拒绝标记。用户手动点「定位到我」并成功时调用：权限既已授予，
 * 旧的拒绝标记已过期，下次进地图恢复自动定位，免得回退到默认位置。
 */
export function clearLocateDeclined(): void {
	try {
		storage()?.removeItem(DENIED_KEY);
	} catch {
		/* 存储不可用则忽略，下次进入再按拒绝处理 */
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
