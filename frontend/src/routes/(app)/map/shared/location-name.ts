// 地图页「浏览位置」地名解析：纯函数、无框架依赖、无副作用，桌面/移动可共用。
//
// 背景：dev 环境逆地理不可用（VITE_AMAP_KEY 为空 + /osm-service/reverse 502）时，
// 旧兜底链会直接落到服务端默认位置名（如「济源市南天门」），与地图实际中心
// （GPS 郑州）不符。这里给「锚点名 / 服务端默认名」加距离门槛：只有中心确实落在
// 该点附近（半径内）才采用其名字，否则宁可不显示（「未命名区域」）。

export interface LatLng {
	lat: number;
	lng: number;
}

export interface NamedPoint extends LatLng {
	name?: string | null;
}

/** 中心与命名点（锚点 / 服务端默认位置）的距离在此半径内，才采用该点的名字。 */
export const NAME_MATCH_RADIUS_M = 500;

/** 中心与浏览器 GPS fix 的距离在此半径内，视为「我的位置」。 */
export const MY_LOCATION_RADIUS_M = 300;

/** 无法确定地名时的显示文案。 */
export const UNNAMED_LOCATION_NAME = '未命名区域';

/** 中心即用户当前 GPS 位置时的显示文案。 */
export const MY_LOCATION_NAME = '我的位置';

const EARTH_RADIUS_M = 6371000;
const DEG_TO_RAD = Math.PI / 180;

/** 球面距离近似（Haversine，米）：只用于数百米级的半径判定，不需要投影精度。 */
export function approxDistanceMeters(a: LatLng, b: LatLng): number {
	const dLat = (b.lat - a.lat) * DEG_TO_RAD;
	const dLng = (b.lng - a.lng) * DEG_TO_RAD;
	const h =
		Math.sin(dLat / 2) ** 2 +
		Math.cos(a.lat * DEG_TO_RAD) * Math.cos(b.lat * DEG_TO_RAD) * Math.sin(dLng / 2) ** 2;
	// clamp：浮点误差可能让 sqrt(h) 略大于 1，asin 会返回 NaN。
	return 2 * EARTH_RADIUS_M * Math.asin(Math.min(1, Math.sqrt(h)));
}

/** 四位小数网格 key（与地图页逆地理缓存的 key 同口径）。 */
export function locationGridKey(point: LatLng): string {
	return `${point.lat.toFixed(4)},${point.lng.toFixed(4)}`;
}

/** 归一化名字：null / undefined / 空白串一律视为「没有名字」。 */
function cleanName(name: string | null | undefined): string | undefined {
	const trimmed = name?.trim();
	return trimmed ? trimmed : undefined;
}

/**
 * 解析地图「浏览位置」显示名。优先级（严格）：
 * 选中地点 > 选中图钉 > 同格逆地理 > 附近锚点名 > 我的位置 > 附近服务端默认名 >
 * 「未命名区域」。
 *
 * 核心回归：中心远离默认位置点时，绝不显示默认名（逆地理不可用时尤其重要）。
 */
export function resolveLocationName(input: {
	center: LatLng | null;
	selectedName?: string | null;
	pinName?: string | null;
	reverseName?: string | null;
	anchor?: NamedPoint | null;
	myLocation?: LatLng | null;
	defaultLocation?: NamedPoint | null;
}): string {
	const { center, selectedName, pinName, reverseName, anchor, myLocation, defaultLocation } = input;
	if (!center) return UNNAMED_LOCATION_NAME;

	// 显式选择（地点/图钉/当前格逆地理结果）不受距离门槛限制。
	const explicitName = cleanName(selectedName) ?? cleanName(pinName) ?? cleanName(reverseName);
	if (explicitName) return explicitName;

	// 锚点名：仅当中心仍在该锚点附近（如上次确认位置未被拖远）才沿用。
	if (anchor) {
		const anchorName = cleanName(anchor.name);
		if (anchorName && approxDistanceMeters(center, anchor) <= NAME_MATCH_RADIUS_M) return anchorName;
	}

	if (myLocation && approxDistanceMeters(center, myLocation) <= MY_LOCATION_RADIUS_M) {
		return MY_LOCATION_NAME;
	}

	// 服务端默认名：仅当中心确实落在默认位置点附近才采用，否则「未命名区域」。
	if (defaultLocation) {
		const defaultName = cleanName(defaultLocation.name);
		if (defaultName && approxDistanceMeters(center, defaultLocation) <= NAME_MATCH_RADIUS_M) {
			return defaultName;
		}
	}

	return UNNAMED_LOCATION_NAME;
}
