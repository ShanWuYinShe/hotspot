// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, pick } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	PaginatedData,
	UserItem,
	SearchSuggestRequest,
	SearchSuggestResponse,
	SearchUsersRequest,
} from '@/api/types';

// Search
export async function searchUsers(params: SearchUsersRequest)         { return request<PaginatedData<UserItem>>(pick(Endpoints.Users.Search, Endpoints.Public.Users.Search), params); }

// Search
// spec api_spec_discovery.md：prefix 必填，reference_point/types/radius 必填可空
export async function searchSuggest(params: SearchSuggestRequest) { return request<SearchSuggestResponse>(pick(Endpoints.Search.Suggest, Endpoints.Public.Search.Suggest), params); }
