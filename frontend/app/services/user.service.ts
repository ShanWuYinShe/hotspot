/**
 * 用户服务 - Composition API 风格
 * 提供用户相关的所有 API 调用功能
 * 
 * 设计原则：
 * 1. 发送前严格验证参数
 * 2. 接收后信任后端数据，直接使用
 */

import { USER_ENDPOINTS, AUTH_ENDPOINTS, buildUserUrl, buildAuthUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";
import { 
  validateLoginId, 
  validatePassword, 
  validateNickname, 
  validateUserId, 
  validateCoordinates,
  validateContent,
  validatePaginationParams,
  validateRadius
} from "~/utils/api-validation";
import { extractApiData, extractPaginatedData } from "~/utils/api-response";
import type { 
  ApiResponse, 
  UpdateNicknameRequest, 
  SearchUserByLocationRequest, 
  SearchUserByLocationResponse,
  UserInfo,
  AuthResponse,
  RefreshTokenResponse,
  User,
  PaginatedResponse,
  BasicSuccess,
  PasswordStatusResult
} from "~/types";

export const useUserService = () => {
  const api = useApi();
  
  // 🔥 使用安全认证服务
  const secureAuth = useSecureAuth();

  // 🔥 安全认证方法（与需求文档对齐：使用 SHA256 + nonce 机制）
  const login = async (loginId: string, password: string): Promise<AuthResponse> => {
    // 发送前验证参数
    const validLoginId = validateLoginId(loginId);
    const validPassword = validatePassword(password);
    
    // 使用安全认证流程
    return await secureAuth.secureLogin(validLoginId, validPassword);
  };

  const register = async (loginId: string, nickname: string, password: string): Promise<AuthResponse> => {
    // 发送前验证参数
    const validLoginId = validateLoginId(loginId);
    const validNickname = validateNickname(nickname);
    const validPassword = validatePassword(password);
    
    // 使用安全认证流程
    return await secureAuth.secureRegister(validLoginId, validNickname, validPassword);
  };

  const createTempUser = async (nickname?: string): Promise<AuthResponse> => {
    // 发送前验证参数（如果提供了昵称）
    const validNickname = nickname ? validateNickname(nickname) : `临时用户${Math.floor(Math.random() * 10000)}`;
    
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.CREATE_TEMP), { 
      nickname: validNickname,
    });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  const refreshToken = async (refreshToken: string): Promise<RefreshTokenResponse> => {
    // 发送前验证参数
    if (!refreshToken || typeof refreshToken !== 'string' || !refreshToken.trim()) {
      throw new Error('刷新令牌不能为空');
    }
    
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.REFRESH_TOKEN), { 
      refresh_token: refreshToken.trim() 
    });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  const checkAuthStatus = async (): Promise<AuthResponse> => {
    const response = await api.post(buildAuthUrl(AUTH_ENDPOINTS.CHECK_STATUS), {}, { auth: true });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  // 用户资料方法（公开访问，无需认证）
  const getUserProfile = async (userId: string): Promise<User> => {
    // 发送前验证参数
    const validUserId = validateUserId(userId);
    
    const response = await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true), { user_id: validUserId });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  const updateUserProfile = async (data: UpdateNicknameRequest): Promise<{ nickname: string }> => {
    // 发送前验证参数
    const validNickname = validateNickname(data.nickname);
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_PROFILE), {
      nickname: validNickname
    }, { auth: true });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  // 位置方法
  const initLocation = async (latitude?: number, longitude?: number): Promise<BasicSuccess> => {
    // 发送前验证参数
    const lat = latitude || DEFAULTS.latitude;
    const lng = longitude || DEFAULTS.longitude;
    validateCoordinates(lat, lng);
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_LOCATION), {
      latitude: lat,
      longitude: lng,
      location_name: DEFAULTS.locationName
    }, { auth: true });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  const updateLocation = async (latitude?: number, longitude?: number, locationName?: string): Promise<BasicSuccess> => {
    // 发送前验证参数
    const lat = latitude ?? DEFAULTS.latitude;
    const lng = longitude ?? DEFAULTS.longitude;
    validateCoordinates(lat, lng);
    
    // 验证位置名称
    const validLocationName = locationName ? validateContent(locationName, 1, 120, '位置名称') : DEFAULTS.locationName;
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_LOCATION), { 
      latitude: lat,
      longitude: lng,
      location_name: validLocationName
    }, { auth: true });
    
    // 接收后直接信任并返回数据
    return extractApiData(response);
  };

  // 搜索方法（公开访问，无需认证）
  const searchUsers = async (query: string, cursor?: number): Promise<PaginatedResponse<User>> => {
    // 发送前验证参数
    const validQuery = validateContent(query, 1, 32, '搜索关键词');
    const paginationParams = validatePaginationParams({ cursor, limit: DEFAULTS.pageSize });
    
    const response = await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_NAME, true), {
      query: validQuery,
      cursor: paginationParams.cursor || 0,
      limit: paginationParams.limit || DEFAULTS.pageSize,
    });
    
    // 接收后直接信任并返回分页数据
    return extractPaginatedData(response);
  };

  const findNearbyUsers = async (params: SearchUserByLocationRequest): Promise<PaginatedResponse<User>> => {
    // 发送前验证参数
    const lat = params.latitude || DEFAULTS.latitude;
    const lng = params.longitude || DEFAULTS.longitude;
    validateCoordinates(lat, lng);
    const validRadius = validateRadius(params.radius ?? DEFAULTS.nearbyRadiusMeters);
    const paginationParams = validatePaginationParams({ 
      cursor: params.cursor ? parseInt(params.cursor) : undefined, 
      limit: params.limit 
    });
    
    const response = await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), {
      latitude: lat,
      longitude: lng,
      radius: validRadius,
      cursor: paginationParams.cursor || 0,
      limit: paginationParams.limit || DEFAULTS.pageSize,
    });
    
    // 接收后直接信任并返回分页数据
    return extractPaginatedData(response);
  };

  const getNearbyUsers = async (
    latitude?: number,
    longitude?: number,
    radiusMeters?: number,
    cursor?: number
  ): Promise<PaginatedResponse<User>> => {
    return await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), {
      latitude: latitude || DEFAULTS.latitude,
      longitude: longitude || DEFAULTS.longitude,
      radius: Math.round(radiusMeters || DEFAULTS.nearbyRadiusMeters),
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    });
  };

  // 🔥 安全密码设置方法
  const setPassword = async (currentPassword: string, newPassword: string, userId: string): Promise<BasicSuccess> => {
    // 发送前验证参数
    const validNewPassword = validatePassword(newPassword);
    const validCurrentPassword = currentPassword ? validatePassword(currentPassword) : undefined;
    const validUserId = validateUserId(userId);
    
    // 使用安全认证流程
    return await secureAuth.secureSetPassword(validNewPassword, validUserId, validCurrentPassword);
  };

  const getPasswordStatus = async (): Promise<PasswordStatusResult> => {
    return await api.post(buildAuthUrl(AUTH_ENDPOINTS.PASSWORD_STATUS), {}, { auth: true });
  };

  // 🔥 安全账户升级方法
  const upgradeAccount = async (loginId: string, password: string): Promise<AuthResponse> => {
    // 发送前验证参数
    const validLoginId = validateLoginId(loginId);
    const validPassword = validatePassword(password);
    
    // 使用安全认证流程
    return await secureAuth.secureUpgradeAccount(validLoginId, validPassword);
  };

  const deleteAccount = async (): Promise<BasicSuccess> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.DELETE_ACCOUNT), {}, { auth: true });
  };

  const healthCheck = async (): Promise<{ status: string }> => {
    return await api.postPublic('/health', {});
  };

  return {
    // 认证
    login,
    register,
    createTempUser,
    refreshToken,
    checkAuthStatus,
    
    // 资料
    getUserProfile,
    updateUserProfile,
    
    // 位置
    initLocation,
    updateLocation,
    
    // 搜索发现
    searchUsers,
    findNearbyUsers,
    getNearbyUsers,
    
    // 安全
    setPassword,
    getPasswordStatus,
    
    // 账户管理
    upgradeAccount,
    deleteAccount,
    
    // 系统
    healthCheck,
  };
};
