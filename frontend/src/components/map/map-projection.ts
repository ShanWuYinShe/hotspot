// Web Mercator 投影与 XYZ 瓦片网格计算（256px 瓦片，EPSG:3857 像素空间）。
// 底图为高德瓦片（GCJ-02 坐标系，浏览器直连高德栅格瓦片服务）。
// 应用数据口径为 WGS-84（GPS/spec）：瓦片像素空间统一先做 WGS-84 → GCJ-02
// 纠偏（本模块 wgs84ToGcj02），组件层只接触 WGS-84，勿散落换算。

export const TILE_SIZE = 256;

/* ── 地图缩放级别界限 ─────────────────────────────────────────────
   画布滚轮/捏合钳制（MapCanvas）与桌面/移动 +/- 按钮（LocationDesktop/Mobile）
   共用，勿在调用处散落字面量。上限 19：高德栅格与 OSM 系瓦片源的支持级别；
   下限 3 放开到洲际视野（此前 13~16 且初始 14，向上仅剩两级）。 */
export const MAP_MIN_ZOOM = 3;
export const MAP_MAX_ZOOM = 19;

/* ── WGS-84 → GCJ-02 纠偏（国测局标准公开算法，中国境外直通） ── */

const GCJ_A = 6378245; // 克拉索夫斯基椭球长半轴
const GCJ_EE = 0.00669342162296594326; // 第一偏心率平方

function transformLat(x: number, y: number): number {
	let ret = -100 + 2 * x + 3 * y + 0.2 * y * y + 0.1 * x * y + 0.2 * Math.sqrt(Math.abs(x));
	ret += ((20 * Math.sin(6 * x * Math.PI) + 20 * Math.sin(2 * x * Math.PI)) * 2) / 3;
	ret += ((20 * Math.sin(y * Math.PI) + 40 * Math.sin((y / 3) * Math.PI)) * 2) / 3;
	ret += ((160 * Math.sin((y / 12) * Math.PI) + 320 * Math.sin((y * Math.PI) / 30)) * 2) / 3;
	return ret;
}

function transformLng(x: number, y: number): number {
	let ret = 300 + x + 2 * y + 0.1 * x * x + 0.1 * x * y + 0.1 * Math.sqrt(Math.abs(x));
	ret += ((20 * Math.sin(6 * x * Math.PI) + 20 * Math.sin(2 * x * Math.PI)) * 2) / 3;
	ret += ((20 * Math.sin(x * Math.PI) + 40 * Math.sin((x / 3) * Math.PI)) * 2) / 3;
	ret += ((150 * Math.sin((x / 12) * Math.PI) + 300 * Math.sin((x / 30) * Math.PI)) * 2) / 3;
	return ret;
}

/** 是否在中国坐标系偏移范围外（境外 GCJ-02 与 WGS-84 一致，直通）。 */
export function outOfChina(lat: number, lng: number): boolean {
	return lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271;
}

/** WGS-84 → GCJ-02（火星坐标）。 */
export function wgs84ToGcj02(lat: number, lng: number): { lat: number; lng: number } {
	if (outOfChina(lat, lng)) return { lat, lng };
	let dLat = transformLat(lng - 105, lat - 35);
	let dLng = transformLng(lng - 105, lat - 35);
	const radLat = (lat / 180) * Math.PI;
	let magic = Math.sin(radLat);
	magic = 1 - GCJ_EE * magic * magic;
	const sqrtMagic = Math.sqrt(magic);
	dLat = (dLat * 180) / (((GCJ_A * (1 - GCJ_EE)) / (magic * sqrtMagic)) * Math.PI);
	dLng = (dLng * 180) / ((GCJ_A / sqrtMagic) * Math.cos(radLat) * Math.PI);
	return { lat: lat + dLat, lng: lng + dLng };
}

/** GCJ-02 → WGS-84（一次反演，误差 ~1m，远小于渲染像素粒度）。 */
export function gcj02ToWgs84(lat: number, lng: number): { lat: number; lng: number } {
	if (outOfChina(lat, lng)) return { lat, lng };
	const gcj = wgs84ToGcj02(lat, lng);
	return { lat: lat - (gcj.lat - lat), lng: lng - (gcj.lng - lng) };
}

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
