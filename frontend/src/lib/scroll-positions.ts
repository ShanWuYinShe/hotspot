// scroll-positions — 路由级滚动位置存取（RoutedContentViewport 的无 DOM 逻辑层）。
// SvelteKit 默认只恢复 window 滚动，本项目滚动 owner 是自定义 overflow 容器
//（RoutedContentViewport 的 <main>），故按 pathname 自管：离开存位，返回恢复，
// 进新页置顶，同路由内导航（如 ?tab= 切换）不动。

export type RestoreDecision =
	| { action: 'restore'; top: number }
	| { action: 'top' }
	| { action: 'keep' };

const MAX_ENTRIES = 30;

export function savePosition(store: Map<string, number>, pathname: string, top: number): void {
	// delete-then-set 刷新 recency（Map 按插入序淘汰最旧）。
	if (store.has(pathname)) store.delete(pathname);
	store.set(pathname, top);
	if (store.size > MAX_ENTRIES) {
		const oldest = store.keys().next();
		if (!oldest.done) store.delete(oldest.value);
	}
}

/** afterNavigate 后的滚动决策：popstate 按存位恢复（无存位则置顶）；跨 pathname 新进置顶；同路由保持不动。 */
export function decideScroll(
	navigationType: string,
	fromPathname: string | null,
	toPathname: string | null,
	store: Map<string, number>
): RestoreDecision {
	if (navigationType === 'popstate') {
		const saved = toPathname == null ? undefined : store.get(toPathname);
		return { action: 'restore', top: saved ?? 0 };
	}
	if (fromPathname !== toPathname) return { action: 'top' };
	return { action: 'keep' };
}

/** 可滚动元素的最小结构（组件传真 DOM，单测传桩）。 */
export interface ScrollableElement {
	scrollTop: number;
	readonly scrollHeight: number;
	readonly clientHeight: number;
	style: { scrollBehavior: string };
	offsetHeight: number;
}

/**
 * 追到为止的位置恢复：图片/瀑布流布局异步长高时，单次赋值会被钳位到当时
 * 的最大高度。每帧重设，直到到达目标、高度连续 5 帧不动，或 40 帧上限。
 * style 恢复调用方原值（覆盖 scroll-smooth 的平滑漂移）。
 */
export function restoreScrollTop(
	el: ScrollableElement | null,
	top: number,
	schedule: (cb: () => void) => void = (cb) => requestAnimationFrame(cb)
): void {
	if (!el) return;
	const prevBehavior = el.style.scrollBehavior;
	el.style.scrollBehavior = 'auto';
	if (top <= 0) {
		el.scrollTop = 0;
		void el.offsetHeight;
		el.style.scrollBehavior = prevBehavior;
		return;
	}
	let attempts = 0;
	let stableHeightCount = 0;
	let lastHeight = -1;
	const tick = () => {
		el.scrollTop = top;
		attempts++;
		if (Math.abs(el.scrollTop - top) <= 2 || attempts >= 40) {
			el.style.scrollBehavior = prevBehavior;
			return;
		}
		if (el.scrollHeight === lastHeight) stableHeightCount++;
		else {
			stableHeightCount = 0;
			lastHeight = el.scrollHeight;
		}
		if (stableHeightCount >= 5) {
			el.style.scrollBehavior = prevBehavior;
			return;
		}
		schedule(tick);
	};
	schedule(tick);
}
