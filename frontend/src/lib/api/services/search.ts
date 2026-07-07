/**
 * 搜索服务
 */

import { post, type RequestOptions } from '../client';
import { SEARCH } from '../endpoints';
import type {
  SearchSuggestResponse,
  GeoPoint,
  SearchSuggestType,
  SearchAllRequest,
  SearchAllResult
} from '$lib/types/api';

export const searchService = {
  /**
   * 综合搜索建议（对齐 spec SearchSuggestRequest）
   */
  suggest(
    params: {
      prefix: string;
      reference_point?: GeoPoint;
      types?: SearchSuggestType[];
      radius?: number;
    },
    options?: RequestOptions
  ) {
    return post<SearchSuggestResponse>(SEARCH.SUGGEST, params, options);
  },

  /**
   * 综合聚合搜索（一次请求返回用户/群组/瞬间）
   */
  all(params: SearchAllRequest, options?: RequestOptions) {
    return post<SearchAllResult>(SEARCH.ALL, params, options);
  }
};
