/**
 * 用户服务 - Composition API 风格
 * 提供用户相关的所有 API 调用功能
 * 
 * 设计原则：
 * 1. 发送前严格验证参数
 * 2. 接收后信任后端数据，直接使用
 */

import { USER_ENDPOINTS, AUTH_ENDPOINTS, STATUS_ENDPOINTS, buildUserUrl, buildAuthUrl, buildStatusUrl } from "~/config/api";
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
import type { 
  ApiResponse, 
  UpdateProfileRequest,
  UpdateProfileResponse,
  SearchUserByLocationRequest, 
  SearchUserByLocationResponse,
  UserResponse,
  AuthResponse,
  RefreshTokenResponse,
  PaginatedResponse,
  UpdateLocationRequest,
  UpdateLocationResponse,
  SearchUserByNameRequest,
  SearchUserByNameResponse,
  SearchUserByIdRequest,
  SearchUserByIdResponse,
  GetSettingsRequest,
  GetSettingsResponse,
  UpdateSettingsRequest,
  UpdateSettingsResponse,
  DeleteAccountRequest,
  DeleteAccountResponse,
  UpgradeAccountRequest,
  UpgradeAccountResponse,
  SetPasswordRequest,
  SetPasswordResponse,
  AddFriendRequest,
  AddFriendResponse,
  AcceptFriendRequest,
  AcceptFriendResponse,
  RejectFriendRequest,
  RejectFriendResponse,
  FollowRequest,
  FollowResponse,
  UnfollowRequest,
  UnfollowResponse,
  BlockRequest,
  BlockResponse,
  GetFriendsListRequest,
  GetFriendsListResponse,
  GetBlockedListRequest,
  GetBlockedListResponse,
  HealthCheckRequest,
  HealthCheckResponse
} from "~/types/api";

