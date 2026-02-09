/**
 * API 客户端
 * 严格按照 api_spec.mdc 定义的 API 端点实现
 */

import { HttpUtil } from '../network/HttpUtil';
import { ApiEndpoints } from '../config/ApiEndpoints';
import common from '@ohos.app.ability.common';
import fs from '@ohos.file.fs';
import {
  // 基础类型
  ApiResponse,
  PaginatedData,
  UploadMediaData,
  PresignUploadRequest,
  PresignUploadData,
  CompleteUploadRequest,
  UploadCategory,
  
  // 认证相关
  AuthData,
  RegisterRequest,
  LoginRequest,
  GuestRequest,
  CreateReportRequest,
  RefreshTokenRequest,
  CheckStatusData,
  RequestPasswordResetRequest,
  RequestPasswordResetData,
  VerifyResetCodeRequest,
  VerifyResetCodeData,
  ResetPasswordRequest,
  ChangePasswordRequest,
  SessionsRequest,
  SessionItem,
  RevokeSessionRequest,
  
  // 用户管理
  UserSearchItem,
  SearchUsersByKeywordRequest,
  SearchUsersByTagRequest,
  SearchUsersByLocationRequest,
  UserOpenTheDoorRequest,
  GetRelationshipRequest,
  UserRelation,
  UserStatsRequest,
  UserStatsData,
  MyProfileData,
  UpdateProfileRequest,
  UpdateLocationRequest,
  UpgradeAccountRequest,
  UserSettingsData,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  FollowRequest,
  UnfollowRequest,
  BlockRequest,
  UnblockRequest,
  FriendListItem,
  FollowingItem,
  FollowerItem,
  BlockedUserItem,
  SetRemarkRequest,
  UserFriendsRequest,
  UserFollowingRequest,
  UserFollowersRequest,
  MyFriendsRequest,
  MyFollowingRequest,
  MyFollowersRequest,
  MyBlockedRequest,
  
  // 生活瞬间
  MomentSearchItem,
  SearchMomentsByKeywordRequest,
  SearchMomentsByTagRequest,
  SearchMomentsByLocationRequest,
  MomentOpenTheDoorRequest,
  UserMomentsRequest,
  CreateMomentRequest,
  EditMomentRequest,
  DeleteMomentRequest,
  LikeMomentRequest,
  UnlikeMomentRequest,
  FavoriteMomentRequest,
  UnfavoriteMomentRequest,
  CheckLikedRequest,
  CheckLikedData,
  CheckFavoritedRequest,
  CheckFavoritedData,
  MomentLikesRequest,
  MomentLikeItem,
  MyMomentsRequest,
  MyFavoritesRequest,
  
  // 评论
  CreateCommentRequest,
  DeleteCommentRequest,
  CommentsRequest,
  CommentListItem,
  CommentRepliesRequest,
  LikeCommentRequest,
  UnlikeCommentRequest,
  CommentLikesRequest,
  CommentLikeItem,
  
  // 私聊管理
  MyPrivateChatsRequest,
  PrivateChatListItem,
  PrivateChatOpenTheDoorRequest,
  PrivateChatDetailData,
  UpdatePrivateChatSettingsRequest,
  DeletePrivateChatRequest,
  
  // 私聊消息
  SendPrivateMessageRequest,
  PrivateMessageItem,
  PrivateMessagesHistoryRequest,
  UpdatePrivateMessageRequest,
  RecallPrivateMessageRequest,
  DeletePrivateMessageRequest,
  MarkPrivateMessagesReadRequest,
  AddPrivateMessageReactionRequest,
  RemovePrivateMessageReactionRequest,
  PrivateMessageReactionsRequest,
  MessageReactionItem,
  
  // 群组管理
  SearchGroupsByKeywordRequest,
  SearchGroupsByTagRequest,
  SearchGroupsByLocationRequest,
  GroupSearchItem,
  GroupOpenTheDoorRequest,
  GroupSettingsRequest,
  GroupSettingsData,
  GroupMembersRequest,
  GroupMemberItem,
  CreateGroupRequest,
  UpdateGroupRequest,
  DeleteGroupRequest,
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
  BannedMembersRequest,
  BannedMemberItem,
  RemoveMemberRequest,
  GroupApplicationsRequest,
  GroupApplicationItem,
  ApproveGroupApplicationRequest,
  RejectGroupApplicationRequest,
  TransferOwnershipRequest,
  MyGroupsRequest,
  JoinedGroupItem,
  CancelApplicationRequest,
  MyInvitesRequest,
  GroupInviteItem,
  AcceptInviteRequest,
  RejectInviteRequest,
  LeaveGroupRequest,
  
  // 群聊消息
  SendGroupMessageRequest,
  GroupMessageItem,
  GroupMessagesHistoryRequest,
  UpdateGroupMessageRequest,
  RecallGroupMessageRequest,
  DeleteGroupMessageRequest,
  MarkGroupMessagesReadRequest,
  MessageReadStatusRequest,
  MessageReadItem,
  AddGroupMessageReactionRequest,
  RemoveGroupMessageReactionRequest,
  GroupMessageReactionsRequest,
  
  // 举报
  CreateReportData,
  ReportItem,
  MyReportsRequest,
  
  // 订阅设置
  SubscriptionsData,
  UpdateSubscriptionsRequest,
  
  // 设备推送
  RegisterDeviceTokenRequest,
  UnregisterDeviceTokenRequest,
  DeviceTokenItem,
  MyDeviceTokensRequest,
  
  // 信息流
  GetFeedMomentsRequest,
  GetFeedGroupsRequest,
  GetFeedUsersRequest,
  GetFeedTrendingRequest,
  FeedGroupItem,
  FeedUserItem,
  TrendingBundle,
  FeedFeedbackRequest,
  
  // 标签
  TrendingTagsRequest,
  TrendingTagsData,
  SuggestTagsRequest,
  TagSuggestions,
  
  // 地图
  MapRenderRequest,
  MapRenderData,
  
  // 配置
  FetchGlobalSettingsRequest,
  FetchSettingsData,
  
  // 系统
  HealthCheckData,
  
  UserInfo,
} from '../types/api';

