/**
 * 搜索服务
 */

import { post } from '../client';
import { SEARCH } from '../endpoints';
import type { SearchSuggestions, GeoPoint, SearchSuggestType } from '$lib/types/api';

export const searchService = {
	/**
	 * 综合搜索建议（对齐 spec SearchSuggestRequest）
	 */
	suggest(params: { prefix: string; reference_point: GeoPoint; types?: SearchSuggestType[] }) {
		return post<SearchSuggestions>(SEARCH.SUGGEST, params);
	}
};