export const useUserService = () => {
  const api = useApi();
  
  // 使用安全认证服务
  const secureAuth = useSecureAuth();
  
  // 获取当前用户信息
  const authStore = useAuthStore();
  const currentUser = computed(() => authStore.user);

  // 安全认证方法（使用 SHA256 + nonce 机制）
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
    
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.CREATE_TEMP), { 
      nickname: validNickname,
    });
    
    // 接收后直接信任并返回数据
    return response;
  };

  const refreshToken = async (refreshToken: string): Promise<RefreshTokenResponse> => {
    // 发送前验证参数
    if (!refreshToken || typeof refreshToken !== 'string' || !refreshToken.trim()) {
      throw new Error('刷新令牌不能为空');
    }
    
    const response = await api.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.REFRESH_TOKEN), { 
      refresh_token: refreshToken.trim() 
    });
    
    // 接收后直接信任并返回数据
    return response;
  };

  const checkAuthStatus = async (): Promise<AuthResponse> => {
    const response = await api.post(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.CHECK_STATUS), {}, { auth: true });
    
    // 接收后直接信任并返回数据
    return response;
  };


  const getUserProfile = async (userId: string): Promise<SearchUserByIdResponse> => {
    // 发送前验证参数
    const validUserId = validateUserId(userId);
    
    const request: SearchUserByIdRequest = {
      user_id: validUserId
    };
    
    const response = await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true), request);
    
    // 接收后直接信任并返回数据
    return response;
  };

  // 搜索方法（公开访问，无需认证）
  const searchUsers = async (query: string, cursor?: string): Promise<PaginatedResponse<SearchUserByNameResponse>> => {
    // 发送前验证参数
    const validQuery = validateContent(query, 1, 32, '搜索关键词');
    const paginationParams = validatePaginationParams({ cursor, limit: DEFAULTS.pageSize });
    
    const request: SearchUserByNameRequest = {
      name: validQuery,
      limit: paginationParams.limit || DEFAULTS.pageSize,
      cursor: paginationParams.cursor
    };
    
    const response = await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_NAME, true), request);
    
    // 接收后直接信任并返回分页数据
    return response;
  };

  const findNearbyUsers = async (params: SearchUserByLocationRequest): Promise<PaginatedResponse<SearchUserByLocationResponse>> => {
    // 发送前验证参数
    const lat = params.latitude || DEFAULTS.latitude;
    const lng = params.longitude || DEFAULTS.longitude;
    validateCoordinates(lat, lng);
    const validRadius = validateRadius(params.radius ?? DEFAULTS.nearbyRadiusMeters);
    const paginationParams = validatePaginationParams({ 
      cursor: params.cursor, 
      limit: params.limit 
    });
    
    const request: SearchUserByLocationRequest = {
      latitude: lat,
      longitude: lng,
      radius: validRadius,
      limit: paginationParams.limit || DEFAULTS.pageSize,
      cursor: paginationParams.cursor
    };
    
    const response = await api.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), request);
    
    // 接收后直接信任并返回分页数据
    return response;
  };

  const updateUserProfile = async (data: UpdateProfileRequest): Promise<UpdateProfileResponse> => {
    // 发送前验证参数
    const validatedData: UpdateProfileRequest = {};
    if (data.nickname) validatedData.nickname = validateNickname(data.nickname);
    if (data.avatar_url) validatedData.avatar_url = data.avatar_url;
    if (data.bio) validatedData.bio = validateContent(data.bio, 1, 500, '个人简介');
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_PROFILE), validatedData, { auth: true });
    
    // 接收后直接信任并返回数据
    return response;
  };

  // 位置方法
  const initLocation = async (latitude?: number, longitude?: number): Promise<UpdateLocationResponse> => {
    // 发送前验证参数
    const lat = latitude || DEFAULTS.latitude;
    const lng = longitude || DEFAULTS.longitude;
    validateCoordinates(lat, lng);
    
    const request: UpdateLocationRequest = {
      latitude: lat,
      longitude: lng,
      location_name: DEFAULTS.locationName
    };
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_LOCATION), request, { auth: true });
    
    // 接收后直接信任并返回数据
    return response;
  };

  const updateLocation = async (latitude?: number, longitude?: number, locationName?: string): Promise<UpdateLocationResponse> => {
    // 发送前验证参数
    const lat = latitude ?? DEFAULTS.latitude;
    const lng = longitude ?? DEFAULTS.longitude;
    validateCoordinates(lat, lng);
    
    // 验证位置名称
    const validLocationName = locationName ? validateContent(locationName, 1, 120, '位置名称') : DEFAULTS.locationName;
    
    const request: UpdateLocationRequest = {
      latitude: lat,
      longitude: lng,
      location_name: validLocationName
    };
    
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_LOCATION), request, { auth: true });
    
    // 接收后直接信任并返回数据
    return response;
  };


  // 用户设置方法
  const getUserSettings = async (): Promise<GetSettingsResponse> => {
    const request: GetSettingsRequest = {};
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.GET_SETTINGS), request, { auth: true });
    return response;
  };

  const updateUserSettings = async (data: UpdateSettingsRequest): Promise<UpdateSettingsResponse> => {
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_SETTINGS), data, { auth: true });
    return response;
  };

  // 安全密码设置方法
  const setPassword = async (password: string): Promise<SetPasswordResponse> => {
    // 发送前验证参数
    const validPassword = validatePassword(password);
    
    // 使用安全认证流程
    const result = await secureAuth.secureSetPassword(validPassword, currentUser.value?.user_id || '', undefined);
    return {
      success: result.success,
      message: result.success ? '密码设置成功' : '密码设置失败',
      can_login: result.success
    };
  };

  // 安全账户升级方法
  const upgradeAccount = async (loginId: string, password: string): Promise<UpgradeAccountResponse> => {
    // 发送前验证参数
    const validLoginId = validateLoginId(loginId);
    const validPassword = validatePassword(password);
    
    // 使用安全认证流程
    return await secureAuth.secureUpgradeAccount(validLoginId, validPassword);
  };

  const deleteAccount = async (): Promise<DeleteAccountResponse> => {
    const request: DeleteAccountRequest = {
      confirmation: "DELETE_MY_ACCOUNT"
    };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.DELETE_ACCOUNT), request, { auth: true });
    return response;
  };

  // 用户关系管理方法
  const addFriend = async (toUserId: string, message?: string): Promise<AddFriendResponse> => {
    const validUserId = validateUserId(toUserId);
    const request: AddFriendRequest = {
      to_user_id: validUserId,
      message: message ? validateContent(message, 1, 200, '申请消息') : undefined
    };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.ADD_FRIEND), request, { auth: true });
    return response;
  };

  const acceptFriend = async (relationshipId: string): Promise<AcceptFriendResponse> => {
    const request: AcceptFriendRequest = { relationship_id: relationshipId };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.ACCEPT_FRIEND), request, { auth: true });
    return response;
  };

  const rejectFriend = async (relationshipId: string): Promise<RejectFriendResponse> => {
    const request: RejectFriendRequest = { relationship_id: relationshipId };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.REJECT_FRIEND), request, { auth: true });
    return response;
  };

  const followUser = async (toUserId: string): Promise<FollowResponse> => {
    const validUserId = validateUserId(toUserId);
    const request: FollowRequest = { to_user_id: validUserId };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.FOLLOW_USER), request, { auth: true });
    return response;
  };

  const unfollowUser = async (toUserId: string): Promise<UnfollowResponse> => {
    const validUserId = validateUserId(toUserId);
    const request: UnfollowRequest = { to_user_id: validUserId };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UNFOLLOW_USER), request, { auth: true });
    return response;
  };

  const blockUser = async (toUserId: string, reason?: string): Promise<BlockResponse> => {
    const validUserId = validateUserId(toUserId);
    const request: BlockRequest = {
      to_user_id: validUserId,
      reason: reason ? validateContent(reason, 1, 200, '拉黑原因') : undefined
    };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCK_USER), request, { auth: true });
    return response;
  };

  const getFriendsList = async (params?: Partial<GetFriendsListRequest>): Promise<PaginatedResponse<GetFriendsListResponse>> => {
    const request: GetFriendsListRequest = {
      limit: params?.limit || DEFAULTS.pageSize,
      cursor: params?.cursor,
      status: params?.status || 'accepted'
    };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.FRIENDS_LIST), request, { auth: true });
    return response;
  };

  const getBlockedList = async (params?: Partial<GetBlockedListRequest>): Promise<PaginatedResponse<GetBlockedListResponse>> => {
    const request: GetBlockedListRequest = {
      limit: params?.limit || DEFAULTS.pageSize,
      cursor: params?.cursor
    };
    const response = await api.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCKED_LIST), request, { auth: true });
    return response;
  };

  const healthCheck = async (): Promise<HealthCheckResponse> => {
    const request: HealthCheckRequest = {};
    const response = await api.postPublic(buildStatusUrl(STATUS_ENDPOINTS.PUBLIC.HEALTH), request);
    return response;
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
    
    // 设置
    getUserSettings,
    updateUserSettings,
    
    // 安全
    setPassword,
    
    // 账户管理
    upgradeAccount,
    deleteAccount,
    
    // 用户关系管理
    addFriend,
    acceptFriend,
    rejectFriend,
    followUser,
    unfollowUser,
    blockUser,
    getFriendsList,
    getBlockedList,
    
    // 系统
    healthCheck,
  };
};
