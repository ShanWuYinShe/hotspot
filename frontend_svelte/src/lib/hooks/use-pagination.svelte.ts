/**
 * 通用分页 Hook
 * 
 * 功能：
 * - 管理分页状态（items, loading, cursor, hasMore）
 * - 统一的数据加载逻辑
 * - 支持刷新和加载更多
 * - 统一的错误处理
 * 
 * 使用示例：
 * ```typescript
 * const pagination = usePagination<UserSearchItem>();
 * 
 * async function loadUsers(refresh = false) {
 *   await pagination.load(
 *     () => usersService.searchByLocation({
 *       reference_point: getAnchor(),
 *       radius: 5000
 *     }),
 *     refresh
 *   );
 * }
 * ```
 */

import { isSuccess } from '$lib/api/client';
import type { ApiResponse, PaginatedData } from '$lib/types/api';

interface UsePaginationOptions<T> {
	/**
	 * 初始加载状态（默认 true）
	 */
	initialLoading?: boolean;
	
	/**
	 * 用于去重的唯一键字段名或获取函数
	 * 如果不提供，则不去重（可能导致 duplicate key 错误）
	 */
	uniqueKey?: keyof T | ((item: T) => string | number);
}

interface PaginationState<T> {
	/**
	 * 数据列表
	 */
	readonly items: T[];
	
	/**
	 * 初始加载中
	 */
	readonly loading: boolean;
	
	/**
	 * 加载更多中
	 */
	readonly loadingMore: boolean;
	
	/**
	 * 是否有更多数据
	 */
	readonly hasMore: boolean;
	
	/**
	 * 错误信息
	 */
	readonly error: string | null;
	
	/**
	 * 加载数据
	 * @param fetcher 数据获取函数
	 * @param refresh 是否刷新（清空现有数据）
	 */
	load: (
		fetcher: (cursor?: string) => Promise<ApiResponse<PaginatedData<T>>>,
		refresh?: boolean
	) => Promise<void>;
	
	/**
	 * 重置状态
	 */
	reset: () => void;
	
	/**
	 * 设置错误
	 */
	setError: (error: string | null) => void;
	
	/**
	 * 添加项目（在列表开头）
	 */
	addItem: (item: T) => void;
	
	/**
	 * 更新项目（根据唯一键）
	 */
	updateItem: (updater: (item: T) => T | null) => void;
	
	/**
	 * 删除项目（根据唯一键）
	 */
	removeItem: (predicate: (item: T) => boolean) => void;
	
	/**
	 * 直接设置 items（用于复杂操作）
	 */
	setItems: (newItems: T[]) => void;
}

/**
 * 创建分页状态管理
 */
export function usePagination<T>(options?: UsePaginationOptions<T>): PaginationState<T> {
	let items = $state.raw<T[]>([]);
	let loading = $state(options?.initialLoading ?? true);
	let loadingMore = $state(false);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(true);
	let error = $state<string | null>(null);
	let isLoadingRequest = false; // 防止并发调用
	let lastLoadTime = 0; // 记录上次加载时间
	const MIN_LOAD_INTERVAL = 100; // 最小加载间隔（毫秒）
	
	async function load(
		fetcher: (cursor?: string) => Promise<ApiResponse<PaginatedData<T>>>,
		refresh = false
	) {
		// 🔒 防止并发调用：如果正在加载，直接返回
		if (isLoadingRequest) {
			return;
		}
		
		// 🕐 防抖：如果距离上次加载时间太短，直接返回
		const now = Date.now();
		if (!refresh && now - lastLoadTime < MIN_LOAD_INTERVAL) {
			return;
		}
		
		isLoadingRequest = true;
		lastLoadTime = now;
		
		// 刷新模式：重置状态
		if (refresh) {
			// Don't clear items immediately to avoid flash, unless you want skeleton
			// But current logic clears it. Let's keep it for now or maybe optimize later.
			// The user didn't complain about flash, but about duplicates.
			// Actually, clearing items causes layout shift. 
			// But let's stick to fixing duplicates first.
			items = [];
			cursor = undefined;
			hasMore = true;
			loading = true;
			error = null;
		} else {
			// 加载更多模式
			loadingMore = true;
		}

		try {
			const response = await fetcher(refresh ? undefined : cursor);

			if (isSuccess(response) && response.data) {
				// 更新数据
				const newItems = response.data.items;
				
				if (refresh) {
					items = newItems;
				} else {
					// 去重逻辑
					if (options?.uniqueKey) {
						const getKey = (item: T) => 
							typeof options.uniqueKey === 'function' 
								? (options.uniqueKey as Function)(item) 
								: item[options.uniqueKey as keyof T];
								
						const existingKeys = new Set(items.map(getKey));
						const uniqueNewItems = newItems.filter(item => !existingKeys.has(getKey(item)));
						items = [...items, ...uniqueNewItems];
					} else {
						items = [...items, ...newItems];
					}
				}
				
				// 更新分页状态
				hasMore = response.data.pagination?.has_more ?? false;
				cursor = response.data.pagination?.next_cursor;
				
				// 清除错误
				error = null;
			} else {
				// 设置错误信息
				error = response.message || '加载失败';
			}
		} catch (e) {
			// 异常处理
			error = e instanceof Error ? e.message : '加载失败';
		} finally {
			loading = false;
			loadingMore = false;
			isLoadingRequest = false; // 🔓 释放锁
		}
	}

	function reset() {
		items = [];
		cursor = undefined;
		hasMore = true;
		loading = options?.initialLoading ?? true;
		error = null;
	}

	function setError(newError: string | null) {
		error = newError;
	}

	function addItem(item: T) {
		items = [item, ...items];
	}

	function updateItem(updater: (item: T) => T | null) {
		items = items.map((item) => updater(item) ?? item);
	}

	function removeItem(predicate: (item: T) => boolean) {
		items = items.filter((item) => !predicate(item));
	}

	function setItems(newItems: T[]) {
		items = newItems;
	}

	return {
		get items() { return items; },
		get loading() { return loading; },
		get loadingMore() { return loadingMore; },
		get hasMore() { return hasMore; },
		get error() { return error; },
		load,
		reset,
		setError,
		addItem,
		updateItem,
		removeItem,
		setItems
	};
}