// 空请求体类
class EmptyRequest {}

export class ApiClient {
  // ==================== 文件上传 API ====================

  /**
   * 申请上传预签名
   */
  static async presignUpload(params: PresignUploadRequest): Promise<ApiResponse<PresignUploadData>> {
    return HttpUtil.post<ApiResponse<PresignUploadData>, PresignUploadRequest>(ApiEndpoints.UPLOAD.PRESIGN_UPLOAD, params);
  }

  /**
   * 完成上传
   */
  static async completeUpload(params: CompleteUploadRequest): Promise<ApiResponse<UploadMediaData>> {
    return HttpUtil.post<ApiResponse<UploadMediaData>, CompleteUploadRequest>(ApiEndpoints.UPLOAD.COMPLETE_UPLOAD, params);
  }

  /**
   * 上传文件 (Presigned Flow)
   */
  static async uploadFile(
    context: common.UIAbilityContext,
    fileUri: string,
    category: UploadCategory
  ): Promise<UploadMediaData> {
    // 1. 读取文件信息
    let file: fs.File | null = null;
    let size = 0;
    let buffer: ArrayBuffer | null = null;
    try {
      file = fs.openSync(fileUri, fs.OpenMode.READ_ONLY);
      const stat = fs.statSync(file.fd);
      size = stat.size;
      buffer = new ArrayBuffer(size);
      fs.readSync(file.fd, buffer);
    } catch (error) {
      if (file) {
        try {
          fs.closeSync(file);
        } catch (e) {}
      }
      throw (error instanceof Error) ? error : new Error(String(error));
    } finally {
      if (file) {
        try {
          fs.closeSync(file);
        } catch (e) {}
      }
    }

    let fileName = 'file';
    try {
        const parts = fileUri.split('/');
        if (parts.length > 0) {
            fileName = parts[parts.length - 1];
        }
    } catch (e) {}

    // 2. 申请预签名
    const presignReq: PresignUploadRequest = {
        file_name: fileName,
        file_size: size,
        mime_type: 'application/octet-stream', // 简化的MIME类型推断
        category: category
    };
    
    const presignRes = await ApiClient.presignUpload(presignReq);
    if (presignRes.code !== 0 || !presignRes.data) {
        throw new Error(presignRes.message || 'Presign failed');
    }
    const presignData = presignRes.data;
    const upload_url = presignData.upload_url;
    const upload_id = presignData.upload_id;
    const presignHeaders = presignData.headers;
    
    // Convert Map to Record<string, string>
    const headers: Record<string, string> = {};
    if (presignHeaders) {
      const headerKeys = Object.keys(presignHeaders);
      for (let i = 0; i < headerKeys.length; i++) {
        const key = headerKeys[i];
        const value = presignHeaders[key];
        if (typeof value === 'string') {
          headers[key] = value;
        }
      }
    }

    // 3. 上传到对象存储 (S3/MinIO)
    if (!buffer) {
      throw new Error('Failed to read file buffer');
    }
    await HttpUtil.uploadToUrl(upload_url, buffer, headers);

    // 4. 完成上传
    const completeReq: CompleteUploadRequest = { upload_id };
    const completeRes = await ApiClient.completeUpload(completeReq);
    if (completeRes.code !== 0 || !completeRes.data) {
        throw new Error(completeRes.message || 'Complete failed');
    }
    return completeRes.data;
  }

  // ==================== 系统健康检查 API ====================

  /**
   * 系统健康检查
   */
  static async healthCheck(): Promise<ApiResponse<HealthCheckData>> {
    return HttpUtil.post<ApiResponse<HealthCheckData>, EmptyRequest>(ApiEndpoints.SYSTEM.HEALTH, new EmptyRequest());
  }

  // ==================== 认证 API ====================

  /**
   * 用户注册
   */
  static async register(params: RegisterRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, RegisterRequest>(ApiEndpoints.AUTH.REGISTER, params);
  }

