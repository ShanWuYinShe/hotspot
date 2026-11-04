import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import InfiniteSentinel from '../InfiniteSentinel.svelte';

// InfiniteSentinel 契约（docs/ui-components.md）：
//   - 哨兵进入视口且空闲（!loading / !error / hasMore）→ 触发 onLoadMore
//   - loading / error / hasMore=false 时不触发
//   - 自动补屏：加载完成后哨兵仍可见 → 继续触发下一页
//   - 卸载时断开观察、清掉待触发定时器

class MockIntersectionObserver {
	static instances: MockIntersectionObserver[] = [];
	/** 模拟浏览器「开始观察时立即上报当前相交状态」：观察后异步上报 intersecting */
	static autoReportOnObserve = false;
	cb: IntersectionObserverCallback;
	els: Element[] = [];
	constructor(cb: IntersectionObserverCallback) {
		this.cb = cb;
		MockIntersectionObserver.instances.push(this);
	}
	observe(el: Element) {
		this.els.push(el);
		if (MockIntersectionObserver.autoReportOnObserve) {
			setTimeout(() => this.trigger(true), 0);
		}
	}
	unobserve() {}
	disconnect() {}
	takeRecords() {
		return [];
	}
	trigger(isIntersecting: boolean) {
		this.cb(
			this.els.map((target) => ({ isIntersecting, target }) as IntersectionObserverEntry),
			this as unknown as IntersectionObserver
		);
	}
}

/** 等待组件内 setTimeout(0) 的触发落地（真实定时器，避免与 Svelte 调度器互相干扰） */
const settle = () => new Promise((resolve) => setTimeout(resolve, 60));

describe('InfiniteSentinel（滚到底自动续载）', () => {
	beforeEach(() => {
		MockIntersectionObserver.instances = [];
		MockIntersectionObserver.autoReportOnObserve = false;
		vi.stubGlobal('IntersectionObserver', MockIntersectionObserver);
	});

	it('进入视口且空闲时触发加载', async () => {
		const onLoadMore = vi.fn();
		render(InfiniteSentinel, { props: { hasMore: true, onLoadMore } });
		const io = MockIntersectionObserver.instances[0];
		expect(io).toBeDefined();

		io.trigger(true);
		await settle();
		expect(onLoadMore).toHaveBeenCalledTimes(1);
	});

	it('loading / error / hasMore=false 时不触发', async () => {
		const onLoadMore = vi.fn();
		render(InfiniteSentinel, { props: { hasMore: true, loading: true, onLoadMore } });
		MockIntersectionObserver.instances[0].trigger(true);
		await settle();
		expect(onLoadMore).not.toHaveBeenCalled();

		const errorCase = vi.fn();
		render(InfiniteSentinel, { props: { hasMore: true, error: true, onLoadMore: errorCase } });
		MockIntersectionObserver.instances[1].trigger(true);
		await settle();
		expect(errorCase).not.toHaveBeenCalled();

		const noMore = vi.fn();
		render(InfiniteSentinel, { props: { hasMore: false, onLoadMore: noMore } });
		// hasMore=false 时不注册观察者
		expect(MockIntersectionObserver.instances).toHaveLength(2);
		await settle();
		expect(noMore).not.toHaveBeenCalled();
	});

	it('加载完成后哨兵仍可见时自动补载下一页（首屏不足一屏）', async () => {
		const onLoadMore = vi.fn();
		const { rerender } = render(InfiniteSentinel, {
			props: { hasMore: true, loading: false, onLoadMore }
		});
		MockIntersectionObserver.instances[0].trigger(true);
		await settle();
		expect(onLoadMore).toHaveBeenCalledTimes(1);

		// 页面进入 loading → 结束后（loading=false）应继续拉下一页。
		// 真实浏览器里重新 observe 会立刻上报当前相交状态（哨兵仍在视口），mock 同步模拟该行为。
		MockIntersectionObserver.autoReportOnObserve = true;
		await rerender({ hasMore: true, loading: true, onLoadMore });
		await rerender({ hasMore: true, loading: false, onLoadMore });
		await settle();
		expect(onLoadMore).toHaveBeenCalledTimes(2);
	});
});
