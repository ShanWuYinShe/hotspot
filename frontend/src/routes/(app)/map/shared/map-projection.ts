// Web Mercator 投影与 XYZ 瓦片网格计算（256px 瓦片，EPSG:3857 像素空间）。
// 底图为高德瓦片（GCJ-02 坐标系，经 nginx /amap-tiles/ 同源代理）：mock/展示
// 场景直接以传入经纬度近似对齐；接入真实 WGS-84 数据时需在此层统一做
// WGS-84 → GCJ-02 纠偏，勿散落到组件。

export const TILE_SIZE = 256;

/** 经度 → 世界像素 x（zoom 级像素坐标）。 */
export function lngToWorldX(lng: number, zoom: number): number {
	return ((lng + 180) / 360) * TILE_SIZE * 2 ** zoom;
}

/** 纬度 → 世界像素 y（墨卡托，越靠极地越拉伸）。 */
export function latToWorldY(lat: number, zoom: number): number {
	const s = Math.sin((Math.max(-85.05112878, Math.min(85.05112878, lat)) * Math.PI) / 180);
	return (0.5 - Math.log((1 + s) / (1 - s)) / (4 * Math.PI)) * TILE_SIZE * 2 ** zoom;
}

/** 世界像素 x → 经度。 */
export function worldXToLng(x: number, zoom: number): number {
	return (x / (TILE_SIZE * 2 ** zoom)) * 360 - 180;
}

/** 世界像素 y → 纬度。 */
export function worldYToLat(y: number, zoom: number): number {
	const n = Math.PI - 2 * Math.PI * (y / (TILE_SIZE * 2 ** zoom));
	return (180 / Math.PI) * Math.atan(0.5 * (Math.exp(n) - Math.exp(-n)));
}

export interface ViewportSize {
	width: number;
	height: number;
}

export interface TileItem {
	/** 瓦片 XYZ 坐标 */
	x: number;
	y: number;
	z: number;
	/** 容器内像素位置（左上角） */
	left: number;
	top: number;
}

/** 覆盖视口所需的瓦片网格（含边缘外扩 1 圈，避免拖拽时露白）。 */
export function tilesForViewport(
	center: { lat: number; lng: number },
	zoom: number,
	size: ViewportSize
): TileItem[] {
	const cx = lngToWorldX(center.lng, zoom);
	const cy = latToWorldY(center.lat, zoom);
	const left = cx - size.width / 2;
	const top = cy - size.height / 2;
	const n = 2 ** zoom;
	const x0 = Math.floor(left / TILE_SIZE) - 1;
	const x1 = Math.floor((left + size.width) / TILE_SIZE) + 1;
	const y0 = Math.floor(top / TILE_SIZE) - 1;
	const y1 = Math.floor((top + size.height) / TILE_SIZE) + 1;
	const tiles: TileItem[] = [];
	for (let ty = y0; ty <= y1; ty++) {
		// 墨卡托 y 越界（极地外）不渲染
		if (ty < 0 || ty >= n) continue;
		for (let tx = x0; tx <= x1; tx++) {
			// 经度环绕（x 取模），保证跨反经线连续
			const wx = ((tx % n) + n) % n;
			tiles.push({ x: wx, y: ty, z: zoom, left: tx * TILE_SIZE - left, top: ty * TILE_SIZE - top });
		}
	}
	return tiles;
}

/** 视口四角 → 地理 bounds（spec MapBounds：north/south/east/west 语义的最小外接矩形）。 */
export function viewportToBounds(
	center: { lat: number; lng: number },
	zoom: number,
	size: ViewportSize
): { minLat: number; maxLat: number; minLng: number; maxLng: number } {
	const cx = lngToWorldX(center.lng, zoom);
	const cy = latToWorldY(center.lat, zoom);
	return {
		minLng: worldXToLng(cx - size.width / 2, zoom),
		maxLng: worldXToLng(cx + size.width / 2, zoom),
		maxLat: worldYToLat(cy - size.height / 2, zoom),
		minLat: worldYToLat(cy + size.height / 2, zoom)
	};
}
