// 地图共享逻辑。

import { getMapCanvas } from '@/api';
import { formatDistance } from '@/lib/utils';
import { MAP_MIN_ZOOM, MAP_MAX_ZOOM } from '@/components/map/map-projection';
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
// 统一 re-export，避免散落直接依赖 '@/api/types'。MapBounds 已上收到
// components/map/types.ts（跨路由地图原语），此处转出保持既有 import 兼容。
import type { MapBounds } from '@/components/map/types';
export type { MapCanvasFilter, MapCanvasRequest, MapCanvasResponse, MapHeatBucket, MapHotspotItem };
export type { MapBounds };
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
	// 白名单校验而非裸 as：未知 object_type（后端新增类型/异常数据）回落 user，
	// 避免图钉取色/标签查表 undefined 静默失色。
	const type: PinType = (['user', 'group', 'moment'] as const).includes(
		item.object_type as PinType
	)
		? (item.object_type as PinType)
		: 'user';
	return {
		id: item.object_id,
		type,
		lat: item.location.point.latitude,
		lng: item.location.point.longitude,
		name: item.title,
		title: item.title,
		tag: item.subtitle ?? undefined,
		image: item.cover_url ?? undefined,
		avatar: item.cover_url ?? undefined,
		distance: item.location.distance != null ? formatDistance(item.location.distance) : undefined
	};
}

/** 热力桶的"附近"列表行标题（无实体明细，只能给出聚合描述）。 */
function heatBucketTitle(counts: Partial<Record<PinType, number>>): string {
	const parts: string[] = [];
	if (counts.user) parts.push(`${counts.user} 位用户`);
	if (counts.group) parts.push(`${counts.group} 个群组`);
	if (counts.moment) parts.push(`${counts.moment} 条动态`);
	return parts.join(' · ') || '附近内容';
}

/**
 * 热力桶 → 列表项 Pin（列表专用，不参与地图图钉渲染）。
 *
 * 视口跨度大时服务端返回 heatmap（S2 网格聚合，无实体明细），此时 pins 为空。
 * 若直接以 pins 驱动"附近"列表，就会出现"缩放到全市后附近列表说没有内容，
 * 但地图上明明有热力点"的自相矛盾——用户坐标没变、数据也没变，只是模式切换了。
 * 这里把每个桶投影成一行聚合条目，保证任何缩放级别下列表与地图口径一致。
 *
 * 注意：热力桶不是真实实体（没有 object_id），因此：
 * - id 用 cell_token 并加前缀，避免与 hotspots 的实体 id 撞键；
 * - type 取桶内占比最大的类型，仅用于列表行的类型徽章/配色；
 * - 不可点击进入详情（见 Pin.isAggregate），由调用方改为定位到该格。
 */
function heatBucketToPin(bucket: MapHeatBucket): Pin {
	const bd = bucket.breakdown;
	const counts: Record<PinType, number> = {
		user: bd?.user ?? 0,
		group: bd?.group ?? 0,
		moment: bd?.moment ?? 0
	};
	let type: PinType = 'user';
	if (counts.group > counts[type]) type = 'group';
	if (counts.moment > counts[type]) type = 'moment';
	return {
		id: `heat:${bucket.cell_token}`,
		type,
		lat: bucket.location.latitude,
		lng: bucket.location.longitude,
		name: heatBucketTitle(counts),
		title: heatBucketTitle(counts),
		count: bucket.total_count,
		isAggregate: true
	};
}

/**
 * 列表行数据源：hotspots 用实体条目；heatmap 用聚合桶投影（见 heatBucketToPin）。
 * 两种模式都返回非空列表，缩放不再把"附近"列表变成空态。
 */
export function canvasToListPins(result: MapCanvasResult): Pin[] {
	if (result.pins.length > 0) return result.pins;
	return result.heatBuckets.map(heatBucketToPin);
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

/** 聚合桶下钻的目标跨度（米）。服务端按视口跨度选择聚合网格，桶边长随缩放变化：下钻后
 *  视口要容纳整格成员，又要稳定低于服务端 50km 热力阈值——span > 50km 会切回
 *  heatmap，下钻就白点了。取 20km 并向下取整后，实际跨度恒落在 [20km, 40km)，
 *  任何屏宽/纬度都同时满足两个条件。 */
const AGGREGATE_DRILL_SPAN_METERS = 20000;

/**
 * 聚合桶下钻的目标缩放级：从当前 zoom 与视口 bounds 换算，使下钻后视口跨度
 * 落在 [AGGREGATE_DRILL_SPAN_METERS, 2×) 区间。
 *
 * 不能用固定 zoom：跨度随视口像素宽度变化（同一 zoom 下 1200px 桌面 ≈37km、
 * 390px 移动 ≈12km）——桌面版曾用固定 14（≈11.6km）勉强可用，移动端在 14 级
 * 只有 ~3km 跨度，整格成员根本不在画内。
 */
export function aggregateDrillZoom(currentZoom: number, bounds: MapBounds): number {
	const centerLat = (bounds.minLat + bounds.maxLat) / 2;
	const latMeters = Math.abs(bounds.maxLat - bounds.minLat) * 111000;
	const lngMeters = Math.abs(bounds.maxLng - bounds.minLng) * 111000 * Math.cos((centerLat * Math.PI) / 180);
	const spanMeters = Math.hypot(latMeters, lngMeters);
	// 跨度随 zoom 每 +1 减半：span(z) = span_now × 2^(currentZoom − z)
	const zoom = currentZoom + Math.log2(spanMeters / AGGREGATE_DRILL_SPAN_METERS);
	const clamped = Math.max(MAP_MIN_ZOOM, Math.min(MAP_MAX_ZOOM, Math.floor(zoom)));
	// 只放大不缩小：徽标只在 heatmap 视口（span > 50km）出现，此时必应放大；
	// 理论混合场景下若已在更细的视口被调到，保持现状而不是把用户拉远。
	return Math.max(currentZoom, clamped);
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
