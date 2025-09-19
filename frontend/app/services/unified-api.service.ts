/**
 * 统一API服务 - 与后端需求文档完全对齐
 * 基于 backend_doc.mdc 中的 API 类型定义
 * 使用统一的 XxxRequest/XxxResponse 命名规范
 */

import { 
  ApiResponse, 
  PaginatedResponse,
  // 认证相关类型
  GetNonceRequest, GetNonceResponse,
  RegisterRequest, RegisterResponse,
  LoginRequest, LoginResponse,
  CreateTempRequest, CreateTempResponse,
  RefreshTokenRequest, RefreshTokenResponse,
  CheckStatusRequest, CheckStatusResponse,
  SetPasswordRequest, SetPasswordResponse,
  // 用户相关类型
  SearchUserByNameRequest, SearchUserByNameResponse,
  SearchUserByIdRequest, SearchUserByIdResponse,
  SearchUserByLocationRequest, SearchUserByLocationResponse,
  UpdateLocationRequest, UpdateLocationResponse,
  UpdateProfileRequest, UpdateProfileResponse,
  UpgradeAccountRequest, UpgradeAccountResponse,
  GetSettingsRequest, GetSettingsResponse,
  UpdateSettingsRequest, UpdateSettingsResponse,
  DeleteAccountRequest, DeleteAccountResponse,
  // 用户关系管理类型
  AddFriendRequest, AddFriendResponse,
  AcceptFriendRequest, AcceptFriendResponse,
  RejectFriendRequest, RejectFriendResponse,
  FollowRequest, FollowResponse,
  UnfollowRequest, UnfollowResponse,
  BlockRequest, BlockResponse,
  GetFriendsListRequest, GetFriendsListResponse,
  GetBlockedListRequest, GetBlockedListResponse,
  // 群组管理类型
  SearchGroupByLocationRequest, SearchGroupByLocationResponse,
  SearchGroupByNameRequest, SearchGroupByNameResponse,
  SearchGroupByIdRequest, SearchGroupByIdResponse,
  GetMembersRequest, GetMembersResponse,
  CreateGroupRequest, CreateGroupResponse,
  JoinGroupRequest, JoinGroupResponse,
  LeaveGroupRequest, LeaveGroupResponse,
  KickGroupRequest, KickGroupResponse,
  TransferOwnershipRequest, TransferOwnershipResponse,
  MyGroupsRequest, MyGroupsResponse,
  // 统一消息类型
  SendMessageRequest, SendMessageResponse,
  GetHistoryRequest, GetHistoryResponse,
  DeleteMessageRequest, DeleteMessageResponse,
  MarkReadRequest, MarkReadResponse,
  UnreadCountRequest, UnreadCountResponse,
  ConversationsRequest, ConversationsResponse,
  // 生活瞬间类型
  CreateMomentRequest, CreateMomentResponse,
  SearchMomentsByLocationRequest, SearchMomentsByLocationResponse,
  SearchMomentsByTagsRequest, SearchMomentsByTagsResponse,
  SearchMomentByIdRequest, SearchMomentByIdResponse,
  GetMomentsHistoryRequest, GetMomentsHistoryResponse,
  DeleteMomentRequest, DeleteMomentResponse,
  LikeMomentRequest, LikeMomentResponse,
  UnlikeMomentRequest, UnlikeMomentResponse,
  // 评论互动类型
  AddCommentRequest, AddCommentResponse,
  ReplyCommentRequest, ReplyCommentResponse,
  DeleteCommentRequest, DeleteCommentResponse,
  GetCommentsRequest, GetCommentsResponse,
  // 内容安全类型
  CreateReportRequest, CreateReportResponse,
  GetMyReportsRequest, GetMyReportsResponse,
  GetReportsRequest, GetReportsResponse,
  HandleReportRequest, HandleReportResponse,
  BlockUserRequest, BlockUserResponse,
  // 通知推送类型
  SendNotificationRequest, SendNotificationResponse,
  GetNotificationsRequest, GetNotificationsResponse,
  MarkNotificationReadRequest, MarkNotificationReadResponse,
  GetUnreadNotificationCountRequest, GetUnreadNotificationCountResponse,
  GetNotificationSettingsRequest, GetNotificationSettingsResponse,
  UpdateNotificationSettingsRequest, UpdateNotificationSettingsResponse,
  // 系统状态类型
  HealthCheckRequest, HealthCheckResponse,
} from '~/types/api-unified'

