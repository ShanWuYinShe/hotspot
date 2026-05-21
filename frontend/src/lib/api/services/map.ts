import { post } from '../client';
import { MAP } from '../endpoints';
import type { MapCanvasRequest, MapCanvasResponse, MapHeatBucket } from '@/lib/types/api';

export const mapService = {
  canvas(data: MapCanvasRequest) {
    return post<MapCanvasResponse>(MAP.CANVAS, data);
  }
};

const COORD_GROUP_PRECISION = 10000;
const OVERLAP_GROUP_PREFIX = 'overlap_';
const OVERLAP_CLUSTER_MAX_ZOOM = 15;

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
  if (total === 2) {
    return index === 0 ? [0, -16] : [0, 16];
  }
  if (total === 3) {
    if (index === 0) return [0, -18];
    if (index === 1) return [0, 0];
    return [0, 18];
  }
  const ringRadius = total <= 6 ? 20 : 28;
  const angle = (Math.PI * 2 * index) / total - Math.PI / 2;
  return [Math.round(Math.cos(angle) * ringRadius), Math.round(Math.sin(angle) * ringRadius)];
}

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
      groupItems.forEach((item, index) => {
        merged.push({
          item,
          offset: getFanOutOffset(index, groupItems.length)
        });
      });
    } else {
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
