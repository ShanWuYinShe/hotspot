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
