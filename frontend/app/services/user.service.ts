/**
 * 用户服务 - 最简化版本
 * 保持基本接口，供现有代码调用
 */


import { USER_ENDPOINTS, AUTH_ENDPOINTS, buildUserUrl, buildAuthUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";
import type { 
  ApiResponse, 
  UpdateNicknameRequest, 
  SearchUserByLocationRequest, 
  SearchUserByLocationResponse,
  UserInfo
} from "~/types";

export const useUserService = () => {
  const api = useApi();

  // 基本认证方法（与文档对齐：/api/v1/auth/*）
  const login = async (loginId: string, password: string): Promise<any> => {
    return await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.LOGIN), { login_id: loginId, password });
  };

  const register = async (loginId: string, nickname: string, password: string): Promise<any> => {
    return await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.REGISTER), { login_id: loginId, nickname, password });
  };

  const createTempUser = async (nickname?: string): Promise<any> => {
    return await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.CREATE_TEMP), { 
      nickname: nickname || `临时用户${Math.floor(Math.random() * 10000)}`,
    });
  };

  const refreshToken = async (refreshToken: string): Promise<any> => {
    return await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.REFRESH_TOKEN), { refresh_token: refreshToken });
  };

  const checkAuthStatus = async (): Promise<any> => {
    return await api.post(buildAuthUrl(AUTH_ENDPOINTS.CHECK_STATUS), {}, { auth: true });
  };

  // 用户资料方法  
  const getUserProfile = async (userId: string): Promise<UserInfo> => {
    return await api.post(buildUserUrl('/get_profile'), { user_id: userId }, { auth: true });
  };

  const updateUserProfile = async (data: UpdateNicknameRequest): Promise<ApiResponse> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.UPDATE_PROFILE), data, { auth: true });
  };

  // 位置方法
  const initLocation = async (latitude?: number, longitude?: number): Promise<any> => {
    return await api.post(buildUserUrl('/init_location'), {
      latitude: latitude || DEFAULTS.latitude,
      longitude: longitude || DEFAULTS.longitude,
    }, { auth: true });
  };

  const updateLocation = async (latitude?: number, longitude?: number, locationName?: string): Promise<any> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.UPDATE_LOCATION), { 
      latitude: latitude ?? DEFAULTS.latitude,
      longitude: longitude ?? DEFAULTS.longitude,
      location_name: locationName || DEFAULTS.locationName
    }, { auth: true });
  };

  // 搜索方法
  const searchUsers = async (query: string, cursor?: number): Promise<any> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.SEARCH_BY_NAME), {
      query,
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  const findNearbyUsers = async (params: SearchUserByLocationRequest): Promise<SearchUserByLocationResponse> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.SEARCH_BY_LOCATION), {
      latitude: params.latitude || DEFAULTS.latitude,
      longitude: params.longitude || DEFAULTS.longitude,
      radius: params.radius ?? DEFAULTS.nearbyRadiusMeters,
      cursor: params.cursor || 0,
      limit: params.limit || DEFAULTS.pageSize,
    }, { auth: true });
  };

  const getNearbyUsers = async (
    latitude?: number,
    longitude?: number,
    radiusMeters?: number,
    cursor?: number
  ): Promise<any> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.SEARCH_BY_LOCATION), {
      latitude: latitude || DEFAULTS.latitude,
      longitude: longitude || DEFAULTS.longitude,
      radius: Math.round(radiusMeters || DEFAULTS.nearbyRadiusMeters),
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 其他方法
  const setPassword = async (currentPassword: string, newPassword: string): Promise<any> => {
    return await api.post(buildAuthUrl(AUTH_ENDPOINTS.SET_PASSWORD), {
      new_password: newPassword,
      old_password: currentPassword || undefined,
    }, { auth: true });
  };

  const getPasswordStatus = async (): Promise<any> => {
    return await api.post(buildAuthUrl(AUTH_ENDPOINTS.PASSWORD_STATUS), {}, { auth: true });
  };

  const upgradeAccount = async (loginId: string, password: string): Promise<any> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.UPGRADE_ACCOUNT), {
      login_id: loginId,
      password: password
    }, { auth: true });
  };

  const deleteAccount = async (): Promise<any> => {
    return await api.post(buildUserUrl(USER_ENDPOINTS.DELETE_ACCOUNT), {}, { auth: true });
  };

  const healthCheck = async (): Promise<any> => {
    return await api.post(buildAuthUrl(AUTH_ENDPOINTS.HEALTH), {});
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
