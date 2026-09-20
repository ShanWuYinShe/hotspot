import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 地图首屏定位 helper：锚点解析、拒绝记忆、自动定位一次的三态（成功/拒绝记/临时失败不记）。
// navigator.geolocation 经 defineProperty 桩化；拒绝/超时错误需自带实例常量
// （PERMISSION_DENIED 等），否则 mapNativeError 的 switch 走 default 分支。
const { autoLocateDeclined, autoLocateOnce, readLastLocation } = await import('./map-auto-locate');

const CODES = { PERMISSION_DENIED: 1, POSITION_UNAVAILABLE: 2, TIMEOUT: 3 };

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

	it('合法 [lat, lng] 解析为锚点', () => {
		localStorage.setItem('hotspot_last_location', '[39.9,116.4]');
		expect(readLastLocation()).toEqual({ lat: 39.9, lng: 116.4 });
	});

	it.each([['缺失', null], ['非法 JSON', 'xxx'], ['非数组', '{"a":1}'], ['非数字', '["a","b"]'], ['越界', '[91,116.4]']])(
		'%s 时返回 null',
		(_label, raw) => {
			if (raw === null) localStorage.removeItem('hotspot_last_location');
			else localStorage.setItem('hotspot_last_location', raw);
			expect(readLastLocation()).toBeNull();
		}
	);
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
