// Cursor Pager Rune Composable — 统一分页加载器（Svelte 5 runes）
//
// 规范：
// 1. 消除页面各自维护 nextCursor, loading, loadingMore, error, seen 去重的重复样板；
// 2. 内置防竞态 token（快速切 tab 或重新加载时废弃未完成的异步响应）；
// 3. 响应式导出（通过 getter 暴露 $state / $derived），便于解构或直接传递。

export interface PageResult<T> {
	items: T[];
	nextCursor: string | null;
}

export interface CursorPagerOptions<T> {
	/** 获取单页数据的异步函数（cursor 为 null 表示首屏） */
	fetchPage: (cursor: string | null) => Promise<PageResult<T>>;
	/** 实体唯一 ID 取值函数（用于去重与局部操作） */
	getKey: (item: T) => string;
	/** 初始条目（可选） */
	initialItems?: T[];
	/** 加载失败的统一错误文案兜底 */
	fallbackErrorMessage?: string;
}

export interface CursorPager<T> {
	readonly items: T[];
	readonly loading: boolean;
	readonly loadingMore: boolean;
	readonly error: string | null;
	readonly loaded: boolean;
	readonly hasMore: boolean;
	readonly nextCursor: string | null;
	load: () => Promise<void>;
	loadMore: () => Promise<void>;
	reset: () => void;
	prepend: (item: T) => void;
	remove: (key: string) => void;
	update: (key: string, updater: (item: T) => T) => void;
}

export function createCursorPagination<T>(options: CursorPagerOptions<T>): CursorPager<T> {
	const { fetchPage, getKey, fallbackErrorMessage = '加载失败' } = options;

	let items = $state<T[]>(options.initialItems ?? []);
	let loading = $state(false);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let nextCursor = $state<string | null>(null);
	let loaded = $state(false);

	// 请求版本号：防止并发竞态（如 reset 后迟到的旧响应回写）
	let requestId = 0;

	async function load(): Promise<void> {
		if (loading) return;
		loading = true;
		error = null;
		const currentReq = ++requestId;

		try {
			const page = await fetchPage(null);
			if (currentReq !== requestId) return;
			items = page.items;
			nextCursor = page.nextCursor;
			loaded = true;
		} catch (e) {
			if (currentReq !== requestId) return;
			error = e instanceof Error ? e.message : fallbackErrorMessage;
		} finally {
			if (currentReq === requestId) {
				loading = false;
			}
		}
	}

	async function loadMore(): Promise<void> {
		if (loadingMore || !nextCursor) return;
		loadingMore = true;
		const currentReq = ++requestId;

		try {
			const page = await fetchPage(nextCursor);
			if (currentReq !== requestId) return;
			const seen = new Set(items.map(getKey));
			const fresh = page.items.filter((item) => !seen.has(getKey(item)));
			items = [...items, ...fresh];
			nextCursor = page.nextCursor;
		} catch (e) {
			if (currentReq !== requestId) return;
			error = e instanceof Error ? e.message : fallbackErrorMessage;
		} finally {
			if (currentReq === requestId) {
				loadingMore = false;
			}
		}
	}

	function reset(): void {
		requestId++;
		items = [];
		nextCursor = null;
		loaded = false;
		error = null;
		loading = false;
		loadingMore = false;
	}

	function prepend(item: T): void {
		const key = getKey(item);
		items = [item, ...items.filter((existing) => getKey(existing) !== key)];
	}

	function remove(key: string): void {
		items = items.filter((existing) => getKey(existing) !== key);
	}

	function update(key: string, updater: (item: T) => T): void {
		items = items.map((existing) => (getKey(existing) === key ? updater(existing) : existing));
	}

	return {
		get items() { return items; },
		get loading() { return loading; },
		get loadingMore() { return loadingMore; },
		get error() { return error; },
		get loaded() { return loaded; },
		get hasMore() { return Boolean(nextCursor); },
		get nextCursor() { return nextCursor; },
		load,
		loadMore,
		reset,
		prepend,
		remove,
		update
	};
}
