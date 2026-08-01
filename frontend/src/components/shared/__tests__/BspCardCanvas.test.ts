// BspCardCanvas 渲染行为：布局格子数可能超过 items 数（fillToFull 补位），
// 补位格必须渲染为空骨架而非重复卡片。
import { describe, it, expect, vi, beforeEach } from 'vitest';
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
