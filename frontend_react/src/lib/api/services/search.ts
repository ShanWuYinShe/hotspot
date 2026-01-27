import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  GeoPoint,
  GroupSearchItem,
  MomentSearchItem,
  PaginatedData,
  PaginationParams,
  UserSearchItem,
} from '@/types/api';

export interface SearchMomentsByKeywordRequest extends PaginationParams {
  kw: string;
  reference_point?: GeoPoint;
}

export interface SearchUsersByKeywordRequest extends PaginationParams {
  kw: string;
  reference_point?: GeoPoint;
}

export interface SearchGroupsByKeywordRequest extends PaginationParams {
  kw: string;
  reference_point?: GeoPoint;
}

export const SearchService = {
  searchMomentsByKeyword: (params: SearchMomentsByKeywordRequest) =>
    apiClient.post<PaginatedData<MomentSearchItem>>(ENDPOINTS.MOMENTS.SEARCH_BY_KEYWORD, params),

  searchMomentsByTag: (params: { tag: string; cursor?: string; limit?: number; reference_point?: GeoPoint }) =>
    apiClient.post<PaginatedData<MomentSearchItem>>(ENDPOINTS.MOMENTS.SEARCH_BY_TAG, params),

  searchMomentsByLocation: (params: { radius: number; kw?: string; cursor?: string; limit?: number; reference_point: GeoPoint }) =>
    apiClient.post<PaginatedData<MomentSearchItem>>(ENDPOINTS.MOMENTS.SEARCH_BY_LOCATION, params),

  searchUsersByKeyword: (params: SearchUsersByKeywordRequest) =>
    apiClient.post<PaginatedData<UserSearchItem>>(ENDPOINTS.USER.SEARCH_BY_KEYWORD, params),

  searchUsersByTag: (params: { tag: string; cursor?: string; limit?: number; reference_point?: GeoPoint }) =>
    apiClient.post<PaginatedData<UserSearchItem>>(ENDPOINTS.USER.SEARCH_BY_TAG, params),

  searchUsersByLocation: (params: { radius: number; kw?: string; cursor?: string; limit?: number; reference_point: GeoPoint }) =>
    apiClient.post<PaginatedData<UserSearchItem>>(ENDPOINTS.USER.SEARCH_BY_LOCATION, params),

  searchGroupsByKeyword: (params: SearchGroupsByKeywordRequest) =>
    apiClient.post<PaginatedData<GroupSearchItem>>(ENDPOINTS.GROUPS.SEARCH_BY_KEYWORD, params),

  searchGroupsByTag: (params: { tag: string; cursor?: string; limit?: number; reference_point?: GeoPoint }) =>
    apiClient.post<PaginatedData<GroupSearchItem>>(ENDPOINTS.GROUPS.SEARCH_BY_TAG, params),

  searchGroupsByLocation: (params: { radius: number; kw?: string; cursor?: string; limit?: number; reference_point: GeoPoint }) =>
    apiClient.post<PaginatedData<GroupSearchItem>>(ENDPOINTS.GROUPS.SEARCH_BY_LOCATION, params),
};
