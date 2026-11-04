// PageHeader 灵动岛接线契约测试：桌面端注册、吸顶标题交接、非吸顶离场触发、显式关闭。
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createRawSnippet, mount, unmount, tick } from 'svelte';
import PageHeader from '../PageHeader.svelte';
import { initPlatform } from '@/stores/platform.svelte';
import { headerIsland } from '@/stores/header-island.svelte';

// jsdom 无 IntersectionObserver：用可控 mock 驱动「标题离场」触发。
class MockIntersectionObserver {
	static instances: MockIntersectionObserver[] = [];
	cb: IntersectionObserverCallback;
	els: Element[] = [];
	constructor(cb: IntersectionObserverCallback) {
		this.cb = cb;
		MockIntersectionObserver.instances.push(this);
	}
	observe(el: Element) {
		this.els.push(el);
	}
	unobserve() {}
	disconnect() {}
	takeRecords() {
		return [];
	}
	trigger(isIntersecting: boolean) {
		this.cb(this.els.map((target) => ({ isIntersecting, target }) as IntersectionObserverEntry), this as unknown as IntersectionObserver);
	}
}

const mounted: Array<() => void> = [];
const platformCleanups: Array<() => void> = [];

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
	return el;
}

beforeEach(() => {
	setViewport(1920, 1080);
	platformCleanups.push(initPlatform());
	MockIntersectionObserver.instances = [];
	vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
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
	vi.unstubAllGlobals();
});

describe('PageHeader 灵动岛（桌面端）', () => {
	it('吸顶页头：滚动 >120 标题交接给岛，回到顶部恢复', async () => {
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

	it('吸顶页头入岛后整条隐去（不留内容区残留标题栏），回顶恢复', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '设置', subtitle: '账号 · 外观 · 地图 · 通知' }
		});
		mounted.push(() => void unmount(comp));
		await tick();

		const header = scroller.querySelector('header') as HTMLElement;
		expect(header.dataset.islandBandHidden).toBeUndefined();

		scroller.scrollTop = 300;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();

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
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '搜索', below }
		});
		mounted.push(() => void unmount(comp));
		await tick();

		scroller.scrollTop = 300;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();

		const header = scroller.querySelector('header') as HTMLElement;
		// 用户契约：入岛后整块页头（含下方筛选/工具栏）一起退场，内容区保持干净
		expect(header.dataset.islandBandHidden).toBe('true');
		expect(header.inert).toBe(true);
		// 工具行仍在 DOM（回顶即恢复），只是随整条页头隐去
		expect(header.querySelector('[data-test-below]')).toBeTruthy();

		scroller.scrollTop = 0;
		scroller.dispatchEvent(new Event('scroll'));
		await pump();
		expect(header.dataset.islandBandHidden).toBeUndefined();
	});

	it('integrated：入岛时标题与 below 筛选行一起交接（不留半截标签）', async () => {
		const scroller = createScroller();
		const below = createRawSnippet(() => ({ render: () => '<div data-test-below>标签</div>' }));
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '附近的人', subtitle: '筛选列表', variant: 'integrated', below }
		});
		mounted.push(() => void unmount(comp));
		await tick();

		const header = scroller.querySelector('header') as HTMLElement;
		const io = MockIntersectionObserver.instances[0];
		io.trigger(false);
		await tick();

		expect(headerIsland.visible).toBe(true);
		// below 行同步进入交接态（淡出 + inert），避免岛已出现、筛选胶囊还挂在内容区
		const handoffs = header.querySelectorAll('[data-island-handoff="true"]');
		expect(handoffs.length).toBe(2);
		expect(header.querySelector('[data-test-below]')?.closest('[data-island-handoff]')).toBeTruthy();
		expect((header.querySelector('[data-test-below]')?.closest('[data-island-handoff]') as HTMLElement).inert).toBe(true);

		io.trigger(true);
		await tick();
		expect(headerIsland.visible).toBe(false);
		expect(header.querySelectorAll('[data-island-handoff="true"]').length).toBe(0);
	});

	it('非吸顶页头：标题离开视口后入岛，回到视口退出', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, {
			target: scroller,
			props: { title: '附近的人', subtitle: '筛选列表', variant: 'integrated' }
		});
		mounted.push(() => void unmount(comp));
		await tick();

		const io = MockIntersectionObserver.instances[0];
		expect(io).toBeTruthy();
		io.trigger(false);
		await tick();
		expect(headerIsland.visible).toBe(true);

		io.trigger(true);
		await tick();
		expect(headerIsland.visible).toBe(false);
	});

	it('integrated 页头被外层 sticky 容器包裹时，改走滚动阈值交接（/notify SplitPageLayout 回归）', async () => {
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

		// sticky 包裹时不做 IO（标题恒在视口内），入岛前不应出现
		expect(MockIntersectionObserver.instances.length).toBe(0);
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

	it('卸载后释放注册（路由切换不残留）', async () => {
		const scroller = createScroller();
		const comp = mount(PageHeader, { target: scroller, props: { title: '设置' } });
		await tick();
		expect(headerIsland.active).toBe(true);
		void unmount(comp);
		await tick();
		expect(headerIsland.active).toBe(false);
	});
});
