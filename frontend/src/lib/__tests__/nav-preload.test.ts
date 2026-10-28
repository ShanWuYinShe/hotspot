// 导航意图预取契约（R16）：去重、失败静默、当前路径跳过、事件目标解析、空闲兜底。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { preloadCode } from '$app/navigation';
import {
	normalizeNavPath,
	preloadRoute,
	resetPreloadedRoutes,
	resolvePreloadTarget,
	scheduleIdlePreload
} from '../nav-preload';

vi.mock('$app/navigation', () => ({ preloadCode: vi.fn(() => Promise.resolve()) }));

const mockedPreload = vi.mocked(preloadCode);

beforeEach(() => {
	resetPreloadedRoutes();
	mockedPreload.mockReset();
	mockedPreload.mockImplementation(() => Promise.resolve());
});

afterEach(() => {
	vi.useRealTimers();
});

describe('normalizeNavPath', () => {
	it('去掉 query/hash；相对路径与非字符串返回 null', () => {
		expect(normalizeNavPath('/notify')).toBe('/notify');
		expect(normalizeNavPath('/me/relationships?tab=following#x')).toBe('/me/relationships');
		expect(normalizeNavPath('notify')).toBeNull();
		expect(normalizeNavPath('')).toBeNull();
		expect(normalizeNavPath(null)).toBeNull();
	});
});

describe('preloadRoute', () => {
	it('同一路径并发/重复意图只预取一次（Set 去重）', async () => {
		await Promise.all([preloadRoute('/notify'), preloadRoute('/notify?from=home'), preloadRoute('/notify')]);
		expect(mockedPreload).toHaveBeenCalledTimes(1);
		expect(mockedPreload).toHaveBeenCalledWith('/notify');
	});

	it('预取失败静默：不 reject，也不重复尝试', async () => {
		mockedPreload.mockRejectedValueOnce(new Error('chunk load failed'));
		await expect(preloadRoute('/search')).resolves.toBeUndefined();
		await preloadRoute('/search');
		expect(mockedPreload).toHaveBeenCalledTimes(1);
	});

	it('当前所在路径不预取', async () => {
		// jsdom 默认 location.pathname === '/'
		await preloadRoute('/');
		expect(mockedPreload).not.toHaveBeenCalled();
	});
});

describe('resolvePreloadTarget（事件委托 → 路径映射）', () => {
	const labels = new Map<string, string>([
		['通知', '/notify'],
		['关系', '/me/relationships'],
		['位置选择', '/map']
	]);

	it('命中按钮 aria-label 时返回路径（含内部子元素事件目标）', () => {
		const button = document.createElement('button');
		button.setAttribute('aria-label', '通知');
		const icon = document.createElement('span');
		button.appendChild(icon);
		expect(resolvePreloadTarget(icon, labels)).toBe('/notify');
	});

	it('非导航区域（无映射标签）返回 null', () => {
		const div = document.createElement('div');
		div.textContent = '随便的文字';
		document.body.appendChild(div);
		expect(resolvePreloadTarget(div, labels)).toBeNull();
		div.remove();
	});

	it('title 兜底：只有 title 的导航按钮同样可解析', () => {
		const button = document.createElement('button');
		button.setAttribute('title', '位置选择');
		expect(resolvePreloadTarget(button, labels)).toBe('/map');
	});
});

describe('scheduleIdlePreload（requestIdleCallback 兜底）', () => {
	it('延迟后批量预取常用路由；取消函数生效', async () => {
		vi.useFakeTimers();
		const stop = scheduleIdlePreload(['/messages', '/notify'], { delayMs: 5000 });
		await vi.advanceTimersByTimeAsync(4999);
		expect(mockedPreload).not.toHaveBeenCalled();
		await vi.advanceTimersByTimeAsync(2);
		// jsdom 无 requestIdleCallback → 直接走 run()
		expect(mockedPreload.mock.calls.map((c) => c[0])).toEqual(['/messages', '/notify']);
		stop();
	});

	it('取消后不再预取', async () => {
		vi.useFakeTimers();
		const stop = scheduleIdlePreload(['/messages'], { delayMs: 5000 });
		stop();
		await vi.advanceTimersByTimeAsync(6000);
		expect(mockedPreload).not.toHaveBeenCalled();
	});
});
