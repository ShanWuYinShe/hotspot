// Web Mercator 投影与 XYZ 瓦片网格计算（256px 瓦片，EPSG:3857 像素空间）。
// 底图为高德瓦片（GCJ-02 坐标系，浏览器直连高德栅格瓦片服务）。
// 应用数据口径为 WGS-84（GPS/spec）：瓦片像素空间统一先做 WGS-84 → GCJ-02
// 纠偏（本模块 wgs84ToGcj02），组件层只接触 WGS-84，勿散落换算。

export const TILE_SIZE = 256;

/* ── 地图缩放级别界限 ─────────────────────────────────────────────
   画布滚轮/捏合钳制（MapCanvas）与桌面/移动 +/- 按钮（LocationDesktop/Mobile）
   共用，勿在调用处散落字面量。上限 18（实测两套瓦片源的有效上限）：
   - 高德栅格 z=19 返回 HTTP 200 但内容为空白瓦片（整屏地图消失）；
   - tile.openstreetmap.de（dev/prod 的 OSM 上游）z=19 返回 404。
   下限 3 放开到洲际视野（此前 13~16 且初始 14，向上仅剩两级）。 */
export const MAP_MIN_ZOOM = 3;
export const MAP_MAX_ZOOM = 18;

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

/** GCJ-02 → WGS-84（不动点迭代反演，两轮后误差 < 1cm）。
 *
 *	正变换 wgs84ToGcj02 无闭式逆，一次反演残留 ~1m 系统性偏差；滚轮/捏合
 *	每缩放一级都要经过一次 WGS↔GCJ 往返，偏差逐级同向累积（14→18 级实测
 *	漂移约 6m，高 zoom 下约 10px、「我的位置」肉眼可见滑 off 房顶），故
 *	必须迭代到厘米级再收敛。境外直通不变。 */
export function gcj02ToWgs84(lat: number, lng: number): { lat: number; lng: number } {
	if (outOfChina(lat, lng)) return { lat, lng };
	let wLat = lat;
	let wLng = lng;
	for (let i = 0; i < 2; i++) {
		const g = wgs84ToGcj02(wLat, wLng);
		wLat = lat - (g.lat - wLat);
		wLng = lng - (g.lng - wLng);
	}
	return { lat: wLat, lng: wLng };
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
	/**
	 * 经度环绕的第几份世界拷贝。世界像素宽不足「视口宽 + 两侧外扩一圈」时，
	 * 同一 (z,x,y) 只有一个世界拷贝的话盖不满视口（低 zoom / 宽屏 / 跨反经线），
	 * 需要相邻拷贝各渲染一份——用 copy 区分身份，避免 keyed each key 重复。
	 */
	copy: number;
}

/**
 * 覆盖视口所需的瓦片网格（含边缘外扩 1 圈，避免拖拽时露白）。
 *
 * 契约：MapCanvas 以 `${source}/${z}/${x}/${y}/${copy}` 作 keyed each 的 key，
 * 任何重复 key 都会抛 each_key_duplicate 并使整层瓦片停止渲染（/admin/stats
 * zoom=3 曾因此整块地图空白）。因此：
 * 1. (z,x,y,copy) 恒唯一；
 * 2. 视口被瓦片完整覆盖：每个 (z,x,y) 通常只保留一份；仅当经度环绕后同一瓦片
 *    的两个屏幕位置都落在视口内（世界宽 2048px 附近的低 zoom、宽屏或跨接缝）时
 *    保留多份（copy 不同）。不能简单「只留相交面积最大的一份」，否则跨接缝一侧露白。
 */
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
	// 先按瓦片身份（wx/ty）分组，再按「与视口相交」取舍世界拷贝：
	// - 相交的拷贝全部保留（跨接缝时可能 2 份，丢任一份都会露白）；
	// - 全部都不是可见拷贝（只来自外扩 padding 的远端环绕副本）时保留 1 份，
	//   维持「拖拽边缘不露白」语义且不无谓放大 DOM。
	const groups = new Map<string, TileItem[]>();
	for (let ty = y0; ty <= y1; ty++) {
		// 墨卡托 y 越界（极地外）不渲染
		if (ty < 0 || ty >= n) continue;
		for (let tx = x0; tx <= x1; tx++) {
			const wx = ((tx % n) + n) % n;
			const tile: TileItem = {
				x: wx,
				y: ty,
				z: zoom,
				left: tx * TILE_SIZE - left,
				top: ty * TILE_SIZE - top,
				copy: Math.floor(tx / n)
			};
			const key = `${wx}/${ty}`;
			const list = groups.get(key);
			if (list) list.push(tile);
			else groups.set(key, [tile]);
		}
	}
	const tiles: TileItem[] = [];
	for (const list of groups.values()) {
		const visible = list.filter((t) => viewportOverlap(t.left, t.top, size) > 0);
		if (visible.length > 0) {
			tiles.push(...visible);
			continue;
		}
		tiles.push(
			list.reduce((a, b) =>
				viewportOverlap(b.left, b.top, size) > viewportOverlap(a.left, a.top, size) ? b : a
			)
		);
	}
	return tiles;
}

/** 瓦片矩形与视口的相交面积（px²）；经度环绕取舍世界拷贝时用于判定可见性。 */
function viewportOverlap(left: number, top: number, size: ViewportSize): number {
	const w = Math.max(0, Math.min(left + TILE_SIZE, size.width) - Math.max(left, 0));
	const h = Math.max(0, Math.min(top + TILE_SIZE, size.height) - Math.max(top, 0));
	return w * h;
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
