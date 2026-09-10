// 平台管理列表页共享的分页加载器（cursor 翻页 + 筛选变化自动重载第一页）。
// deps() 的返回值参与 $effect 依赖：筛选条件变化时自动重新拉取并清空列表。

import type { PaginatedData } from '@/api/types';
import type { ApiResponse } from '@/api';
import { netError } from '@/api';

export function createAdminList<T>(
	fetchPage: (cursor: string | undefined, limit: number) => Promise<ApiResponse<PaginatedData<T>>>,
	deps: () => unknown,
	limit = 20
) {
	let items = $state<T[]>([]);
	let nextCursor = $state<string | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);

	$effect(() => {
		void deps();
		void reloadKey;
		let cancelled = false;
		async function run() {
			loading = true;
			error = null;
			nextCursor = null;
			const res = await fetchPage(undefined, limit).catch(() => netError<PaginatedData<T>>({ items: [], next_cursor: null }));
			if (cancelled) return;
			if (res.code === 0 && res.data) {
				items = res.data.items ?? [];
				nextCursor = res.data.next_cursor ?? null;
			} else {
				items = [];
				error = res.message ?? '加载失败';
			}
			loading = false;
		}
		void run();
		return () => {
			cancelled = true;
		};
	});

	async function loadMore(): Promise<void> {
		if (loadingMore || !nextCursor) return;
		loadingMore = true;
		try {
			const res = await fetchPage(nextCursor ?? undefined, limit);
			if (res.code === 0 && res.data) {
				items = [...items, ...(res.data.items ?? [])];
				nextCursor = res.data.next_cursor ?? null;
			}
		} finally {
			loadingMore = false;
		}
	}

	return {
		get items() { return items; },
		get loading() { return loading; },
		get loadingMore() { return loadingMore; },
		get error() { return error; },
		get hasMore() { return nextCursor !== null; },
		get count() { return items.length; },
		reload: () => { reloadKey++; },
		loadMore: () => void loadMore()
	};
}
