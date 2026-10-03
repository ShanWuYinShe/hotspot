// map/canvas 响应解析与合并逻辑测试（spec api_spec_public.md：mode 判别联合
// + RelativeLocation）。
import { describe, it, expect, vi, afterEach } from 'vitest';
import { fetchMapCanvas, canvasToListPins, aggregateDrillZoom } from './location-shared';
import type { MapBounds } from './location-shared';
import { getMapCanvas } from '@/api';
import type { MapCanvasResponse } from '@/api/types';

vi.mock('@/api', async (importOriginal) => {
	const actual = await importOriginal<typeof import('@/api')>();
	return { ...actual, getMapCanvas: vi.fn() };
});

const CENTER = { lat: 34.2604, lng: 117.1855 };
const ZOOM = 0.045;

function canvasOnce(data: MapCanvasResponse, code = 0) {
	return { code, data };
}

afterEach(() => {
	vi.mocked(getMapCanvas).mockReset();
});

describe('fetchMapCanvas', () => {
	it('hotspots 响应映射为图钉（moment→moment、距离格式化）', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(
			canvasOnce({
				mode: 'hotspots',
				hotspots: {
					items: [
						{
							object_type: 'moment',
							object_id: '42',
							title: '云龙湖日落',
							subtitle: '云龙湖',
							cover_url: 'https://example.com/a.jpg',
							location: {
								point: { latitude: 34.26, longitude: 117.19 },
								location_name: '云龙湖',
								distance: 850,
								bearing_degrees: 45
							},
							created_at: null
						},
						{
							object_type: 'user',
							object_id: '7',
							title: '鹿鸣',
							subtitle: null,
							cover_url: null,
							location: {
								point: { latitude: 34.27, longitude: 117.18 },
								location_name: '',
								distance: null,
								bearing_degrees: null
							},
							created_at: null
						}
					],
					next_cursor: null
				}
			})
		);

		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		expect(result.mode).toBe('hotspots');
		expect(result.pins).toHaveLength(2);
		const moment = result.pins[0];
		expect(moment.type).toBe('moment');
		expect(moment.id).toBe('42');
		expect(moment.lat).toBe(34.26);
		expect(moment.distance).toBe('850m');
		expect(moment.image).toBe('https://example.com/a.jpg');
		expect(moment.tag).toBe('云龙湖');
		expect(result.counts).toEqual({ user: 1, group: 0, moment: 1 });
	});

	it('heatmap 响应按 breakdown 聚合统计', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(
			canvasOnce({
				mode: 'heatmap',
				heatmap: {
					items: [
						{
							cell_token: 't1',
							level: 12,
							location: { latitude: 34.26, longitude: 117.19 },
							total_count: 5,
							breakdown: { user: 3, group: 1, moment: 1 }
						},
						{
							cell_token: 't2',
							level: 12,
							location: { latitude: 34.27, longitude: 117.18 },
							total_count: 2,
							breakdown: { user: 0, group: 2, moment: 0 }
						}
					]
				}
			})
		);

		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		expect(result.mode).toBe('heatmap');
		expect(result.heatBuckets).toHaveLength(2);
		expect(result.pins).toHaveLength(0);
		expect(result.counts).toEqual({ user: 3, group: 3, moment: 1 });
	});

	it('空响应返回 empty 结果', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(canvasOnce({ mode: 'empty' }));

		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		expect(result.mode).toBe('empty');
		expect(result.pins).toHaveLength(0);
		expect(result.heatBuckets).toHaveLength(0);
		expect(result.counts).toEqual({ user: 0, group: 0, moment: 0 });
	});

	it('部分类型选择时按类型分别请求并合并', async () => {
		vi.mocked(getMapCanvas)
			.mockResolvedValueOnce(
				canvasOnce({
					mode: 'hotspots',
					hotspots: {
						items: [
							{
								object_type: 'user',
								object_id: '1',
								title: 'A',
								subtitle: null,
								cover_url: null,
								location: { point: { latitude: 34.26, longitude: 117.19 }, location_name: '', distance: null, bearing_degrees: null },
								created_at: null
							}
						],
						next_cursor: null
					}
				})
			)
			.mockResolvedValueOnce(
				canvasOnce({
					mode: 'hotspots',
					hotspots: {
						items: [
							{
								object_type: 'group',
								object_id: '2',
								title: 'B',
								subtitle: null,
								cover_url: null,
								location: { point: { latitude: 34.27, longitude: 117.18 }, location_name: '', distance: null, bearing_degrees: null },
								created_at: null
							}
						],
						next_cursor: null
					}
				})
			);

		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group']);
		expect(result.mode).toBe('hotspots');
		expect(result.pins).toHaveLength(2);
		expect(result.counts).toEqual({ user: 1, group: 1, moment: 0 });
		expect(getMapCanvas).toHaveBeenCalledTimes(2);
		// data_type 为单值：user/group 各自请求（spec：filter 必填可空）
		expect(vi.mocked(getMapCanvas).mock.calls[0][0]).toMatchObject({ filter: { data_type: 'user' } });
		expect(vi.mocked(getMapCanvas).mock.calls[1][0]).toMatchObject({ filter: { data_type: 'group' } });
	});

	it('全选时单次请求且 filter 为 null', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(canvasOnce({ mode: 'empty' }));

		await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		expect(getMapCanvas).toHaveBeenCalledTimes(1);
		const params = vi.mocked(getMapCanvas).mock.calls[0][0];
		expect(params.viewport.bounds).toEqual({
			north: CENTER.lat + ZOOM,
			south: CENTER.lat - ZOOM,
			east: CENTER.lng + ZOOM,
			west: CENTER.lng - ZOOM
		});
		// spec MapCanvasRequest：filter/pagination 必填可空，全选时 filter=null
		expect(params.filter).toBeNull();
		expect(params.pagination).toEqual({ limit: 100 });
	});

	it('全部类型关闭时不发请求', async () => {
		const result = await fetchMapCanvas(CENTER, ZOOM, []);
		expect(result.mode).toBe('empty');
		expect(getMapCanvas).not.toHaveBeenCalled();
	});

	it('code!=0 或请求失败时返回 empty', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(canvasOnce({ mode: 'empty' }, 10600));
		const errRes = await fetchMapCanvas(CENTER, ZOOM, ['user']);
		expect(errRes.mode).toBe('empty');
		expect(errRes.counts).toEqual({ user: 0, group: 0, moment: 0 });

		vi.mocked(getMapCanvas).mockRejectedValueOnce(new Error('network down'));
		const rejectRes = await fetchMapCanvas(CENTER, ZOOM, ['user']);
		expect(rejectRes.mode).toBe('empty');
		expect(rejectRes.pins).toHaveLength(0);
	});

	it('heatmap 模式下 canvasToListPins 把聚合桶投影为列表行（缩放到城市级不再空态）', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(
			canvasOnce({
				mode: 'heatmap',
				heatmap: {
					items: [
						{
							cell_token: '35d801',
							level: 10,
							location: { latitude: 35.154922, longitude: 112.682524 },
							total_count: 54,
							breakdown: { user: 54, group: null, moment: null }
						},
						{
							cell_token: '367801',
							level: 10,
							location: { latitude: 35.0976, longitude: 112.6124 },
							total_count: 34,
							breakdown: { user: 22, group: 4, moment: 8 }
						}
					]
				}
			})
		);

		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		// 契约：heatmap 不产生实体 pins（地图图钉另有热力渲染）
		expect(result.pins).toHaveLength(0);

		const rows = canvasToListPins(result);
		expect(rows).toHaveLength(2);
		// 聚合行必须可辨识，避免被当成真实实体跳详情
		expect(rows.every((r) => r.isAggregate)).toBe(true);
		// id 加前缀，防与 hotspots 实体 id 撞键
		expect(rows[0].id).toBe('heat:35d801');
		expect(rows[0].lat).toBe(35.154922);
		expect(rows[0].count).toBe(54);
		// breakdown 全为 null 的三分类不崩，且取占比最大类型（user=54 → user）
		expect(rows[0].type).toBe('user');
		expect(rows[0].title).toBe('54 位用户');
		// 第二桶 user=22 为最大分类 → user（徽章配色取占比最大类型）
		expect(rows[1].type).toBe('user');
		expect(rows[1].title).toBe('22 位用户 · 4 个群组 · 8 条动态');
	});

	it('hotspots 模式下 canvasToListPins 原样返回实体条目（不被聚合投影污染）', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(
			canvasOnce({
				mode: 'hotspots',
				hotspots: {
					items: [
						{
							object_type: 'user',
							object_id: 'u1',
							title: '鹿鸣',
							subtitle: null,
							cover_url: null,
							location: { point: { latitude: 34.26, longitude: 117.19 }, location_name: '', distance: null, bearing_degrees: null },
							created_at: null
						}
					],
					next_cursor: null
				}
			})
		);

		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		const rows = canvasToListPins(result);
		expect(rows).toHaveLength(1);
		expect(rows[0].id).toBe('u1');
		expect(rows[0].isAggregate).toBeUndefined();
	});

	it('empty 模式下 canvasToListPins 返回空列表（真实无数据仍显示空态）', async () => {
		vi.mocked(getMapCanvas).mockResolvedValueOnce(canvasOnce({ mode: 'empty' }));
		const result = await fetchMapCanvas(CENTER, ZOOM, ['user', 'group', 'moment']);
		expect(canvasToListPins(result)).toHaveLength(0);
	});
});

