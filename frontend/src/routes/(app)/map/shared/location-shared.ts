// 地图共享逻辑。

import { getMapCanvas } from '@/api';
import { formatDistance } from '@/lib/utils';
import type {
	Bounds,
	MapCanvasFilter,
	MapCanvasRequest,
	MapCanvasResponse,
	MapHeatBucket,
	MapHotspotItem,
	MapViewport
} from '@/api/types';

// 既有消费方（desktop/mobile/MapHeatDot）从本模块导入这些类型——
// 统一 re-export，避免散落直接依赖 '@/api/types'。
export type { MapCanvasFilter, MapCanvasRequest, MapCanvasResponse, MapHeatBucket, MapHotspotItem };
import type { Pin, PinType } from './location-data';
import markerUserRaw from '@/assets/markers/marker_user.svg?raw';
import markerGroupRaw from '@/assets/markers/marker_group.svg?raw';
import markerMomentRaw from '@/assets/markers/marker_moment.svg?raw';
import markerUserClusterRaw from '@/assets/markers/marker_user_cluster.svg?raw';
import markerGroupClusterRaw from '@/assets/markers/marker_group_cluster.svg?raw';
import markerMomentClusterRaw from '@/assets/markers/marker_moment_cluster.svg?raw';

export const SNAP_MIN = 148;
export const SNAP_MID = 318;
export const SNAP_MAX = 540;
export type SnapState = 'min' | 'mid' | 'max';
export const SNAP_HEIGHTS: Record<SnapState, number> = { min: SNAP_MIN, mid: SNAP_MID, max: SNAP_MAX };

export const PIN_COLORS: Record<PinType, string> = {
	user: 'var(--cst-marker-user)',
	group: 'var(--cst-marker-group)',
	moment: 'var(--cst-marker-moment)'
};
// 图钉淡底/边框/光晕档位：直接引用主题预定义 token，不自行 color-mix 派生
export const PIN_TONES: Record<PinType, { bg: string; border: string; glow: string }> = {
	user: { bg: 'var(--cst-marker-user-bg)', border: 'var(--cst-marker-user-border)', glow: 'var(--cst-marker-user-glow)' },
	group: { bg: 'var(--cst-marker-group-bg)', border: 'var(--cst-marker-group-border)', glow: 'var(--cst-marker-group-glow)' },
	moment: { bg: 'var(--cst-marker-moment-bg)', border: 'var(--cst-marker-moment-border)', glow: 'var(--cst-marker-moment-glow)' }
};
export const PIN_LABELS: Record<PinType, string> = { user: '用户', group: '群组', moment: '瞬间' };
export const floatBtnStyle =
	'background:var(--cst-surface-primary);border:var(--cst-border-width-hairline) solid var(--cst-border-default);box-shadow:var(--cst-elevation-fab);';

export interface MapBounds {
	minLat: number;
	maxLat: number;
	minLng: number;
	maxLng: number;
}

/** 内联渲染图钉 SVG（currentColor 主题化：容器 color 走 --cst-marker-* token，
    深底/白点保留资源固定对比色）。返回完整 <svg> 字符串供 {@html} 使用。 */
export function markerSvg(type: PinType, count: number, size?: { w: number; h: number }) {
	const raw = type === 'user' ? (count >= 3 ? markerUserClusterRaw : markerUserRaw)
		: type === 'group' ? (count >= 3 ? markerGroupClusterRaw : markerGroupRaw)
		: (count >= 3 ? markerMomentClusterRaw : markerMomentRaw);
	const dim = size ? `width="${size.w}" height="${size.h}"` : '';
	const svg = raw.replace('<svg ', `<svg ${dim} `);
	return `<span style="display:inline-block;color:${PIN_COLORS[type]};">${svg}</span>`;
}

