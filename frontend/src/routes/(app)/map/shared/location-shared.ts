// 地图共享逻辑。

import { getMapCanvas } from '@/api';
import { formatDistance } from '@/lib/utils';
import type { Pin, PinType } from './location-data';
import markerUserUrl from '@/assets/markers/marker_user.svg';
import markerGroupUrl from '@/assets/markers/marker_group.svg';
import markerMomentUrl from '@/assets/markers/marker_moment.svg';
import markerUserClusterUrl from '@/assets/markers/marker_user_cluster.svg';
import markerGroupClusterUrl from '@/assets/markers/marker_group_cluster.svg';
import markerMomentClusterUrl from '@/assets/markers/marker_moment_cluster.svg';

export {
	markerUserUrl,
	markerGroupUrl,
	markerMomentUrl,
	markerUserClusterUrl,
	markerGroupClusterUrl,
	markerMomentClusterUrl
};

export const SNAP_MIN = 148;
export const SNAP_MID = 318;
export const SNAP_MAX = 540;
export type SnapState = 'min' | 'mid' | 'max';
export const SNAP_HEIGHTS: Record<SnapState, number> = { min: SNAP_MIN, mid: SNAP_MID, max: SNAP_MAX };

export const PIN_COLORS: Record<PinType, string> = {
	user: 'var(--cst-marker-user)',
	group: 'var(--cst-marker-group)',
	moment: 'var(--cst-marker-content)'
};
// 图钉淡底/边框/光晕档位：直接引用主题预定义 token，不自行 color-mix 派生
export const PIN_TONES: Record<PinType, { bg: string; border: string; glow: string }> = {
	user: { bg: 'var(--cst-marker-user-bg)', border: 'var(--cst-marker-user-border)', glow: 'var(--cst-marker-user-glow)' },
	group: { bg: 'var(--cst-marker-group-bg)', border: 'var(--cst-marker-group-border)', glow: 'var(--cst-marker-group-glow)' },
	moment: { bg: 'var(--cst-marker-moment-bg)', border: 'var(--cst-marker-moment-border)', glow: 'var(--cst-marker-moment-glow)' }
};
export const PIN_LABELS: Record<PinType, string> = { user: '用户', group: '群组', moment: '瞬间' };
export const floatBtnStyle =
	'background:var(--cst-surface-primary);border:1px solid var(--cst-border-default);box-shadow:0 2px 12px var(--cst-elevation-shadow-dark), 0 1px 3px var(--cst-elevation-shadow-dark);';

export interface MapBounds {
	minLat: number;
	maxLat: number;
	minLng: number;
	maxLng: number;
}

export function getMarkerUrl(type: PinType, count: number) {
	if (type === 'user') return count >= 3 ? markerUserClusterUrl : markerUserUrl;
	if (type === 'group') return count >= 3 ? markerGroupClusterUrl : markerGroupUrl;
	return count >= 3 ? markerMomentClusterUrl : markerMomentUrl;
}

export function buildMapData(center: { lat: number; lng: number }, zoom: number) {
	const bounds: MapBounds = {
		minLat: center.lat - zoom,
		maxLat: center.lat + zoom,
		minLng: center.lng - zoom,
		maxLng: center.lng + zoom
	};
	return {
		url: `https://www.openstreetmap.org/export/embed.html?bbox=${bounds.minLng},${bounds.minLat},${bounds.maxLng},${bounds.maxLat}&layer=mapnik`,
		bounds
	};
}

/* ── map/canvas API 接入（spec: api_spec_map.md）────────────────────── */

export interface MapHeatBucket {
	cell_token: string;
	level: number;
	location: { latitude: number; longitude: number };
	total_count: number;
	breakdown?: { user?: number; group?: number; moment?: number };
}

export interface MapHotspotItem {
	object_type: 'user' | 'group' | 'moment';
	object_id: string;
	title: string;
	subtitle?: string;
	cover_url?: string;
	location: { latitude: number; longitude: number };
	distance_meters?: number;
	created_at?: string;
}

// 注意：spec（api_spec_map.md）为带 mode 字段的 tagged union，且 location 为
// RelativeLocation；当前后端实现（backend-go/internal/modules/map）返回
// `{ heatmap? | hotspots? }`（均缺省 = empty），location 为 GeoPoint +
// distance_meters。前端按实际实现解析，两处偏差待后端对齐 spec 后简化。
type MapCanvasResponse = {
	heatmap?: { items: MapHeatBucket[] };
	hotspots?: { items: MapHotspotItem[]; next_cursor?: string };
};

