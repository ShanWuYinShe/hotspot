/**
 * 通用分页Store基础组合式函数
 * 
 * 提供统一的分页、缓存、错误处理逻辑，减少Store间的代码重复
 */

export interface PaginationState {
  hasMore: boolean;
  nextCursor?: string;
}

export interface PaginatedStoreConfig<T> {
  /** Store名称，用于日志和调试 */
  storeName: string;
  /** 获取项目ID的函数 */
  getItemId: (item: T) => string;
  /** 默认分页大小 */
  defaultLimit?: number;
}

export interface PaginatedList<T> {
  items: Ref<T[]>;
  pagination: Ref<PaginationState>;
  loading: Ref<boolean>;
}

/**
 * 创建基础分页Store逻辑
 */
export function useBasePaginatedStore<T>(config: PaginatedStoreConfig<T>) {
  const { storeName, getItemId, defaultLimit = 20 } = config;
  const errorHandler = useErrorHandler();

  /**
   * 创建分页列表状态
   */
  function createPaginatedList(): PaginatedList<T> {
    return {
      items: ref([]) as Ref<T[]>,
      pagination: ref<PaginationState>({
        hasMore: false,
        nextCursor: undefined,
      }),
      loading: ref(false),
    };
  }

  /**
   * 通用的分页数据获取逻辑
   */
  async function fetchPaginatedData<TResponse>(
    list: PaginatedList<T>,
    fetchFn: (cursor?: string, limit?: number) => Promise<TResponse>,
    extractData: (response: TResponse) => { items: T[]; hasMore: boolean; nextCursor?: string },
    loadMore = false
  ): Promise<T[]> {
    if (list.loading.value) return [];

    try {
      list.loading.value = true;

      const cursor = loadMore ? list.pagination.value.nextCursor : undefined;
      const response = await fetchFn(cursor, defaultLimit);
      const { items, hasMore, nextCursor } = extractData(response);

      if (loadMore) {
        list.items.value.push(...items);
      } else {
        list.items.value = items;
      }

      list.pagination.value = {
        hasMore,
        nextCursor,
      };

      return items;
    } catch (error) {
      errorHandler.handleApiError(error, `${storeName}.${loadMore ? 'loadMore' : 'fetch'}`);
      return [];
    } finally {
      list.loading.value = false;
    }
  }

  /**
   * 在列表中查找项目
   */
  function findItemInList(list: PaginatedList<T>, itemId: string): T | undefined {
    return list.items.value.find(item => getItemId(item) === itemId);
  }

  /**
   * 在列表中更新项目
   */
  function updateItemInList(
    list: PaginatedList<T>,
    itemId: string,
    updates: Partial<T>
  ): boolean {
    const index = list.items.value.findIndex(item => getItemId(item) === itemId);
    if (index !== -1) {
      Object.assign(list.items.value[index] as any, updates);
      return true;
    }
    return false;
  }

  /**
   * 从列表中移除项目
   */
  function removeItemFromList(list: PaginatedList<T>, itemId: string): boolean {
    const index = list.items.value.findIndex(item => getItemId(item) === itemId);
    if (index !== -1) {
      list.items.value.splice(index, 1);
      return true;
    }
    return false;
  }

  /**
   * 在列表开头添加项目
   */
  function prependItemToList(list: PaginatedList<T>, item: T): void {
    list.items.value.unshift(item);
  }

  /**
   * 清空列表
   */
  function clearList(list: PaginatedList<T>): void {
    list.items.value = [];
    list.pagination.value = {
      hasMore: false,
      nextCursor: undefined,
    };
  }

  /**
   * 跨多个列表更新项目
   */
  function updateItemInMultipleLists(
    lists: PaginatedList<T>[],
    itemId: string,
    updates: Partial<T>
  ): void {
    lists.forEach(list => updateItemInList(list, itemId, updates));
  }

  /**
   * 跨多个列表移除项目
   */
  function removeItemFromMultipleLists(
    lists: PaginatedList<T>[],
    itemId: string
  ): void {
    lists.forEach(list => removeItemFromList(list, itemId));
  }

  /**
   * 创建只读的计算属性
   */
  function createReadonlyList<T>(list: PaginatedList<T>) {
    return {
      items: computed(() => list.items.value.slice()),
      pagination: computed(() => ({ ...list.pagination.value })),
      loading: computed(() => list.loading.value),
    };
  }

  return {
    // 核心功能
    createPaginatedList,
    fetchPaginatedData,
    
    // 列表操作
    findItemInList,
    updateItemInList,
    removeItemFromList,
    prependItemToList,
    clearList,
    
    // 多列表操作
    updateItemInMultipleLists,
    removeItemFromMultipleLists,
    
    // 工具函数
    createReadonlyList,
  };
}

/**
 * 地理位置分页Store扩展
 */
export interface GeoPaginatedStoreConfig<T> extends PaginatedStoreConfig<T> {
  /** 默认搜索半径（米） */
  defaultRadius?: number;
}

export function useGeoPaginatedStore<T>(config: GeoPaginatedStoreConfig<T>) {
  const base = useBasePaginatedStore(config);
  const { defaultRadius = 5000 } = config;

  /**
   * 地理位置分页数据获取
   */
  async function fetchGeoPaginatedData<TResponse>(
    list: PaginatedList<T>,
    fetchFn: (latitude: number, longitude: number, radius: number, cursor?: string, limit?: number) => Promise<TResponse>,
    extractData: (response: TResponse) => { items: T[]; hasMore: boolean; nextCursor?: string },
    latitude: number,
    longitude: number,
    radius = defaultRadius,
    loadMore = false
  ): Promise<T[]> {
    return base.fetchPaginatedData(
      list,
      (cursor, limit) => fetchFn(latitude, longitude, radius, cursor, limit),
      extractData,
      loadMore
    );
  }

  return {
    ...base,
    fetchGeoPaginatedData,
  };
}

/**
 * 搜索分页Store扩展
 */
export function useSearchPaginatedStore<T>(config: PaginatedStoreConfig<T>) {
  const base = useBasePaginatedStore(config);

  /**
   * 搜索分页数据获取
   */
  async function fetchSearchPaginatedData<TResponse>(
    list: PaginatedList<T>,
    fetchFn: (query: string, cursor?: string, limit?: number) => Promise<TResponse>,
    extractData: (response: TResponse) => { items: T[]; hasMore: boolean; nextCursor?: string },
    query: string,
    loadMore = false
  ): Promise<T[]> {
    return base.fetchPaginatedData(
      list,
      (cursor, limit) => fetchFn(query, cursor, limit),
      extractData,
      loadMore
    );
  }

  return {
    ...base,
    fetchSearchPaginatedData,
  };
}
