/**
 * 动态相关 API 服务
 * 使用与后端完全对齐的类型定义
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
  MomentInfo,
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
} from "~/types";

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
    description: string,
    latitude?: number,
    longitude?: number,
    locationName?: string,
    tags?: string[],
  ): Promise<CreateMomentResponse> => {
    try {
      const request: CreateMomentRequest = {
        description,
        latitude: latitude || DEFAULTS.latitude,
        longitude: longitude || DEFAULTS.longitude,
        location_name: locationName || DEFAULTS.locationName,
        tags,
      };
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.CREATE),
        request,
        { auth: true },
      );
      return response as CreateMomentResponse;
    } catch (e: any) {
      throw createApiError({
        status: e?.status || 400,
        code: e?.code || "CREATE_MOMENT_FAILED",
        message: e?.message || "创建动态失败",
        details: e?.details,
      });
    }
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
        buildMomentUrl(MOMENT_ENDPOINTS.DELETE),
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

  // 根据ID搜索动态
  const getMomentById = async (momentId: string): Promise<MomentInfo> => {
    try {
      const request: SearchMomentByIdRequest = {
        moment_id: momentId,
      };
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.SEARCH_BY_ID),
        request,
        { auth: true },
      );
      return response as MomentInfo;
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
    cursor?: number,
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
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.SEARCH_BY_LOCATION),
        request,
        { auth: true },
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
    userId?: number,
    cursor?: number,
    limit?: number,
  ): Promise<UserMomentsHistoryResponse> => {
    try {
      const request: UserMomentsHistoryRequest = {
        user_id: userId,
        limit: limit || DEFAULTS.pageSize,
        cursor,
      };
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.USER_HISTORY),
        request,
        { auth: true },
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
    cursor?: number,
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
      const response = await api.post(
        buildMomentUrl(MOMENT_ENDPOINTS.SEARCH_BY_TAGS),
        request,
        { auth: true },
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
        buildMomentUrl(MOMENT_ENDPOINTS.LIKE),
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
        buildMomentUrl(MOMENT_ENDPOINTS.UNLIKE),
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
    cursor?: number,
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
    cursor?: number,
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
      userId?: number;
    },
    cursor?: number,
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
