/**
 * 完整的 API 客户端
 * 使用后端文档定义的完整类型系统
 */

import { HttpUtil } from '../network/HttpUtil';
import {
  // 基础类型
  ApiResponse,
  PaginatedResponse,
  BasicSuccess,

  // 认证相关
  GetNonceRequest,
  GetNonceResponse,
  RegisterRequest,
  RegisterResponse,
  LoginRequest,
  LoginResponse,
  CreateTempRequest,
  CreateTempResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  CheckStatusRequest,
  CheckStatusResponse,
  SetPasswordRequest,
  SetPasswordResponse,

  // 用户管理
  SearchUserByNameRequest,
  SearchUserByNameResponse,
  SearchUserByIdRequest,
  SearchUserByIdResponse,
  SearchUserByLocationRequest,
  SearchUserByLocationResponse,
  UpdateLocationRequest,
  UpdateLocationResponse,
  UpdateProfileRequest,
  UpdateProfileResponse,
  UpgradeAccountRequest,
  UpgradeAccountResponse,
  GetSettingsRequest,
  GetSettingsResponse,
  UpdateSettingsRequest,
  UpdateSettingsResponse,
  DeleteAccountRequest,
  DeleteAccountResponse,

  // 用户关系管理
  AddFriendRequest,
  AddFriendResponse,
  AcceptFriendRequest,
  AcceptFriendResponse,
  RejectFriendRequest,
  RejectFriendResponse,
  FollowUserRequest,
  FollowUserResponse,
  UnfollowUserRequest,
  UnfollowUserResponse,
  BlockUserRequest,
  BlockUserResponse,
  GetFriendsListRequest,
  GetFriendsListResponse,
  GetBlockedListRequest,
  GetBlockedListResponse,

  // 群组管理
  SearchGroupByLocationRequest,
  SearchGroupByLocationResponse,
  SearchGroupByNameRequest,
  SearchGroupByNameResponse,
  SearchGroupByIdRequest,
  SearchGroupByIdResponse,
  GetMembersRequest,
  GetMembersResponse,
  CreateGroupRequest,
  CreateGroupResponse,
  JoinGroupRequest,
  JoinGroupResponse,
  LeaveGroupRequest,
  LeaveGroupResponse,
  KickMemberRequest,
  KickMemberResponse,
  TransferOwnershipRequest,
  TransferOwnershipResponse,
  GetMyGroupsRequest,
  GetMyGroupsResponse,

  // 消息通讯
  SendMessageRequest,
  SendMessageResponse,
  GetHistoryRequest,
  GetHistoryResponse,
  DeleteMessageRequest,
  DeleteMessageResponse,
  MarkMessagesReadRequest,
  MarkMessagesReadResponse,
  GetUnreadCountRequest,
  GetUnreadCountResponse,
  GetConversationsRequest,
  GetConversationsResponse,

  // 生活瞬间
  CreateMomentRequest,
  CreateMomentResponse,
  SearchMomentsByLocationRequest,
  SearchMomentsByLocationResponse,
  LikeMomentRequest,
  LikeMomentResponse,
  UnlikeMomentRequest,
  UnlikeMomentResponse,
  DeleteMomentRequest,
  DeleteMomentResponse,
  SearchMomentsByTagRequest,
  SearchMomentsByTagResponse,
  SearchMomentByIdRequest,
  SearchMomentByIdResponse,
  GetMomentsHistoryRequest,
  GetMomentsHistoryResponse,

  // 评论互动
  AddCommentRequest,
  AddCommentResponse,
  ReplyCommentRequest,
  ReplyCommentResponse,
  DeleteCommentRequest,
  DeleteCommentResponse,
  GetCommentsListRequest,
  GetCommentsListResponse,

  // 通知推送
  GetNotificationsRequest,
  GetNotificationsResponse,
  MarkNotificationsReadRequest,
  MarkNotificationsReadResponse,

  // 内容安全
  ReportContentRequest,
  ReportContentResponse,

  // 系统状态
  HealthCheckRequest,
  HealthCheckResponse
} from '../types/BackendApiTypes';

export class ApiClient {
  // ==================== 系统状态 API ====================

  /**
   * 系统健康检查
   */
  static async healthCheck(): Promise<ApiResponse<HealthCheckResponse>> {
    const request = {};
    return HttpUtil.post<ApiResponse<HealthCheckResponse>, HealthCheckRequest>('/api/v1/public/status/health', request);
  }

