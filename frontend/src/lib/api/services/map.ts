/**
 * 地图服务
 */

import { post } from '../client';
import { MAP } from '../endpoints';
import type { MapCanvasRequest, MapCanvasResponse, MapHeatBucket } from '$lib/types/api';

export const mapService = {
  /**
   * 地图画布：根据视口自动选择返回统计或热点明细，对应 /map/canvas
   */
  canvas(data: MapCanvasRequest) {
    return post<MapCanvasResponse>(MAP.CANVAS, data);
  }
};

// 坐标分组精度：4 位小数 ≈ 11 米
const COORD_GROUP_PRECISION = 10000;
const OVERLAP_GROUP_PREFIX = 'overlap_';

// 当地图缩放级别小于等于该值时，仍将同一点附近的多个标记合并为“重叠 cluster”；
// 当 zoom 大于该值时，改为在原位置周围打散为多个带像素偏移的独立标记，便于查看细节。
const OVERLAP_CLUSTER_MAX_ZOOM = 15;

// 前端内部用于聚合展示的渲染项类型（在 MapHeatBucket 基础上增加本地标识符 render_id）
export interface RenderItem extends MapHeatBucket {
  render_id: string;
}

export interface OverlappingGroup {
  groupId: string;
  items: RenderItem[];
  latitude: number;
  longitude: number;
}

export interface PreparedRenderMarker {
  item: RenderItem;
  offset?: [number, number];
}

export function isOverlapGroupRenderId(renderId: string): boolean {
  return renderId.startsWith(OVERLAP_GROUP_PREFIX);
}

function getFanOutOffset(index: number, total: number): [number, number] {
  // 在高缩放级别下，为同一点的多个标记提供一个简单且“顺眼”的像素偏移方案：
  // - 2~3 个点时采用垂直堆叠，避免绕圈分布导致认知负担；
  // - 4~6 个点时围绕原点做小半径放射状布局；
  // - 更多点时使用稍大半径，保证标记之间不互相遮挡。
  if (total === 2) {
    // 上下各一个
    return index === 0 ? [0, -16] : [0, 16];
  }
  if (total === 3) {
    // 上、中、下三列
    if (index === 0) return [0, -18];
    if (index === 1) return [0, 0];
    return [0, 18];
  }

  const ringRadius = total <= 6 ? 20 : 28;
  const angle = (Math.PI * 2 * index) / total - Math.PI / 2;
  return [Math.round(Math.cos(angle) * ringRadius), Math.round(Math.sin(angle) * ringRadius)];
}

/**
 * 将同坐标（~11m 内）的非 cluster 渲染项在客户端进行处理：
 * - 当 zoom 不大时（≤ OVERLAP_CLUSTER_MAX_ZOOM），合并为 OverlappingGroup，统一展示为一个 pseudo-cluster；
 * - 当 zoom 很大时（> OVERLAP_CLUSTER_MAX_ZOOM），不再合并为 cluster，而是在原位置附近按像素打散为多个标记，便于直接查看具体对象。
 * 返回新的 items 数组和分组映射（groupId → 原始 items）。
 */
export function groupOverlappingRenderItems(
  items: RenderItem[],
  zoom: number
): {
  merged: PreparedRenderMarker[];
  groups: Map<string, RenderItem[]>;
} {
  const coordGroups = new Map<string, RenderItem[]>();

  for (const item of items) {
    const key = `${Math.floor(item.location.latitude * COORD_GROUP_PRECISION)}_${Math.floor(item.location.longitude * COORD_GROUP_PRECISION)}`;
    const group = coordGroups.get(key);
    if (group) {
      group.push(item);
    } else {
      coordGroups.set(key, [item]);
    }
  }

  const merged: PreparedRenderMarker[] = [];
  const groups = new Map<string, RenderItem[]>();

  for (const [key, groupItems] of coordGroups) {
    const first = groupItems[0];
    if (!first) continue;
    if (groupItems.length === 1) {
      merged.push({ item: first });
    } else if (zoom > OVERLAP_CLUSTER_MAX_ZOOM) {
      // 高缩放级别：将同一点附近的多个聚合点在地图上打散为多个独立标记，仅使用像素偏移，便于查看细节。
      groupItems.forEach((item, index) => {
        merged.push({
          item,
          offset: getFanOutOffset(index, groupItems.length)
        });
      });
    } else {
      // 较低缩放级别：将同一坐标附近的多个格网进一步汇总为一个“重叠单元”，用于统计展示
      const groupId = `${OVERLAP_GROUP_PREFIX}${key}`;
      groups.set(groupId, groupItems);
      const totalCount = groupItems.reduce((sum, item) => sum + item.total_count, 0);
      merged.push({
        item: {
          ...first,
          render_id: groupId,
          total_count: totalCount
        }
      });
    }
  }

  return { merged, groups };
}
