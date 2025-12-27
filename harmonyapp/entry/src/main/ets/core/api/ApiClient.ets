/**
 * API 客户端
 * 严格按照 api_spec.mdc 定义的 API 端点实现
 */

import { HttpUtil } from '../network/HttpUtil';
import common from '@ohos.app.ability.common';
import {
  // 基础类型
  ApiResponse,
  PaginatedData,
  UploadMediaData,

  // 认证相关
  AppAuthData,
  RegisterRequest,
  LoginRequest,
  CreateTempRequest,
  RefreshTokenRequest,
  CheckStatusData,
  RequestPasswordResetRequest,
  RequestPasswordResetData,
  VerifyResetCodeRequest,
  VerifyResetCodeData,
  ResetPasswordRequest,
  ChangePasswordRequest,
  GetSessionsRequest,
  SessionItem,
  RevokeSessionRequest,

  // 用户管理
  UserSearchItem,
  SearchUsersByKeywordRequest,
  SearchUsersByTagRequest,
  SearchUsersByLocationRequest,
  UserOpenTheDoorRequest,
  GetRelationshipRequest,
  UserRelationship,
  GetUserStatsRequest,
  UserStatsData,
  MyProfileData,
  UpdateProfileRequest,
  UpdateLocationRequest,
  UpgradeAccountRequest,
  UserSettingsData,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  FollowUserRequest,
  UnfollowUserRequest,
  BlockUserRequest,
  UnblockUserRequest,
  FriendListItem,
  FollowingItem,
  FollowerItem,
  BlockedUserItem,
  SetFriendRemarkRequest,
  GetUserFriendsListRequest,
  GetUserFollowingListRequest,
  GetUserFollowersListRequest,

  // 生活瞬间
  MomentSearchItem,
  SearchMomentsByKeywordRequest,
  SearchMomentsByTagRequest,
  SearchMomentsByLocationRequest,
  MomentOpenTheDoorRequest,
  GetUserMomentsRequest,
  CreateMomentRequest,
  CreateMomentData,
  EditMomentRequest,
  DeleteMomentRequest,
  LikeMomentRequest,
  UnlikeMomentRequest,
  FavoriteMomentRequest,
  UnfavoriteMomentRequest,
  CheckMomentLikedRequest,
  CheckLikedData,
  CheckMomentFavoritedRequest,
  CheckFavoritedData,
  GetMomentLikesRequest,
  MomentLikeItem,

  // 评论
  CreateCommentRequest,
  CreateCommentData,
  DeleteCommentRequest,
  GetCommentsRequest,
  CommentListItem,
  GetRepliesRequest,
  LikeCommentRequest,
  UnlikeCommentRequest,
  GetCommentLikesRequest,
  CommentLikeItem,

  // 私聊管理
  CreatePrivateChatRequest,
  CreatePrivateChatData,
  GetPrivateChatsRequest,
  PrivateChatListItem,
  PrivateChatOpenTheDoorRequest,
  PrivateChatDetailData,
  UpdatePrivateChatSettingsRequest,
  DeletePrivateChatRequest,

  // 群组管理
  GroupSearchItem,
  SearchGroupsByKeywordRequest,
  SearchGroupsByTagRequest,
  SearchGroupsByLocationRequest,
  GroupOpenTheDoorRequest,
  GetGroupSettingsRequest,
  GroupSettingsData,
  CreateGroupChatRequest,
  CreateGroupChatData,
  UpdateGroupChatRequest,
  DeleteGroupChatRequest,
  GetMembersRequest,
  GroupMemberItem,
  InviteMemberRequest,
  InviteMemberData,
  JoinGroupRequest,
  ApplyToGroupRequest,
  ApplyToGroupData,
  SetMemberRoleRequest,
  MuteMemberRequest,
  UnmuteMemberRequest,
  BanMemberRequest,
  UnbanMemberRequest,
  GetBannedMembersRequest,
  BannedMemberItem,
  RemoveMemberRequest,
  GetGroupApplicationsRequest,
  GroupApplicationItem,
  ApproveGroupApplicationRequest,
  RejectGroupApplicationRequest,
  TransferOwnershipRequest,
  GetJoinedGroupChatsRequest,
  JoinedGroupItem,
  CancelGroupApplicationRequest,
  GetGroupInvitesRequest,
  GroupInviteItem,
  AcceptGroupInviteRequest,
  RejectGroupInviteRequest,
  LeaveGroupRequest,

  // 私聊消息
  SendPrivateMessageRequest,
  SendPrivateMessageData,
  GetPrivateMessagesRequest,
  PrivateMessageItem,
  UpdatePrivateMessageRequest,
  RecallPrivateMessageRequest,
  DeletePrivateMessageRequest,
  MarkPrivateMessagesReadRequest,
  AddPrivateMessageReactionRequest,
  RemovePrivateMessageReactionRequest,
  GetPrivateMessageReactionsRequest,
  MessageReactionItem,

  // 群聊消息
  SendGroupMessageRequest,
  SendGroupMessageData,
  GetGroupMessagesRequest,
  GroupMessageItem,
  UpdateGroupMessageRequest,
  RecallGroupMessageRequest,
  DeleteGroupMessageRequest,
  MarkGroupMessagesReadRequest,
  GetMessageReadStatusRequest,
  MessageReadItem,
  AddGroupMessageReactionRequest,
  RemoveGroupMessageReactionRequest,
  GetGroupMessageReactionsRequest,

  // 举报
  CreateReportRequest,
  CreateReportData,
  GetMyReportsRequest,
  ReportItem,

  // 订阅设置
  SubscriptionsData,
  UpdateSubscriptionsRequest,

  // 设备推送
  RegisterDeviceTokenRequest,
  UnregisterDeviceTokenRequest,
  GetDeviceTokensRequest,
  DeviceTokenItem,

  // 系统健康
  HealthCheckData,

  // 信息流
  GetFeedMomentsRequest,
  FeedMomentItem,
  GetFeedGroupsRequest,
  FeedGroupItem,
  GetFeedUsersRequest,
  FeedUserItem,
  GetFeedTrendingRequest,
  FeedTrendingData,
  FeedFeedbackRequest,

  // 标签
  GetTrendingTagsRequest,
  TrendingTagsData,
  SuggestTagsRequest,
  TagSuggestData,
  UserInfo,
} from '../types/api';

