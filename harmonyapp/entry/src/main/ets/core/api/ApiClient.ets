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
  
  // 评论
  CreateCommentRequest,
  DeleteCommentRequest,
  CommentsRequest,
  CommentListItem,
  CommentRepliesRequest,
  CommentLikeRequest,
  UnlikeCommentRequest,
  CommentLikesRequest,
  CommentLikeItem,
  
  // 私聊管理
  MyPrivateChatsRequest,
  GetGroupInvitesRequest,
  GetPrivateMessagesRequest,
  GetPrivateMessageReactionsRequest,
  GetGroupMessagesRequest,
  GetMessageReadStatusRequest,
  GetGroupMessageReactionsRequest,
  MyReportsRequest,
  MyDeviceTokensRequest,
  TrendingTagsRequest,
  TrendingTagsData,
  SuggestTagsRequest,
  TagSuggestData,
  UserInfo,

  // 地图
  MapRenderRequest,
  MapRenderData,

  // 配置
  FetchGlobalSettingsRequest,
  FetchSettingsData,
} from '../types/api';

export class ApiClient {
  // ==================== 文件上传 API ====================

  /**
   * 申请上传预签名
   */
  static async presignUpload(params: PresignUploadRequest): Promise<ApiResponse<PresignUploadData>> {
    return HttpUtil.post<ApiResponse<PresignUploadData>, PresignUploadRequest>(ApiEndpoints.protected.media.presignUpload, params);
  }

  /**
   * 完成上传
   */
  static async completeUpload(params: CompleteUploadRequest): Promise<ApiResponse<UploadMediaData>> {
    return HttpUtil.post<ApiResponse<UploadMediaData>, CompleteUploadRequest>(ApiEndpoints.protected.media.completeUpload, params);
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
    let file = fs.openSync(fileUri, fs.OpenMode.READ_ONLY);
    let stat = fs.statSync(file.fd);
    let size = stat.size;
    let buffer = new ArrayBuffer(size);
    fs.readSync(file.fd, buffer);
    fs.closeSync(file);

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
    const { upload_url, upload_id, headers } = presignRes.data;

    // 3. 上传到对象存储 (S3/MinIO)
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
    return HttpUtil.post<ApiResponse<HealthCheckData>, object>(ApiEndpoints.public.system.health, {});
  }

  // ==================== 认证 API ====================

  /**
   * 用户注册
   */
  static async register(params: RegisterRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, RegisterRequest>(ApiEndpoints.public.auth.register, params);
  }

  /**
   * 用户登录
   */
  static async login(params: LoginRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, LoginRequest>(ApiEndpoints.public.auth.login, params);
  }

  /**
   * 创建临时用户
   */
  static async createTempUser(params: CreateTempRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, CreateTempRequest>(ApiEndpoints.public.auth.guest, params);
  }

  /**
   * 刷新Token
   */
  static async refreshToken(params: RefreshTokenRequest): Promise<ApiResponse<AppAuthData>> {
    return HttpUtil.post<ApiResponse<AppAuthData>, RefreshTokenRequest>(ApiEndpoints.public.auth.refreshToken, params);
  }

  /**
   * 检查认证状态
   */
  static async checkAuthStatus(): Promise<ApiResponse<CheckStatusData>> {
    return HttpUtil.post<ApiResponse<CheckStatusData>, object>(ApiEndpoints.protected.auth.checkStatus, {});
  }

  /**
   * 用户登出
   */
  static async logout(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, object>(ApiEndpoints.protected.auth.logout, {});
  }

  /**
   * 请求密码重置
   */
  static async requestPasswordReset(params: RequestPasswordResetRequest): Promise<ApiResponse<RequestPasswordResetData>> {
    return HttpUtil.post<ApiResponse<RequestPasswordResetData>, RequestPasswordResetRequest>(ApiEndpoints.public.auth.requestPasswordReset, params);
  }

  /**
   * 验证重置码
   */
  static async verifyResetCode(params: VerifyResetCodeRequest): Promise<ApiResponse<VerifyResetCodeData>> {
    return HttpUtil.post<ApiResponse<VerifyResetCodeData>, VerifyResetCodeRequest>(ApiEndpoints.public.auth.verifyResetCode, params);
  }

  /**
   * 重置密码
   */
  static async resetPassword(params: ResetPasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ResetPasswordRequest>(ApiEndpoints.public.auth.resetPassword, params);
  }

