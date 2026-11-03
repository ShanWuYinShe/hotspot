// 瓦片网格契约：MapCanvas 以 `source/z/x/y/copy` 作 keyed each 的 key，任何重复 key
// 都会抛 each_key_duplicate 并使整层瓦片停止渲染（/admin/stats zoom=3 曾在 1596×320
// 画布上整块地图空白）。同时「低 zoom 经度环绕」不能靠丢弃重复瓦片来消除 key 冲突：
// 世界宽 2048px 时跨接缝/宽视口两侧的同一瓦片两份都落在视口内，丢一份就露白。
// 本文件锁定：复合 key 恒唯一 + 视口（含跨接缝、宽屏）完整覆盖。
import { describe, it, expect } from 'vitest';
import { tilesForViewport, TILE_SIZE, type TileItem, type ViewportSize } from '../map-projection';

/** MapCanvas keyed each 实际使用的 key。 */
const renderKey = (t: TileItem) => `${t.z}/${t.x}/${t.y}/${t.copy}`;
/** 瓦片身份（不含世界拷贝）。 */
const tileKey = (t: TileItem) => `${t.z}/${t.x}/${t.y}`;

function expectUniqueRenderKeys(tiles: TileItem[]): void {
	const keys = tiles.map(renderKey);
	expect(new Set(keys).size).toBe(keys.length);
}

/** 在视口上采样 9×6 个点（含四角/边），断言每个点都落在某块瓦片内。 */
function expectViewportCovered(tiles: TileItem[], size: ViewportSize): void {
	const cols = 8;
	const rows = 5;
	for (let i = 0; i <= cols; i++) {
		for (let j = 0; j <= rows; j++) {
			const px = ((size.width - 1) * i) / cols;
			const py = ((size.height - 1) * j) / rows;
			const hit = tiles.some(
				(t) => px >= t.left && px < t.left + TILE_SIZE && py >= t.top && py < t.top + TILE_SIZE
			);
			if (!hit) expect.fail(`视口点 (${px}, ${py}) 未被任何瓦片覆盖`);
		}
	}
}

describe('tilesForViewport：key 唯一 + 视口覆盖', () => {
	it('zoom=3 中国中心 1596×320（/admin/stats 实际画布）：复合 key 与 (z,x,y) 均唯一、视口完整覆盖（回归 each_key_duplicate）', () => {
		const size: ViewportSize = { width: 1596, height: 320 };
		const tiles = tilesForViewport({ lat: 35.0, lng: 105.0 }, 3, size);

		expect(tiles.length).toBeGreaterThan(0);
		expectUniqueRenderKeys(tiles);
		// 该视口下同一瓦片只有一份可见拷贝：不应出现 copy 多份
		const identities = tiles.map(tileKey);
		expect(new Set(identities).size).toBe(identities.length);
		expectViewportCovered(tiles, size);
		for (const t of tiles) {
			expect(t.z).toBe(3);
			expect(t.x).toBeGreaterThanOrEqual(0);
			expect(t.x).toBeLessThan(8);
			expect(t.y).toBeGreaterThanOrEqual(0);
			expect(t.y).toBeLessThan(8);
		}
	});

	it('zoom=3 中国中心 1920×1080（跨接缝、同瓦片两份都可见）：复合 key 唯一且视口完整覆盖', () => {
		const size: ViewportSize = { width: 1920, height: 1080 };
		const tiles = tilesForViewport({ lat: 35.0, lng: 105.0 }, 3, size);

		expect(tiles.length).toBeGreaterThan(0);
		expectUniqueRenderKeys(tiles);
		expectViewportCovered(tiles, size);
		// 跨接缝场景确实需要同一 (z,x,y) 的世界拷贝两份，靠 copy 区分
		const counts = new Map<string, number>();
		for (const t of tiles) counts.set(tileKey(t), (counts.get(tileKey(t)) ?? 0) + 1);
		expect([...counts.values()].some((n) => n === 2)).toBe(true);
	});

	it('zoom=3 宽屏 2560×1080（视口宽于世界宽 2048）：复合 key 唯一且视口完整覆盖', () => {
		const size: ViewportSize = { width: 2560, height: 1080 };
		const tiles = tilesForViewport({ lat: 35.0, lng: 105.0 }, 3, size);

		expectUniqueRenderKeys(tiles);
		expectViewportCovered(tiles, size);
	});

	it('高 zoom 常规视口 1288×980（/map 桌面画布）：复合 key 与 (z,x,y) 均唯一、视口完整覆盖', () => {
		const size: ViewportSize = { width: 1288, height: 980 };
		const tiles = tilesForViewport({ lat: 34.25, lng: 117.36 }, 14, size);

		expect(tiles.length).toBeGreaterThan(0);
		expectUniqueRenderKeys(tiles);
		const identities = tiles.map(tileKey);
		expect(new Set(identities).size).toBe(identities.length);
		expectViewportCovered(tiles, size);
		for (const t of tiles) expect(t.z).toBe(14);
	});

	it('跨反经线（lng=180）zoom=3 1596×320：复合 key 唯一且视口完整覆盖', () => {
		const size: ViewportSize = { width: 1596, height: 320 };
		const tiles = tilesForViewport({ lat: 0, lng: 180 }, 3, size);

		expectUniqueRenderKeys(tiles);
		expectViewportCovered(tiles, size);
	});

	it('极地外 y 越界不渲染：所有 y 落在 [0, 2^zoom)', () => {
		const tiles = tilesForViewport({ lat: 85, lng: 105 }, 3, { width: 1288, height: 980 });
		expect(tiles.length).toBeGreaterThan(0);
		for (const t of tiles) {
			expect(t.y).toBeGreaterThanOrEqual(0);
			expect(t.y).toBeLessThan(8);
		}
	});
});
