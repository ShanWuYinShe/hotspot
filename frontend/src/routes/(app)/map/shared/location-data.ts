/* Location types & static data — 地图页私有类型与静态数据。 */

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

// 地图图钉与统计来自 POST /api/v1/map/canvas（见 location-shared.ts 的 fetchMapCanvas）。
// 热门/最近地点与地标搜索：服务端暂无对应端点，此处用与页面同域的静态演示数据
// （mock 阶段保证 Tab 与搜索 UI 可用；后端就绪后替换为端点数据）。
export const RECENT_LOCATIONS: LocationResult[] = [
  { displayName: '老街手冲馆', lat: 34.26, lng: 117.18, emoji: '☕', desc: '刚刚去过 · 咖啡与旧街', users: 12 },
  { displayName: '云龙湖', lat: 34.25, lng: 117.19, emoji: '🌊', desc: '昨天 · 环湖步道', users: 48 },
  { displayName: '汉文化景区', lat: 34.27, lng: 117.2, emoji: '🏛️', desc: '3 天前 · 历史漫步', users: 21 }
];
export const HOT_LOCATIONS: LocationResult[] = [
  { displayName: '云龙湖', lat: 34.25, lng: 117.19, emoji: '🌊', desc: '今天 214 人打卡', users: 214 },
  { displayName: '户部山弹唱角', lat: 34.26, lng: 117.18, emoji: '🎸', desc: '今天 156 人打卡', users: 156 },
  { displayName: '植物园', lat: 34.24, lng: 117.21, emoji: '🌿', desc: '今天 132 人打卡', users: 132 },
  { displayName: '彭城广场', lat: 34.26, lng: 117.19, emoji: '⛲', desc: '今天 98 人打卡', users: 98 }
];
export const SEARCH_DB: LocationResult[] = [
  ...HOT_LOCATIONS,
  { displayName: '回龙窝', lat: 34.26, lng: 117.18, emoji: '🏘️', desc: '老城街区 · 胶片摄影', users: 64 },
  { displayName: '黄楼夜读角', lat: 34.26, lng: 117.19, emoji: '📚', desc: '夜读与沙龙', users: 37 },
  { displayName: '市中心', lat: 34.26, lng: 117.19, emoji: '🏙️', desc: '商业街区', users: 180 }
];