  /**
   * 用户登录
   */
  static async login(params: LoginRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, LoginRequest>(ApiEndpoints.AUTH.LOGIN, params);
  }

  /**
   * 创建临时用户
   */
  static async createTempUser(params: GuestRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, GuestRequest>(ApiEndpoints.AUTH.GUEST, params);
  }

  /**
   * 刷新Token
   */
  static async refreshToken(params: RefreshTokenRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, RefreshTokenRequest>(ApiEndpoints.AUTH.REFRESH_TOKEN, params);
  }

  /**
   * 检查认证状态
   */
  static async checkAuthStatus(): Promise<ApiResponse<CheckStatusData>> {
    return HttpUtil.post<ApiResponse<CheckStatusData>, EmptyRequest>(ApiEndpoints.AUTH.CHECK_STATUS, new EmptyRequest());
  }

  /**
   * 用户登出
   */
  static async logout(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, EmptyRequest>(ApiEndpoints.AUTH.LOGOUT, new EmptyRequest());
  }

  /**
   * 请求密码重置
   */
  static async requestPasswordReset(params: RequestPasswordResetRequest): Promise<ApiResponse<RequestPasswordResetData>> {
    return HttpUtil.post<ApiResponse<RequestPasswordResetData>, RequestPasswordResetRequest>(ApiEndpoints.AUTH.REGISTER, params);
  }

  /**
   * 验证重置码
   */
  static async verifyResetCode(params: VerifyResetCodeRequest): Promise<ApiResponse<VerifyResetCodeData>> {
    return HttpUtil.post<ApiResponse<VerifyResetCodeData>, VerifyResetCodeRequest>(ApiEndpoints.AUTH.REGISTER, params);
  }

  /**
   * 重置密码
   */
  static async resetPassword(params: ResetPasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ResetPasswordRequest>(ApiEndpoints.AUTH.REGISTER, params);
  }

  /**
   * 修改密码
   */
  static async changePassword(params: ChangePasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ChangePasswordRequest>(ApiEndpoints.AUTH.ME.CHANGE_PASSWORD, params);
  }

