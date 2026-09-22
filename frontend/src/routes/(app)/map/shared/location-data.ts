/* Location types — 地图页私有类型。 */

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

// 图钉与统计来自 POST /public/v1/map/canvas（见 location-shared.ts 的 fetchMapCanvas）；
// 地点选择来自真实地理编码搜索（geo-service.ts，高德/OSM 随地图源）。
// 此处曾放「热门/最近/搜索兜底」静态演示数据（含虚构打卡人数），已按产品决定移除；
// 服务端就绪热门地点端点前，勿回填任何本地假数据。