export interface MapCanvasResult {
	mode: 'heatmap' | 'hotspots' | 'empty';
	/** hotspots 模式：视口内实体图钉（已按 activeTypes 过滤） */
	pins: Pin[];
	/** heatmap 模式：S2 网格热力点 */
	heatBuckets: MapHeatBucket[];
	/** 视口内各类型数量（heatmap 按 breakdown 聚合，hotspots 按条目统计） */
	counts: Record<PinType, number>;
}

/** 视口 bounds（spec Bounds 结构：north/south/east/west） */
export function buildViewportBounds(center: { lat: number; lng: number }, zoom: number) {
	return {
		north: center.lat + zoom,
		south: center.lat - zoom,
		east: center.lng + zoom,
		west: center.lng - zoom
	};
}

function emptyResult(): MapCanvasResult {
	return { mode: 'empty', pins: [], heatBuckets: [], counts: { user: 0, group: 0, moment: 0 } };
}

function hotspotToPin(item: MapHotspotItem): Pin {
	const type: PinType = item.object_type as PinType;
	return {
		id: item.object_id,
		type,
		lat: item.location.latitude,
		lng: item.location.longitude,
		name: item.title,
		title: item.title,
		tag: item.subtitle,
		image: item.cover_url,
		distance: item.distance_meters != null ? formatDistance(item.distance_meters) : undefined
	};
}

function normalizeCanvas(res: { code: number; data?: unknown }): MapCanvasResult {
	if (res.code !== 0 || !res.data) return emptyResult();
	const data = res.data as MapCanvasResponse;
	if (data.heatmap?.items?.length) {
		const items = data.heatmap.items;
		const counts: Record<PinType, number> = { user: 0, group: 0, moment: 0 };
		for (const b of items) {
			counts.user += b.breakdown?.user ?? 0;
			counts.group += b.breakdown?.group ?? 0;
			counts.moment += b.breakdown?.moment ?? 0;
		}
		return { mode: 'heatmap', pins: [], heatBuckets: items, counts };
	}
	if (data.hotspots?.items?.length) {
		const items = data.hotspots.items;
		// next_cursor：密集视口时分页取下一页。当前固定 zoom 下单页足够，
		// 图钉与统计同页口径一致；后续接入视口交互时再做游标翻页。
		const counts: Record<PinType, number> = { user: 0, group: 0, moment: 0 };
		const pins = items.map(hotspotToPin);
		for (const p of pins) counts[p.type] += 1;
		return { mode: 'hotspots', pins, heatBuckets: [], counts };
	}
	return emptyResult();
}

/**
 * 拉取当前视口地图数据（spec: POST /api/v1/map/canvas）。
 * - 三种类型全选：data_type 省略（服务端按 all 处理），单次请求
 * - 部分选择：spec 的 data_type 为单值，按选中类型分别请求后合并
 */
export async function fetchMapCanvas(
	center: { lat: number; lng: number },
	zoom: number,
	activeTypes: PinType[]
): Promise<MapCanvasResult> {
	const viewport = { bounds: buildViewportBounds(center, zoom) };
	if (activeTypes.length === 0) return emptyResult();

	const buildParams = (t?: PinType) => ({
		viewport,
		pagination: { limit: 100 },
		...(t ? { filter: { data_type: t } } : {})
	});

	if (activeTypes.length === 3) {
		try {
			return normalizeCanvas(await getMapCanvas(buildParams()));
		} catch {
			return emptyResult();
		}
	}

	try {
		const results = await Promise.all(activeTypes.map((t) => getMapCanvas(buildParams(t))));
		const parts = results.map(normalizeCanvas);
		const merged: MapCanvasResult = {
			mode: 'empty',
			pins: [],
			heatBuckets: [],
			counts: { user: 0, group: 0, moment: 0 }
		};
		for (const part of parts) {
			merged.pins.push(...part.pins);
			merged.heatBuckets.push(...part.heatBuckets);
			merged.counts.user += part.counts.user;
			merged.counts.group += part.counts.group;
			merged.counts.moment += part.counts.moment;
			// 混合模式合并（理论场景：同视口下不同类型被服务端判为不同模式）：
			// 热力与图钉叠加渲染，mode 以热力优先
			if (part.mode === 'heatmap') merged.mode = 'heatmap';
			else if (part.mode === 'hotspots' && merged.mode === 'empty') merged.mode = 'hotspots';
		}
		return merged;
	} catch {
		return emptyResult();
	}
}
