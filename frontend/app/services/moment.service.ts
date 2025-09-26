/**
 * 动态服务 - Composition API 风格
 * 提供动态相关的所有 API 调用功能
 */

import { MOMENT_ENDPOINTS, buildMomentUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";


import type {
  ApiResponse,
  CreateMomentRequest,
  CreateMomentResponse,
  DeleteMomentRequest,
  DeleteMomentResponse,
  SearchMomentByIdRequest,
  MomentResponse,
  SearchMomentsByLocationRequest,
  SearchMomentsByLocationResponse,
  UserMomentsHistoryRequest,
  UserMomentsHistoryResponse,
  SearchMomentsByTagsRequest,
  SearchMomentsByTagsResponse,
  LikeMomentRequest,
  LikeMomentResponse,
  UnlikeMomentRequest,
  UnlikeMomentResponse,
} from "~/types/api";

// 临时错误创建函数
const createApiError = (params: {
  status?: number;
  code?: string;
  message: string;
  details?: any;
}) => {
  const error = new Error(params.message) as any;
  error.status = params.status || 400;
  error.code = params.code || "API_ERROR";
  error.details = params.details;
  return error;
};

export const useMomentService = () => {
  const api = useApi();

  // 创建动态
  const createMoment = async (
    latitude: number,
    longitude: number,
    description?: string,
    tags?: string[],
    imageUrls?: string[],
  ): Promise<CreateMomentResponse> => {
    const request: CreateMomentRequest = {
      latitude,
      longitude,
      description: description || '',
      location_name: '', // 需要从坐标获取位置名称
      tags,
      images: imageUrls,
    };
    
    // 直接使用useApi的统一错误处理和响应处理
    const response = await api.post(
      buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.CREATE),
      request,
      { auth: true },
    );
    
    return response as CreateMomentResponse;
  };

  // 删除动态
  const deleteMoment = async (
    momentId: string,
  ): Promise<DeleteMomentResponse> => {
    try {
      const request: DeleteMomentRequest = {
        moment_id: momentId,
      };
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.DELETE),
        request,
        { auth: true },
      );
      return response as DeleteMomentResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "DELETE_MOMENT_FAILED",
        message: e?.message || "删除动态失败",
        details: e?.details,
      });
    }
  };

  // 根据ID搜索动态（公开访问，无需认证）
  const getMomentById = async (momentId: string): Promise<MomentResponse> => {
    try {
      const request: SearchMomentByIdRequest = {
        moment_id: momentId,
      };
      const response = await api.postPublic(
        buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true),
        request,
      );
      return response as MomentResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "GET_MOMENT_FAILED",
        message: e?.message || "获取动态详情失败",
        details: e?.details,
      });
    }
  };

  // 根据位置搜索动态
  const searchMomentsByLocation = async (
    latitude: number,
    longitude: number,
    radius?: number,
    cursor?: string,
    limit?: number,
  ): Promise<SearchMomentsByLocationResponse> => {
    try {
      const request: SearchMomentsByLocationRequest = {
        latitude,
        longitude,
        radius: radius || DEFAULTS.nearbyRadiusMeters,
        limit: limit || DEFAULTS.pageSize,
        cursor,
      };
      const response = await api.postPublic(
        buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true),
        request,
      );
      return response as SearchMomentsByLocationResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "SEARCH_MOMENTS_BY_LOCATION_FAILED",
        message: e?.message || "按位置搜索动态失败",
        details: e?.details,
      });
    }
  };

  // 获取用户动态历史
  const getUserMomentsHistory = async (
    userId?: string,
    cursor?: string,
    limit?: number,
  ): Promise<UserMomentsHistoryResponse> => {
    try {
      const request: UserMomentsHistoryRequest = {
        user_id: userId?.toString() || '',
        limit: limit || DEFAULTS.pageSize,
        cursor,
      };
      const response = await api.postPublic(
        buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.HISTORY, true),
        request,
      );
      return response as UserMomentsHistoryResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "GET_USER_MOMENTS_FAILED",
        message: e?.message || "获取用户动态历史失败",
        details: e?.details,
      });
    }
  };

  // 根据标签搜索动态
  const searchMomentsByTags = async (
    tags: string[],
    latitude: number,
    longitude: number,
    cursor?: string,
    limit?: number,
  ): Promise<SearchMomentsByTagsResponse> => {
    try {
      const request: SearchMomentsByTagsRequest = {
        tags,
        latitude,
        longitude,
        limit: limit || DEFAULTS.pageSize,
        cursor,
      };
      const response = await api.postPublic(
        buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.SEARCH_BY_TAGS, true),
        request,
      );
      return response as SearchMomentsByTagsResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "SEARCH_MOMENTS_BY_TAGS_FAILED",
        message: e?.message || "按标签搜索动态失败",
        details: e?.details,
      });
    }
  };

  // 点赞动态
  const likeMoment = async (momentId: string): Promise<LikeMomentResponse> => {
    try {
      const request: LikeMomentRequest = {
        moment_id: momentId,
      };
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.LIKE),
        request,
        { auth: true },
      );
      return response as LikeMomentResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "LIKE_MOMENT_FAILED",
        message: e?.message || "点赞动态失败",
        details: e?.details,
      });
    }
  };

  // 取消点赞动态
  const unlikeMoment = async (
    momentId: string,
  ): Promise<UnlikeMomentResponse> => {
    try {
      const request: UnlikeMomentRequest = {
        moment_id: momentId,
      };
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.UNLIKE),
        request,
        { auth: true },
      );
      return response as UnlikeMomentResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "UNLIKE_MOMENT_FAILED",
        message: e?.message || "取消点赞动态失败",
        details: e?.details,
      });
    }
  };

  // 获取附近的动态（默认使用当前位置）
  const getNearbyMoments = async (
    latitude?: number,
    longitude?: number,
    radius?: number,
    cursor?: string,
    limit?: number,
  ): Promise<SearchMomentsByLocationResponse> => {
    return await searchMomentsByLocation(
      latitude || DEFAULTS.latitude,
      longitude || DEFAULTS.longitude,
      radius,
      cursor,
      limit,
    );
  };

  // 获取我的动态
  const getMyMoments = async (
    cursor?: string,
    limit?: number,
  ): Promise<UserMomentsHistoryResponse> => {
    return await getUserMomentsHistory(undefined, cursor, limit);
  };

  // 切换点赞状态
  const toggleLike = async (
    momentId: string,
    isCurrentlyLiked: boolean,
  ): Promise<LikeMomentResponse | UnlikeMomentResponse> => {
    if (isCurrentlyLiked) {
      return await unlikeMoment(momentId);
    } else {
      return await likeMoment(momentId);
    }
  };

  // 搜索动态（综合搜索）
  const searchMoments = async (
    query: {
      type: "location" | "tags" | "user";
      latitude?: number;
      longitude?: number;
      radius?: number;
      tags?: string[];
      userId?: string;
    },
    cursor?: string,
    limit?: number,
  ): Promise<
    | SearchMomentsByLocationResponse
    | SearchMomentsByTagsResponse
    | UserMomentsHistoryResponse
  > => {
    switch (query.type) {
      case "location":
        return await searchMomentsByLocation(
          query.latitude!,
          query.longitude!,
          query.radius,
          cursor,
          limit,
        );
      case "tags":
        return await searchMomentsByTags(
          query.tags!,
          query.latitude || DEFAULTS.latitude,
          query.longitude || DEFAULTS.longitude,
          cursor,
          limit,
        );
      case "user":
        return await getUserMomentsHistory(query.userId, cursor, limit);
      default:
        throw new Error("不支持的搜索类型");
    }
  };

  return {
    // 基础CRUD
    createMoment,
    deleteMoment,
    getMomentById,

    // 搜索相关
    searchMomentsByLocation,
    searchMomentsByTags,
    getUserMomentsHistory,

    // 交互相关
    likeMoment,
    unlikeMoment,
    toggleLike,

    // 便捷方法
    getNearbyMoments,
    getMyMoments,
    searchMoments,
  };
};
