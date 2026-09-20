import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// 真实位置上报：未登录跳过、网格去重、失败静默（不抛错）、名称兜底与非法坐标拒绝。
// updateLocation 经 vi.mock 桩化，避免真实网络；模块内通过静态 import 绑定 mock。
vi.mock('@/api', () => ({ updateLocation: vi.fn() }));

vi.mock('@/lib/default-location', () => ({ loadDefaultLocation: vi.fn() }));

const { updateLocation } = await import('@/api');
const { loadDefaultLocation } = await import('@/lib/default-location');

const { reportLocation, resetReportedLocation } = await import('./location-report');

const mocked = vi.mocked(updateLocation);
const mockedDefault = vi.mocked(loadDefaultLocation);

describe('reportLocation', () => {
	beforeEach(() => {
		localStorage.clear();
		mocked.mockReset();
		mocked.mockResolvedValue({ code: 0, message: '', data: undefined } as never);
		mockedDefault.mockResolvedValue({ point: { latitude: 0, longitude: 0 }, location_name: '默认位置' } as never);
	});
	afterEach(() => localStorage.clear());

	it('登录用户首次定位即上报，并带上坐标', async () => {
		const sent = await reportLocation({ lat: 30.66, lng: 104.06 }, true);
		expect(sent).toBe(true);
		expect(mocked).toHaveBeenCalledOnce();
		expect(mocked.mock.calls[0][0]).toMatchObject({
			location: { point: { latitude: 30.66, longitude: 104.06 } }
		});
	});

	it('未登录不发请求（该端点需要身份）', async () => {
		const sent = await reportLocation({ lat: 30.66, lng: 104.06 }, false);
		expect(sent).toBe(false);
		expect(mocked).not.toHaveBeenCalled();
	});

	it('同一网格内第二次不再上报（去重）', async () => {
		await reportLocation({ lat: 30.66, lng: 104.06 }, true);
		const again = await reportLocation({ lat: 30.6601, lng: 104.0601 }, true);
		expect(again).toBe(false);
		expect(mocked).toHaveBeenCalledOnce();
	});

	it('跨网格后再次上报（用户移动）', async () => {
		await reportLocation({ lat: 30.66, lng: 104.06 }, true);
		const moved = await reportLocation({ lat: 30.7, lng: 104.1 }, true);
		expect(moved).toBe(true);
		expect(mocked).toHaveBeenCalledTimes(2);
	});

	it('接口返回非 0 时不写去重标记，下次仍会重试', async () => {
		mocked.mockResolvedValueOnce({ code: 10600, message: '服务暂时不可用' } as never);
		const first = await reportLocation({ lat: 30.66, lng: 104.06 }, true);
		expect(first).toBe(false);
		const second = await reportLocation({ lat: 30.66, lng: 104.06 }, true);
		expect(second).toBe(true);
		expect(mocked).toHaveBeenCalledTimes(2);
	});

	it('网络异常静默：不抛错且不写标记', async () => {
		mocked.mockRejectedValueOnce(new Error('network down'));
		await expect(reportLocation({ lat: 30.66, lng: 104.06 }, true)).resolves.toBe(false);
	});

	it('非法/越界坐标不发请求', async () => {
		await expect(reportLocation({ lat: Number.NaN, lng: 104.06 }, true)).resolves.toBe(false);
		await expect(reportLocation({ lat: 999, lng: 104.06 }, true)).resolves.toBe(false);
		expect(mocked).not.toHaveBeenCalled();
	});

	it('带名称时用该名称；无名称回退默认位置名（契约：无名称用 ClientConfig 默认）', async () => {
		await reportLocation({ lat: 30.66, lng: 104.06, name: '成都' }, true);
		expect(mocked.mock.calls[0][0]).toMatchObject({ location: { location_name: '成都' } });

		resetReportedLocation();
		await reportLocation({ lat: 31.0, lng: 105.0 }, true);
		expect(mocked.mock.calls[1][0]).toMatchObject({ location: { location_name: '默认位置' } });
	});

	it('配置也拿不到名称时才用兜底文案', async () => {
		mockedDefault.mockResolvedValueOnce(null as never);
		await reportLocation({ lat: 30.66, lng: 104.06 }, true);
		expect(mocked.mock.calls[0][0]).toMatchObject({ location: { location_name: '未知位置' } });
	});
});
