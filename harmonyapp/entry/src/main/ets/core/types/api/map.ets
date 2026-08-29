// ==================== 地图 API ====================
import { GeoPoint, MapViewport, PaginatedData, PaginationParams, RelativeLocation } from './common';


export type MapDataType = 'user' | 'group' | 'moment' | 'all';

export interface TimeRangeParams {
  start_time?: string;
  end_time?: string;
}

export interface MapHeatBreakdown {
  user: number | null;
  group: number | null;
  moment: number | null;
}

export interface MapHeatBucket {
  cell_token: string;
  level: number;
  location: GeoPoint;
  total_count: number;
  breakdown: MapHeatBreakdown;
}

export interface MapHeatCanvas {
  items: MapHeatBucket[];
}

export type HotspotObjectType = 'user' | 'group' | 'moment';

export interface MapHotspotItem {
  object_type: HotspotObjectType;
  object_id: string;
  title: string;
  subtitle: string | null;
  cover_url: string | null;
  location: RelativeLocation;
  created_at: string | null;
}

export interface MapHeatFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;
}

export interface MapHotspotFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;
}

export interface MapHeatRequest {
	viewport: MapViewport;
	filter?: MapHeatFilter;
}

export interface MapHotspotsRequest {
	viewport: MapViewport;
	filter?: MapHotspotFilter;
	pagination?: PaginationParams;
}

// spec public.md → map/canvas hotspots 分支 = PaginatedData<MapHotspotItem>（next_cursor 恒在）
export interface MapHotspotsCanvas {
  items: MapHotspotItem[];
  next_cursor: string | null;
}

export interface MapCanvasFilter {
  data_type?: MapDataType;
  time_range?: TimeRangeParams;
}

export interface MapCanvasRequest {
  viewport: MapViewport;
  filter?: MapCanvasFilter;
  pagination?: PaginationParams;
}

export type MapCanvasMode = 'heatmap' | 'hotspots' | 'empty';

export interface MapCanvasResponse {
  mode: MapCanvasMode;
  heatmap?: MapHeatCanvas;
  hotspots?: MapHotspotsCanvas;
}
