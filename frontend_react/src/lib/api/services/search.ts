/**
 * 搜索服务
 * 用户、瞬间、群组搜索 API
 * 
 * 基于 @api_spec_user.mdc, @api_spec_moment.mdc, @api_spec_group.mdc 规范
 */

import { apiRequestPublic } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  UserSearchItem,
  MomentSearchItem,
  GroupSearchItem,
} from '@/types/api';

// ============== 基础搜索参数 ==============

interface SearchBaseParams {
  limit?: number;
  cursor?: string;
}

// ============== 用户搜索 ==============

interface UserSearchByKeywordParams extends SearchBaseParams {
  kw: string;
}

interface UserSearchByTagParams extends SearchBaseParams {
  tag: string;
}

interface UserSearchByLocationParams extends SearchBaseParams {
  kw?: string;
  radius: number;
}

/**
 * 按关键字搜索用户
 * Token 可选；登录后可见更多用户
 * @api POST /api/v1/user/search_by_keyword
 */
export async function searchUsersByKeyword(
  params: UserSearchByKeywordParams
): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
  return apiRequestPublic<PaginatedData<UserSearchItem>>(
    ENDPOINTS.USER.SEARCH_BY_KEYWORD,
    params
  );
}

/**
 * 按标签搜索用户
 * Token 可选；登录后可见更多用户
 * @api POST /api/v1/user/search_by_tag
 */
export async function searchUsersByTag(
  params: UserSearchByTagParams
): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
  return apiRequestPublic<PaginatedData<UserSearchItem>>(
    ENDPOINTS.USER.SEARCH_BY_TAG,
    params
  );
}

/**
 * 按位置搜索用户
 * Token 可选；登录后可见更多用户
 * @api POST /api/v1/user/search_by_location
 */
export async function searchUsersByLocation(
  params: UserSearchByLocationParams
): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
  return apiRequestPublic<PaginatedData<UserSearchItem>>(
    ENDPOINTS.USER.SEARCH_BY_LOCATION,
    params
  );
}

// ============== 瞬间搜索 ==============

interface MomentSearchByKeywordParams extends SearchBaseParams {
  kw: string;
}

interface MomentSearchByTagParams extends SearchBaseParams {
  tag: string;
}

interface MomentSearchByLocationParams extends SearchBaseParams {
  kw?: string;
  radius: number;
}

/**
 * 按关键字搜索瞬间
 * Token 可选；登录后可见更多内容
 * @api POST /api/v1/moments/search_by_keyword
 */
export async function searchMomentsByKeyword(
  params: MomentSearchByKeywordParams
): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
  return apiRequestPublic<PaginatedData<MomentSearchItem>>(
    ENDPOINTS.MOMENTS.SEARCH_BY_KEYWORD,
    params
  );
}

/**
 * 按标签搜索瞬间
 * Token 可选；登录后可见更多内容
 * @api POST /api/v1/moments/search_by_tag
 */
export async function searchMomentsByTag(
  params: MomentSearchByTagParams
): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
  return apiRequestPublic<PaginatedData<MomentSearchItem>>(
    ENDPOINTS.MOMENTS.SEARCH_BY_TAG,
    params
  );
}

/**
 * 按位置搜索瞬间
 * Token 可选；登录后可见更多内容
 * @api POST /api/v1/moments/search_by_location
 */
export async function searchMomentsByLocation(
  params: MomentSearchByLocationParams
): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
  return apiRequestPublic<PaginatedData<MomentSearchItem>>(
    ENDPOINTS.MOMENTS.SEARCH_BY_LOCATION,
    params
  );
}

// ============== 群组搜索 ==============

interface GroupSearchByKeywordParams extends SearchBaseParams {
  kw: string;
}

interface GroupSearchByTagParams extends SearchBaseParams {
  tag: string;
}

interface GroupSearchByLocationParams extends SearchBaseParams {
  kw?: string;
  radius: number;
}

/**
 * 按关键字搜索群组
 * Token 可选；登录后可见更多群组
 * @api POST /api/v1/group_chats/search_by_keyword
 */
export async function searchGroupsByKeyword(
  params: GroupSearchByKeywordParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.GROUP_CHATS.SEARCH_BY_KEYWORD,
    params
  );
}

/**
 * 按标签搜索群组
 * Token 可选；登录后可见更多群组
 * @api POST /api/v1/group_chats/search_by_tag
 */
export async function searchGroupsByTag(
  params: GroupSearchByTagParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.GROUP_CHATS.SEARCH_BY_TAG,
    params
  );
}

/**
 * 按位置搜索群组
 * Token 可选；登录后可见更多群组
 * @api POST /api/v1/group_chats/search_by_location
 */
export async function searchGroupsByLocation(
  params: GroupSearchByLocationParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.GROUP_CHATS.SEARCH_BY_LOCATION,
    params
  );
}
