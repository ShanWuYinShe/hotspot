// msw map/canvas handler 集成测试：确保 mock 响应结构对齐 spec
// （api_spec_public.md → map/canvas：mode tagged union + RelativeLocation）。
// 直接调用 handler resolver（msw/node 网络层与 vitest 4 fetch 不兼容）。
import { describe, it, expect } from 'vitest';
import { handlers } from './handlers';

const mapHandler = handlers.find((h) => h.info.path === '/public/v1/map/canvas')!;

async function canvas(body: unknown): Promise<any> {
	const result = await mapHandler.run({
		request: new Request('http://test.local/public/v1/map/canvas', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body)
		}),
		requestId: 'test',
		// node 环境无 location，需显式 baseUrl 让相对路径 handler 匹配绝对 URL
		resolutionContext: { baseUrl: 'http://test.local' }
	});
	if (!result?.response) throw new Error('map/canvas handler did not match');
	return result.response.json();
}

const VIEWPORT = { viewport: { bounds: { north: 34.3054, south: 34.2154, east: 117.2305, west: 117.1405 } }, pagination: { limit: 100 } };

describe('msw map/canvas handler', () => {
	it('缺省 data_type 返回三种类型混合的 hotspots（spec tagged union：mode=hotspots）', async () => {
		const res = await canvas(VIEWPORT);
		expect(res.code).toBe(0);
		// spec：响应为 { mode: 'heatmap' | 'hotspots' | 'empty', ... }，mock 恒返回 hotspots 明细
		expect(res.data.mode).toBe('hotspots');
		const items = res.data.hotspots.items as any[];
		expect(items.length).toBeGreaterThan(0);
		expect(new Set(items.map((i) => i.object_type))).toEqual(new Set(['user', 'group', 'moment']));
		// PaginatedData：next_cursor 恒在
		expect(res.data.hotspots.next_cursor).toBeNull();
		// 结构对齐 spec：location 为 RelativeLocation（point 嵌套 + 米数 distance/bearing_degrees）
		const first = items[0];
		expect(typeof first.location.point.latitude).toBe('number');
		expect(typeof first.location.point.longitude).toBe('number');
		expect(typeof first.location.distance).toBe('number');
		expect(typeof first.location.bearing_degrees).toBe('number');
		expect(first.distance_meters).toBeUndefined();
	});

	it('data_type=user / group / moment 分别返回对应类型', async () => {
		for (const t of ['user', 'group', 'moment']) {
			const res = await canvas({ ...VIEWPORT, filter: { data_type: t } });
			const items = res.data.hotspots.items as any[];
			expect(items.length).toBeGreaterThan(0);
			expect(items.every((i) => i.object_type === t)).toBe(true);
		}
	});

	it('请求体缺失时按全部返回（不报错）', async () => {
		const res = await canvas({});
		expect(res.code).toBe(0);
		expect(Array.isArray(res.data.hotspots.items)).toBe(true);
	});
});
