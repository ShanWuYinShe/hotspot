// 跨路由复用的地图原语共享类型（components/map/：MapCanvas / MapHeatDot /
// map-projection）。location-shared.ts 从此处再导出保持既有 import 兼容。

export interface MapBounds {
	minLat: number;
	maxLat: number;
	minLng: number;
	maxLng: number;
}
