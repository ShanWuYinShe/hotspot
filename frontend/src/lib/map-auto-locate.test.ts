import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 地图首屏定位 helper：锚点读写（含历史数组格式兼容）、拒绝记忆、自动定位三态。
// navigator.geolocation 经 defineProperty 桩化；拒绝/超时错误需自带实例常量
// （PERMISSION_DENIED 等），否则 mapNativeError 的 switch 走 default 分支。
const { autoLocateDeclined, autoLocateOnce, readLastLocation, writeLastLocation } = await import(
	'./map-auto-locate'
);

const CODES = { PERMISSION_DENIED: 1, POSITION_UNAVAILABLE: 2, TIMEOUT: 3 };
const KEY = 'hotspot_last_location';

function stubGeolocation(impl: GeolocationPosition | null, err?: { code: number }) {
	const mock = impl
		? { getCurrentPosition: vi.fn((ok: PositionCallback) => ok(impl)) }
		: {
				getCurrentPosition: vi.fn((_ok: PositionCallback, fail: PositionErrorCallback) =>
					fail({ ...CODES, code: err!.code } as unknown as GeolocationPositionError)
				)
			};
	Object.defineProperty(navigator, 'geolocation', { value: mock, configurable: true });
	return mock;
}

function mockFix(): GeolocationPosition {
	return { coords: { latitude: 39.9, longitude: 116.4, accuracy: 25 } } as GeolocationPosition;
}

describe('readLastLocation', () => {
	beforeEach(() => localStorage.clear());

	it('历史数组格式 [lat, lng] 仍可解析（向后兼容）', () => {
		localStorage.setItem(KEY, '[39.9,116.4]');
		expect(readLastLocation()).toEqual({ lat: 39.9, lng: 116.4 });
	});

	it('对象格式带名称时保留名称', () => {
		localStorage.setItem(KEY, JSON.stringify({ lat: 30.66, lng: 104.06, name: '成都' }));
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06, name: '成都' });
	});

	it('名称为空串时按无名称处理', () => {
		localStorage.setItem(KEY, JSON.stringify({ lat: 30.66, lng: 104.06, name: '  ' }));
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06 });
	});

	const invalid: Array<[string, string | null]> = [
		['缺失', null],
		['非法 JSON', 'xxx'],
		['非坐标对象', '{"a":1}'],
		['非数字数组', '["a","b"]'],
		['数组越界', '[91,116.4]'],
		['对象越界', '{"lat":91,"lng":116.4}'],
		['JSON null', 'null']
	];

	it.each(invalid)('%s 时返回 null', (_label, raw) => {
		if (raw === null) localStorage.removeItem(KEY);
		else localStorage.setItem(KEY, raw);
		expect(readLastLocation()).toBeNull();
	});
});

describe('writeLastLocation', () => {
	beforeEach(() => localStorage.clear());

	it('写入坐标与名称，供附近页/侧栏同一口径读取', () => {
		writeLastLocation({ lat: 30.66, lng: 104.06, name: '成都' });
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06, name: '成都' });
	});

	it('名称为空白时不写入该字段（避免显示空串）', () => {
		writeLastLocation({ lat: 30.66, lng: 104.06, name: '   ' });
		expect(readLastLocation()).toEqual({ lat: 30.66, lng: 104.06 });
	});

	it('越界坐标拒绝写入（后端 S2 索引会拒，不应固化到本地）', () => {
		writeLastLocation({ lat: 999, lng: 104.06 });
		expect(localStorage.getItem(KEY)).toBeNull();
	});
});

describe('autoLocateOnce', () => {
	const original = Object.getOwnPropertyDescriptor(navigator, 'geolocation');

	beforeEach(() => {
		localStorage.clear();
		Object.defineProperty(navigator, 'geolocation', { value: undefined, configurable: true });
	});
	afterEach(() => {
		if (original) Object.defineProperty(navigator, 'geolocation', original);
		else Reflect.deleteProperty(navigator, 'geolocation');
	});

	it('成功返回 fix 且不记拒绝', async () => {
		const mock = stubGeolocation(mockFix());
		const fix = await autoLocateOnce();
		expect(fix).toMatchObject({ lat: 39.9, lng: 116.4 });
		expect(mock.getCurrentPosition).toHaveBeenCalledOnce();
		expect(autoLocateDeclined()).toBe(false);
	});

	it('拒绝授权返回 null 并记住不再打扰', async () => {
		stubGeolocation(null, { code: 1 });
		await expect(autoLocateOnce()).resolves.toBeNull();
		expect(autoLocateDeclined()).toBe(true);
	});

	it('超时返回 null 但不记（下次再试）', async () => {
		stubGeolocation(null, { code: 3 });
		await expect(autoLocateOnce()).resolves.toBeNull();
		expect(autoLocateDeclined()).toBe(false);
	});

	it('无定位能力返回 null 并记住', async () => {
		Reflect.deleteProperty(navigator, 'geolocation');
		await expect(autoLocateOnce()).resolves.toBeNull();
		expect(autoLocateDeclined()).toBe(true);
	});

	it('已拒绝过则直接短路，不碰 geolocation', async () => {
		localStorage.setItem('hotspot:locate-denied', '1');
		const mock = stubGeolocation(mockFix());
		await expect(autoLocateOnce()).resolves.toBeNull();
		expect(mock.getCurrentPosition).not.toHaveBeenCalled();
	});
});
