/* Location types & static data — extracted from the legacy page file to keep the page lean. */

export type PinType = "user" | "group" | "moment";

export interface Pin {
  id: string | number; type: PinType; lat: number; lng: number;
  count?: number;
  name?: string; avatar?: string; distance?: string; online?: boolean;
  members?: number; tag?: string; title?: string; likes?: number; image?: string;
}

export interface LocationResult {
  displayName: string; lat: number; lng: number;
  emoji?: string; desc?: string; users?: number;
}

export const XUZHOU_CENTER = { lat: 34.2604, lng: 117.1855 };

// 地图图钉与统计来自 POST /api/v1/map/canvas（见 location-shared.ts 的 fetchMapCanvas）。
// 以下为暂无数据源的静态空态：热门/最近列表（服务端无对应端点）、地点搜索（search/suggest
// 仅覆盖 tag/user/group/moment，不含地标）。页面在数据源就绪前显示空状态提示。
export const RECENT_LOCATIONS: LocationResult[] = [];
export const HOT_LOCATIONS: LocationResult[] = [];
export const SEARCH_DB: LocationResult[] = [];
