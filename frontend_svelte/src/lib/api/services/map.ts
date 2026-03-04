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

/**
 * 将同坐标（~11m 内）的非 cluster 渲染项合并为 OverlappingGroup，
 * 合并后的条目使用 type='cluster' 便于统一渲染。
 * 返回新的 items 数组和分组映射（groupId → 原始 items）。
 */
export function groupOverlappingRenderItems(items: MapRenderItem[]): {
	merged: MapRenderItem[];
	groups: Map<string, MapRenderItem[]>;
} {
	const coordGroups = new Map<string, MapRenderItem[]>();
	const passThrough: MapRenderItem[] = [];

	for (const item of items) {
		if (item.type === 'cluster') {
			passThrough.push(item);
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

	const merged: MapRenderItem[] = [...passThrough];
	const groups = new Map<string, MapRenderItem[]>();

	for (const [key, groupItems] of coordGroups) {
		const first = groupItems[0];
		if (!first) continue;
		if (groupItems.length === 1) {
			merged.push(first);
		} else {
			const groupId = `${OVERLAP_GROUP_PREFIX}${key}`;
			groups.set(groupId, groupItems);
			merged.push({
				render_id: groupId,
				type: 'cluster',
				location: first.location,
				cluster: {
					count: groupItems.length,
					label: `${groupItems.length} 个内容`
				}
			});
		}
	}

	return { merged, groups };
}
