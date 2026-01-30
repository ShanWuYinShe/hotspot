/**
 * 搜索服务
 */

import { post } from '../client';
import { SEARCH } from '../endpoints';
import type { SearchSuggestions, GeoPoint } from '$lib/types/api';

export const searchService = {
	/**
	 * 综合搜索建议
	 */
	suggest(params: { prefix: string; limit?: number; location?: GeoPoint }) {
		return post<SearchSuggestions>(SEARCH.SUGGEST, params);
	}
};
