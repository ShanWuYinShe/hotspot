/**
 * 🎯 最优雅的API响应处理工具
 * useApi已经统一处理了所有响应格式和错误，这里只提供便利函数
 */

import type { PaginatedResponse } from '~/types';

/**
 * ✅ 直接返回数据 - useApi已经处理了所有复杂逻辑
 * 这个函数主要用于类型提示和代码可读性
 */
export const extractApiData = <T>(data: T): T => data;

/**
 * ✅ 直接返回分页数据 - useApi已经处理了所有复杂逻辑
 */
export const extractPaginatedData = <T>(data: PaginatedResponse<T>): PaginatedResponse<T> => data;

/**
 * 🛡️ 安全提取数据，失败时返回默认值
 */
export const safeExtractApiData = <T>(data: T, defaultValue: T): T => {
  try {
    return data ?? defaultValue;
  } catch {
    return defaultValue;
  }
};

/**
 * 🛡️ 安全提取分页数据，失败时返回空结果
 */
export const safeExtractPaginatedData = <T>(data: PaginatedResponse<T>): PaginatedResponse<T> => {
  try {
    return data ?? {
      items: [],
      has_more: false,
      next_cursor: undefined
    };
  } catch {
    return {
      items: [],
      has_more: false,
      next_cursor: undefined
    };
  }
};