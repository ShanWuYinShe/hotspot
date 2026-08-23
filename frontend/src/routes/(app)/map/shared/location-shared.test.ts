// map/canvas 响应解析与合并逻辑测试（spec api_spec_public.md：mode 判别联合
// + RelativeLocation）。
import { describe, it, expect, vi, afterEach } from 'vitest';
import { fetchMapCanvas } from './location-shared';
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
});
