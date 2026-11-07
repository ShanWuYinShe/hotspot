// PageHeader 灵动岛接线契约测试：
//   - 桌面端注册、吸顶页头滚动窗口交接、integrated 页头锚点位置交接、sticky 外包容器、显式关闭。
//   - 交接是滚动驱动（p=0..1）：滚动位置决定交接进度，回到顶部必须完全复原。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createRawSnippet, mount, unmount, tick } from 'svelte';
import PageHeader from '../PageHeader.svelte';
import { initPlatform } from '@/stores/platform.svelte';
import { headerIsland } from '@/stores/header-island.svelte';

const mounted: Array<() => void> = [];
const platformCleanups: Array<() => void> = [];
/** 当前测试的滚动容器；getBoundingClientRect 桩用它换算锚点在视口中的位置。 */
let activeScroller: HTMLElement | null = null;
/** 标题锚点的文档绝对位置（px）：jsdom 无布局，用桩提供稳定坐标。 */
const ANCHOR_ABS_TOP = 100;

function setViewport(w: number, h: number) {
	Object.defineProperty(window, 'innerWidth', { value: w, configurable: true });
	Object.defineProperty(window, 'innerHeight', { value: h, configurable: true });
}

async function pump() {
	await new Promise((r) => setTimeout(r, 10));
	await tick();
}

function createScroller() {
	const el = document.createElement('div');
	el.style.overflowY = 'auto';
	document.body.appendChild(el);
	activeScroller = el;
	return el;
}

beforeEach(() => {
	setViewport(1920, 1080);
	platformCleanups.push(initPlatform());
	activeScroller = null;
	vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function () {
		const scrollTop = activeScroller?.scrollTop ?? 0;
		const top = ANCHOR_ABS_TOP - scrollTop;
		return {
			x: 100,
			y: top,
			top,
			left: 100,
			right: 180,
			bottom: top + 28,
			width: 80,
			height: 28,
			toJSON: () => ({})
		} as DOMRect;
	});
	vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => setTimeout(() => cb(0), 0) as unknown as number);
	vi.stubGlobal('cancelAnimationFrame', (id: number) => clearTimeout(id));
});

afterEach(() => {
	while (mounted.length) mounted.pop()?.();
	platformCleanups.forEach((c) => c());
	platformCleanups.length = 0;
	document.querySelectorAll('body > div').forEach((el) => el.remove());
	setViewport(1024, 768);
	initPlatform()();
	vi.restoreAllMocks();
	vi.unstubAllGlobals();
	activeScroller = null;
});

