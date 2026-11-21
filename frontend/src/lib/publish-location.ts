// 发布器位置解析（Web CreateSheet 唯一入口）。
//
// 语义（与鸿蒙 CreateMomentPage 对齐）：发布瞬间的位置优先用「当前浏览锚点」
// （localStorage hotspot_last_location，用户在附近/地图页显式确认过的位置），
// 缺失时才回退 ClientConfig 默认位置，两者皆缺才退化为零点。
// 浏览锚点是用户正在浏览的区域中心，比服务端默认位置更符合「我在这里发」的意图；
// 只取默认位置会把在别处浏览时发布的瞬间错标到默认地点。

import type { MapPoint } from '@/lib/map-auto-locate';

/** 发布请求的 location 形状（结构对齐 api/types.md 的 Location，避免纯函数耦合 api 层）。 */
export interface PublishLocation {
	point: { latitude: number; longitude: number };
	location_name: string;
}

/** ClientConfig.defaults.default_location 的最小结构（避免把 api 类型耦合进纯函数）。 */
export interface PublishFallbackLocation {
	point: { latitude: number; longitude: number };
	location_name: string;
}

/** 锚点无名称时的展示/落库回退名（与地图页「自定义位置」口径一致）。 */
export const ANCHOR_FALLBACK_NAME = '自定义位置';

/**
 * 解析发布瞬间的 location：
 * 1. 浏览锚点存在 → 锚点坐标 + （锚点名 | 自定义位置）；
 * 2. 否则 → ClientConfig 默认位置；
 * 3. 两者皆缺 → 零点 + 空名称（仅在配置不可用时出现，保持可发布性）。
 */
export function resolvePublishLocation(
	anchor: MapPoint | null,
	fallback: PublishFallbackLocation | null
): PublishLocation {
	if (anchor) {
		return {
			point: { latitude: anchor.lat, longitude: anchor.lng },
			location_name: anchor.name?.trim() || ANCHOR_FALLBACK_NAME
		};
	}
	if (fallback) {
		return {
			point: { latitude: fallback.point.latitude, longitude: fallback.point.longitude },
			location_name: fallback.location_name
		};
	}
	return { point: { latitude: 0, longitude: 0 }, location_name: '' };
}