  // ==================== 认证相关 API ====================

  /**
   * 获取认证随机数
   */
  static async getNonce(): Promise<ApiResponse<GetNonceResponse>> {
    const request = {};
    return HttpUtil.post<ApiResponse<GetNonceResponse>, GetNonceRequest>('/api/v1/public/auth/get_nonce', request);
  }

  /**
   * 用户注册
   */
  static async register(params: RegisterRequest): Promise<ApiResponse<RegisterResponse>> {
    return HttpUtil.post<ApiResponse<RegisterResponse>, RegisterRequest>('/api/v1/public/auth/register', params);
  }

  /**
   * 用户登录
   */
  static async login(params: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return HttpUtil.post<ApiResponse<LoginResponse>, LoginRequest>('/api/v1/public/auth/login', params);
  }

  /**
   * 创建临时用户
   */
  static async createTemp(params: CreateTempRequest): Promise<ApiResponse<CreateTempResponse>> {
    return HttpUtil.post<ApiResponse<CreateTempResponse>, CreateTempRequest>('/api/v1/public/auth/create_temp', params);
  }

  /**
   * 刷新Token
   */
  static async refreshToken(params: RefreshTokenRequest): Promise<ApiResponse<RefreshTokenResponse>> {
    return HttpUtil.post<ApiResponse<RefreshTokenResponse>, RefreshTokenRequest>('/api/v1/public/auth/refresh_token',
      params);
  }

  /**
   * 检查认证状态
   */
  static async checkAuthStatus(): Promise<ApiResponse<CheckStatusResponse>> {
    const request = {};
    return HttpUtil.post<ApiResponse<CheckStatusResponse>, CheckStatusRequest>('/api/v1/public/auth/check_status',
      request);
  }

  /**
   * 设置密码
   */
  static async setPassword(params: SetPasswordRequest): Promise<ApiResponse<SetPasswordResponse>> {
    return HttpUtil.post<ApiResponse<SetPasswordResponse>, SetPasswordRequest>('/api/v1/protected/auth/set_password',
      params);
  }

  // ==================== 用户管理 API ====================

