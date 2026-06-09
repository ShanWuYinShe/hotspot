/**
 * 搜索服务
 */

import { post, type RequestOptions } from '../client';
import { SEARCH } from '../endpoints';
import type { SearchSuggestResponse, GeoPoint, SearchSuggestType } from '$lib/types/api';

export const searchService = {
  /**
   * 综合搜索建议（对齐 spec SearchSuggestRequest）
   */
  suggest(
    params: {
      prefix: string;
      reference_point: GeoPoint;
      types?: SearchSuggestType[];
      radius?: number;
    },
    options?: RequestOptions
  ) {
    return post<SearchSuggestResponse>(SEARCH.SUGGEST, params, options);
  }
};