export class ApiClient {
  // ==================== 文件上传 API ====================
  /**
   * 上传文件
   */
  static async uploadFile(
    context: common.UIAbilityContext,
    fileUri: string,
    category: UploadCategory
  ): Promise<UploadMediaData> {
    return HttpUtil.uploadFile(context, fileUri, category);
  }

  // ==================== 系统健康检查 API ====================

  /**
   * 系统健康检查
   */
  static async healthCheck(): Promise<ApiResponse<HealthCheckData>> {
    return HttpUtil.post<ApiResponse<HealthCheckData>, object>('/api/v1/public/system/health', {});
  }

  // ==================== 认证 API ====================

  /**
   * 用户注册
   */
  static async register(params: RegisterRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, RegisterRequest>('/api/v1/public/auth/register', params);
  }

  /**
   * 用户登录
   */
  static async login(params: LoginRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, LoginRequest>('/api/v1/public/auth/login', params);
  }

  /**
   * 创建临时用户
   */
  static async createTempUser(params: CreateTempRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, CreateTempRequest>('/api/v1/public/auth/guest', params);
  }

  /**
   * 刷新Token
   */
  static async refreshToken(params: RefreshTokenRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, RefreshTokenRequest>('/api/v1/public/auth/refresh_token', params);
  }

  /**
   * 检查认证状态
   */
  static async checkAuthStatus(): Promise<ApiResponse<CheckStatusData>> {
    return HttpUtil.post<ApiResponse<CheckStatusData>, object>('/api/v1/protected/auth/check_status', {});
  }

  /**
   * 用户登出
   */
  static async logout(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, object>('/api/v1/protected/auth/logout', {});
  }

  /**
   * 请求密码重置
   */
  static async requestPasswordReset(params: RequestPasswordResetRequest): Promise<ApiResponse<RequestPasswordResetData>> {
    return HttpUtil.post<ApiResponse<RequestPasswordResetData>, RequestPasswordResetRequest>('/api/v1/public/auth/request_password_reset', params);
  }

  /**
   * 验证重置码
   */
  static async verifyResetCode(params: VerifyResetCodeRequest): Promise<ApiResponse<VerifyResetCodeData>> {
    return HttpUtil.post<ApiResponse<VerifyResetCodeData>, VerifyResetCodeRequest>('/api/v1/public/auth/verify_reset_code', params);
  }

  /**
   * 重置密码
   */
  static async resetPassword(params: ResetPasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ResetPasswordRequest>('/api/v1/public/auth/reset_password', params);
  }

  /**
   * 修改密码
   */
  static async changePassword(params: ChangePasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ChangePasswordRequest>('/api/v1/protected/auth/me/change_password', params);
  }

