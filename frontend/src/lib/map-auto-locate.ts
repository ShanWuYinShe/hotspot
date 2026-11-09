// 地图锚点 helper（桌面/移动地图页共用）。
//
// 两件事：
//   1. 上次锚点同步初值 —— 确认页写入 localStorage 的浏览锚点（hotspot_last_location），
//      页面初始化即用它渲染底图，不再干等 client_config 回来；
//   2. 锚点读写 —— 存 { lat, lng, name }；兼容历史 [lat, lng] 数组格式。
//      名称随锚点一起持久化，附近页与侧栏才能显示与锚点一致的位置名
//      （只存坐标会让「XX · 5km 内」的语义与真实距离基准脱节）。

export interface MapPoint {
	lat: number;
	lng: number;
	/** 位置名（可选）：来源于选中地点的 displayName；纯拖拽设锚点时为 undefined。 */
	name?: string;
}

const LAST_KEY = 'hotspot_last_location';

/** 锚点写入后派发（同窗口）：侧栏位置卡等常驻组件据此重读锚点。 */
export const LOCATION_ANCHOR_UPDATED_EVENT = 'location:anchor-updated';
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

/**
 * 清除浏览锚点（附近页「改用默认位置」等场景）：随后派发锚点更新事件，
 * 侧栏位置卡等常驻组件据此回退默认位置。存储不可用时静默忽略。
 */
export function clearLastLocation(): void {
	const store = storage();
	if (!store) return;
	try {
		store.removeItem(LAST_KEY);
	} catch {
		/* 隐私模式等存储不可用场景：不打断交互 */
	}
	window.dispatchEvent(new CustomEvent(LOCATION_ANCHOR_UPDATED_EVENT));
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
		window.dispatchEvent(new CustomEvent(LOCATION_ANCHOR_UPDATED_EVENT));
	} catch {
		/* 隐私模式等存储不可用场景：不打断交互 */
	}
}
