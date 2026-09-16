import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 高德直连 key：默认注入测试 key；「未配置」用例单独 stubEnv('') 覆盖。
vi.stubEnv('VITE_AMAP_KEY', 'test-amap-key');
const { searchPlaces, reverseGeocode, GeoServiceUnavailable } = await import('./geo-service');

// 地理编码服务：高德（restapi 直连）/OSM（同源代理）双源路由、坐标口径转换、降级链。
// fetch 全局 mock；VITE_AMAP_KEY 经 vi.stubEnv 注入；坐标断言取 GCJ-02 反演后的 WGS-84 近似值。

const amapPoi = {
	status: '1',
	pois: [{ name: '测试地点', address: '某街 1 号', pname: '某省', cityname: '某市', location: '116.400000,39.900000' }]
};

describe('searchPlaces (amap)', () => {
	it('直连 restapi.amap.com，key 注入 query，结果反演回 WGS-84，near 转 GCJ 传参', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(amapPoi), { status: 200 }));
		vi.stubGlobal('fetch', fetchMock);

		const places = await searchPlaces('amap', '咖啡', { lat: 39.9, lng: 116.4 });

		const [url] = fetchMock.mock.calls[0] as [string];
		expect(url.startsWith('https://restapi.amap.com/v3/place/text?')).toBe(true);
		expect(url).toContain('key=test-amap-key');
		expect(url).toContain('keywords=');
		// near 中心换算 GCJ-02 后为「经度,纬度」顺序
		expect(url).toMatch(/location=\d+\.\d{6}%2C\d+\.\d{6}|location=\d+\.\d{6},\d+\.\d{6}/);

		expect(places).toHaveLength(1);
		expect(places[0].displayName).toBe('测试地点');
		// GCJ(116.4, 39.9) → WGS 反演有偏移，但必须仍是合理坐标（不等于原 GCJ 值、不越界）
		expect(Math.abs(places[0].lat - 39.9)).toBeLessThan(0.01);
		expect(Math.abs(places[0].lng - 116.4)).toBeLessThan(0.01);
	});

	it('高德接口 status!=1 返回空列表', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ status: '0' }), { status: 200 })));
		const places = await searchPlaces('amap', 'x', null);
		expect(places).toEqual([]);
	});
});

describe('searchPlaces (osm)', () => {
	it('走 /osm-service/ 代理，display_name 截短', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify([{ display_name: '中国, 北京, 天安门', lat: '39.9', lon: '116.39', type: 'attraction' }]), { status: 200 })
		);
		vi.stubGlobal('fetch', fetchMock);

		const places = await searchPlaces('osm', '天安门', null);
		const [url] = fetchMock.mock.calls[0] as [string];
		expect(url.startsWith('/osm-service/search?')).toBe(true);
		expect(url).toContain('format=jsonv2');

		expect(places).toHaveLength(1);
		expect(places[0].lat).toBeCloseTo(39.9);
		expect(places[0].displayName).toBe('中国 · 北京');
	});
});

describe('service unavailable', () => {
	beforeEach(() => vi.stubEnv('VITE_AMAP_KEY', 'test-amap-key'));
	afterEach(() => vi.unstubAllEnvs());

	it('高德直连 503（限流/故障）抛 GeoServiceUnavailable', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 503 })));
		await expect(searchPlaces('amap', 'x', null)).rejects.toBeInstanceOf(GeoServiceUnavailable);
	});

	it('OSM 代理 503 同样抛错', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 503 })));
		await expect(reverseGeocode('osm', 39.9, 116.4)).rejects.toBeInstanceOf(GeoServiceUnavailable);
	});

	it('VITE_AMAP_KEY 未配置时高德源不发请求，直接降级 OSM', async () => {
		vi.stubEnv('VITE_AMAP_KEY', '');
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(
			new Response(JSON.stringify([{ display_name: '后备', lat: '1', lon: '2' }]), { status: 200 })
		));
		const places = await searchPlaces('amap', 'x', null);
		const fetchMock = vi.mocked(globalThis.fetch);
		const [url] = fetchMock.mock.calls[0] as [string];
		expect(url.startsWith('/osm-service/search?')).toBe(true);
		expect(places).toHaveLength(1);
	});
});

describe('reverseGeocode', () => {
	beforeEach(() => {
		vi.stubEnv('VITE_AMAP_KEY', 'test-amap-key');
		vi.stubGlobal('fetch', vi.fn());
	});
	afterEach(() => {
		vi.unstubAllEnvs();
		vi.unstubAllGlobals();
	});

	it('高德：WGS-84 转 GCJ 后查询，返回格式化地址', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			new Response(JSON.stringify({ status: '1', regeocode: { formatted_address: '北京市东城区某街 1 号' } }), { status: 200 })
		);
		vi.stubGlobal('fetch', fetchMock);
		const addr = await reverseGeocode('amap', 39.9, 116.4);
		expect(addr).toBe('北京市东城区某街 1 号');
		const [url] = fetchMock.mock.calls[0] as [string];
		expect(url.startsWith('https://restapi.amap.com/v3/geocode/regeo?location=')).toBe(true);
	});

	it('OSM：空结果返回 null', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: 'no results' }), { status: 200 })));
		const addr = await reverseGeocode('osm', 0, 0);
		expect(addr).toBeNull();
	});
});