  /**
   * 获取活跃会话列表
   */
  static async getSessions(params: GetSessionsRequest): Promise<ApiResponse<PaginatedData<SessionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<SessionItem>>, GetSessionsRequest>('/api/v1/protected/auth/me/sessions', params);
  }

  /**
   * 撤销会话
   */
  static async revokeSession(params: RevokeSessionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeSessionRequest>('/api/v1/protected/auth/me/sessions/revoke', params);
  }

  // ==================== 用户 API ====================

  /**
   * 按关键字搜索用户
   */
  static async searchUsersByKeyword(params: SearchUsersByKeywordRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByKeywordRequest>('/api/v1/public/user/search_by_keyword', params);
  }

  /**
   * 按标签搜索用户
   */
  static async searchUsersByTag(params: SearchUsersByTagRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByTagRequest>('/api/v1/public/user/search_by_tag', params);
  }

  /**
   * 按位置搜索用户
   */
  static async searchUsersByLocation(params: SearchUsersByLocationRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByLocationRequest>('/api/v1/public/user/search_by_location', params);
  }

  /**
   * 获取用户资料（包含在线状态，受隐私设置控制）
   */
  static async getUserProfile(params: UserOpenTheDoorRequest): Promise<ApiResponse<UserSearchItem>> {
    return HttpUtil.post<ApiResponse<UserSearchItem>, UserOpenTheDoorRequest>('/api/v1/public/user/open_the_door', params);
  }

  /**
   * 获取用户关系
   */
  static async getRelationship(params: GetRelationshipRequest): Promise<ApiResponse<UserRelationship>> {
    return HttpUtil.post<ApiResponse<UserRelationship>, GetRelationshipRequest>('/api/v1/protected/user/relationship', params);
  }

  /**
   * 获取用户统计
   */
  static async getUserStats(params: GetUserStatsRequest): Promise<ApiResponse<UserStatsData>> {
    return HttpUtil.post<ApiResponse<UserStatsData>, GetUserStatsRequest>('/api/v1/public/user/stats', params);
  }

  /**
   * 获取指定用户好友列表
   */
  static async getUserFriendsList(params: GetUserFriendsListRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, GetUserFriendsListRequest>('/api/v1/public/user/friends', params);
  }

  /**
   * 获取指定用户关注列表
   */
  static async getUserFollowingList(params: GetUserFollowingListRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, GetUserFollowingListRequest>('/api/v1/public/user/following', params);
  }

  /**
   * 获取指定用户粉丝列表
   */
  static async getUserFollowersList(params: GetUserFollowersListRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, GetUserFollowersListRequest>('/api/v1/public/user/followers', params);
  }

  // ==================== 当前用户 API（/me/）====================

  /**
   * 获取当前用户资料
   */
  static async getMyProfile(): Promise<ApiResponse<MyProfileData>> {
    return HttpUtil.post<ApiResponse<MyProfileData>, object>('/api/v1/protected/user/me/profile', {});
  }

  /**
   * 更新用户资料
   */
  static async updateProfile(params: UpdateProfileRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateProfileRequest>('/api/v1/protected/user/me/profile/update', params);
  }

  /**
   * 更新用户位置
   */
  static async updateLocation(params: UpdateLocationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateLocationRequest>('/api/v1/protected/user/me/location', params);
  }

  /**
   * 账户升级
   */
  static async upgradeAccount(params: UpgradeAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpgradeAccountRequest>('/api/v1/protected/user/me/upgrade', params);
  }

  /**
   * 获取我的统计数据
   */
  static async getMyStats(): Promise<ApiResponse<UserStatsData>> {
    return HttpUtil.post<ApiResponse<UserStatsData>, object>('/api/v1/protected/user/me/stats', {});
  }

  /**
   * 获取用户设置
   */
  static async getSettings(): Promise<ApiResponse<UserSettingsData>> {
    return HttpUtil.post<ApiResponse<UserSettingsData>, object>('/api/v1/protected/user/me/settings', {});
  }

  /**
   * 更新用户设置
   */
  static async updateSettings(params: UpdateSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSettingsRequest>('/api/v1/protected/user/me/settings/update', params);
  }

  /**
   * 删除账户
   */
  static async deleteAccount(params: DeleteAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteAccountRequest>('/api/v1/protected/user/me/delete', params);
  }

  /**
   * 关注用户
   */
  static async followUser(params: FollowUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FollowUserRequest>('/api/v1/protected/user/me/follow', params);
  }

  /**
   * 取消关注
   */
  static async unfollowUser(params: UnfollowUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnfollowUserRequest>('/api/v1/protected/user/me/unfollow', params);
  }

  /**
   * 拉黑用户
   */
  static async blockUser(params: BlockUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BlockUserRequest>('/api/v1/protected/user/me/block', params);
  }

  /**
   * 解除拉黑
   */
  static async unblockUser(params: UnblockUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnblockUserRequest>('/api/v1/protected/user/me/unblock', params);
  }

  /**
   * 获取好友列表
   */
  static async getFriendsList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FriendListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FriendListItem>>, object>('/api/v1/protected/user/me/friends', params);
  }

  /**
   * 获取关注列表
   */
  static async getFollowingList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowingItem>>, object>('/api/v1/protected/user/me/following', params);
  }

  /**
   * 获取粉丝列表
   */
  static async getFollowersList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowerItem>>, object>('/api/v1/protected/user/me/followers', params);
  }

  /**
   * 获取拉黑列表
   */
  static async getBlockedList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BlockedUserItem>>, object>('/api/v1/protected/user/me/blocked', params);
  }

  /**
   * 设置好友备注
   */
  static async setFriendRemark(params: SetFriendRemarkRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetFriendRemarkRequest>('/api/v1/protected/user/me/remark', params);
  }

  // ==================== 生活瞬间 API ====================

  /**
   * 按关键字搜索瞬间
   */
  static async searchMomentsByKeyword(params: SearchMomentsByKeywordRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByKeywordRequest>('/api/v1/public/moments/search_by_keyword', params);
  }

  /**
   * 按标签搜索瞬间
   */
  static async searchMomentsByTag(params: SearchMomentsByTagRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByTagRequest>('/api/v1/public/moments/search_by_tag', params);
  }

  /**
   * 按位置搜索瞬间
   */
  static async searchMomentsByLocation(params: SearchMomentsByLocationRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByLocationRequest>('/api/v1/public/moments/search_by_location', params);
  }

  /**
   * 获取瞬间详情
   */
  static async getMomentDetail(params: MomentOpenTheDoorRequest): Promise<ApiResponse<MomentSearchItem>> {
    return HttpUtil.post<ApiResponse<MomentSearchItem>, MomentOpenTheDoorRequest>('/api/v1/public/moments/open_the_door', params);
  }

  /**
   * 获取用户瞬间列表
   */
  static async getUserMoments(params: GetUserMomentsRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, GetUserMomentsRequest>('/api/v1/public/user/moments', params);
  }

  /**
   * 获取瞬间评论列表
   */
  static async getMomentComments(params: GetCommentsRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, GetCommentsRequest>('/api/v1/public/moments/comments', params);
  }

  /**
   * 获取瞬间点赞列表
   */
  static async getMomentLikes(params: GetMomentLikesRequest): Promise<ApiResponse<PaginatedData<MomentLikeItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentLikeItem>>, GetMomentLikesRequest>('/api/v1/public/moments/likes', params);
  }

  /**
   * 创建瞬间
   */
  static async createMoment(params: CreateMomentRequest): Promise<ApiResponse<CreateMomentData>> {
    return HttpUtil.post<ApiResponse<CreateMomentData>, CreateMomentRequest>('/api/v1/protected/moments/create', params);
  }

  /**
   * 编辑瞬间
   */
  static async updateMoment(params: EditMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, EditMomentRequest>('/api/v1/protected/moments/update', params);
  }

  /**
   * 删除瞬间
   */
  static async deleteMoment(params: DeleteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteMomentRequest>('/api/v1/protected/moments/delete', params);
  }

  /**
   * 点赞瞬间
   */
  static async likeMoment(params: LikeMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LikeMomentRequest>('/api/v1/protected/moments/like', params);
  }

  /**
   * 取消点赞瞬间
   */
  static async unlikeMoment(params: UnlikeMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnlikeMomentRequest>('/api/v1/protected/moments/unlike', params);
  }

  /**
   * 收藏瞬间
   */
  static async favoriteMoment(params: FavoriteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FavoriteMomentRequest>('/api/v1/protected/moments/favorite', params);
  }

  /**
   * 取消收藏瞬间
   */
  static async unfavoriteMoment(params: UnfavoriteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnfavoriteMomentRequest>('/api/v1/protected/moments/unfavorite', params);
  }

  // ==================== 当前用户瞬间 API（/me/）====================

  /**
   * 获取我发布的瞬间
   */
  static async getMyMoments(params: { latitude: number; longitude: number; limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, object>('/api/v1/protected/moments/me', params);
  }

  /**
   * 获取我的收藏
   */
  static async getMyFavorites(params: { latitude: number; longitude: number; limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, object>('/api/v1/protected/moments/me/favorites', params);
  }

  /**
   * 检查瞬间是否已点赞
   */
  static async checkMomentLiked(params: CheckMomentLikedRequest): Promise<ApiResponse<CheckLikedData>> {
    return HttpUtil.post<ApiResponse<CheckLikedData>, CheckMomentLikedRequest>('/api/v1/protected/moments/me/liked', params);
  }

  /**
   * 检查瞬间是否已收藏
   */
  static async checkMomentFavorited(params: CheckMomentFavoritedRequest): Promise<ApiResponse<CheckFavoritedData>> {
    return HttpUtil.post<ApiResponse<CheckFavoritedData>, CheckMomentFavoritedRequest>('/api/v1/protected/moments/me/favorited', params);
  }

  // ==================== 评论 API ====================

  /**
   * 获取评论列表
   */
  static async getComments(params: GetCommentsRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, GetCommentsRequest>('/api/v1/public/comments', params);
  }

  /**
   * 获取子评论
   */
  static async getReplies(params: GetRepliesRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, GetRepliesRequest>('/api/v1/public/comments/replies', params);
  }

  /**
   * 获取评论点赞列表
   */
  static async getCommentLikes(params: GetCommentLikesRequest): Promise<ApiResponse<PaginatedData<CommentLikeItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentLikeItem>>, GetCommentLikesRequest>('/api/v1/public/comments/likes', params);
  }

  /**
   * 添加评论
   */
  static async createComment(params: CreateCommentRequest): Promise<ApiResponse<CreateCommentData>> {
    return HttpUtil.post<ApiResponse<CreateCommentData>, CreateCommentRequest>('/api/v1/protected/comments/create', params);
  }

  /**
   * 删除评论
   */
  static async deleteComment(params: DeleteCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteCommentRequest>('/api/v1/protected/comments/delete', params);
  }

  /**
   * 点赞评论
   */
  static async likeComment(params: LikeCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LikeCommentRequest>('/api/v1/protected/comments/like', params);
  }

  /**
   * 取消点赞评论
   */
  static async unlikeComment(params: UnlikeCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnlikeCommentRequest>('/api/v1/protected/comments/unlike', params);
  }

  // ==================== 私聊管理 API ====================

  /**
   * 发起私聊
   */
  static async createPrivateChat(params: CreatePrivateChatRequest): Promise<ApiResponse<CreatePrivateChatData>> {
    return HttpUtil.post<ApiResponse<CreatePrivateChatData>, CreatePrivateChatRequest>('/api/v1/protected/private_chats/create', params);
  }

  /**
   * 获取私聊列表
   */
  static async getPrivateChats(params: GetPrivateChatsRequest): Promise<ApiResponse<PaginatedData<PrivateChatListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateChatListItem>>, GetPrivateChatsRequest>('/api/v1/protected/private_chats/me', params);
  }

  /**
   * 获取私聊详情 - open_the_door
   */
  static async privateChatOpenTheDoor(params: PrivateChatOpenTheDoorRequest): Promise<ApiResponse<PrivateChatDetailData>> {
    return HttpUtil.post<ApiResponse<PrivateChatDetailData>, PrivateChatOpenTheDoorRequest>('/api/v1/protected/private_chats/open_the_door', params);
  }

  /**
   * 更新私聊设置
   */
  static async updatePrivateChatSettings(params: UpdatePrivateChatSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateChatSettingsRequest>('/api/v1/protected/private_chats/update_settings', params);
  }

  /**
   * 删除私聊
   */
  static async deletePrivateChat(params: DeletePrivateChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateChatRequest>('/api/v1/protected/private_chats/delete', params);
  }

  // ==================== 群组管理 API ====================

  /**
   * 按关键字搜索群组
   */
  static async searchGroupsByKeyword(params: SearchGroupsByKeywordRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByKeywordRequest>('/api/v1/public/group_chats/search_by_keyword', params);
  }

  /**
   * 按标签搜索群组
   */
  static async searchGroupsByTag(params: SearchGroupsByTagRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByTagRequest>('/api/v1/public/group_chats/search_by_tag', params);
  }

  /**
   * 按位置搜索群组
   */
  static async searchGroupsByLocation(params: SearchGroupsByLocationRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByLocationRequest>('/api/v1/public/group_chats/search_by_location', params);
  }

  /**
   * 获取群组详情
   */
  static async getGroupDetail(params: GroupOpenTheDoorRequest): Promise<ApiResponse<GroupSearchItem>> {
    return HttpUtil.post<ApiResponse<GroupSearchItem>, GroupOpenTheDoorRequest>('/api/v1/public/group_chats/open_the_door', params);
  }

  /**
   * 获取群组设置
   */
  static async getGroupSettings(params: GetGroupSettingsRequest): Promise<ApiResponse<GroupSettingsData>> {
    return HttpUtil.post<ApiResponse<GroupSettingsData>, GetGroupSettingsRequest>('/api/v1/public/group_chats/settings', params);
  }

  /**
   * 获取群组成员列表
   */
  static async getGroupMembers(params: GetMembersRequest): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMemberItem>>, GetMembersRequest>('/api/v1/public/group_chats/members', params);
  }

  /**
   * 创建群组
   */
  static async createGroupChat(params: CreateGroupChatRequest): Promise<ApiResponse<CreateGroupChatData>> {
    return HttpUtil.post<ApiResponse<CreateGroupChatData>, CreateGroupChatRequest>('/api/v1/protected/group_chats/create', params);
  }

  /**
   * 更新群组
   */
  static async updateGroupChat(params: UpdateGroupChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupChatRequest>('/api/v1/protected/group_chats/update', params);
  }

  /**
   * 解散群组
   */
  static async deleteGroupChat(params: DeleteGroupChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupChatRequest>('/api/v1/protected/group_chats/delete', params);
  }

  /**
   * 邀请成员
   */
  static async inviteMember(params: InviteMemberRequest): Promise<ApiResponse<InviteMemberData>> {
    return HttpUtil.post<ApiResponse<InviteMemberData>, InviteMemberRequest>('/api/v1/protected/group_chats/members/invite', params);
  }

  /**
   * 加入群组
   */
  static async joinGroup(params: JoinGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, JoinGroupRequest>('/api/v1/protected/group_chats/members/join', params);
  }

  /**
   * 申请加入群组
   */
  static async applyToGroup(params: ApplyToGroupRequest): Promise<ApiResponse<ApplyToGroupData>> {
    return HttpUtil.post<ApiResponse<ApplyToGroupData>, ApplyToGroupRequest>('/api/v1/protected/group_chats/members/apply', params);
  }

  /**
   * 设置成员角色
   */
  static async setMemberRole(params: SetMemberRoleRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetMemberRoleRequest>('/api/v1/protected/group_chats/members/set_role', params);
  }

  /**
   * 禁言成员
   */
  static async muteMember(params: MuteMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MuteMemberRequest>('/api/v1/protected/group_chats/members/mute', params);
  }

  /**
   * 取消禁言
   */
  static async unmuteMember(params: UnmuteMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnmuteMemberRequest>('/api/v1/protected/group_chats/members/unmute', params);
  }

  /**
   * 封禁成员
   */
  static async banMember(params: BanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BanMemberRequest>('/api/v1/protected/group_chats/members/ban', params);
  }

  /**
   * 解除封禁
   */
  static async unbanMember(params: UnbanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnbanMemberRequest>('/api/v1/protected/group_chats/members/unban', params);
  }

  /**
   * 获取封禁列表
   */
  static async getBannedMembers(params: GetBannedMembersRequest): Promise<ApiResponse<PaginatedData<BannedMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BannedMemberItem>>, GetBannedMembersRequest>('/api/v1/protected/group_chats/members/banned', params);
  }

  /**
   * 移除成员
   */
  static async removeMember(params: RemoveMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveMemberRequest>('/api/v1/protected/group_chats/members/remove_member', params);
  }

  /**
   * 获取群组申请列表
   */
  static async getGroupApplications(params: GetGroupApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, GetGroupApplicationsRequest>('/api/v1/protected/group_chats/applications', params);
  }

  /**
   * 审批群组申请
   */
  static async approveGroupApplication(params: ApproveGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ApproveGroupApplicationRequest>('/api/v1/protected/group_chats/applications/approve', params);
  }

  /**
   * 拒绝群组申请
   */
  static async rejectGroupApplication(params: RejectGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectGroupApplicationRequest>('/api/v1/protected/group_chats/applications/reject', params);
  }

  /**
   * 转让群主
   */
  static async transferOwnership(params: TransferOwnershipRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, TransferOwnershipRequest>('/api/v1/protected/group_chats/transfer_ownership', params);
  }

  // ==================== 当前用户群组 API（/me/）====================

  /**
   * 获取我加入的群组
   */
  static async getJoinedGroups(params: GetJoinedGroupChatsRequest): Promise<ApiResponse<PaginatedData<JoinedGroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<JoinedGroupItem>>, GetJoinedGroupChatsRequest>('/api/v1/protected/group_chats/me', params);
  }

  /**
   * 取消群组申请
   */
  static async cancelGroupApplication(params: CancelGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, CancelGroupApplicationRequest>('/api/v1/protected/group_chats/me/applications/cancel', params);
  }

  /**
   * 获取我的群组邀请
   */
  static async getGroupInvites(params: GetGroupInvitesRequest): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupInviteItem>>, GetGroupInvitesRequest>('/api/v1/protected/group_chats/me/invites', params);
  }

  /**
   * 接受群组邀请
   */
  static async acceptGroupInvite(params: AcceptGroupInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AcceptGroupInviteRequest>('/api/v1/protected/group_chats/me/invites/accept', params);
  }

  /**
   * 拒绝群组邀请
   */
  static async rejectGroupInvite(params: RejectGroupInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectGroupInviteRequest>('/api/v1/protected/group_chats/me/invites/reject', params);
  }

  /**
   * 退出群组
   */
  static async leaveGroup(params: LeaveGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LeaveGroupRequest>('/api/v1/protected/group_chats/me/leave', params);
  }

  // ==================== 私聊消息 API ====================

  /**
   * 发送私聊消息
   */
  static async sendPrivateMessage(params: SendPrivateMessageRequest): Promise<ApiResponse<SendPrivateMessageData>> {
    return HttpUtil.post<ApiResponse<SendPrivateMessageData>, SendPrivateMessageRequest>('/api/v1/protected/private_messages/send', params);
  }

  /**
   * 获取私聊消息历史
   */
  static async getPrivateMessages(params: GetPrivateMessagesRequest): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateMessageItem>>, GetPrivateMessagesRequest>('/api/v1/protected/private_messages/history', params);
  }

  /**
   * 编辑私聊消息
   */
  static async updatePrivateMessage(params: UpdatePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateMessageRequest>('/api/v1/protected/private_messages/update', params);
  }

  /**
   * 撤回私聊消息
   */
  static async recallPrivateMessage(params: RecallPrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallPrivateMessageRequest>('/api/v1/protected/private_messages/recall', params);
  }

  /**
   * 删除私聊消息
   */
  static async deletePrivateMessage(params: DeletePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateMessageRequest>('/api/v1/protected/private_messages/delete', params);
  }

  /**
   * 标记私聊消息已读
   */
  static async markPrivateMessagesRead(params: MarkPrivateMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkPrivateMessagesReadRequest>('/api/v1/protected/private_messages/mark_as_read', params);
  }

  /**
   * 添加私聊消息反应
   */
  static async addPrivateMessageReaction(params: AddPrivateMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AddPrivateMessageReactionRequest>('/api/v1/protected/private_messages/add_reaction', params);
  }

  /**
   * 移除私聊消息反应
   */
  static async removePrivateMessageReaction(params: RemovePrivateMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemovePrivateMessageReactionRequest>('/api/v1/protected/private_messages/remove_reaction', params);
  }

  /**
   * 获取私聊消息反应列表
   */
  static async getPrivateMessageReactions(params: GetPrivateMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, GetPrivateMessageReactionsRequest>('/api/v1/protected/private_messages/reactions', params);
  }

  // ==================== 群聊消息 API ====================

  /**
   * 发送群聊消息
   */
  static async sendGroupMessage(params: SendGroupMessageRequest): Promise<ApiResponse<SendGroupMessageData>> {
    return HttpUtil.post<ApiResponse<SendGroupMessageData>, SendGroupMessageRequest>('/api/v1/protected/group_messages/send', params);
  }

  /**
   * 获取群聊消息历史
   */
  static async getGroupMessages(params: GetGroupMessagesRequest): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMessageItem>>, GetGroupMessagesRequest>('/api/v1/protected/group_messages/history', params);
  }

  /**
   * 编辑群聊消息
   */
  static async updateGroupMessage(params: UpdateGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupMessageRequest>('/api/v1/protected/group_messages/update', params);
  }

  /**
   * 撤回群聊消息
   */
  static async recallGroupMessage(params: RecallGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallGroupMessageRequest>('/api/v1/protected/group_messages/recall', params);
  }

  /**
   * 删除群聊消息
   */
  static async deleteGroupMessage(params: DeleteGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupMessageRequest>('/api/v1/protected/group_messages/delete', params);
  }

  /**
   * 标记群聊消息已读
   */
  static async markGroupMessagesRead(params: MarkGroupMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkGroupMessagesReadRequest>('/api/v1/protected/group_messages/mark_as_read', params);
  }

  /**
   * 获取消息已读状态
   */
  static async getMessageReadStatus(params: GetMessageReadStatusRequest): Promise<ApiResponse<PaginatedData<MessageReadItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReadItem>>, GetMessageReadStatusRequest>('/api/v1/protected/group_messages/read_status', params);
  }

  /**
   * 添加群聊消息反应
   */
  static async addGroupMessageReaction(params: AddGroupMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AddGroupMessageReactionRequest>('/api/v1/protected/group_messages/add_reaction', params);
  }

  /**
   * 移除群聊消息反应
   */
  static async removeGroupMessageReaction(params: RemoveGroupMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveGroupMessageReactionRequest>('/api/v1/protected/group_messages/remove_reaction', params);
  }

  /**
   * 获取群聊消息反应列表
   */
  static async getGroupMessageReactions(params: GetGroupMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, GetGroupMessageReactionsRequest>('/api/v1/protected/group_messages/reactions', params);
  }

  // ==================== 消息通用 API ====================

  /**
   * 标记所有消息已读
   */
  static async markAllMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, object>('/api/v1/protected/messages/mark_all_as_read', {});
  }

  // ==================== 举报 API ====================

  /**
   * 创建举报
   */
  static async createReport(params: CreateReportRequest): Promise<ApiResponse<CreateReportData>> {
    return HttpUtil.post<ApiResponse<CreateReportData>, CreateReportRequest>('/api/v1/protected/reports/create', params);
  }

  /**
   * 获取我的举报记录
   */
  static async getMyReports(params: GetMyReportsRequest): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<ReportItem>>, GetMyReportsRequest>('/api/v1/protected/reports/me', params);
  }

  // ==================== 订阅设置 API ====================

  /**
   * 获取订阅配置
   */
  static async getSubscriptions(): Promise<ApiResponse<SubscriptionsData>> {
    return HttpUtil.post<ApiResponse<SubscriptionsData>, object>('/api/v1/protected/subscriptions/me', {});
  }

  /**
   * 更新订阅配置
   */
  static async updateSubscriptions(params: UpdateSubscriptionsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSubscriptionsRequest>('/api/v1/protected/subscriptions/me/update', params);
  }

  // ==================== 设备推送 API ====================

  /**
   * 注册设备推送Token
   */
  static async registerDeviceToken(params: RegisterDeviceTokenRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RegisterDeviceTokenRequest>('/api/v1/protected/device_tokens/me/register', params);
  }

  /**
   * 注销设备推送Token
   */
  static async unregisterDeviceToken(params: UnregisterDeviceTokenRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnregisterDeviceTokenRequest>('/api/v1/protected/device_tokens/me/unregister', params);
  }

  /**
   * 获取设备推送Token列表
   */
  static async getDeviceTokens(params: GetDeviceTokensRequest): Promise<ApiResponse<PaginatedData<DeviceTokenItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<DeviceTokenItem>>, GetDeviceTokensRequest>('/api/v1/protected/device_tokens/me', params);
  }

  // ==================== 信息流 API ====================

  /**
   * 获取瞬间信息流
   */
  static async getFeedMoments(params: GetFeedMomentsRequest): Promise<ApiResponse<PaginatedData<FeedMomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedMomentItem>>, GetFeedMomentsRequest>('/api/v1/protected/feed/moments', params);
  }

  /**
   * 获取群组推荐
   */
  static async getFeedGroups(params: GetFeedGroupsRequest): Promise<ApiResponse<PaginatedData<FeedGroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedGroupItem>>, GetFeedGroupsRequest>('/api/v1/protected/feed/groups', params);
  }

  /**
   * 获取用户推荐
   */
  static async getFeedUsers(params: GetFeedUsersRequest): Promise<ApiResponse<PaginatedData<FeedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedUserItem>>, GetFeedUsersRequest>('/api/v1/protected/feed/users', params);
  }

  /**
   * 获取热门内容
   */
  static async getFeedTrending(params: GetFeedTrendingRequest): Promise<ApiResponse<FeedTrendingData>> {
    return HttpUtil.post<ApiResponse<FeedTrendingData>, GetFeedTrendingRequest>('/api/v1/protected/feed/trending', params);
  }

  /**
   * 反馈信息流内容
   */
  static async feedFeedback(params: FeedFeedbackRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FeedFeedbackRequest>('/api/v1/protected/feed/feedback', params);
  }

  // ==================== 标签 API ====================

  /**
   * 获取热门标签
   */
  static async getTrendingTags(params: GetTrendingTagsRequest): Promise<ApiResponse<TrendingTagsData>> {
    return HttpUtil.post<ApiResponse<TrendingTagsData>, GetTrendingTagsRequest>('/api/v1/public/tags/trending', params);
  }

  /**
   * 标签搜索建议
   */
  static async suggestTags(params: SuggestTagsRequest): Promise<ApiResponse<TagSuggestData>> {
    return HttpUtil.post<ApiResponse<TagSuggestData>, SuggestTagsRequest>('/api/v1/public/tags/suggest', params);
  }
}