/* ── map/canvas API 接入（spec: api_spec_public.md → map/canvas）──────────
   类型统一从 '@/api/types' 引入（mode 判别联合 + RelativeLocation），
   不再维护本地副本。 */

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
export const MAP_DELTA_DEG = 0.045;
export function buildViewportBounds(center: { lat: number; lng: number }, deltaDeg: number = MAP_DELTA_DEG) {
	const clampLat = (v: number) => Math.max(-90, Math.min(90, v));
	const clampLng = (v: number) => Math.max(-180, Math.min(180, v));
	return {
		north: clampLat(center.lat + deltaDeg),
		south: clampLat(center.lat - deltaDeg),
		east: clampLng(center.lng + deltaDeg),
		west: clampLng(center.lng - deltaDeg)
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
		lat: item.location.point.latitude,
		lng: item.location.point.longitude,
		name: item.title,
		title: item.title,
		tag: item.subtitle ?? undefined,
		image: item.cover_url ?? undefined,
		distance: item.location.distance != null ? formatDistance(item.location.distance) : undefined
	};
}

function normalizeCanvas(res: { code: number; data?: unknown }): MapCanvasResult {
	if (res.code !== 0 || !res.data) return emptyResult();
	const data = res.data as MapCanvasResponse;
	if (data.mode === 'heatmap' && data.heatmap.items.length) {
		const items = data.heatmap.items;
		const counts: Record<PinType, number> = { user: 0, group: 0, moment: 0 };
		for (const b of items) {
			counts.user += b.breakdown?.user ?? 0;
			counts.group += b.breakdown?.group ?? 0;
			counts.moment += b.breakdown?.moment ?? 0;
		}
		return { mode: 'heatmap', pins: [], heatBuckets: items, counts };
	}
	if (data.mode === 'hotspots' && data.hotspots.items.length) {
		const items = data.hotspots.items;
		// TODO(next_cursor 翻页): 当 hotspots.next_cursor !== null 时表示还有下一页，
		// 应以 pagination: { cursor: next_cursor, limit: 100 } 循环拉取并合并 pins/counts。
		// 当前首屏固定视口下单页足够（limit 100），与统计同页口径一致；后续接入视口交互/密集区域时补全游标翻页。
		const counts: Record<PinType, number> = { user: 0, group: 0, moment: 0 };
		const pins = items.map(hotspotToPin);
		for (const p of pins) counts[p.type] += 1;
		return { mode: 'hotspots', pins, heatBuckets: [], counts };
	}
	return emptyResult();
}

/**
 * 直接以视口 bounds 拉取地图数据（spec: POST /public/v1/map/canvas），不经过固定 delta 换算。
 * 供已持有真实视口 bounds（如 MapCanvas 计算的 mapBounds）的调用方使用，随 zoom/拖拽自动变化。
 */
export async function fetchMapCanvasByViewport(
	viewport: MapViewport,
	activeTypes: PinType[]
): Promise<MapCanvasResult> {
	if (activeTypes.length === 0) return emptyResult();

	const buildParams = (t?: PinType): MapCanvasRequest => ({
		viewport,
		// spec：filter/pagination 必填可空——未选单类型时 filter 传 null（服务端按 all 处理）
		filter: (t ? { data_type: t, time_range: null } : null) satisfies MapCanvasFilter | null,
		pagination: { limit: 100 }
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

/** 将 MapBounds（minLat/maxLat/minLng/maxLng）转换为 spec Bounds（north/south/east/west）。 */
export function mapBoundsToViewport(bounds: MapBounds): MapViewport {
	return {
		bounds: {
			north: bounds.maxLat,
			south: bounds.minLat,
			east: bounds.maxLng,
			west: bounds.minLng
		} satisfies Bounds
	};
}

/**
 * 拉取当前视口地图数据（spec: POST /public/v1/map/canvas）。
 * - 三种类型全选：data_type 省略（服务端按 all 处理），单次请求
 * - 部分选择：spec 的 data_type 为单值，按选中类型分别请求后合并
 * @deprecated 固定 delta 不随 zoom 变化，仅用于首屏兜底；已有真实视口 bounds 时请用 fetchMapCanvasByViewport。
 */
export async function fetchMapCanvas(
	center: { lat: number; lng: number },
	deltaDeg: number = MAP_DELTA_DEG,
	activeTypes: PinType[]
): Promise<MapCanvasResult> {
	const viewport = { bounds: buildViewportBounds(center, deltaDeg) };
	return fetchMapCanvasByViewport(viewport, activeTypes);
}
