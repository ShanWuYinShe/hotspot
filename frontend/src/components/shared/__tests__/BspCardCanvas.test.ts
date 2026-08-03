// BspCardCanvas 渲染行为：布局格子数可能超过 items 数（fillToFull 补位），
// 补位格必须渲染为空骨架而非重复卡片。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import BspHost from './BspHost.svelte';

class MockResizeObserver {
	constructor(_cb: ResizeObserverCallback) {}
	observe() {}
	unobserve() {}
	disconnect() {}
}
class MockIntersectionObserver {
	constructor(private cb: IntersectionObserverCallback) {}
	observe(el: Element) {
		// 立即标记可见（模拟所有槽位在视口内），让虚拟化不拦截渲染
		this.cb([{ target: el, isIntersecting: true }] as unknown as IntersectionObserverEntry[], this as unknown as IntersectionObserver);
	}
	unobserve() {}
	disconnect() {}
}

beforeEach(() => {
	vi.stubGlobal('ResizeObserver', MockResizeObserver);
	vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
	vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
		width: 1044,
		height: 0,
		top: 0,
		left: 0,
		right: 1044,
		bottom: 0,
		x: 0,
		y: 0,
		toJSON: () => ({})
	} as DOMRect);
});

describe('BspCardCanvas', () => {
	it('布局恰好由 n 张卡铺满网格：无重复卡片、无补位骨架', async () => {
		// 5 个用户：plan 给出 5 张卡恰好铺满的布局（无 fillToFull 补位）
		const items = Array.from({ length: 5 }, (_, i) => ({ name: `用户${i + 1}` }));
		const { container } = render(BspHost, { props: { items } });

		const cards = await screen.findAllByText(/^用户\d+$/);
		expect(cards).toHaveLength(5);
		// 无重复：每个用户恰好出现一次
		for (let i = 1; i <= 5; i++) {
			expect(screen.getAllByText(`用户${i}`)).toHaveLength(1);
		}
		// 满铺：没有补位骨架
		expect(container.querySelectorAll('div[aria-hidden="true"]').length).toBe(0);
	});

	it('items 数变化时同样无重复卡片', async () => {
		const items = Array.from({ length: 8 }, (_, i) => ({ name: `用户${i + 1}` }));
		const { container } = render(BspHost, { props: { items } });

		const cards = await screen.findAllByText(/^用户\d+$/);
		expect(cards.length).toBeGreaterThanOrEqual(8);
		for (let i = 1; i <= 8; i++) {
			expect(screen.getAllByText(`用户${i}`)).toHaveLength(1);
		}
		// 即便存在补位骨架（兜底路径），也绝不能是重复的用户卡
		const placeholders = container.querySelectorAll('div[aria-hidden="true"]');
		placeholders.forEach((p) => expect(p.textContent).toBe(''));
	});
});

describe('BspCardCanvas 容器自适应', () => {
	afterEach(() => {
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it('容器宽度变化时 1×1 卡片尺寸随之变化，且卡片总宽与容器一致（完整填充）', async () => {
		// 可触发回调的 ResizeObserver：模拟容器 resize 事件
		let resizeCb: ResizeObserverCallback | null = null;
		class TriggerableResizeObserver {
			constructor(cb: ResizeObserverCallback) {
				resizeCb = cb;
			}
			observe() {}
			unobserve() {}
			disconnect() {}
		}
		vi.stubGlobal('ResizeObserver', TriggerableResizeObserver);

		let containerWidth = 1044;
		// BspCardCanvas 只对容器 section 调 getBoundingClientRect（通过 bind:this 的 containerEl）；
		// 其余元素返回 0 以模拟 jsdom 无布局（避免虚高导致列数失真）。
		vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
			const w = this.tagName === 'SECTION' ? containerWidth : 0;
			return { width: w, height: 0, top: 0, left: 0, right: w, bottom: 0, x: 0, y: 0, toJSON: () => ({}) } as DOMRect;
		});

		const items = Array.from({ length: 12 }, (_, i) => ({ name: `用户${i + 1}` }));
		const { container } = render(BspHost, { props: { items } });

		// 初始宽度 1044 的布局
		await screen.findAllByText(/^用户\d+$/);
		const bsp = container.querySelector('section[aria-label="测试布局"]') as HTMLElement;
		expect(bsp).not.toBeNull();

		const widthOf = (el: Element) => Math.round(parseFloat((el as HTMLElement).style.width));
		const cardAt = (i: number) => bsp.querySelectorAll(':scope > div')[i] as HTMLElement;
		const firstWide = widthOf(cardAt(0));
		const firstCanvasH = Math.round(parseFloat(bsp.style.height));

		// 缩窄容器到 700px → 触发 ResizeObserver
		containerWidth = 700;
		resizeCb!([], {} as ResizeObserver);

		// 重排后：1×1 单元 = 容器宽 / 列数，卡片变小并重新铺满
		await new Promise((r) => setTimeout(r, 50));
		const secondWide = widthOf(cardAt(0));
		const secondCanvasH = Math.round(parseFloat(bsp.style.height));

		// 容器变窄 → 卡片宽变小（1×1 单元随容器缩放）
		expect(secondWide).toBeLessThan(firstWide);

		// 完整填充：布局几何 = 卡片绝对坐标（unit = 容器宽/列数，inset = gap/2），
		// 因此最右卡片右边缘恒为 canvasWidth - gap/2（与卡片形状无关）。
		const gap = 24;
		const rightEdges = [...bsp.querySelectorAll(':scope > div')]
			.filter((d) => (d as HTMLElement).style.width)
			.map((d) => Math.round(parseFloat((d as HTMLElement).style.left) + parseFloat((d as HTMLElement).style.width)));
		expect(Math.max(...rightEdges)).toBe(containerWidth - gap / 2);

		// 布局保持满铺：真实卡片数不变，高度重新计算
		const visibleCards = [...bsp.querySelectorAll(':scope > div')].filter((d) => d.style.width && (d as HTMLElement).style.height !== '');
		expect(visibleCards.length).toBeGreaterThanOrEqual(12);
		expect(secondCanvasH).toBeGreaterThan(0);
		void firstCanvasH;
	});
});