  /**
   * 按昵称搜索用户
   */
  static async searchUserByName(params: SearchUserByNameRequest): Promise<ApiResponse<PaginatedResponse<SearchUserByNameResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<SearchUserByNameResponse>>, SearchUserByNameRequest>('/api/v1/public/user/search_by_name',
      params);
  }

  /**
   * 按ID获取用户资料
   */
  static async searchUserById(params: SearchUserByIdRequest): Promise<ApiResponse<SearchUserByIdResponse>> {
    return HttpUtil.post<ApiResponse<SearchUserByIdResponse>, SearchUserByIdRequest>('/api/v1/public/user/search_by_id',
      params);
  }

  /**
   * 搜索附近用户
   */
  static async searchUserByLocation(params: SearchUserByLocationRequest): Promise<ApiResponse<PaginatedResponse<SearchUserByLocationResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<SearchUserByLocationResponse>>, SearchUserByLocationRequest>('/api/v1/public/user/search_by_location',
      params);
  }

  /**
   * 更新用户位置
   */
  static async updateLocation(params: UpdateLocationRequest): Promise<ApiResponse<UpdateLocationResponse>> {
    return HttpUtil.post<ApiResponse<UpdateLocationResponse>, UpdateLocationRequest>('/api/v1/protected/user/update_location',
      params);
  }

  /**
   * 更新用户资料
   */
  static async updateProfile(params: UpdateProfileRequest): Promise<ApiResponse<UpdateProfileResponse>> {
    return HttpUtil.post<ApiResponse<UpdateProfileResponse>, UpdateProfileRequest>('/api/v1/protected/user/update_profile',
      params);
  }

  /**
   * 账户升级
   */
  static async upgradeAccount(params: UpgradeAccountRequest): Promise<ApiResponse<UpgradeAccountResponse>> {
    return HttpUtil.post<ApiResponse<UpgradeAccountResponse>, UpgradeAccountRequest>('/api/v1/protected/user/upgrade_account',
      params);
  }

  /**
   * 获取用户设置
   */
  static async getSettings(): Promise<ApiResponse<GetSettingsResponse>> {
    const request = {};
    return HttpUtil.post<ApiResponse<GetSettingsResponse>, GetSettingsRequest>('/api/v1/protected/user/get_settings',
      request);
  }

  /**
   * 更新用户设置
   */
  static async updateSettings(params: UpdateSettingsRequest): Promise<ApiResponse<UpdateSettingsResponse>> {
    return HttpUtil.post<ApiResponse<UpdateSettingsResponse>, UpdateSettingsRequest>('/api/v1/protected/user/update_settings',
      params);
  }

  /**
   * 删除账户
   */
  static async deleteAccount(params: DeleteAccountRequest): Promise<ApiResponse<DeleteAccountResponse>> {
    return HttpUtil.post<ApiResponse<DeleteAccountResponse>, DeleteAccountRequest>('/api/v1/protected/user/delete_account',
      params);
  }

  // ==================== 用户关系管理 API ====================

  /**
   * 发送好友申请
   */
  static async addFriend(params: AddFriendRequest): Promise<ApiResponse<AddFriendResponse>> {
    return HttpUtil.post<ApiResponse<AddFriendResponse>, AddFriendRequest>('/api/v1/protected/user/add_friend', params);
  }

  /**
   * 接受好友申请
   */
  static async acceptFriend(params: AcceptFriendRequest): Promise<ApiResponse<AcceptFriendResponse>> {
    return HttpUtil.post<ApiResponse<AcceptFriendResponse>, AcceptFriendRequest>('/api/v1/protected/user/accept_friend', params);
  }

  /**
   * 拒绝好友申请
   */
  static async rejectFriend(params: RejectFriendRequest): Promise<ApiResponse<RejectFriendResponse>> {
    return HttpUtil.post<ApiResponse<RejectFriendResponse>, RejectFriendRequest>('/api/v1/protected/user/reject_friend', params);
  }

  /**
   * 关注用户
   */
  static async followUser(params: FollowUserRequest): Promise<ApiResponse<FollowUserResponse>> {
    return HttpUtil.post<ApiResponse<FollowUserResponse>, FollowUserRequest>('/api/v1/protected/user/follow_user', params);
  }

  /**
   * 取消关注
   */
  static async unfollowUser(params: UnfollowUserRequest): Promise<ApiResponse<UnfollowUserResponse>> {
    return HttpUtil.post<ApiResponse<UnfollowUserResponse>, UnfollowUserRequest>('/api/v1/protected/user/unfollow_user', params);
  }

  /**
   * 拉黑用户
   */
  static async blockUser(params: BlockUserRequest): Promise<ApiResponse<BlockUserResponse>> {
    return HttpUtil.post<ApiResponse<BlockUserResponse>, BlockUserRequest>('/api/v1/protected/user/block_user', params);
  }

  /**
   * 获取好友列表
   */
  static async friendsList(params: GetFriendsListRequest): Promise<ApiResponse<PaginatedResponse<GetFriendsListResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetFriendsListResponse>>, GetFriendsListRequest>('/api/v1/protected/user/friends_list', params);
  }

  /**
   * 获取拉黑列表
   */
  static async blockedList(params: GetBlockedListRequest): Promise<ApiResponse<PaginatedResponse<GetBlockedListResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetBlockedListResponse>>, GetBlockedListRequest>('/api/v1/protected/user/blocked_list', params);
  }

  // ==================== 群组管理 API ====================

  /**
   * 按位置搜索群组
   */
  static async searchGroupByLocation(params: SearchGroupByLocationRequest): Promise<ApiResponse<PaginatedResponse<SearchGroupByLocationResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<SearchGroupByLocationResponse>>, SearchGroupByLocationRequest>('/api/v1/public/group/search_by_location',
      params);
  }

  /**
   * 按名称搜索群组
   */
  static async searchGroupByName(params: SearchGroupByNameRequest): Promise<ApiResponse<PaginatedResponse<SearchGroupByNameResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<SearchGroupByNameResponse>>, SearchGroupByNameRequest>('/api/v1/public/group/search_by_name',
      params);
  }

  /**
   * 按ID获取群组详情
   */
  static async searchGroupById(params: SearchGroupByIdRequest): Promise<ApiResponse<SearchGroupByIdResponse>> {
    return HttpUtil.post<ApiResponse<SearchGroupByIdResponse>, SearchGroupByIdRequest>('/api/v1/public/group/search_by_id',
      params);
  }

  /**
   * 获取群组成员
   */
  static async getMembers(params: GetMembersRequest): Promise<ApiResponse<PaginatedResponse<GetMembersResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetMembersResponse>>, GetMembersRequest>('/api/v1/public/group/members',
      params);
  }

  /**
   * 创建群组
   */
  static async createGroup(params: CreateGroupRequest): Promise<ApiResponse<CreateGroupResponse>> {
    return HttpUtil.post<ApiResponse<CreateGroupResponse>, CreateGroupRequest>('/api/v1/protected/group/create',
      params);
  }

  /**
   * 加入群组
   */
  static async joinGroup(params: JoinGroupRequest): Promise<ApiResponse<JoinGroupResponse>> {
    return HttpUtil.post<ApiResponse<JoinGroupResponse>, JoinGroupRequest>('/api/v1/protected/group/join', params);
  }

  /**
   * 离开群组
   */
  static async leaveGroup(params: LeaveGroupRequest): Promise<ApiResponse<LeaveGroupResponse>> {
    return HttpUtil.post<ApiResponse<LeaveGroupResponse>, LeaveGroupRequest>('/api/v1/protected/group/leave', params);
  }

  /**
   * 踢出群组成员
   */
  static async kickMember(params: KickMemberRequest): Promise<ApiResponse<KickMemberResponse>> {
    return HttpUtil.post<ApiResponse<KickMemberResponse>, KickMemberRequest>('/api/v1/protected/group/kick', params);
  }

  /**
   * 转移群组所有权
   */
  static async transferOwnership(params: TransferOwnershipRequest): Promise<ApiResponse<TransferOwnershipResponse>> {
    return HttpUtil.post<ApiResponse<TransferOwnershipResponse>, TransferOwnershipRequest>('/api/v1/protected/group/transfer_ownership', params);
  }

  /**
   * 获取用户参与的群组列表
   */
  static async myGroups(params: GetMyGroupsRequest): Promise<ApiResponse<PaginatedResponse<GetMyGroupsResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetMyGroupsResponse>>, GetMyGroupsRequest>('/api/v1/protected/group/my_groups', params);
  }

  // ==================== 消息通讯 API（统一架构）====================

  /**
   * 发送消息（统一接口）
   */
  static async sendMessage(params: SendMessageRequest): Promise<ApiResponse<SendMessageResponse>> {
    return HttpUtil.post<ApiResponse<SendMessageResponse>, SendMessageRequest>('/api/v1/protected/chat/send', params);
  }

  /**
   * 获取消息历史（统一接口）
   */
  static async getHistory(params: GetHistoryRequest): Promise<ApiResponse<PaginatedResponse<GetHistoryResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetHistoryResponse>>, GetHistoryRequest>('/api/v1/protected/chat/history',
      params);
  }

  /**
   * 删除消息
   */
  static async deleteMessage(params: DeleteMessageRequest): Promise<ApiResponse<DeleteMessageResponse>> {
    return HttpUtil.post<ApiResponse<DeleteMessageResponse>, DeleteMessageRequest>('/api/v1/protected/chat/delete', params);
  }

  /**
   * 标记消息已读
   */
  static async markRead(params: MarkMessagesReadRequest): Promise<ApiResponse<MarkMessagesReadResponse>> {
    return HttpUtil.post<ApiResponse<MarkMessagesReadResponse>, MarkMessagesReadRequest>('/api/v1/protected/chat/mark-read', params);
  }

  /**
   * 获取未读消息数
   */
  static async unreadCount(params: GetUnreadCountRequest): Promise<ApiResponse<GetUnreadCountResponse>> {
    return HttpUtil.post<ApiResponse<GetUnreadCountResponse>, GetUnreadCountRequest>('/api/v1/protected/chat/unread_count', params);
  }

  /**
   * 获取会话列表
   */
  static async conversations(params: GetConversationsRequest): Promise<ApiResponse<PaginatedResponse<GetConversationsResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetConversationsResponse>>, GetConversationsRequest>('/api/v1/protected/chat/conversations', params);
  }

  // ==================== 生活瞬间 API ====================

  /**
   * 创建生活瞬间
   */
  static async createMoment(params: CreateMomentRequest): Promise<ApiResponse<CreateMomentResponse>> {
    return HttpUtil.post<ApiResponse<CreateMomentResponse>, CreateMomentRequest>('/api/v1/protected/moments/create',
      params);
  }

  /**
   * 按位置搜索生活瞬间
   */
  static async searchMomentByLocation(params: SearchMomentsByLocationRequest): Promise<ApiResponse<PaginatedResponse<SearchMomentsByLocationResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<SearchMomentsByLocationResponse>>, SearchMomentsByLocationRequest>('/api/v1/public/moments/search_by_location',
      params);
  }

  /**
   * 点赞生活瞬间
   */
  static async likeMoment(params: LikeMomentRequest): Promise<ApiResponse<LikeMomentResponse>> {
    return HttpUtil.post<ApiResponse<LikeMomentResponse>, LikeMomentRequest>('/api/v1/protected/moments/like', params);
  }

  /**
   * 取消点赞生活瞬间
   */
  static async unlikeMoment(params: UnlikeMomentRequest): Promise<ApiResponse<UnlikeMomentResponse>> {
    return HttpUtil.post<ApiResponse<UnlikeMomentResponse>, UnlikeMomentRequest>('/api/v1/protected/moments/unlike',
      params);
  }

  /**
   * 删除生活瞬间
   */
  static async deleteMoment(params: DeleteMomentRequest): Promise<ApiResponse<DeleteMomentResponse>> {
    return HttpUtil.post<ApiResponse<DeleteMomentResponse>, DeleteMomentRequest>('/api/v1/protected/moments/delete', params);
  }

  /**
   * 按标签搜索生活瞬间
   */
  static async searchMomentByTag(params: SearchMomentsByTagRequest): Promise<ApiResponse<PaginatedResponse<SearchMomentsByTagResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<SearchMomentsByTagResponse>>, SearchMomentsByTagRequest>('/api/v1/public/moments/search_by_tag', params);
  }

  /**
   * 按ID获取生活瞬间详情
   */
  static async searchMomentById(params: SearchMomentByIdRequest): Promise<ApiResponse<SearchMomentByIdResponse>> {
    return HttpUtil.post<ApiResponse<SearchMomentByIdResponse>, SearchMomentByIdRequest>('/api/v1/public/moments/search_by_id', params);
  }

  /**
   * 获取用户历史瞬间
   */
  static async momentsHistory(params: GetMomentsHistoryRequest): Promise<ApiResponse<PaginatedResponse<GetMomentsHistoryResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetMomentsHistoryResponse>>, GetMomentsHistoryRequest>('/api/v1/public/moments/history', params);
  }

  // ==================== 评论互动 API ====================

  /**
   * 添加评论
   */
  static async addComment(params: AddCommentRequest): Promise<ApiResponse<AddCommentResponse>> {
    return HttpUtil.post<ApiResponse<AddCommentResponse>, AddCommentRequest>('/api/v1/protected/comments/add', params);
  }

  /**
   * 回复评论
   */
  static async replyComment(params: ReplyCommentRequest): Promise<ApiResponse<ReplyCommentResponse>> {
    return HttpUtil.post<ApiResponse<ReplyCommentResponse>, ReplyCommentRequest>('/api/v1/protected/comments/reply', params);
  }

  /**
   * 删除评论
   */
  static async deleteComment(params: DeleteCommentRequest): Promise<ApiResponse<DeleteCommentResponse>> {
    return HttpUtil.post<ApiResponse<DeleteCommentResponse>, DeleteCommentRequest>('/api/v1/protected/comments/delete', params);
  }

  /**
   * 获取评论列表
   */
  static async commentsList(params: GetCommentsListRequest): Promise<ApiResponse<PaginatedResponse<GetCommentsListResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetCommentsListResponse>>, GetCommentsListRequest>('/api/v1/public/comments/list', params);
  }

  // ==================== 通知推送 API ====================

  /**
   * 获取通知列表
   */
  static async notifications(params: GetNotificationsRequest): Promise<ApiResponse<PaginatedResponse<GetNotificationsResponse>>> {
    return HttpUtil.post<ApiResponse<PaginatedResponse<GetNotificationsResponse>>, GetNotificationsRequest>('/api/v1/protected/notifications/list', params);
  }

  /**
   * 标记通知已读
   */
  static async markNotificationRead(params: MarkNotificationsReadRequest): Promise<ApiResponse<MarkNotificationsReadResponse>> {
    return HttpUtil.post<ApiResponse<MarkNotificationsReadResponse>, MarkNotificationsReadRequest>('/api/v1/protected/notifications/mark_read', params);
  }

  // ==================== 内容安全 API ====================

  /**
   * 举报内容
   */
  static async reportContent(params: ReportContentRequest): Promise<ApiResponse<ReportContentResponse>> {
    return HttpUtil.post<ApiResponse<ReportContentResponse>, ReportContentRequest>('/api/v1/protected/security/report', params);
  }

  // ==================== 系统状态 API ====================

}