  /**
   * 获取活跃会话列表
   */
  static async getSessions(params: SessionsRequest): Promise<ApiResponse<PaginatedData<SessionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<SessionItem>>, SessionsRequest>(ApiEndpoints.AUTH.ME.SESSIONS.LIST, params);
  }

  /**
   * 撤销会话
   */
  static async revokeSession(params: RevokeSessionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeSessionRequest>(ApiEndpoints.AUTH.ME.SESSIONS.REVOKE, params);
  }

  // ==================== 用户 API ====================

  /**
   * 按关键字搜索用户
   */
  static async searchUsersByKeyword(params: SearchUsersByKeywordRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByKeywordRequest>(ApiEndpoints.USER.SEARCH, params);
  }

  /**
   * 按标签搜索用户
   */
  static async searchUsersByTag(params: SearchUsersByTagRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByTagRequest>(ApiEndpoints.USER.SEARCH, params);
  }

  /**
   * 按位置搜索用户
   */
  static async searchUsersByLocation(params: SearchUsersByLocationRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByLocationRequest>(ApiEndpoints.USER.SEARCH, params);
  }

  /**
   * 获取用户资料（包含在线状态，受隐私设置控制）
   */
  static async getUserProfile(params: UserOpenTheDoorRequest): Promise<ApiResponse<UserSearchItem>> {
    return HttpUtil.post<ApiResponse<UserSearchItem>, UserOpenTheDoorRequest>(ApiEndpoints.USER.OPEN_THE_DOOR, params);
  }

  /**
   * 获取用户关系
   */
  static async getRelationship(params: GetRelationshipRequest): Promise<ApiResponse<UserRelation>> {
    return HttpUtil.post<ApiResponse<UserRelation>, GetRelationshipRequest>(ApiEndpoints.USER.OPEN_THE_DOOR, params);
  }

  /**
   * 获取用户统计
   */
  static async getUserStats(params: UserStatsRequest): Promise<ApiResponse<UserStatsData>> {
    return HttpUtil.post<ApiResponse<UserStatsData>, UserStatsRequest>(ApiEndpoints.USER.OPEN_THE_DOOR, params);
  }

  /**
   * 获取指定用户好友列表
   */
  static async getUserFriendsList(params: UserFriendsRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, UserFriendsRequest>(ApiEndpoints.USER.LIST_FRIENDS, params);
  }

  /**
   * 获取指定用户关注列表
   */
  static async getUserFollowingList(params: UserFollowingRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, UserFollowingRequest>(ApiEndpoints.USER.LIST_FOLLOWING, params);
  }

  /**
   * 获取指定用户粉丝列表
   */
  static async getUserFollowersList(params: UserFollowersRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, UserFollowersRequest>(ApiEndpoints.USER.LIST_FOLLOWERS, params);
  }

  // ==================== 当前用户 API（/me/）====================

  /**
   * 获取当前用户资料
   */
  static async getMyProfile(): Promise<ApiResponse<MyProfileData>> {
    return HttpUtil.post<ApiResponse<MyProfileData>, EmptyRequest>(ApiEndpoints.USER.ME.PROFILE.GET, new EmptyRequest());
  }

  /**
   * 更新用户资料
   */
  static async updateProfile(params: UpdateProfileRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateProfileRequest>(ApiEndpoints.USER.ME.PROFILE.UPDATE, params);
  }

  /**
   * 更新用户位置
   */
  static async updateLocation(params: UpdateLocationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateLocationRequest>(ApiEndpoints.USER.ME.LOCATION.UPDATE, params);
  }

  /**
   * 账户升级
   */
  static async upgradeAccount(params: UpgradeAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpgradeAccountRequest>(ApiEndpoints.USER.ME.UPGRADE, params);
  }

  /**
   * 获取我的统计数据
   */
  static async getMyStats(): Promise<ApiResponse<UserStatsData>> {
    return HttpUtil.post<ApiResponse<UserStatsData>, EmptyRequest>(ApiEndpoints.USER.SEARCH, new EmptyRequest());
  }

  /**
   * 获取用户设置
   */
  static async getSettings(): Promise<ApiResponse<UserSettingsData>> {
    return HttpUtil.post<ApiResponse<UserSettingsData>, EmptyRequest>(ApiEndpoints.USER.ME.SETTINGS.GET, new EmptyRequest());
  }

  /**
   * 更新用户设置
   */
  static async updateSettings(params: UpdateSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSettingsRequest>(ApiEndpoints.USER.ME.SETTINGS.UPDATE, params);
  }

  /**
   * 删除账户
   */
  static async deleteAccount(params: DeleteAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteAccountRequest>(ApiEndpoints.USER.ME.DELETE, params);
  }

  /**
   * 关注用户
   */
  static async followUser(params: FollowRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FollowRequest>(ApiEndpoints.USER.ME.FOLLOW, params);
  }

  /**
   * 取消关注
   */
  static async unfollowUser(params: UnfollowRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnfollowRequest>(ApiEndpoints.USER.ME.UNFOLLOW, params);
  }

  /**
   * 拉黑用户
   */
  static async blockUser(params: BlockRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BlockRequest>(ApiEndpoints.USER.ME.BLOCK, params);
  }

  /**
   * 解除拉黑
   */
  static async unblockUser(params: UnblockRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnblockRequest>(ApiEndpoints.USER.ME.UNBLOCK, params);
  }

  /**
   * 获取好友列表
   */
  static async getFriendsList(params: MyFriendsRequest): Promise<ApiResponse<PaginatedData<FriendListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FriendListItem>>, MyFriendsRequest>(ApiEndpoints.USER.ME.LIST_FRIENDS, params);
  }

  /**
   * 获取关注列表
   */
  static async getFollowingList(params: MyFollowingRequest): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowingItem>>, MyFollowingRequest>(ApiEndpoints.USER.ME.LIST_FOLLOWING, params);
  }

  /**
   * 获取粉丝列表
   */
  static async getFollowersList(params: MyFollowersRequest): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowerItem>>, MyFollowersRequest>(ApiEndpoints.USER.ME.LIST_FOLLOWERS, params);
  }

  /**
   * 获取拉黑列表
   */
  static async getBlockedList(params: MyBlockedRequest): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BlockedUserItem>>, MyBlockedRequest>(ApiEndpoints.USER.ME.LIST_BLOCKED, params);
  }

  /**
   * 设置好友备注
   */
  static async setFriendRemark(params: SetRemarkRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetRemarkRequest>(ApiEndpoints.USER.ME.SET_REMARK, params);
  }

  // ==================== 生活瞬间 API ====================

  /**
   * 按关键字搜索瞬间
   */
  static async searchMomentsByKeyword(params: SearchMomentsByKeywordRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByKeywordRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  /**
   * 按标签搜索瞬间
   */
  static async searchMomentsByTag(params: SearchMomentsByTagRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByTagRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  /**
   * 按位置搜索瞬间
   */
  static async searchMomentsByLocation(params: SearchMomentsByLocationRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByLocationRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  /**
   * 获取瞬间详情
   */
  static async getMomentDetail(params: MomentOpenTheDoorRequest): Promise<ApiResponse<MomentSearchItem>> {
    return HttpUtil.post<ApiResponse<MomentSearchItem>, MomentOpenTheDoorRequest>(ApiEndpoints.MOMENT.OPEN_THE_DOOR, params);
  }

  /**
   * 获取用户瞬间列表
   */
  static async getUserMoments(params: UserMomentsRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, UserMomentsRequest>(ApiEndpoints.USER.LIST_MOMENTS, params);
  }

  /**
   * 获取瞬间评论列表
   */
  static async getMomentComments(params: CommentsRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, CommentsRequest>(ApiEndpoints.COMMENT.LIST, params);
  }

  /**
   * 获取瞬间点赞列表
   */
  static async getMomentLikes(params: MomentLikesRequest): Promise<ApiResponse<PaginatedData<MomentLikeItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentLikeItem>>, MomentLikesRequest>(ApiEndpoints.MOMENT.LIST_LIKES, params);
  }

  /**
   * 创建瞬间
   */
  static async createMoment(params: CreateMomentRequest): Promise<ApiResponse<MomentSearchItem>> {
    return HttpUtil.post<ApiResponse<MomentSearchItem>, CreateMomentRequest>(ApiEndpoints.MOMENT.CREATE, params);
  }

  /**
   * 编辑瞬间
   */
  static async updateMoment(params: EditMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, EditMomentRequest>(ApiEndpoints.MOMENT.UPDATE, params);
  }

  /**
   * 删除瞬间
   */
  static async deleteMoment(params: DeleteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteMomentRequest>(ApiEndpoints.MOMENT.DELETE, params);
  }

  /**
   * 点赞瞬间
   */
  static async likeMoment(params: LikeMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LikeMomentRequest>(ApiEndpoints.MOMENT.ME.LIKE, params);
  }

  /**
   * 取消点赞瞬间
   */
  static async unlikeMoment(params: UnlikeMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnlikeMomentRequest>(ApiEndpoints.MOMENT.ME.UNLIKE, params);
  }

  /**
   * 收藏瞬间
   */
  static async favoriteMoment(params: FavoriteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FavoriteMomentRequest>(ApiEndpoints.MOMENT.ME.FAVORITE, params);
  }

  /**
   * 取消收藏瞬间
   */
  static async unfavoriteMoment(params: UnfavoriteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnfavoriteMomentRequest>(ApiEndpoints.MOMENT.ME.UNFAVORITE, params);
  }

  // ==================== 当前用户瞬间 API（/me/）====================

  /**
   * 获取我发布的瞬间
   */
  static async getMyMoments(params: MyMomentsRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, MyMomentsRequest>(ApiEndpoints.MOMENT.ME.LIST, params);
  }

  /**
   * 获取我的收藏
   */
  static async getMyFavorites(params: MyFavoritesRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, MyFavoritesRequest>(ApiEndpoints.MOMENT.ME.LIST_FAVORITES, params);
  }

  /**
   * 检查瞬间是否已点赞
   */
  static async checkMomentLiked(params: CheckLikedRequest): Promise<ApiResponse<CheckLikedData>> {
    return HttpUtil.post<ApiResponse<CheckLikedData>, CheckLikedRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  /**
   * 检查瞬间是否已收藏
   */
  static async checkMomentFavorited(params: CheckFavoritedRequest): Promise<ApiResponse<CheckFavoritedData>> {
    return HttpUtil.post<ApiResponse<CheckFavoritedData>, CheckFavoritedRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  // ==================== 评论 API ====================

  /**
   * 获取评论列表
   */
  static async getComments(params: CommentsRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, CommentsRequest>(ApiEndpoints.COMMENT.LIST, params);
  }

  /**
   * 获取子评论
   */
  static async getReplies(params: CommentRepliesRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, CommentRepliesRequest>(ApiEndpoints.COMMENT.LIST_REPLIES, params);
  }

  /**
   * 获取评论点赞列表
   */
  static async getCommentLikes(params: CommentLikesRequest): Promise<ApiResponse<PaginatedData<CommentLikeItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentLikeItem>>, CommentLikesRequest>(ApiEndpoints.COMMENT.LIST_LIKES, params);
  }

  /**
   * 添加评论
   */
  static async createComment(params: CreateCommentRequest): Promise<ApiResponse<CommentListItem>> {
    return HttpUtil.post<ApiResponse<CommentListItem>, CreateCommentRequest>(ApiEndpoints.COMMENT.CREATE, params);
  }

  /**
   * 删除评论
   */
  static async deleteComment(params: DeleteCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteCommentRequest>(ApiEndpoints.COMMENT.DELETE, params);
  }

  /**
   * 点赞评论
   */
  static async likeComment(params: LikeCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LikeCommentRequest>(ApiEndpoints.COMMENT.ME.LIKE, params);
  }

  /**
   * 取消点赞评论
   */
  static async unlikeComment(params: UnlikeCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnlikeCommentRequest>(ApiEndpoints.COMMENT.ME.UNLIKE, params);
  }

  // ==================== 私聊管理 API ====================

  /**
   * 获取私聊列表
   */
  static async getPrivateChats(params: MyPrivateChatsRequest): Promise<ApiResponse<PaginatedData<PrivateChatListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateChatListItem>>, MyPrivateChatsRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.ME.LIST, params);
  }

  /**
   * 获取私聊详情 - open_the_door
   */
  static async privateChatOpenTheDoor(params: PrivateChatOpenTheDoorRequest): Promise<ApiResponse<PrivateChatDetailData>> {
    return HttpUtil.post<ApiResponse<PrivateChatDetailData>, PrivateChatOpenTheDoorRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.OPEN_THE_DOOR, params);
  }

  /**
   * 更新私聊设置
   */
  static async updatePrivateChatSettings(params: UpdatePrivateChatSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateChatSettingsRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.ME.UPDATE_SETTINGS, params);
  }

  /**
   * 删除私聊
   */
  static async deletePrivateChat(params: DeletePrivateChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateChatRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.ME.DELETE, params);
  }

  // ==================== 群组管理 API ====================

  /**
   * 按关键字搜索群组
   */
  static async searchGroupsByKeyword(params: SearchGroupsByKeywordRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByKeywordRequest>(ApiEndpoints.GROUP.SEARCH, params);
  }

  /**
   * 按标签搜索群组
   */
  static async searchGroupsByTag(params: SearchGroupsByTagRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByTagRequest>(ApiEndpoints.GROUP.SEARCH, params);
  }

  /**
   * 按位置搜索群组
   */
  static async searchGroupsByLocation(params: SearchGroupsByLocationRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByLocationRequest>(ApiEndpoints.GROUP.SEARCH, params);
  }

  /**
   * 获取群组详情
   */
  static async getGroupDetail(params: GroupOpenTheDoorRequest): Promise<ApiResponse<GroupSearchItem>> {
    return HttpUtil.post<ApiResponse<GroupSearchItem>, GroupOpenTheDoorRequest>(ApiEndpoints.GROUP.OPEN_THE_DOOR, params);
  }

  /**
   * 获取群组设置
   */
  static async getGroupSettings(params: GroupSettingsRequest): Promise<ApiResponse<GroupSettingsData>> {
    return HttpUtil.post<ApiResponse<GroupSettingsData>, GroupSettingsRequest>(ApiEndpoints.GROUP.OPEN_THE_DOOR, params);
  }

  /**
   * 获取群组成员列表
   */
  static async getGroupMembers(params: GroupMembersRequest): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMemberItem>>, GroupMembersRequest>(ApiEndpoints.GROUP.MEMBERS.LIST, params);
  }

  /**
   * 创建群组
   */
  static async createGroupChat(params: CreateGroupRequest): Promise<ApiResponse<JoinedGroupItem>> {
    return HttpUtil.post<ApiResponse<JoinedGroupItem>, CreateGroupRequest>(ApiEndpoints.GROUP.CREATE, params);
  }

  /**
   * 更新群组
   */
  static async updateGroupChat(params: UpdateGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupRequest>(ApiEndpoints.GROUP.UPDATE, params);
  }

  /**
   * 解散群组
   */
  static async deleteGroupChat(params: DeleteGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupRequest>(ApiEndpoints.GROUP.DISMISS, params);
  }

  /**
   * 邀请成员
   */
  static async inviteMember(params: InviteMemberRequest): Promise<ApiResponse<InviteMemberData>> {
    return HttpUtil.post<ApiResponse<InviteMemberData>, InviteMemberRequest>(ApiEndpoints.GROUP.MEMBERS.INVITE, params);
  }

  /**
   * 加入群组
   */
  static async joinGroup(params: JoinGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, JoinGroupRequest>(ApiEndpoints.GROUP.ME.JOIN, params);
  }

  /**
   * 申请加入群组
   */
  static async applyToGroup(params: ApplyToGroupRequest): Promise<ApiResponse<ApplyToGroupData>> {
    return HttpUtil.post<ApiResponse<ApplyToGroupData>, ApplyToGroupRequest>(ApiEndpoints.GROUP.ME.JOIN, params);
  }

  /**
   * 设置成员角色
   */
  static async setMemberRole(params: SetMemberRoleRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetMemberRoleRequest>(ApiEndpoints.GROUP.MEMBERS.SET_ROLE, params);
  }

  /**
   * 禁言成员
   */
  static async muteMember(params: MuteMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MuteMemberRequest>(ApiEndpoints.GROUP.MEMBERS.MUTE, params);
  }

  /**
   * 取消禁言
   */
  static async unmuteMember(params: UnmuteMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnmuteMemberRequest>(ApiEndpoints.GROUP.MEMBERS.UNMUTE, params);
  }

  /**
   * 封禁成员
   */
  static async banMember(params: BanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BanMemberRequest>(ApiEndpoints.GROUP.MEMBERS.BAN, params);
  }

  /**
   * 解除封禁
   */
  static async unbanMember(params: UnbanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnbanMemberRequest>(ApiEndpoints.GROUP.MEMBERS.UNBAN, params);
  }

  /**
   * 获取封禁列表
   */
  static async getBannedMembers(params: BannedMembersRequest): Promise<ApiResponse<PaginatedData<BannedMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BannedMemberItem>>, BannedMembersRequest>(ApiEndpoints.GROUP.MEMBERS.LIST_BANNED, params);
  }

  /**
   * 移除成员
   */
  static async removeMember(params: RemoveMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveMemberRequest>(ApiEndpoints.GROUP.MEMBERS.REMOVE_MEMBER, params);
  }

  /**
   * 获取群组申请列表
   */
  static async getGroupApplications(params: GroupApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, GroupApplicationsRequest>(ApiEndpoints.GROUP.APPLICATIONS.LIST, params);
  }

  /**
   * 审批群组申请
   */
  static async approveGroupApplication(params: ApproveGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ApproveGroupApplicationRequest>(ApiEndpoints.GROUP.APPLICATIONS.APPROVE, params);
  }

  /**
   * 拒绝群组申请
   */
  static async rejectGroupApplication(params: RejectGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectGroupApplicationRequest>(ApiEndpoints.GROUP.APPLICATIONS.REJECT, params);
  }

  /**
   * 转让群主
   */
  static async transferOwnership(params: TransferOwnershipRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, TransferOwnershipRequest>(ApiEndpoints.GROUP.ME.TRANSFER_OWNERSHIP, params);
  }

  // ==================== 当前用户群组 API（/me/）====================

  /**
   * 获取我加入的群组
   */
  static async getJoinedGroups(params: MyGroupsRequest): Promise<ApiResponse<PaginatedData<JoinedGroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<JoinedGroupItem>>, MyGroupsRequest>(ApiEndpoints.GROUP.ME.LIST, params);
  }

  /**
   * 取消群组申请
   */
  static async cancelGroupApplication(params: CancelApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, CancelApplicationRequest>(ApiEndpoints.GROUP.ME.APPLICATIONS.CANCEL, params);
  }

  /**
   * 获取我的群组邀请
   */
  static async getGroupInvites(params: MyInvitesRequest): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupInviteItem>>, MyInvitesRequest>(ApiEndpoints.GROUP.ME.INVITES.LIST, params);
  }

  /**
   * 接受群组邀请
   */
  static async acceptGroupInvite(params: AcceptInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AcceptInviteRequest>(ApiEndpoints.GROUP.ME.INVITES.ACCEPT, params);
  }

  /**
   * 拒绝群组邀请
   */
  static async rejectGroupInvite(params: RejectInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectInviteRequest>(ApiEndpoints.GROUP.ME.INVITES.REJECT, params);
  }

  /**
   * 退出群组
   */
  static async leaveGroup(params: LeaveGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LeaveGroupRequest>(ApiEndpoints.GROUP.ME.LEAVE, params);
  }

  // ==================== 私聊消息 API ====================

  /**
   * 发送私聊消息
   */
  static async sendPrivateMessage(params: SendPrivateMessageRequest): Promise<ApiResponse<PrivateMessageItem>> {
    return HttpUtil.post<ApiResponse<PrivateMessageItem>, SendPrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND, params);
  }

  /**
   * 获取私聊消息历史
   */
  static async getPrivateMessages(params: PrivateMessagesHistoryRequest): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateMessageItem>>, PrivateMessagesHistoryRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.GET_HISTORY, params);
  }

  /**
   * 编辑私聊消息
   */
  static async updatePrivateMessage(params: UpdatePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.UPDATE, params);
  }

  /**
   * 撤回私聊消息
   */
  static async recallPrivateMessage(params: RecallPrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallPrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.RECALL, params);
  }

  /**
   * 删除私聊消息
   */
  static async deletePrivateMessage(params: DeletePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.RECALL, params);
  }

  /**
   * 标记私聊消息已读
   */
  static async markPrivateMessagesRead(params: MarkPrivateMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkPrivateMessagesReadRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.ME.MARK_AS_READ, params);
  }

  /**
   * 添加私聊消息反应
   */
  static async addPrivateMessageReaction(params: AddPrivateMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AddPrivateMessageReactionRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND, params);
  }

  /**
   * 移除私聊消息反应
   */
  static async removePrivateMessageReaction(params: RemovePrivateMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemovePrivateMessageReactionRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.UPDATE, params);
  }

  /**
   * 获取私聊消息反应列表
   */
  static async getPrivateMessageReactions(params: PrivateMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, PrivateMessageReactionsRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.GET_HISTORY, params);
  }

  // ==================== 群聊消息 API ====================

  /**
   * 发送群聊消息
   */
  static async sendGroupMessage(params: SendGroupMessageRequest): Promise<ApiResponse<GroupMessageItem>> {
    return HttpUtil.post<ApiResponse<GroupMessageItem>, SendGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.SEND, params);
  }

  /**
   * 获取群聊消息历史
   */
  static async getGroupMessages(params: GroupMessagesHistoryRequest): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMessageItem>>, GroupMessagesHistoryRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.GET_HISTORY, params);
  }

  /**
   * 编辑群聊消息
   */
  static async updateGroupMessage(params: UpdateGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.UPDATE, params);
  }

  /**
   * 撤回群聊消息
   */
  static async recallGroupMessage(params: RecallGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.RECALL, params);
  }

  /**
   * 删除群聊消息
   */
  static async deleteGroupMessage(params: DeleteGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.RECALL, params);
  }

  /**
   * 标记群聊消息已读
   */
  static async markGroupMessagesRead(params: MarkGroupMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkGroupMessagesReadRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.ME.MARK_AS_READ, params);
  }

  /**
   * 获取消息已读状态
   */
  static async getMessageReadStatus(params: MessageReadStatusRequest): Promise<ApiResponse<PaginatedData<MessageReadItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReadItem>>, MessageReadStatusRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.GET_HISTORY, params);
  }

  /**
   * 添加群聊消息反应
   */
  static async addGroupMessageReaction(params: AddGroupMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AddGroupMessageReactionRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.SEND, params);
  }

  /**
   * 移除群聊消息反应
   */
  static async removeGroupMessageReaction(params: RemoveGroupMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveGroupMessageReactionRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.UPDATE, params);
  }

  /**
   * 获取群聊消息反应列表
   */
  static async getGroupMessageReactions(params: GroupMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, GroupMessageReactionsRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.GET_HISTORY, params);
  }

  // ==================== 消息通用 API ====================

  /**
   * 标记所有消息已读
   */
  static async markAllMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, EmptyRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.ME.MARK_AS_READ, new EmptyRequest());
  }

  // ==================== 举报 API ====================

  /**
   * 创建举报
   */
  static async createReport(params: CreateReportRequest): Promise<ApiResponse<CreateReportData>> {
    return HttpUtil.post<ApiResponse<CreateReportData>, CreateReportRequest>(ApiEndpoints.REPORT.CREATE, params);
  }

  /**
   * 获取我的举报记录
   */
  static async getMyReports(params: MyReportsRequest): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<ReportItem>>, MyReportsRequest>(ApiEndpoints.REPORT.ME.LIST, params);
  }

  // ==================== 订阅设置 API ====================

  /**
   * 获取订阅配置
   */
  static async getSubscriptions(): Promise<ApiResponse<SubscriptionsData>> {
    return HttpUtil.post<ApiResponse<SubscriptionsData>, EmptyRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, new EmptyRequest());
  }

  /**
   * 更新订阅配置
   */
  static async updateSubscriptions(params: UpdateSubscriptionsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSubscriptionsRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, params);
  }

  // ==================== 设备推送 API ====================

  /**
   * 注册设备推送Token
   */
  static async registerDeviceToken(params: RegisterDeviceTokenRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RegisterDeviceTokenRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, params);
  }

  /**
   * 注销设备推送Token
   */
  static async unregisterDeviceToken(params: UnregisterDeviceTokenRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnregisterDeviceTokenRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, params);
  }

  /**
   * 获取设备推送Token列表
   */
  static async getDeviceTokens(params: MyDeviceTokensRequest): Promise<ApiResponse<PaginatedData<DeviceTokenItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<DeviceTokenItem>>, MyDeviceTokensRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, params);
  }

  // ==================== 信息流 API ====================

  /**
   * 获取瞬间信息流
   */
  static async getFeedMoments(params: GetFeedMomentsRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, GetFeedMomentsRequest>(ApiEndpoints.FEED.LIST_MOMENTS, params);
  }

  /**
   * 获取群组推荐
   */
  static async getFeedGroups(params: GetFeedGroupsRequest): Promise<ApiResponse<PaginatedData<FeedGroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedGroupItem>>, GetFeedGroupsRequest>(ApiEndpoints.FEED.LIST_GROUPS, params);
  }

  /**
   * 获取用户推荐
   */
  static async getFeedUsers(params: GetFeedUsersRequest): Promise<ApiResponse<PaginatedData<FeedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedUserItem>>, GetFeedUsersRequest>(ApiEndpoints.FEED.LIST_USERS, params);
  }

  /**
   * 获取热门内容
   */
  static async getFeedTrending(params: GetFeedTrendingRequest): Promise<ApiResponse<TrendingBundle>> {
    return HttpUtil.post<ApiResponse<TrendingBundle>, GetFeedTrendingRequest>(ApiEndpoints.FEED.LIST_TRENDING, params);
  }

  /**
   * 反馈信息流内容
   */
  static async feedFeedback(params: FeedFeedbackRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FeedFeedbackRequest>(ApiEndpoints.FEED.ME.FEEDBACK, params);
  }

  // ==================== 标签 API ====================

  /**
   * 获取热门标签
   */
  static async getTrendingTags(params: TrendingTagsRequest): Promise<ApiResponse<TrendingTagsData>> {
    return HttpUtil.post<ApiResponse<TrendingTagsData>, TrendingTagsRequest>(ApiEndpoints.TAG.LIST_TRENDING, params);
  }

  /**
   * 标签搜索建议
   */
  static async suggestTags(params: SuggestTagsRequest): Promise<ApiResponse<TagSuggestions>> {
    return HttpUtil.post<ApiResponse<TagSuggestions>, SuggestTagsRequest>(ApiEndpoints.SEARCH.SUGGEST, params);
  }

  // ==================== 地图 API ====================

  /**
   * 地图聚合/视口渲染
   */
  static async renderMap(params: MapRenderRequest): Promise<ApiResponse<MapRenderData>> {
    return HttpUtil.post<ApiResponse<MapRenderData>, MapRenderRequest>(ApiEndpoints.MAP.RENDER, params);
  }

  // ==================== 全局配置 API ====================

  /**
   * 拉取全局配置
   */
  static async fetchGlobalSettings(params: FetchGlobalSettingsRequest): Promise<ApiResponse<FetchSettingsData>> {
    return HttpUtil.post<ApiResponse<FetchSettingsData>, FetchGlobalSettingsRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, params);
  }
}
