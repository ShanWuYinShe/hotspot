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
 * const pagination = usePagination<UserItem>();
 *
 * async function loadUsers(refresh = false) {
 *   await pagination.load(
 *     (cursor) => usersService.search({
 *       reference_point: getAnchor(),
 *       radius: 5000,
 *       pagination: { limit: 20, cursor }
 *     }),
 *     refresh
 *   );
 * }
 * ```
 */

	import { isSuccess } from '$lib/api/client';
	import type { ApiResponse, PaginatedData } from '$lib/types/api';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import type { RequestOptions } from '$lib/api/client';
	import { t } from '$lib/i18n';

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

  /**
   * 加载失败回调（替代调用方监听 error 字段写 $effect）
   * 每次加载失败时触发一次，参数为错误信息
   */
  onError?: (error: string) => void;
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
	   * @param fetcher 数据获取函数 (cursor, opts?) => Promise
	   * @param refresh 是否刷新（清空现有数据）
	   */
	  load: (
	    fetcher: (cursor?: string, opts?: { signal?: AbortSignal }) => Promise<ApiResponse<PaginatedData<T>>>,
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

	// 竞态保护：仅最新请求的结果会被写入状态
	let loadVersion = 0;
	let isLoadingMore = false; // 仅限制"加载更多"并发
	let currentAbortController: AbortController | undefined;

	async function load(
	  fetcher: (cursor?: string, opts?: { signal?: AbortSignal }) => Promise<ApiResponse<PaginatedData<T>>>,
	  refresh = false
	) {
	  // 取消上一轮未完成的请求
	  currentAbortController?.abort();
	  currentAbortController = new AbortController();
	  const signal = currentAbortController.signal;

	  if (!refresh) {
	    // 加载更多：防止并发（连续点击"加载更多"按钮）
	    if (isLoadingMore) return;
	    isLoadingMore = true;
	  }

	  // 刷新：递增版本号，后续用于丢弃过期响应
	  const version = ++loadVersion;

	  if (refresh) {
	    items = [];
	    cursor = undefined;
	    hasMore = true;
	    loading = true;
	    error = null;
	  } else {
	    loadingMore = true;
	  }

	  try {
	    const response = refresh
	      ? await withMinimumLoadingTime(() => fetcher(undefined, { signal }))
	      : await fetcher(cursor, { signal });

	    // 竞态检查：丢弃非最新请求的结果
	    if (version !== loadVersion) return;

      if (isSuccess(response) && response.data) {
        const newItems = response.data.items ?? [];

        if (refresh) {
          items = newItems;
        } else {
          // 去重逻辑
          if (options?.uniqueKey) {
            const uniqueKeyOpt = options.uniqueKey;
            const getKey = (item: T): unknown =>
              typeof uniqueKeyOpt === 'function'
                ? uniqueKeyOpt(item)
                : item[uniqueKeyOpt as keyof T];

            const existingKeys = new Set(items.map(getKey));
            const uniqueNewItems = newItems.filter((item) => !existingKeys.has(getKey(item)));
            items = [...items, ...uniqueNewItems];
          } else {
            items = [...items, ...newItems];
          }
        }

        hasMore = !!response.data.next_cursor;
        cursor = response.data.next_cursor;
        error = null;
      } else {
        error = response.message || t('errors.loadFailed');
        options?.onError?.(error);
      }
    } catch (e) {
      // 竞态检查
      if (version !== loadVersion) return;
      error = e instanceof Error ? e.message : t('errors.loadFailed');
      options?.onError?.(error);
    } finally {
      // 仅最新请求负责重置 loading 状态
      if (version === loadVersion) {
        loading = false;
        loadingMore = false;
      }
      if (!refresh) isLoadingMore = false;
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
    get items() {
      return items;
    },
    get loading() {
      return loading;
    },
    get loadingMore() {
      return loadingMore;
    },
    get hasMore() {
      return hasMore;
    },
    get error() {
      return error;
    },
    load,
    reset,
    setError,
    addItem,
    updateItem,
    removeItem,
    setItems
  };
}