describe('PageHeader 灵动岛（桌面端，滚动驱动交接）', () => {
	it('吸顶页头：滚动进入交接窗口后岛可见，回到顶部完全复原', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '附近的人', subtitle: '20 人 · 5km 内', onBack: undefined }
		});
		mounted.push(() => void unmount(comp));
		await tick();
		expect(headerIsland.title).toBe('附近的人');
		expect(headerIsland.meta).toBe('20 人 · 5km 内');

		const header = scroller.querySelector('header') as HTMLElement;
		expect(header.dataset.islandVisible).toBeUndefined();

		scroller.scrollTop = 300;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();

		expect(header.dataset.islandVisible).toBe('true');
		expect(headerIsland.visible).toBe(true);
		const titleCol = header.querySelector('[data-island-handoff]') as HTMLElement | null;
		expect(titleCol).toBeTruthy();

		scroller.scrollTop = 0;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(header.dataset.islandVisible).toBeUndefined();
		expect(headerIsland.visible).toBe(false);
	});

	it('吸顶页头：滚动越过阈值后触发灵动岛并隐去页头，回到顶部完全恢复', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, { target: scroller, props: { title: '设置', subtitle: '账号 · 外观' } });
		mounted.push(() => void unmount(comp));
		await tick();
		const header = scroller.querySelector('header') as HTMLElement;

		// 滚动未达阈值（scrollTop=100）：岛不显示、页头不隐去
		scroller.scrollTop = 100;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(headerIsland.visible).toBe(false);
		expect(header.dataset.islandBandHidden).toBeUndefined();

		// 滚动跨过阈值（scrollTop=300）：触发灵动岛落位、页头整条隐去
		scroller.scrollTop = 300;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(headerIsland.visible).toBe(true);
		expect(header.dataset.islandBandHidden).toBe('true');
		expect(header.getAttribute('aria-hidden')).toBe('true');
		// Svelte 以 DOM 属性方式落 inert（浏览器会反射为同名属性；jsdom 不反射，只留属性值）
		expect(header.inert).toBe(true);

		scroller.scrollTop = 0;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(header.dataset.islandBandHidden).toBeUndefined();
		expect(header.getAttribute('aria-hidden')).toBeNull();
	});

	it('带 below 工具行的吸顶页头入岛后整条隐去（沉浸浏览），回顶恢复', async () => {
		const scroller = createScroller();
		const below = createRawSnippet(() => ({ render: () => '<div data-test-below>工具行</div>' }));
		const comp = mount(PageHeader, { target: scroller, props: { title: '搜索', below } });
		mounted.push(() => void unmount(comp));
		await tick();

		scroller.scrollTop = 300;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();

		const header = scroller.querySelector('header') as HTMLElement;
		expect(header.dataset.islandBandHidden).toBe('true');
		expect(header.inert).toBe(true);
		// 工具行仍在 DOM（回顶即恢复），只是随整条页头隐去
		expect(header.querySelector('[data-test-below]')).toBeTruthy();

		scroller.scrollTop = 0;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(header.dataset.islandBandHidden).toBeUndefined();
	});

	it('integrated：标题滚进顶栏时按锚点位置交接（无 IO、无时长动画），回顶复原', async () => {
		const scroller = createScroller();
		const below = createRawSnippet(() => ({ render: () => '<div data-test-below>标签</div>' }));
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '附近的人', subtitle: '筛选列表', variant: 'integrated', below }
		});
		mounted.push(() => void unmount(comp));
		await tick();
		const header = scroller.querySelector('header') as HTMLElement;

		// 标题从 y=100 滚到 y=40：已越过顶栏下沿（52），交接进度过半 → 岛出现
		scroller.scrollTop = 60;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(headerIsland.visible).toBe(true);
		expect(header.dataset.islandVisible).toBe('true');
		// 交接中：标题与 below 都带交接钩子（供动效/测试定位），但不再用 opacity class 硬切
		const handoffs = header.querySelectorAll('[data-island-handoff="true"]');
		expect(handoffs.length).toBe(2);
		expect(header.querySelector('[data-test-below]')?.closest('[data-island-handoff]')).toBeTruthy();

		scroller.scrollTop = 0;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(headerIsland.visible).toBe(false);
		expect(header.querySelectorAll('[data-island-handoff="true"]').length).toBe(0);
	});

	it('integrated 页头被外层 sticky 容器包裹时，改走滚动窗口交接（/notify SplitPageLayout 回归）', async () => {
		const scroller = createScroller();
		const stickyWrap = document.createElement('div');
		stickyWrap.style.position = 'sticky';
		scroller.appendChild(stickyWrap);
		const comp = mount(PageHeader, {
			target: stickyWrap,
			props: { title: '通知', subtitle: '4 条未读', variant: 'integrated' }
		});
		mounted.push(() => void unmount(comp));
		await tick();

		// sticky 包裹时标题恒在视口内：入岛前不应出现
		expect(headerIsland.visible).toBe(false);

		scroller.scrollTop = 300;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();

		const header = stickyWrap.querySelector('header') as HTMLElement;
		expect(header.dataset.islandVisible).toBe('true');
		expect(headerIsland.visible).toBe(true);
		expect(header.querySelector('[data-island-handoff]')).toBeTruthy();
	});

	it('island={false} 时不注册（面板内嵌页头可显式关闭）', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '位置选择', island: false }
		});
		mounted.push(() => void unmount(comp));
		await tick();
		expect(headerIsland.active).toBe(false);
	});

	it('卸载后释放注册（路由切换不残留，交接归零）', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, { target: scroller, props: { title: '设置' } });
		await tick();
		expect(headerIsland.active).toBe(true);
		void unmount(comp);
		await tick();
		expect(headerIsland.active).toBe(false);
		expect(headerIsland.visible).toBe(false);
	});
});
