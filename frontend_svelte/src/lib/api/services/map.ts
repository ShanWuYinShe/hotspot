/**
 * 地图服务
 */

import { post } from '../client';
import { MAP } from '../endpoints';
import type {
	MapCanvas,
	MapDataType,
	MapRenderItem,
	MapViewport,
	TimeRangeParams
} from '$lib/types/api';

export const mapService = {
	/**
	 * 获取地图渲染数据
	 */
	render(data: {
		viewport: MapViewport;
		filter?: {
			data_type?: MapDataType;
			time_range?: TimeRangeParams;
			only_friends?: boolean;
		};
	}) {
		return post<MapCanvas>(MAP.RENDER, data);
	}
};

// 坐标分组精度：4 位小数 ≈ 11 米
const COORD_GROUP_PRECISION = 10000;
const OVERLAP_GROUP_PREFIX = 'overlap_';

export interface OverlappingGroup {
	groupId: string;
	items: MapRenderItem[];
	latitude: number;
	longitude: number;
}

export interface PreparedRenderMarker {
	item: MapRenderItem;
	offset?: [number, number];
}

const OVERLAP_CLUSTER_MAX_ZOOM = 16;

export function isOverlapGroupRenderId(renderId: string): boolean {
	return renderId.startsWith(OVERLAP_GROUP_PREFIX);
}

export function getOverlapClusterExpandZoom(): number {
	return OVERLAP_CLUSTER_MAX_ZOOM + 1;
}

function getFanOutOffset(index: number, total: number): [number, number] {
	const ringRadius = total <= 4 ? 18 : 26;
	const angle = (Math.PI * 2 * index) / total - Math.PI / 2;
	return [Math.round(Math.cos(angle) * ringRadius), Math.round(Math.sin(angle) * ringRadius)];
}

/**
 * 将同坐标（~11m 内）的非 cluster 渲染项合并为 OverlappingGroup，
 * 合并后的条目使用 type='cluster' 便于统一渲染。
 * 返回新的 items 数组和分组映射（groupId → 原始 items）。
 */
export function groupOverlappingRenderItems(items: MapRenderItem[], zoom: number): {
	merged: PreparedRenderMarker[];
	groups: Map<string, MapRenderItem[]>;
} {
	const coordGroups = new Map<string, MapRenderItem[]>();
	const passThrough: PreparedRenderMarker[] = [];

	for (const item of items) {
		if (item.type === 'cluster') {
			passThrough.push({ item });
			continue;
		}
		const key = `${Math.round(item.location.latitude * COORD_GROUP_PRECISION)}_${Math.round(item.location.longitude * COORD_GROUP_PRECISION)}`;
		const group = coordGroups.get(key);
		if (group) {
			group.push(item);
		} else {
			coordGroups.set(key, [item]);
		}
	}

	const merged: PreparedRenderMarker[] = [...passThrough];
	const groups = new Map<string, MapRenderItem[]>();

	for (const [key, groupItems] of coordGroups) {
		const first = groupItems[0];
		if (!first) continue;
		if (groupItems.length === 1) {
			merged.push({ item: first });
		} else if (zoom > OVERLAP_CLUSTER_MAX_ZOOM) {
			groupItems.forEach((item, index) => {
				merged.push({
					item,
					offset: getFanOutOffset(index, groupItems.length)
				});
			});
		} else {
			const groupId = `${OVERLAP_GROUP_PREFIX}${key}`;
			groups.set(groupId, groupItems);
			merged.push({
				item: {
					render_id: groupId,
					type: 'cluster',
					location: first.location,
					cluster: {
						count: groupItems.length,
						label: `${groupItems.length} 个内容`
					}
				}
			});
		}
	}

	return { merged, groups };
}