// 聚合桶下钻的目标缩放级：跨度必须落在 [20km, 40km)——整格（level-10 ≈25km）
// 成员入画，且稳定低于服务端 50km 热力阈值（span > 50km 会切回 heatmap，
// 下钻后看到的还是徽标而不是明细，等于白点）。
describe('aggregateDrillZoom', () => {
	const CENTER_LAT = 35.1;

	function boundsAround(spanDeg: number): MapBounds {
		return {
			maxLat: CENTER_LAT + spanDeg / 2,
			minLat: CENTER_LAT - spanDeg / 2,
			maxLng: 112.61 + spanDeg / 2,
			minLng: 112.61 - spanDeg / 2
		};
	}

	// 与后端 ApproxViewportSpanMeters 同口径的跨度复算（米）
	function spanMeters(b: MapBounds): number {
		const latM = Math.abs(b.maxLat - b.minLat) * 111000;
		const lngM = Math.abs(b.maxLng - b.minLng) * 111000 * Math.cos((((b.maxLat + b.minLat) / 2) * Math.PI) / 180);
		return Math.hypot(latM, lngM);
	}

	it.each([
		// [当前跨度(度), 当前 zoom] —— 物理上成立的配对：span ≈ w×360/(256×2^z)。
		// 桌面（1200px）：6.59°@8 / 1.65°@10 / 0.82°@11
		// 移动（390px）：2.14°@8 / 0.535°@10 / 0.268°@11
		// 均为 heatmap 视口（span > 50km），即徽标可见的状态。
		[6.59, 8],
		[1.65, 10],
		[0.82, 11],
		[2.14, 8],
		[0.535, 10],
		[0.268, 11]
	])('当前跨度 %d 度 / zoom %d：下钻后跨度 ∈ [20km, 40km) 且不缩小地图', (spanDeg, currentZoom) => {
		const bounds = boundsAround(spanDeg);
		const zoom = aggregateDrillZoom(currentZoom, bounds);
		expect(zoom).toBeGreaterThanOrEqual(currentZoom);
		expect(zoom).toBeLessThanOrEqual(19);
		const after = spanMeters(bounds) / Math.pow(2, zoom - currentZoom);
		expect(after).toBeGreaterThanOrEqual(20000);
		expect(after).toBeLessThan(40000);
	});

	it('已是街区级（跨度小于目标，理论混合场景）时不缩小地图', () => {
		const bounds = boundsAround(0.1); // ≈14km，已比目标细
		expect(aggregateDrillZoom(16, bounds)).toBe(16);
	});
});
