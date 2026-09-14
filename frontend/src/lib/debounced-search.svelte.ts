/**
 * 搜索输入防抖共享原语（地图桌面/移动端地理编码搜索与搜索页话题联想共用）。
 *
 * 三处原先各持一份「effect + setTimeout 防抖 + seq/cancelled 竞态防护」的复制实现，
 * 防抖时长还不一致（地图 400ms / 搜索页 300ms），现统一为 SEARCH_DEBOUNCE_MS。
 * 组件在初始化时调用一次 createDebouncedSearch，内部 effect 负责：
 *   查询为空 → 立即 clear 并短路；
 *   非空     → 防抖到期后发起请求，seq 保证过期响应一律丢弃；
 *   cleanup  → 清除定时器并 abort 进行中的请求。
 */
export const SEARCH_DEBOUNCE_MS = 300;

export interface DebouncedSearchOptions<T> {
	/** 响应式查询词 getter（effect 依赖；是否 trim 由调用方决定）。 */
	query: () => string;
	/** 防抖到期后的请求；signal 供支持中止的请求传入，可安全读取最新外部快照。 */
	fetch: (query: string, signal: AbortSignal) => Promise<T>;
	/** 请求成功且未被更新请求淘汰时应用结果。 */
	apply: (data: T) => void;
	/** 查询为空时立即清空外部状态。 */
	clear: () => void;
	/** 请求失败时回调（未提供则静默）；过期响应不会进入本回调。 */
	onError?: () => void;
	/** 额外响应式依赖（如地图源切换），变化时重新防抖。 */
	deps?: () => void;
}

export function createDebouncedSearch<T>(options: DebouncedSearchOptions<T>): void {
	// seq 竞态防护：只有最新一次触发的响应允许落地
	let seq = 0;
	$effect(() => {
		const query = options.query();
		if (options.deps) options.deps();
		if (!query) {
			options.clear();
			return;
		}
		const current = ++seq;
		const controller = new AbortController();
		const timer = setTimeout(() => {
			options
				.fetch(query, controller.signal)
				.then((data) => {
					if (current === seq) options.apply(data);
				})
				.catch(() => {
					if (current === seq) options.onError?.();
				});
		}, SEARCH_DEBOUNCE_MS);
		return () => {
			clearTimeout(timer);
			controller.abort();
		};
	});
}