  /**
   * 修改密码
   */
  static async changePassword(params: ChangePasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ChangePasswordRequest>(ApiEndpoints.protected.auth.me.changePassword, params);
  }

  /**
   * 获取活跃会话列表
   */
  static async getSessions(params: GetSessionsRequest): Promise<ApiResponse<PaginatedData<SessionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<SessionItem>>, GetSessionsRequest>(ApiEndpoints.protected.auth.me.sessions, params);
  }

  /**
   * 撤销会话
   */
  static async revokeSession(params: RevokeSessionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeSessionRequest>(ApiEndpoints.protected.auth.me.revokeSession, params);
  }

  // ==================== 用户 API ====================

  /**
   * 按关键字搜索用户
   */
  static async searchUsersByKeyword(params: SearchUsersByKeywordRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByKeywordRequest>(ApiEndpoints.public.user.searchByKeyword, params);
  }

  /**
   * 按标签搜索用户
   */
  static async searchUsersByTag(params: SearchUsersByTagRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByTagRequest>(ApiEndpoints.public.user.searchByTag, params);
  }

  /**
   * 按位置搜索用户
   */
  static async searchUsersByLocation(params: SearchUsersByLocationRequest): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserSearchItem>>, SearchUsersByLocationRequest>(ApiEndpoints.public.user.searchByLocation, params);
  }

  /**
   * 获取用户资料（包含在线状态，受隐私设置控制）
   */
  static async getUserProfile(params: UserOpenTheDoorRequest): Promise<ApiResponse<UserSearchItem>> {
    return HttpUtil.post<ApiResponse<UserSearchItem>, UserOpenTheDoorRequest>(ApiEndpoints.public.user.openTheDoor, params);
  }

  /**
   * 获取用户关系
   */
  static async getRelationship(params: GetRelationshipRequest): Promise<ApiResponse<UserRelation>> {
    return HttpUtil.post<ApiResponse<UserRelation>, GetRelationshipRequest>(ApiEndpoints.protected.user.relationship, params);
  }

  /**
   * 获取用户统计
   */
  static async getUserStats(params: GetUserStatsRequest): Promise<ApiResponse<UserStatsData>> {
    return HttpUtil.post<ApiResponse<UserStatsData>, GetUserStatsRequest>(ApiEndpoints.public.user.stats, params);
  }

  /**
   * 获取指定用户好友列表
   */
  static async getUserFriendsList(params: GetUserFriendsListRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, GetUserFriendsListRequest>(ApiEndpoints.public.user.friends, params);
  }

  /**
   * 获取指定用户关注列表
   */
  static async getUserFollowingList(params: GetUserFollowingListRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, GetUserFollowingListRequest>(ApiEndpoints.public.user.following, params);
  }

  /**
   * 获取指定用户粉丝列表
   */
  static async getUserFollowersList(params: GetUserFollowersListRequest): Promise<ApiResponse<PaginatedData<UserInfo>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserInfo>>, GetUserFollowersListRequest>(ApiEndpoints.public.user.followers, params);
  }

  // ==================== 当前用户 API（/me/）====================

  /**
   * 获取当前用户资料
   */
  static async getMyProfile(): Promise<ApiResponse<MyProfileData>> {
    return HttpUtil.post<ApiResponse<MyProfileData>, object>(ApiEndpoints.protected.user.me.profile, {});
  }

  /**
   * 更新用户资料
   */
  static async updateProfile(params: UpdateProfileRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateProfileRequest>(ApiEndpoints.protected.user.me.updateProfile, params);
  }

  /**
   * 更新用户位置
   */
  static async updateLocation(params: UpdateLocationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateLocationRequest>(ApiEndpoints.protected.user.me.location, params);
  }

  /**
   * 账户升级
   */
  static async upgradeAccount(params: UpgradeAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpgradeAccountRequest>(ApiEndpoints.protected.user.me.upgrade, params);
  }

  /**
   * 获取我的统计数据
   */
  static async getMyStats(): Promise<ApiResponse<UserStatsData>> {
    return HttpUtil.post<ApiResponse<UserStatsData>, object>(ApiEndpoints.protected.user.me.stats, {});
  }

  /**
   * 获取用户设置
   */
  static async getSettings(): Promise<ApiResponse<UserSettingsData>> {
    return HttpUtil.post<ApiResponse<UserSettingsData>, object>(ApiEndpoints.protected.user.me.settings, {});
  }

  /**
   * 更新用户设置
   */
  static async updateSettings(params: UpdateSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSettingsRequest>(ApiEndpoints.protected.user.me.updateSettings, params);
  }

  /**
   * 删除账户
   */
  static async deleteAccount(params: DeleteAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteAccountRequest>(ApiEndpoints.protected.user.me.deleteAccount, params);
  }

  /**
   * 关注用户
   */
  static async followUser(params: FollowUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FollowUserRequest>(ApiEndpoints.protected.user.me.follow, params);
  }

  /**
   * 取消关注
   */
  static async unfollowUser(params: UnfollowUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnfollowUserRequest>(ApiEndpoints.protected.user.me.unfollow, params);
  }

  /**
   * 拉黑用户
   */
  static async blockUser(params: BlockUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BlockUserRequest>(ApiEndpoints.protected.user.me.block, params);
  }

  /**
   * 解除拉黑
   */
  static async unblockUser(params: UnblockUserRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnblockUserRequest>(ApiEndpoints.protected.user.me.unblock, params);
  }

  /**
   * 获取好友列表
   */
  static async getFriendsList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FriendListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FriendListItem>>, object>(ApiEndpoints.protected.user.me.friends, params);
  }

  /**
   * 获取关注列表
   */
  static async getFollowingList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowingItem>>, object>(ApiEndpoints.protected.user.me.following, params);
  }

  /**
   * 获取粉丝列表
   */
  static async getFollowersList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowerItem>>, object>(ApiEndpoints.protected.user.me.followers, params);
  }

  /**
   * 获取拉黑列表
   */
  static async getBlockedList(params: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BlockedUserItem>>, object>(ApiEndpoints.protected.user.me.blocked, params);
  }

  /**
   * 设置好友备注
   */
  static async setFriendRemark(params: SetFriendRemarkRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetFriendRemarkRequest>(ApiEndpoints.protected.user.me.remark, params);
  }

  // ==================== 生活瞬间 API ====================

  /**
   * 按关键字搜索瞬间
   */
  static async searchMomentsByKeyword(params: SearchMomentsByKeywordRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByKeywordRequest>(ApiEndpoints.public.moments.searchByKeyword, params);
  }

  /**
   * 按标签搜索瞬间
   */
  static async searchMomentsByTag(params: SearchMomentsByTagRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByTagRequest>(ApiEndpoints.public.moments.searchByTag, params);
  }

  /**
   * 按位置搜索瞬间
   */
  static async searchMomentsByLocation(params: SearchMomentsByLocationRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, SearchMomentsByLocationRequest>(ApiEndpoints.public.moments.searchByLocation, params);
  }

  /**
   * 获取瞬间详情
   */
  static async getMomentDetail(params: MomentOpenTheDoorRequest): Promise<ApiResponse<MomentSearchItem>> {
    return HttpUtil.post<ApiResponse<MomentSearchItem>, MomentOpenTheDoorRequest>(ApiEndpoints.public.moments.openTheDoor, params);
  }

  /**
   * 获取用户瞬间列表
   */
  static async getUserMoments(params: GetUserMomentsRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, GetUserMomentsRequest>(ApiEndpoints.public.user.moments, params);
  }

  /**
   * 获取瞬间评论列表
   */
  static async getMomentComments(params: GetCommentsRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, GetCommentsRequest>(ApiEndpoints.public.moments.comments, params);
  }

  /**
   * 获取瞬间点赞列表
   */
  static async getMomentLikes(params: GetMomentLikesRequest): Promise<ApiResponse<PaginatedData<MomentLikeItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentLikeItem>>, GetMomentLikesRequest>(ApiEndpoints.public.moments.likes, params);
  }

  /**
   * 创建瞬间
   */
  static async createMoment(params: CreateMomentRequest): Promise<ApiResponse<MomentSearchItem>> {
    return HttpUtil.post<ApiResponse<MomentSearchItem>, CreateMomentRequest>(ApiEndpoints.protected.moments.create, params);
  }

  /**
   * 编辑瞬间
   */
  static async updateMoment(params: EditMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, EditMomentRequest>(ApiEndpoints.protected.moments.update, params);
  }

  /**
   * 删除瞬间
   */
  static async deleteMoment(params: DeleteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteMomentRequest>(ApiEndpoints.protected.moments.delete, params);
  }

  /**
   * 点赞瞬间
   */
  static async likeMoment(params: LikeMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LikeMomentRequest>(ApiEndpoints.protected.moments.like, params);
  }

  /**
   * 取消点赞瞬间
   */
  static async unlikeMoment(params: UnlikeMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnlikeMomentRequest>(ApiEndpoints.protected.moments.unlike, params);
  }

  /**
   * 收藏瞬间
   */
  static async favoriteMoment(params: FavoriteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FavoriteMomentRequest>(ApiEndpoints.protected.moments.favorite, params);
  }

  /**
   * 取消收藏瞬间
   */
  static async unfavoriteMoment(params: UnfavoriteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnfavoriteMomentRequest>(ApiEndpoints.protected.moments.unfavorite, params);
  }

  // ==================== 当前用户瞬间 API（/me/）====================

  /**
   * 获取我发布的瞬间
   */
  static async getMyMoments(params: { latitude: number; longitude: number; limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, object>(ApiEndpoints.protected.moments.me, params);
  }

  /**
   * 获取我的收藏
   */
  static async getMyFavorites(params: { latitude: number; longitude: number; limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, object>(ApiEndpoints.protected.moments.myFavorites, params);
  }

  /**
   * 检查瞬间是否已点赞
   */
  static async checkMomentLiked(params: CheckMomentLikedRequest): Promise<ApiResponse<CheckLikedData>> {
    return HttpUtil.post<ApiResponse<CheckLikedData>, CheckMomentLikedRequest>(ApiEndpoints.protected.moments.checkLiked, params);
  }

  /**
   * 检查瞬间是否已收藏
   */
  static async checkMomentFavorited(params: CheckMomentFavoritedRequest): Promise<ApiResponse<CheckFavoritedData>> {
    return HttpUtil.post<ApiResponse<CheckFavoritedData>, CheckMomentFavoritedRequest>(ApiEndpoints.protected.moments.checkFavorited, params);
  }

  // ==================== 评论 API ====================

  /**
   * 获取评论列表
   */
  static async getComments(params: GetCommentsRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, GetCommentsRequest>(ApiEndpoints.public.comments.list, params);
  }

  /**
   * 获取子评论
   */
  static async getReplies(params: GetRepliesRequest): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentListItem>>, GetRepliesRequest>(ApiEndpoints.public.comments.replies, params);
  }

  /**
   * 获取评论点赞列表
   */
  static async getCommentLikes(params: GetCommentLikesRequest): Promise<ApiResponse<PaginatedData<CommentLikeItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentLikeItem>>, GetCommentLikesRequest>(ApiEndpoints.public.comments.likes, params);
  }

  /**
   * 添加评论
   */
  static async createComment(params: CreateCommentRequest): Promise<ApiResponse<CommentListItem>> {
    return HttpUtil.post<ApiResponse<CommentListItem>, CreateCommentRequest>(ApiEndpoints.protected.comments.create, params);
  }

  /**
   * 删除评论
   */
  static async deleteComment(params: DeleteCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteCommentRequest>(ApiEndpoints.protected.comments.delete, params);
  }

  /**
   * 点赞评论
   */
  static async likeComment(params: LikeCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LikeCommentRequest>(ApiEndpoints.protected.comments.like, params);
  }

  /**
   * 取消点赞评论
   */
  static async unlikeComment(params: UnlikeCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnlikeCommentRequest>(ApiEndpoints.protected.comments.unlike, params);
  }

  // ==================== 私聊管理 API ====================

  /**
   * 获取私聊列表
   */
  static async getPrivateChats(params: MyPrivateChatsRequest): Promise<ApiResponse<PaginatedData<PrivateChatListItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateChatListItem>>, MyPrivateChatsRequest>(ApiEndpoints.protected.privateChats.me, params);
  }

  /**
   * 获取私聊详情 - open_the_door
   */
  static async privateChatOpenTheDoor(params: PrivateChatOpenTheDoorRequest): Promise<ApiResponse<PrivateChatDetailData>> {
    return HttpUtil.post<ApiResponse<PrivateChatDetailData>, PrivateChatOpenTheDoorRequest>(ApiEndpoints.protected.privateChats.openTheDoor, params);
  }

  /**
   * 更新私聊设置
   */
  static async updatePrivateChatSettings(params: UpdatePrivateChatSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateChatSettingsRequest>(ApiEndpoints.protected.privateChats.updateSettings, params);
  }

  /**
   * 删除私聊
   */
  static async deletePrivateChat(params: DeletePrivateChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateChatRequest>(ApiEndpoints.protected.privateChats.delete, params);
  }

  // ==================== 群组管理 API ====================

  /**
   * 按关键字搜索群组
   */
  static async searchGroupsByKeyword(params: SearchGroupsByKeywordRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByKeywordRequest>(ApiEndpoints.public.groups.searchByKeyword, params);
  }

  /**
   * 按标签搜索群组
   */
  static async searchGroupsByTag(params: SearchGroupsByTagRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByTagRequest>(ApiEndpoints.public.groups.searchByTag, params);
  }

  /**
   * 按位置搜索群组
   */
  static async searchGroupsByLocation(params: SearchGroupsByLocationRequest): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSearchItem>>, SearchGroupsByLocationRequest>(ApiEndpoints.public.groups.searchByLocation, params);
  }

  /**
   * 获取群组详情
   */
  static async getGroupDetail(params: GroupOpenTheDoorRequest): Promise<ApiResponse<GroupSearchItem>> {
    return HttpUtil.post<ApiResponse<GroupSearchItem>, GroupOpenTheDoorRequest>(ApiEndpoints.public.groups.openTheDoor, params);
  }

  /**
   * 获取群组设置
   */
  static async getGroupSettings(params: GetGroupSettingsRequest): Promise<ApiResponse<GroupSettingsData>> {
    return HttpUtil.post<ApiResponse<GroupSettingsData>, GetGroupSettingsRequest>(ApiEndpoints.public.groups.settings, params);
  }

  /**
   * 获取群组成员列表
   */
  static async getGroupMembers(params: GetMembersRequest): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMemberItem>>, GetMembersRequest>(ApiEndpoints.public.groups.members, params);
  }

  /**
   * 创建群组
   */
  static async createGroupChat(params: CreateGroupRequest): Promise<ApiResponse<JoinedGroupItem>> {
    return HttpUtil.post<ApiResponse<JoinedGroupItem>, CreateGroupRequest>(ApiEndpoints.protected.groups.create, params);
  }

  /**
   * 更新群组
   */
  static async updateGroupChat(params: UpdateGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupRequest>(ApiEndpoints.protected.groups.update, params);
  }

  /**
   * 解散群组
   */
  static async deleteGroupChat(params: DeleteGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupRequest>(ApiEndpoints.protected.groups.delete, params);
  }

  /**
   * 邀请成员
   */
  static async inviteMember(params: InviteMemberRequest): Promise<ApiResponse<InviteMemberData>> {
    return HttpUtil.post<ApiResponse<InviteMemberData>, InviteMemberRequest>(ApiEndpoints.protected.groups.members.invite, params);
  }

  /**
   * 加入群组
   */
  static async joinGroup(params: JoinGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, JoinGroupRequest>(ApiEndpoints.protected.groups.members.join, params);
  }

  /**
   * 申请加入群组
   */
  static async applyToGroup(params: ApplyToGroupRequest): Promise<ApiResponse<ApplyToGroupData>> {
    return HttpUtil.post<ApiResponse<ApplyToGroupData>, ApplyToGroupRequest>(ApiEndpoints.protected.groups.members.apply, params);
  }

  /**
   * 设置成员角色
   */
  static async setMemberRole(params: SetMemberRoleRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetMemberRoleRequest>(ApiEndpoints.protected.groups.members.setRole, params);
  }

  /**
   * 禁言成员
   */
  static async muteMember(params: MuteMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MuteMemberRequest>(ApiEndpoints.protected.groups.members.mute, params);
  }

  /**
   * 取消禁言
   */
  static async unmuteMember(params: UnmuteMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnmuteMemberRequest>(ApiEndpoints.protected.groups.members.unmute, params);
  }

  /**
   * 封禁成员
   */
  static async banMember(params: BanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BanMemberRequest>(ApiEndpoints.protected.groups.members.ban, params);
  }

  /**
   * 解除封禁
   */
  static async unbanMember(params: UnbanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnbanMemberRequest>(ApiEndpoints.protected.groups.members.unban, params);
  }

  /**
   * 获取封禁列表
   */
  static async getBannedMembers(params: GetBannedMembersRequest): Promise<ApiResponse<PaginatedData<BannedMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BannedMemberItem>>, GetBannedMembersRequest>(ApiEndpoints.protected.groups.members.banned, params);
  }

  /**
   * 移除成员
   */
  static async removeMember(params: RemoveMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveMemberRequest>(ApiEndpoints.protected.groups.members.removeMember, params);
  }

  /**
   * 获取群组申请列表
   */
  static async getGroupApplications(params: GetGroupApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, GetGroupApplicationsRequest>(ApiEndpoints.protected.groups.applications.list, params);
  }

  /**
   * 审批群组申请
   */
  static async approveGroupApplication(params: ApproveGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ApproveGroupApplicationRequest>(ApiEndpoints.protected.groups.applications.approve, params);
  }

  /**
   * 拒绝群组申请
   */
  static async rejectGroupApplication(params: RejectGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectGroupApplicationRequest>(ApiEndpoints.protected.groups.applications.reject, params);
  }

  /**
   * 转让群主
   */
  static async transferOwnership(params: TransferOwnershipRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, TransferOwnershipRequest>(ApiEndpoints.protected.groups.transferOwnership, params);
  }

  // ==================== 当前用户群组 API（/me/）====================

  /**
   * 获取我加入的群组
   */
  static async getJoinedGroups(params: MyJoinedGroupsRequest): Promise<ApiResponse<PaginatedData<JoinedGroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<JoinedGroupItem>>, MyJoinedGroupsRequest>(ApiEndpoints.protected.groups.me.list, params);
  }

  static async getGroupInvites(params: MyGroupInvitesRequest): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupInviteItem>>, MyGroupInvitesRequest>(ApiEndpoints.protected.groups.me.invites, params);
  }

  static async getPrivateMessages(params: GetPrivateMessagesRequest): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateMessageItem>>, GetPrivateMessagesRequest>(ApiEndpoints.protected.privateMessages.history, params);
  }

  static async getPrivateMessageReactions(params: GetPrivateMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, GetPrivateMessageReactionsRequest>(ApiEndpoints.protected.privateMessages.reactions, params);
  }

  /**
   * 取消群组申请
   */
  static async cancelGroupApplication(params: CancelGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, CancelGroupApplicationRequest>(ApiEndpoints.protected.groups.applications.cancel, params);
  }

  /**
   * 获取我的群组邀请
   */
  static async getGroupInvites(params: GetGroupInvitesRequest): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupInviteItem>>, GetGroupInvitesRequest>(ApiEndpoints.protected.groups.me.invites, params);
  }

  /**
   * 接受群组邀请
   */
  static async acceptGroupInvite(params: AcceptGroupInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AcceptGroupInviteRequest>(ApiEndpoints.protected.groups.me.acceptInvite, params);
  }

  /**
   * 拒绝群组邀请
   */
  static async rejectGroupInvite(params: RejectGroupInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectGroupInviteRequest>(ApiEndpoints.protected.groups.me.rejectInvite, params);
  }

  /**
   * 退出群组
   */
  static async leaveGroup(params: LeaveGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LeaveGroupRequest>(ApiEndpoints.protected.groups.me.leave, params);
  }

  // ==================== 私聊消息 API ====================

  /**
   * 发送私聊消息
   */
  static async sendPrivateMessage(params: SendPrivateMessageRequest): Promise<ApiResponse<PrivateMessageItem>> {
    return HttpUtil.post<ApiResponse<PrivateMessageItem>, SendPrivateMessageRequest>(ApiEndpoints.protected.privateMessages.send, params);
  }

  /**
   * 获取私聊消息历史
   */
  static async getPrivateMessages(params: GetPrivateMessagesRequest): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateMessageItem>>, GetPrivateMessagesRequest>(ApiEndpoints.protected.privateMessages.history, params);
  }

  /**
   * 编辑私聊消息
   */
  static async updatePrivateMessage(params: UpdatePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateMessageRequest>(ApiEndpoints.protected.privateMessages.update, params);
  }

  /**
   * 撤回私聊消息
   */
  static async recallPrivateMessage(params: RecallPrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallPrivateMessageRequest>(ApiEndpoints.protected.privateMessages.recall, params);
  }

  /**
   * 删除私聊消息
   */
  static async deletePrivateMessage(params: DeletePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateMessageRequest>(ApiEndpoints.protected.privateMessages.delete, params);
  }

  /**
   * 标记私聊消息已读
   */
  static async markPrivateMessagesRead(params: MarkPrivateMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkPrivateMessagesReadRequest>(ApiEndpoints.protected.privateMessages.markAsRead, params);
  }

  /**
   * 添加私聊消息反应
   */
  static async addPrivateMessageReaction(params: AddPrivateMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AddPrivateMessageReactionRequest>(ApiEndpoints.protected.privateMessages.addReaction, params);
  }

  /**
   * 移除私聊消息反应
   */
  static async removePrivateMessageReaction(params: RemovePrivateMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemovePrivateMessageReactionRequest>(ApiEndpoints.protected.privateMessages.removeReaction, params);
  }

  /**
   * 获取私聊消息反应列表
   */
  static async getPrivateMessageReactions(params: GetPrivateMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, GetPrivateMessageReactionsRequest>(ApiEndpoints.protected.privateMessages.reactions, params);
  }

  // ==================== 群聊消息 API ====================

  /**
   * 发送群聊消息
   */
  static async sendGroupMessage(params: SendGroupMessageRequest): Promise<ApiResponse<GroupMessageItem>> {
    return HttpUtil.post<ApiResponse<GroupMessageItem>, SendGroupMessageRequest>(ApiEndpoints.protected.groupMessages.send, params);
  }

  /**
   * 获取群聊消息历史
   */
  static async getGroupMessages(params: GetGroupMessagesRequest): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMessageItem>>, GetGroupMessagesRequest>(ApiEndpoints.protected.groupMessages.history, params);
  }

  /**
   * 编辑群聊消息
   */
  static async updateGroupMessage(params: UpdateGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupMessageRequest>(ApiEndpoints.protected.groupMessages.update, params);
  }

  /**
   * 撤回群聊消息
   */
  static async recallGroupMessage(params: RecallGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallGroupMessageRequest>(ApiEndpoints.protected.groupMessages.recall, params);
  }

  /**
   * 删除群聊消息
   */
  static async deleteGroupMessage(params: DeleteGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupMessageRequest>(ApiEndpoints.protected.groupMessages.delete, params);
  }

  /**
   * 标记群聊消息已读
   */
  static async markGroupMessagesRead(params: MarkGroupMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkGroupMessagesReadRequest>(ApiEndpoints.protected.groupMessages.markAsRead, params);
  }

  /**
   * 获取消息已读状态
   */
  static async getMessageReadStatus(params: GetMessageReadStatusRequest): Promise<ApiResponse<PaginatedData<MessageReadItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReadItem>>, GetMessageReadStatusRequest>(ApiEndpoints.protected.groupMessages.readStatus, params);
  }

  /**
   * 添加群聊消息反应
   */
  static async addGroupMessageReaction(params: AddGroupMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, AddGroupMessageReactionRequest>(ApiEndpoints.protected.groupMessages.addReaction, params);
  }

  /**
   * 移除群聊消息反应
   */
  static async removeGroupMessageReaction(params: RemoveGroupMessageReactionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveGroupMessageReactionRequest>(ApiEndpoints.protected.groupMessages.removeReaction, params);
  }

  /**
   * 获取群聊消息反应列表
   */
  static async getGroupMessageReactions(params: GetGroupMessageReactionsRequest): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MessageReactionItem>>, GetGroupMessageReactionsRequest>(ApiEndpoints.protected.groupMessages.reactions, params);
  }

  // ==================== 消息通用 API ====================

  /**
   * 标记所有消息已读
   */
  static async markAllMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, object>(ApiEndpoints.protected.messages.markAllAsRead, {});
  }

  // ==================== 举报 API ====================

  /**
   * 创建举报
   */
  static async createReport(params: CreateReportRequest): Promise<ApiResponse<CreateReportData>> {
    return HttpUtil.post<ApiResponse<CreateReportData>, CreateReportRequest>(ApiEndpoints.protected.reports.create, params);
  }

  /**
   * 获取我的举报记录
   */
  static async getMyReports(params: GetMyReportsRequest): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<ReportItem>>, GetMyReportsRequest>(ApiEndpoints.protected.reports.me, params);
  }

  // ==================== 订阅设置 API ====================

  /**
   * 获取订阅配置
   */
  static async getSubscriptions(): Promise<ApiResponse<SubscriptionsData>> {
    return HttpUtil.post<ApiResponse<SubscriptionsData>, object>(ApiEndpoints.protected.subscriptions.me, {});
  }

  /**
   * 更新订阅配置
   */
  static async updateSubscriptions(params: UpdateSubscriptionsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSubscriptionsRequest>(ApiEndpoints.protected.subscriptions.update, params);
  }

  // ==================== 设备推送 API ====================

  /**
   * 注册设备推送Token
   */
  static async registerDeviceToken(params: RegisterDeviceTokenRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RegisterDeviceTokenRequest>(ApiEndpoints.protected.deviceTokens.me.register, params);
  }

  /**
   * 注销设备推送Token
   */
  static async unregisterDeviceToken(params: UnregisterDeviceTokenRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnregisterDeviceTokenRequest>(ApiEndpoints.protected.deviceTokens.me.unregister, params);
  }

  /**
   * 获取设备推送Token列表
   */
  static async getDeviceTokens(params: GetDeviceTokensRequest): Promise<ApiResponse<PaginatedData<DeviceTokenItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<DeviceTokenItem>>, GetDeviceTokensRequest>(ApiEndpoints.protected.deviceTokens.me.list, params);
  }

  // ==================== 信息流 API ====================

  /**
   * 获取瞬间信息流
   */
  static async getFeedMoments(params: GetFeedMomentsRequest): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentSearchItem>>, GetFeedMomentsRequest>(ApiEndpoints.protected.feed.moments, params);
  }

  /**
   * 获取群组推荐
   */
  static async getFeedGroups(params: GetFeedGroupsRequest): Promise<ApiResponse<PaginatedData<FeedGroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedGroupItem>>, GetFeedGroupsRequest>(ApiEndpoints.protected.feed.groups, params);
  }

  /**
   * 获取用户推荐
   */
  static async getFeedUsers(params: GetFeedUsersRequest): Promise<ApiResponse<PaginatedData<FeedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FeedUserItem>>, GetFeedUsersRequest>(ApiEndpoints.protected.feed.users, params);
  }

  /**
   * 获取热门内容
   */
  static async getFeedTrending(params: GetFeedTrendingRequest): Promise<ApiResponse<FeedTrendingData>> {
    return HttpUtil.post<ApiResponse<FeedTrendingData>, GetFeedTrendingRequest>(ApiEndpoints.protected.feed.trending, params);
  }

  /**
   * 反馈信息流内容
   */
  static async feedFeedback(params: FeedFeedbackRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, FeedFeedbackRequest>(ApiEndpoints.protected.feed.feedback, params);
  }

  // ==================== 标签 API ====================

  /**
   * 获取热门标签
   */
  static async getTrendingTags(params: GetTrendingTagsRequest): Promise<ApiResponse<TrendingTagsData>> {
    return HttpUtil.post<ApiResponse<TrendingTagsData>, GetTrendingTagsRequest>(ApiEndpoints.public.tags.trending, params);
  }

  /**
   * 标签搜索建议
   */
  static async suggestTags(params: SuggestTagsRequest): Promise<ApiResponse<TagSuggestData>> {
    return HttpUtil.post<ApiResponse<TagSuggestData>, SuggestTagsRequest>(ApiEndpoints.public.tags.suggest, params);
  }

  // ==================== 地图 API ====================

  /**
   * 地图聚合/视口渲染
   */
  static async renderMap(params: MapRenderRequest): Promise<ApiResponse<MapRenderData>> {
    return HttpUtil.post<ApiResponse<MapRenderData>, MapRenderRequest>(ApiEndpoints.public.map.render, params);
  }

  // ==================== 全局配置 API ====================

  /**
   * 拉取全局配置
   */
  static async fetchGlobalSettings(params: FetchGlobalSettingsRequest): Promise<ApiResponse<FetchSettingsData>> {
    return HttpUtil.post<ApiResponse<FetchSettingsData>, FetchGlobalSettingsRequest>(ApiEndpoints.public.globalSettings.fetch, params);
  }
}