import { 
  AUTH_ENDPOINTS, 
  USER_ENDPOINTS, 
  GROUP_ENDPOINTS, 
  CHAT_ENDPOINTS,
  MOMENT_ENDPOINTS,
  REPORT_ENDPOINTS,
  NOTIFICATION_ENDPOINTS,
  STATUS_ENDPOINTS,
  buildAuthUrl,
  buildUserUrl,
  buildGroupUrl,
  buildChatUrl,
  buildMomentUrl,
} from '~/config/api'

/**
 * 统一API服务类
 * 提供与后端需求文档完全对齐的API调用方法
 */
export class UnifiedApiService {
  private apiService: any // 注入现有的API服务

  constructor(apiService: any) {
    this.apiService = apiService
  }

  // ============================================================================
  // 认证API
  // ============================================================================

  /**
   * 获取认证随机数
   */
  async getNonce(request: GetNonceRequest): Promise<ApiResponse<GetNonceResponse>> {
    return this.apiService.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.GET_NONCE), request)
  }

  /**
   * 用户注册
   */
  async register(request: RegisterRequest): Promise<ApiResponse<RegisterResponse>> {
    return this.apiService.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.REGISTER), request)
  }

  /**
   * 用户登录
   */
  async login(request: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return this.apiService.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.LOGIN), request)
  }

  /**
   * 创建临时用户
   */
  async createTemp(request: CreateTempRequest): Promise<ApiResponse<CreateTempResponse>> {
    return this.apiService.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.CREATE_TEMP), request)
  }

  /**
   * 刷新Token
   */
  async refreshToken(request: RefreshTokenRequest): Promise<ApiResponse<RefreshTokenResponse>> {
    return this.apiService.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.REFRESH_TOKEN), request)
  }

  /**
   * 检查认证状态
   */
  async checkStatus(request: CheckStatusRequest): Promise<ApiResponse<CheckStatusResponse>> {
    return this.apiService.postPublic(buildAuthUrl(AUTH_ENDPOINTS.PUBLIC.CHECK_STATUS), request)
  }

  /**
   * 设置密码
   */
  async setPassword(request: SetPasswordRequest): Promise<ApiResponse<SetPasswordResponse>> {
    return this.apiService.post(buildAuthUrl(AUTH_ENDPOINTS.PROTECTED.SET_PASSWORD, true), request, { auth: true })
  }

  // ============================================================================
  // 用户API
  // ============================================================================

  /**
   * 按昵称搜索用户
   */
  async searchUserByName(request: SearchUserByNameRequest): Promise<ApiResponse<PaginatedResponse<SearchUserByNameResponse>>> {
    return this.apiService.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_NAME, true), request)
  }

  /**
   * 获取用户资料
   */
  async searchUserById(request: SearchUserByIdRequest): Promise<ApiResponse<SearchUserByIdResponse>> {
    return this.apiService.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true), request)
  }

  /**
   * 搜索附近用户
   */
  async searchUserByLocation(request: SearchUserByLocationRequest): Promise<ApiResponse<PaginatedResponse<SearchUserByLocationResponse>>> {
    return this.apiService.postPublic(buildUserUrl(USER_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), request)
  }

  /**
   * 更新用户位置
   */
  async updateLocation(request: UpdateLocationRequest): Promise<ApiResponse<UpdateLocationResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_LOCATION), request, { auth: true })
  }

  /**
   * 更新用户资料
   */
  async updateProfile(request: UpdateProfileRequest): Promise<ApiResponse<UpdateProfileResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_PROFILE), request, { auth: true })
  }

  /**
   * 账户升级
   */
  async upgradeAccount(request: UpgradeAccountRequest): Promise<ApiResponse<UpgradeAccountResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPGRADE_ACCOUNT), request, { auth: true })
  }

  /**
   * 获取用户设置
   */
  async getSettings(request: GetSettingsRequest): Promise<ApiResponse<GetSettingsResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.GET_SETTINGS), request, { auth: true })
  }

  /**
   * 更新用户设置
   */
  async updateSettings(request: UpdateSettingsRequest): Promise<ApiResponse<UpdateSettingsResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UPDATE_SETTINGS), request, { auth: true })
  }

  /**
   * 删除账户
   */
  async deleteAccount(request: DeleteAccountRequest): Promise<ApiResponse<DeleteAccountResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.DELETE_ACCOUNT), request, { auth: true })
  }

  // ============================================================================
  // 用户关系管理API
  // ============================================================================

  /**
   * 发送好友申请
   */
  async addFriend(request: AddFriendRequest): Promise<ApiResponse<AddFriendResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.ADD_FRIEND), request, { auth: true })
  }

  /**
   * 接受好友申请
   */
  async acceptFriend(request: AcceptFriendRequest): Promise<ApiResponse<AcceptFriendResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.ACCEPT_FRIEND), request, { auth: true })
  }

  /**
   * 拒绝好友申请
   */
  async rejectFriend(request: RejectFriendRequest): Promise<ApiResponse<RejectFriendResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.REJECT_FRIEND), request, { auth: true })
  }

  /**
   * 关注用户
   */
  async followUser(request: FollowRequest): Promise<ApiResponse<FollowResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.FOLLOW_USER), request, { auth: true })
  }

  /**
   * 取消关注
   */
  async unfollowUser(request: UnfollowRequest): Promise<ApiResponse<UnfollowResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.UNFOLLOW_USER), request, { auth: true })
  }

  /**
   * 拉黑用户
   */
  async blockUser(request: BlockRequest): Promise<ApiResponse<BlockResponse>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCK_USER), request, { auth: true })
  }

  /**
   * 获取好友列表
   */
  async getFriendsList(request: GetFriendsListRequest): Promise<ApiResponse<PaginatedResponse<GetFriendsListResponse>>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.FRIENDS_LIST), request, { auth: true })
  }

  /**
   * 获取拉黑列表
   */
  async getBlockedList(request: GetBlockedListRequest): Promise<ApiResponse<PaginatedResponse<GetBlockedListResponse>>> {
    return this.apiService.post(buildUserUrl(USER_ENDPOINTS.PROTECTED.BLOCKED_LIST), request, { auth: true })
  }

  // ============================================================================
  // 统一消息API
  // ============================================================================

  /**
   * 发送消息（统一接口，支持群聊和私聊）
   */
  async sendMessage(request: SendMessageRequest): Promise<ApiResponse<SendMessageResponse>> {
    return this.apiService.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.SEND), request, { auth: true })
  }

  /**
   * 获取消息历史（统一接口）
   */
  async getHistory(request: GetHistoryRequest): Promise<ApiResponse<PaginatedResponse<GetHistoryResponse>>> {
    return this.apiService.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.HISTORY), request, { auth: true })
  }

  /**
   * 删除消息（统一接口）
   */
  async deleteMessage(request: DeleteMessageRequest): Promise<ApiResponse<DeleteMessageResponse>> {
    return this.apiService.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.DELETE), request, { auth: true })
  }

  /**
   * 标记消息已读（统一接口）
   */
  async markRead(request: MarkReadRequest): Promise<ApiResponse<MarkReadResponse>> {
    return this.apiService.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.MARK_READ), request, { auth: true })
  }

  /**
   * 获取未读消息数（统一接口）
   */
  async getUnreadCount(request: UnreadCountRequest): Promise<ApiResponse<UnreadCountResponse>> {
    return this.apiService.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.UNREAD_COUNT), request, { auth: true })
  }

  /**
   * 获取会话列表（统一接口）
   */
  async getConversations(request: ConversationsRequest): Promise<ApiResponse<PaginatedResponse<ConversationsResponse>>> {
    return this.apiService.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.CONVERSATIONS), request, { auth: true })
  }

  // ============================================================================
  // 群组管理API
  // ============================================================================

  /**
   * 按位置搜索群组
   */
  async searchGroupByLocation(request: SearchGroupByLocationRequest): Promise<ApiResponse<PaginatedResponse<SearchGroupByLocationResponse>>> {
    return this.apiService.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), request)
  }

  /**
   * 按名称搜索群组
   */
  async searchGroupByName(request: SearchGroupByNameRequest): Promise<ApiResponse<PaginatedResponse<SearchGroupByNameResponse>>> {
    return this.apiService.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.SEARCH_BY_NAME, true), request)
  }

  /**
   * 按ID获取群组详情
   */
  async searchGroupById(request: SearchGroupByIdRequest): Promise<ApiResponse<SearchGroupByIdResponse>> {
    return this.apiService.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true), request)
  }

  /**
   * 获取群组成员
   */
  async getMembers(request: GetMembersRequest): Promise<ApiResponse<PaginatedResponse<GetMembersResponse>>> {
    return this.apiService.postPublic(buildGroupUrl(GROUP_ENDPOINTS.PUBLIC.MEMBERS, true), request)
  }

  /**
   * 创建群组
   */
  async createGroup(request: CreateGroupRequest): Promise<ApiResponse<CreateGroupResponse>> {
    return this.apiService.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.CREATE), request, { auth: true })
  }

  /**
   * 加入群组
   */
  async joinGroup(request: JoinGroupRequest): Promise<ApiResponse<JoinGroupResponse>> {
    return this.apiService.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.JOIN), request, { auth: true })
  }

  /**
   * 离开群组
   */
  async leaveGroup(request: LeaveGroupRequest): Promise<ApiResponse<LeaveGroupResponse>> {
    return this.apiService.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.LEAVE), request, { auth: true })
  }

  /**
   * 踢出群组成员
   */
  async kickGroup(request: KickGroupRequest): Promise<ApiResponse<KickGroupResponse>> {
    return this.apiService.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.KICK), request, { auth: true })
  }

  /**
   * 转移群组所有权
   */
  async transferOwnership(request: TransferOwnershipRequest): Promise<ApiResponse<TransferOwnershipResponse>> {
    return this.apiService.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.TRANSFER_OWNERSHIP), request, { auth: true })
  }

  /**
   * 获取用户参与的群组列表
   */
  async getMyGroups(request: MyGroupsRequest): Promise<ApiResponse<PaginatedResponse<MyGroupsResponse>>> {
    return this.apiService.post(buildGroupUrl(GROUP_ENDPOINTS.PROTECTED.MY_GROUPS), request, { auth: true })
  }

  // ============================================================================
  // 生活瞬间API
  // ============================================================================

  /**
   * 创建生活瞬间
   */
  async createMoment(request: CreateMomentRequest): Promise<ApiResponse<CreateMomentResponse>> {
    return this.apiService.post(buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.CREATE), request, { auth: true })
  }

  /**
   * 按位置搜索生活瞬间
   */
  async searchMomentsByLocation(request: SearchMomentsByLocationRequest): Promise<ApiResponse<PaginatedResponse<SearchMomentsByLocationResponse>>> {
    return this.apiService.postPublic(buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.SEARCH_BY_LOCATION, true), request)
  }

  /**
   * 按标签搜索生活瞬间
   */
  async searchMomentsByTags(request: SearchMomentsByTagsRequest): Promise<ApiResponse<PaginatedResponse<SearchMomentsByTagsResponse>>> {
    return this.apiService.postPublic(buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.SEARCH_BY_TAGS, true), request)
  }

  /**
   * 按ID搜索生活瞬间
   */
  async searchMomentById(request: SearchMomentByIdRequest): Promise<ApiResponse<SearchMomentByIdResponse>> {
    return this.apiService.postPublic(buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.SEARCH_BY_ID, true), request)
  }

  /**
   * 获取生活瞬间历史
   */
  async getMomentsHistory(request: GetMomentsHistoryRequest): Promise<ApiResponse<PaginatedResponse<GetMomentsHistoryResponse>>> {
    return this.apiService.postPublic(buildMomentUrl(MOMENT_ENDPOINTS.PUBLIC.HISTORY, true), request)
  }

  /**
   * 删除生活瞬间
   */
  async deleteMoment(request: DeleteMomentRequest): Promise<ApiResponse<DeleteMomentResponse>> {
    return this.apiService.post(buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.DELETE), request, { auth: true })
  }

  /**
   * 点赞生活瞬间
   */
  async likeMoment(request: LikeMomentRequest): Promise<ApiResponse<LikeMomentResponse>> {
    return this.apiService.post(buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.LIKE), request, { auth: true })
  }

  /**
   * 取消点赞
   */
  async unlikeMoment(request: UnlikeMomentRequest): Promise<ApiResponse<UnlikeMomentResponse>> {
    return this.apiService.post(buildMomentUrl(MOMENT_ENDPOINTS.PROTECTED.UNLIKE), request, { auth: true })
  }
}

/**
 * 创建统一API服务实例
 */
export function createUnifiedApiService(apiService: any): UnifiedApiService {
  return new UnifiedApiService(apiService)
}

/**
 * 导出统一API服务的单例实例
 */
export const unifiedApiService = createUnifiedApiService(null) // 需要在运行时注入apiService
