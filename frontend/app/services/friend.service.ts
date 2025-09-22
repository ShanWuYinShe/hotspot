/**
 * 好友服务 - Composition API 风格
 * 提供用户关系管理相关的所有 API 调用功能
 * 
 * 基于 backend_doc.md 5.2.2 用户关系管理API
 */

import { 
  USER_ENDPOINTS, 
  buildUserUrl 
} from "~/config/api";
import { 
  validateUserId,
  validateContent,
  validatePaginationParams 
} from "~/utils/api-validation";
import type { 
  ApiResponse,
  PaginatedResponse,
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
  UserResponse
} from "~/types/api";

export const useFriendService = () => {
  const api = useApi();

  /**
   * 发送好友请求
   */
  const sendFriendRequest = async (
    userId: string, 
    message?: string
  ): Promise<AddFriendResponse> => {
    // 发送前验证参数
    const validUserId = validateUserId(userId);
    const validMessage = message ? validateContent(message, 1, 200) : undefined;

    const request: AddFriendRequest = {
      target_user_id: validUserId,
      message: validMessage
    };

    const response = await api.post<ApiResponse<AddFriendResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.ADD_FRIEND, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 接受好友请求
   */
  const acceptFriendRequest = async (
    requestId: string
  ): Promise<AcceptFriendResponse> => {
    const validRequestId = validateUserId(requestId); // 通用ID验证

    const request: AcceptFriendRequest = {
      request_id: validRequestId
    };

    const response = await api.post<ApiResponse<AcceptFriendResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.ACCEPT_FRIEND, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 拒绝好友请求
   */
  const rejectFriendRequest = async (
    requestId: string
  ): Promise<RejectFriendResponse> => {
    const validRequestId = validateUserId(requestId); // 通用ID验证

    const request: RejectFriendRequest = {
      request_id: validRequestId
    };

    const response = await api.post<ApiResponse<RejectFriendResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.REJECT_FRIEND, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 关注用户
   */
  const followUser = async (userId: string): Promise<FollowResponse> => {
    const validUserId = validateUserId(userId);

    const request: FollowRequest = {
      target_user_id: validUserId
    };

    const response = await api.post<ApiResponse<FollowResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.FOLLOW_USER, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 取消关注用户
   */
  const unfollowUser = async (userId: string): Promise<UnfollowResponse> => {
    const validUserId = validateUserId(userId);

    const request: UnfollowRequest = {
      target_user_id: validUserId
    };

    const response = await api.post<ApiResponse<UnfollowResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.UNFOLLOW_USER, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 拉黑用户
   */
  const blockUser = async (
    userId: string, 
    reason?: string
  ): Promise<BlockResponse> => {
    const validUserId = validateUserId(userId);
    const validReason = reason ? validateContent(reason, 1, 200) : undefined;

    const request: BlockRequest = {
      target_user_id: validUserId,
      reason: validReason
    };

    const response = await api.post<ApiResponse<BlockResponse>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCK_USER, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取好友列表
   */
  const getFriendsList = async (
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<UserResponse>> => {
    // 发送前验证参数
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetFriendsListRequest = {
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<UserResponse>>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.FRIENDS_LIST, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取拉黑列表
   */
  const getBlockedList = async (
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<UserResponse>> => {
    // 发送前验证参数
    const validLimit = validatePaginationParams(cursor, limit).limit;

    const request: GetBlockedListRequest = {
      cursor,
      limit: validLimit
    };

    const response = await api.post<ApiResponse<PaginatedResponse<UserResponse>>>(
      buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCKED_LIST, false),
      request,
      { auth: true }
    );

    return response.data!;
  };

  /**
   * 获取好友请求列表（通过好友列表API实现）
   */
  const getFriendRequests = async (
    type: 'received' | 'sent' = 'received', 
    cursor?: string,
    limit: number = 20
  ): Promise<PaginatedResponse<UserResponse>> => {
    // 注意：这里需要根据实际后端API调整
    // 当前使用好友列表API，实际可能需要单独的好友请求API
    return await getFriendsList(cursor, limit);
  };

  /**
   * 获取待处理好友请求数量
   */
  const getPendingRequestsCount = async (): Promise<{ count: number }> => {
    try {
      // 获取好友请求列表的第一页
      const requests = await getFriendRequests('received', undefined, 1);
      // 这里应该有专门的计数API，暂时返回0
      return { count: 0 };
    } catch (error) {
      console.warn('获取待处理好友请求数量失败:', error);
      return { count: 0 };
    }
  };

  /**
   * 删除好友（通过拉黑实现）
   */
  const removeFriend = async (userId: string): Promise<BlockResponse> => {
    return await blockUser(userId, '删除好友');
  };

  /**
   * 获取用户关系状态
   */
  const getUserRelationshipStatus = (userId: string): Promise<{
    is_friend: boolean;
    is_following: boolean;
    is_blocked: boolean;
    is_blocked_by: boolean;
    has_pending_request: boolean;
  }> => {
    // 这里需要后端提供专门的关系状态查询API
    // 暂时返回默认状态
    return Promise.resolve({
      is_friend: false,
      is_following: false,
      is_blocked: false,
      is_blocked_by: false,
      has_pending_request: false
    });
  };

  return {
    // 核心API方法
    sendFriendRequest,
    acceptFriendRequest,
    rejectFriendRequest,
    followUser,
    unfollowUser,
    blockUser,
    getFriendsList,
    getBlockedList,
    
    // 兼容性方法
    getFriendRequests,
    getPendingRequestsCount,
    removeFriend,
    
    // 扩展方法
    getUserRelationshipStatus
  };
};
