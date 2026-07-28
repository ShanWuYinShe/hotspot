/**
 * API 客户端
 * 严格按照 api_spec_*.md 定义的 API 端点实现
 */

import { HttpUtil } from '../network/HttpUtil';
import { ApiEndpoints } from '../config/ApiEndpoints';
import { GlobalStateManager } from '../state/GlobalStateManager';
import common from '@ohos.app.ability.common';
import fs from '@ohos.file.fs';
import { hilog } from '@kit.PerformanceAnalysisKit';
import {
  // 基础类型
  ApiResponse,
  PaginatedData,
  PresignUploadRequest,
  PresignedUpload,
  CompleteUploadRequest,
  UploadedMedia,
  BatchCompleteUploadRequest,
  BatchCompleteUpload,
  MediaCategory,
  BatchPresignUploadRequest,
  BatchPresignedUpload,

  // 认证相关
  AuthData,
  RegisterRequest,
  LoginRequest,
  GuestRequest,
  RefreshTokenRequest,
  AuthStatus,
  RequestPasswordResetRequest,
  ResetPasswordRequest,
  ChangePasswordRequest,
  SessionsRequest,
  SessionItem,
  RevokeSessionRequest,
  RevokeAllResult,

  // 用户
  UserItem,
  SearchUsersRequest,
  UserOpenTheDoorRequest,
  MyProfile,
  UpdateProfileRequest,
  UpdateLocationRequest,
  UpgradeAccountRequest,
  UserSettings,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  FollowRequest,
  UnfollowRequest,
  BlockRequest,
  UnblockRequest,
  FriendItem,
  FollowingItem,
  FollowerItem,
  BlockedUserItem,
  SetRemarkRequest,
  ListFriendsRequest,
  ListFollowingRequest,
  ListFollowersRequest,
  MyBlockedRequest,
  UserGroupsRequest,
  UserMomentsRequest,
  UserLikedMomentsRequest,
  GroupItem,
  UserProfileBundle,

  // 生活瞬间
  MomentItem,
  SearchMomentsRequest,
  MomentOpenTheDoorRequest,
  CreateMomentRequest,
  EditMomentRequest,
  DeleteMomentRequest,
  LikeMomentRequest,
  UnlikeMomentRequest,
  FavoriteMomentRequest,
  UnfavoriteMomentRequest,
  MomentLikesRequest,
  ReactionItem,
  MyMomentsRequest,
  MyFavoritesRequest,

  // 评论
  CreateCommentRequest,
  DeleteCommentRequest,
  CommentsRequest,
  CommentItem,
  CommentRepliesRequest,
  LikeCommentRequest,
  UnlikeCommentRequest,
  CommentLikesRequest,

  // Reactions
  SetReactionRequest,
  ListReactionsRequest,

  // 私聊管理
  MyPrivateChatsRequest,
  PrivateChatItem,
  PrivateChatOpenTheDoorRequest,
  PrivateChatOpenResult,
  UpdatePrivateChatSettingsRequest,
  DeletePrivateChatRequest,

  // 私聊消息
  SendPrivateMessageRequest,
  MessageSent,
  PrivateMessageItem,
  GetPrivateMessagesHistoryRequest,
  UpdatePrivateMessageRequest,
  RecallPrivateMessageRequest,
  MarkPrivateMessagesReadRequest,

  // 群组管理
  SearchGroupsRequest,
  GroupOpenTheDoorRequest,
  CreateGroupRequest,
  UpdateGroupRequest,
  DismissGroupRequest,
  UpdateGroupMySettingsRequest,
  GroupMembersRequest,
  GroupMemberItem,
  InviteMemberRequest,
  Invitation,
  JoinGroupRequest,
  JoinGroupResult,
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
  MyApplicationsRequest,
  CancelApplicationRequest,
  GroupSentInvitesRequest,
  SentInviteItem,
  RevokeSentInviteRequest,
  MyInvitesRequest,
  GroupInviteItem,
  AcceptInviteRequest,
  InviteAcceptedResult,
  RejectInviteRequest,
  LeaveGroupRequest,
  DismissFeedItemRequest,

  // 群公告
  CreateGroupAnnouncementRequest,
  GroupAnnouncementsRequest,
  GroupAnnouncementItem,
  UpdateGroupAnnouncementRequest,
  DeleteGroupAnnouncementRequest,

  // 群聊消息
  SendGroupMessageRequest,
  GroupMessageItem,
  GetGroupMessagesHistoryRequest,
  UpdateGroupMessageRequest,
  RecallGroupMessageRequest,
  MarkGroupMessagesReadRequest,

  // 举报
  CreateReportRequest,
  ReportCreated,
  ReportItem,
  MyReportsRequest,

  // 信息流
  GetFeedMomentsRequest,
  GetFeedGroupsRequest,
  GetFeedUsersRequest,
  GetFeedTrendingRequest,
  TrendingBundle,

  TagItem,

	// 搜索
	SearchSuggestRequest,
	SearchSuggestResponse,
	SearchAllRequest,
	SearchAllResult,

	// 地图
	MapHeatCanvas,
	MapHotspotsCanvas,
	MapCanvasRequest,
	MapCanvasResponse,

  // 系统
  HealthStatus,
  ClientConfig,
  SystemAnnouncementItem,
  AnnouncementListRequest,
  ApiVersionMeta,
  AdminPermissions,

  // 分享
  CreateShareRequest,
  ShareCreated,
  GetShareRequest,
  ShareDetail,
  RevokeShareRequest,
  ShareListRequest,
  ShareListItem,

  // Response 包装类型
  SystemAnnouncementListData,
  SessionListData,
  GroupAnnouncementListData,

  // 通知中心
  NotificationItem,
  NotificationListRequest,
  UnreadCountResult,
  MarkAsReadRequest,
} from '../types/api';

const DOMAIN_NUMBER: number = 0xFF21;
const TAG: string = 'ApiClient';

// 空请求体类
class EmptyRequest {}

export class ApiClient {
  // ==================== 文件上传 API ====================

  private static resolveReadableFilePath(context: common.UIAbilityContext, fileUri: string): string {
    if (fileUri.startsWith('internal://cache/')) {
      const cacheRelativePath = fileUri.substring('internal://cache/'.length);
      return `${context.cacheDir}/${cacheRelativePath}`;
    }
    if (fileUri.startsWith('file://')) {
      const strippedPath = fileUri.substring('file://'.length);
      // 仅处理 file:///absolute/path；file://media/... 等 URI 保留给文件系统 API 解析
      if (strippedPath.startsWith('/')) {
        return strippedPath;
      }
      return fileUri;
    }
    return fileUri;
  }

  private static extractFileName(fileUri: string, resolvedPath: string): string {
    const source = fileUri.length > 0 ? fileUri : resolvedPath;
    const normalized = source.endsWith('/') ? source.substring(0, source.length - 1) : source;
    const parts = normalized.split('/');
    const candidate = parts.length > 0 ? parts[parts.length - 1] : '';
    const cleanCandidate = candidate.split('?')[0].split('#')[0];
    if (!cleanCandidate) {
      return `upload_${Date.now()}`;
    }
    return cleanCandidate;
  }

  private static inferMimeType(fileName: string, category: MediaCategory): string {
    const lower = fileName.toLowerCase();
    if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) return 'image/jpeg';
    if (lower.endsWith('.png')) return 'image/png';
    if (lower.endsWith('.webp')) return 'image/webp';
    if (lower.endsWith('.gif')) return 'image/gif';
    if (lower.endsWith('.mp4')) return 'video/mp4';
    if (lower.endsWith('.mov')) return 'video/quicktime';
    if (lower.endsWith('.mp3')) return 'audio/mpeg';
    if (lower.endsWith('.aac') || lower.endsWith('.m4a')) return 'audio/aac';
    if (lower.endsWith('.ogg')) return 'audio/ogg';

    switch (category) {
      case 'avatar':
      case 'group_avatar':
      case 'moment_image':
      case 'chat_image':
        return 'image/jpeg';
      case 'chat_video':
        return 'video/mp4';
      case 'chat_audio':
        return 'audio/aac';
      default:
        return 'application/octet-stream';
    }
  }

  static async presignUpload(params: PresignUploadRequest): Promise<ApiResponse<PresignedUpload>> {
    return HttpUtil.post<ApiResponse<PresignedUpload>, PresignUploadRequest>(ApiEndpoints.UPLOAD.PRESIGN_UPLOAD, params);
  }

  static async batchPresignUpload(params: BatchPresignUploadRequest): Promise<ApiResponse<BatchPresignedUpload>> {
    return HttpUtil.post<ApiResponse<BatchPresignedUpload>, BatchPresignUploadRequest>(ApiEndpoints.UPLOAD.BATCH_PRESIGN_UPLOAD, params);
  }

  static async completeUpload(params: CompleteUploadRequest): Promise<ApiResponse<UploadedMedia>> {
    return HttpUtil.post<ApiResponse<UploadedMedia>, CompleteUploadRequest>(ApiEndpoints.UPLOAD.COMPLETE_UPLOAD, params);
  }

  static async batchCompleteUpload(params: BatchCompleteUploadRequest): Promise<ApiResponse<BatchCompleteUpload>> {
    return HttpUtil.post<ApiResponse<BatchCompleteUpload>, BatchCompleteUploadRequest>(ApiEndpoints.UPLOAD.BATCH_COMPLETE_UPLOAD, params);
  }

  static async uploadFile(
    context: common.UIAbilityContext,
    fileUri: string,
    category: MediaCategory
  ): Promise<UploadedMedia> {
    const readablePath = ApiClient.resolveReadableFilePath(context, fileUri);
    let file: fs.File | null = null;
    let size = 0;
    let buffer: ArrayBuffer | null = null;
    try {
      file = fs.openSync(readablePath, fs.OpenMode.READ_ONLY);
      const stat = fs.statSync(file.fd);
      size = stat.size;
      if (size <= 0) {
        throw new Error('文件内容为空');
      }
      buffer = new ArrayBuffer(size);
      fs.readSync(file.fd, buffer);
    } catch (error) {
      throw (error instanceof Error) ? error : new Error(String(error));
    } finally {
      if (file) {
        try {
          fs.closeSync(file);
        } catch (error) {
          hilog.warn(DOMAIN_NUMBER, TAG, `关闭上传文件失败: ${JSON.stringify(error)}`);
        }
      }
    }

    const fileName = ApiClient.extractFileName(fileUri, readablePath);
    const mimeType = ApiClient.inferMimeType(fileName, category);

    const presignReq: PresignUploadRequest = {
      file_name: fileName,
      file_size: size,
      mime_type: mimeType,
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

    if (!buffer) {
      throw new Error('Failed to read file buffer');
    }
    await HttpUtil.uploadToUrl(upload_url, buffer, headers);

    const completeReq: CompleteUploadRequest = { upload_id };
    const completeRes = await ApiClient.completeUpload(completeReq);
    if (completeRes.code !== 0 || !completeRes.data) {
      throw new Error(completeRes.message || 'Complete failed');
    }
    return completeRes.data;
  }

  // ==================== 系统 API ====================

  static async healthCheck(): Promise<ApiResponse<HealthStatus>> {
    return HttpUtil.post<ApiResponse<HealthStatus>, EmptyRequest>(ApiEndpoints.SYSTEM.HEALTH, new EmptyRequest());
  }

  static async getSystemMeta(): Promise<ApiResponse<ApiVersionMeta>> {
    return HttpUtil.post<ApiResponse<ApiVersionMeta>, EmptyRequest>(ApiEndpoints.SYSTEM.GET_META, new EmptyRequest());
  }

  static async getClientConfig(): Promise<ApiResponse<ClientConfig>> {
    return HttpUtil.post<ApiResponse<ClientConfig>, EmptyRequest>(ApiEndpoints.SYSTEM.GET_CLIENT_CONFIG, new EmptyRequest());
  }

  static async getMyAdminPermissions(): Promise<ApiResponse<AdminPermissions>> {
    return HttpUtil.post<ApiResponse<AdminPermissions>, EmptyRequest>(ApiEndpoints.ADMIN.RBAC.GET_ME, new EmptyRequest());
  }

  static async getSystemAnnouncements(params: AnnouncementListRequest): Promise<ApiResponse<SystemAnnouncementListData>> {
    return HttpUtil.post<ApiResponse<SystemAnnouncementListData>, AnnouncementListRequest>(ApiEndpoints.SYSTEM.LIST_ANNOUNCEMENTS, params);
  }

  // ==================== 认证 API ====================

  static async register(params: RegisterRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, RegisterRequest>(ApiEndpoints.AUTH.REGISTER, params);
  }

  static async login(params: LoginRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, LoginRequest>(ApiEndpoints.AUTH.LOGIN, params);
  }

  static async createTempUser(params: GuestRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, GuestRequest>(ApiEndpoints.AUTH.GUEST, params);
  }

  static async refreshToken(params: RefreshTokenRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, RefreshTokenRequest>(ApiEndpoints.AUTH.REFRESH_TOKEN, params);
  }

  static async checkAuthStatus(): Promise<ApiResponse<AuthStatus>> {
    return HttpUtil.post<ApiResponse<AuthStatus>, EmptyRequest>(ApiEndpoints.AUTH.CHECK_STATUS, new EmptyRequest());
  }

  static async logout(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, EmptyRequest>(ApiEndpoints.AUTH.LOGOUT, new EmptyRequest());
  }

  static async requestPasswordReset(params: RequestPasswordResetRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RequestPasswordResetRequest>(ApiEndpoints.AUTH.REQUEST_PASSWORD_RESET, params);
  }

  static async resetPassword(params: ResetPasswordRequest): Promise<ApiResponse<AuthData>> {
    return HttpUtil.post<ApiResponse<AuthData>, ResetPasswordRequest>(ApiEndpoints.AUTH.RESET_PASSWORD, params);
  }

  static async changePassword(params: ChangePasswordRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ChangePasswordRequest>(ApiEndpoints.AUTH.CHANGE_PASSWORD, params);
  }

  static async getSessions(params: SessionsRequest): Promise<ApiResponse<SessionListData>> {
    return HttpUtil.post<ApiResponse<SessionListData>, SessionsRequest>(ApiEndpoints.AUTH.LIST_SESSIONS, params);
  }

  static async revokeSession(params: RevokeSessionRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeSessionRequest>(ApiEndpoints.AUTH.REVOKE_SESSION, params);
  }

  static async revokeAllSessions(): Promise<ApiResponse<RevokeAllResult>> {
    return HttpUtil.post<ApiResponse<RevokeAllResult>, EmptyRequest>(ApiEndpoints.AUTH.REVOKE_ALL_SESSIONS, new EmptyRequest());
  }

  // ==================== 用户 API ====================

  static async searchUsers(params: SearchUsersRequest): Promise<ApiResponse<PaginatedData<UserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserItem>>, SearchUsersRequest>(ApiEndpoints.USER.SEARCH, params);
  }

  static async getUserProfile(params: UserOpenTheDoorRequest): Promise<ApiResponse<UserItem>> {
    return HttpUtil.post<ApiResponse<UserItem>, UserOpenTheDoorRequest>(ApiEndpoints.USER.GET_DETAIL, params);
  }

  static async getUserProfileBundle(params: UserOpenTheDoorRequest): Promise<ApiResponse<UserProfileBundle>> {
    return HttpUtil.post<ApiResponse<UserProfileBundle>, UserOpenTheDoorRequest>(ApiEndpoints.USER.GET_PROFILE, params);
  }

  static async getUserGroups(params: UserGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, UserGroupsRequest>(ApiEndpoints.USER.LIST_GROUPS, params);
  }

  static async getUserMoments(params: UserMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, UserMomentsRequest>(ApiEndpoints.USER.LIST_MOMENTS, params);
  }

  static async getUserLikedMoments(params: UserLikedMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, UserLikedMomentsRequest>(ApiEndpoints.USER.LIST_REACTED_MOMENTS, params);
  }

  // ==================== 当前用户 API（/me/）====================

  static async getMyProfile(): Promise<ApiResponse<MyProfile>> {
    return HttpUtil.post<ApiResponse<MyProfile>, EmptyRequest>(ApiEndpoints.USER.GET_ACCOUNT, new EmptyRequest());
  }

  static async updateProfile(params: UpdateProfileRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateProfileRequest>(ApiEndpoints.USER.UPDATE_ACCOUNT, params);
  }

  static async updateLocation(params: UpdateLocationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateLocationRequest>(ApiEndpoints.USER.UPDATE_LOCATION, params);
  }

  static async upgradeAccount(params: UpgradeAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpgradeAccountRequest>(ApiEndpoints.USER.UPGRADE, params);
  }

  static async getSettings(): Promise<ApiResponse<UserSettings>> {
    return HttpUtil.post<ApiResponse<UserSettings>, EmptyRequest>(ApiEndpoints.USER.GET_SETTINGS, new EmptyRequest());
  }

  static async updateSettings(params: UpdateSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateSettingsRequest>(ApiEndpoints.USER.UPDATE_SETTINGS, params);
  }

  static async deleteAccount(params: DeleteAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteAccountRequest>(ApiEndpoints.USER.DELETE_ACCOUNT, params);
  }

  static async followUser(params: FollowRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, FollowRequest>(ApiEndpoints.USER.FOLLOW, params);
  }

  static async unfollowUser(params: UnfollowRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, UnfollowRequest>(ApiEndpoints.USER.FOLLOW, params);
  }

  static async blockUser(params: BlockRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, BlockRequest>(ApiEndpoints.USER.BLOCK, params);
  }

  static async unblockUser(params: UnblockRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, UnblockRequest>(ApiEndpoints.USER.BLOCK, params);
  }

  static async getFriendsList(params: ListFriendsRequest): Promise<ApiResponse<PaginatedData<FriendItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FriendItem>>, ListFriendsRequest>(ApiEndpoints.USER.LIST_FRIENDS, params);
  }

  static async getFollowingList(params: ListFollowingRequest): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowingItem>>, ListFollowingRequest>(ApiEndpoints.USER.LIST_FOLLOWING, params);
  }

  static async getFollowersList(params: ListFollowersRequest): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<FollowerItem>>, ListFollowersRequest>(ApiEndpoints.USER.LIST_FOLLOWERS, params);
  }

  static async getBlockedList(params: MyBlockedRequest): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BlockedUserItem>>, MyBlockedRequest>(ApiEndpoints.USER.LIST_BLOCKED, params);
  }

  static async setFriendRemark(params: SetRemarkRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetRemarkRequest>(ApiEndpoints.USER.SET_REMARK, params);
  }

  // ==================== 生活瞬间 API ====================

  static async searchMoments(params: SearchMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, SearchMomentsRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  static async getMomentDetail(params: MomentOpenTheDoorRequest): Promise<ApiResponse<MomentItem>> {
    return HttpUtil.post<ApiResponse<MomentItem>, MomentOpenTheDoorRequest>(ApiEndpoints.MOMENT.GET_DETAIL, params);
  }

  static async createMoment(params: CreateMomentRequest): Promise<ApiResponse<MomentItem>> {
    return HttpUtil.post<ApiResponse<MomentItem>, CreateMomentRequest>(ApiEndpoints.MOMENT.CREATE, params);
  }

  static async updateMoment(params: EditMomentRequest): Promise<ApiResponse<MomentItem>> {
    return HttpUtil.post<ApiResponse<MomentItem>, EditMomentRequest>(ApiEndpoints.MOMENT.UPDATE, params);
  }

  static async deleteMoment(params: DeleteMomentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteMomentRequest>(ApiEndpoints.MOMENT.DELETE, params);
  }

  static async favoriteMoment(params: FavoriteMomentRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, FavoriteMomentRequest>(ApiEndpoints.MOMENT.FAVORITE, params);
  }

  static async unfavoriteMoment(params: UnfavoriteMomentRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, UnfavoriteMomentRequest>(ApiEndpoints.MOMENT.FAVORITE, params);
  }

  static async setMomentReaction(targetId: string, emoji: string, active: boolean): Promise<ApiResponse<void>> {
    const req: SetReactionRequest = { target_type: 'moment', target_id: targetId, emoji: emoji, active: active };
    return HttpUtil.post<ApiResponse<void>, SetReactionRequest>(ApiEndpoints.REACTIONS.SET, req);
  }

  /**
   * 「点赞」使用的默认 emoji。spec 要求 emoji 必须在 AppConfig.limits.reaction_emojis
   * 白名单内，否则后端返回 10000。优先取白名单首项；配置未加载时退化为 '👍'（规范默认白名单含此项）。
   */
  private static getDefaultLikeEmoji(): string {
    const emojis = GlobalStateManager.getInstance().getLimits()?.reaction_emojis;
    return (emojis && emojis.length > 0) ? emojis[0] : '👍';
  }

  static async likeMoment(params: LikeMomentRequest): Promise<ApiResponse<void>> {
    return ApiClient.setMomentReaction(params.moment_id, ApiClient.getDefaultLikeEmoji(), true);
  }

  static async unlikeMoment(params: UnlikeMomentRequest): Promise<ApiResponse<void>> {
    // 取消点赞必须用与点赞时相同的 emoji（reactions/set 按 emoji 维度去重）。
    return ApiClient.setMomentReaction(params.moment_id, ApiClient.getDefaultLikeEmoji(), false);
  }

  static async getMomentLikes(params: MomentLikesRequest): Promise<ApiResponse<PaginatedData<ReactionItem>>> {
    const req: ListReactionsRequest = { target_type: 'moment', target_id: params.moment_id, pagination: params.pagination };
    return HttpUtil.post<ApiResponse<PaginatedData<ReactionItem>>, ListReactionsRequest>(ApiEndpoints.REACTIONS.LIST, req);
  }

  static async getMyMoments(params: MyMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, MyMomentsRequest>(ApiEndpoints.MOMENT.LIST_MY, params);
  }

  static async getMyFavorites(params: MyFavoritesRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, MyFavoritesRequest>(ApiEndpoints.MOMENT.LIST_FAVORITES, params);
  }

  // ==================== 评论 API ====================

  static async getComments(params: CommentsRequest): Promise<ApiResponse<PaginatedData<CommentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentItem>>, CommentsRequest>(ApiEndpoints.COMMENT.LIST, params);
  }

  static async getReplies(params: CommentRepliesRequest): Promise<ApiResponse<PaginatedData<CommentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentItem>>, CommentRepliesRequest>(ApiEndpoints.COMMENT.LIST_REPLIES, params);
  }

  static async getCommentLikes(params: CommentLikesRequest): Promise<ApiResponse<PaginatedData<ReactionItem>>> {
    const req: ListReactionsRequest = { target_type: 'comment', target_id: params.comment_id, pagination: params.pagination };
    return HttpUtil.post<ApiResponse<PaginatedData<ReactionItem>>, ListReactionsRequest>(ApiEndpoints.REACTIONS.LIST, req);
  }

  static async createComment(params: CreateCommentRequest): Promise<ApiResponse<CommentItem>> {
    return HttpUtil.post<ApiResponse<CommentItem>, CreateCommentRequest>(ApiEndpoints.COMMENT.CREATE, params);
  }

  static async deleteComment(params: DeleteCommentRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteCommentRequest>(ApiEndpoints.COMMENT.DELETE, params);
  }

  static async setCommentReaction(commentId: string, emoji: string, active: boolean): Promise<ApiResponse<void>> {
    const req: SetReactionRequest = { target_type: 'comment', target_id: commentId, emoji: emoji, active: active };
    return HttpUtil.post<ApiResponse<void>, SetReactionRequest>(ApiEndpoints.REACTIONS.SET, req);
  }

  static async likeComment(params: LikeCommentRequest): Promise<ApiResponse<void>> {
    return ApiClient.setCommentReaction(params.comment_id, '👍', true);
  }

  static async unlikeComment(params: UnlikeCommentRequest): Promise<ApiResponse<void>> {
    return ApiClient.setCommentReaction(params.comment_id, '👍', false);
  }

  // ==================== 私聊管理 API ====================

  static async getPrivateChats(params: MyPrivateChatsRequest): Promise<ApiResponse<PaginatedData<PrivateChatItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateChatItem>>, MyPrivateChatsRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.LIST, params);
  }

  static async privateChatOpenTheDoor(params: PrivateChatOpenTheDoorRequest): Promise<ApiResponse<PrivateChatOpenResult>> {
    return HttpUtil.post<ApiResponse<PrivateChatOpenResult>, PrivateChatOpenTheDoorRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.GET_DETAIL, params);
  }

  static async updatePrivateChatSettings(params: UpdatePrivateChatSettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateChatSettingsRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.UPDATE_PREFERENCES, params);
  }

  static async deletePrivateChat(params: DeletePrivateChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateChatRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.DELETE, params);
  }

  // ==================== 私聊消息 API ====================

  static async sendPrivateMessage(params: SendPrivateMessageRequest): Promise<ApiResponse<MessageSent>> {
    return HttpUtil.post<ApiResponse<MessageSent>, SendPrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND, params);
  }

  static async getPrivateMessages(params: GetPrivateMessagesHistoryRequest): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateMessageItem>>, GetPrivateMessagesHistoryRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.GET_HISTORY, params);
  }

  static async updatePrivateMessage(params: UpdatePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.UPDATE, params);
  }

  static async recallPrivateMessage(params: RecallPrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallPrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.RECALL, params);
  }

  static async markPrivateMessagesRead(params: MarkPrivateMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkPrivateMessagesReadRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.MARK_READ, params);
  }

  static async markAllPrivateMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkAsReadRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.MARK_READ, { scope: 'all' });
  }

  // ==================== 群组管理 API ====================

  static async searchGroups(params: SearchGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, SearchGroupsRequest>(ApiEndpoints.GROUP.SEARCH, params);
  }

  static async getGroupDetail(params: GroupOpenTheDoorRequest): Promise<ApiResponse<GroupItem>> {
    return HttpUtil.post<ApiResponse<GroupItem>, GroupOpenTheDoorRequest>(ApiEndpoints.GROUP.GET_DETAIL, params);
  }

  static async createGroup(params: CreateGroupRequest): Promise<ApiResponse<GroupItem>> {
    return HttpUtil.post<ApiResponse<GroupItem>, CreateGroupRequest>(ApiEndpoints.GROUP.CREATE, params);
  }

  static async updateGroup(params: UpdateGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupRequest>(ApiEndpoints.GROUP.UPDATE, params);
  }

  static async dismissGroup(params: DismissGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DismissGroupRequest>(ApiEndpoints.GROUP.DISMISS, params);
  }

  static async updateGroupMySettings(params: UpdateGroupMySettingsRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupMySettingsRequest>(ApiEndpoints.GROUP.UPDATE_PREFERENCES, params);
  }

  static async getGroupMembers(params: GroupMembersRequest): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMemberItem>>, GroupMembersRequest>(ApiEndpoints.GROUP.MEMBERS.LIST, params);
  }

  static async inviteMember(params: InviteMemberRequest): Promise<ApiResponse<Invitation>> {
    return HttpUtil.post<ApiResponse<Invitation>, InviteMemberRequest>(ApiEndpoints.GROUP.MEMBERS.INVITE, params);
  }

  static async getSentInvites(params: GroupSentInvitesRequest): Promise<ApiResponse<PaginatedData<SentInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<SentInviteItem>>, GroupSentInvitesRequest>(ApiEndpoints.GROUP.MEMBERS.INVITES_LIST, params);
  }

  static async revokeSentInvite(params: RevokeSentInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeSentInviteRequest>(ApiEndpoints.GROUP.MEMBERS.INVITES_REVOKE, params);
  }

  static async joinGroup(params: JoinGroupRequest): Promise<ApiResponse<JoinGroupResult>> {
    return HttpUtil.post<ApiResponse<JoinGroupResult>, JoinGroupRequest>(ApiEndpoints.GROUP.JOIN, params);
  }

  static async setMemberRole(params: SetMemberRoleRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetMemberRoleRequest>(ApiEndpoints.GROUP.MEMBERS.SET_ROLE, params);
  }

  static async muteMember(params: MuteMemberRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, MuteMemberRequest>(ApiEndpoints.GROUP.MEMBERS.SET_MUTE, params);
  }

  static async unmuteMember(params: UnmuteMemberRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, UnmuteMemberRequest>(ApiEndpoints.GROUP.MEMBERS.SET_MUTE, params);
  }

  static async banMember(params: BanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, BanMemberRequest>(ApiEndpoints.GROUP.MEMBERS.BAN, params);
  }

  static async unbanMember(params: UnbanMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UnbanMemberRequest>(ApiEndpoints.GROUP.MEMBERS.UNBAN, params);
  }

  static async getBannedMembers(params: BannedMembersRequest): Promise<ApiResponse<PaginatedData<BannedMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BannedMemberItem>>, BannedMembersRequest>(ApiEndpoints.GROUP.MEMBERS.LIST_BANNED, params);
  }

  static async removeMember(params: RemoveMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveMemberRequest>(ApiEndpoints.GROUP.MEMBERS.REMOVE, params);
  }

  static async getGroupApplications(params: GroupApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, GroupApplicationsRequest>(ApiEndpoints.GROUP.APPLICATIONS.LIST, params);
  }

  static async approveGroupApplication(params: ApproveGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ApproveGroupApplicationRequest>(ApiEndpoints.GROUP.APPLICATIONS.APPROVE, params);
  }

  static async rejectGroupApplication(params: RejectGroupApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectGroupApplicationRequest>(ApiEndpoints.GROUP.APPLICATIONS.REJECT, params);
  }

  static async transferOwnership(params: TransferOwnershipRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, TransferOwnershipRequest>(ApiEndpoints.GROUP.TRANSFER_OWNERSHIP, params);
  }

  // ==================== 群公告 API ====================

  static async createGroupAnnouncement(params: CreateGroupAnnouncementRequest): Promise<ApiResponse<GroupAnnouncementItem>> {
    return HttpUtil.post<ApiResponse<GroupAnnouncementItem>, CreateGroupAnnouncementRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.CREATE, params);
  }

  static async getGroupAnnouncements(params: GroupAnnouncementsRequest): Promise<ApiResponse<GroupAnnouncementListData>> {
    return HttpUtil.post<ApiResponse<GroupAnnouncementListData>, GroupAnnouncementsRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.LIST, params);
  }

  static async updateGroupAnnouncement(params: UpdateGroupAnnouncementRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupAnnouncementRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.UPDATE, params);
  }

  static async deleteGroupAnnouncement(params: DeleteGroupAnnouncementRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupAnnouncementRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.DELETE, params);
  }

  // ==================== 当前用户群组 API（/me/）====================

  static async getJoinedGroups(params: MyGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, MyGroupsRequest>(ApiEndpoints.GROUP.LIST_MY, params);
  }

  static async getMyGroupApplications(params: MyApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, MyApplicationsRequest>(ApiEndpoints.GROUP.LIST_APPLICATIONS, params);
  }

  static async cancelGroupApplication(params: CancelApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, CancelApplicationRequest>(ApiEndpoints.GROUP.CANCEL_APPLICATION, params);
  }

  static async getGroupInvites(params: MyInvitesRequest): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupInviteItem>>, MyInvitesRequest>(ApiEndpoints.GROUP.LIST_INVITES, params);
  }

  static async acceptGroupInvite(params: AcceptInviteRequest): Promise<ApiResponse<InviteAcceptedResult>> {
    return HttpUtil.post<ApiResponse<InviteAcceptedResult>, AcceptInviteRequest>(ApiEndpoints.GROUP.ACCEPT_INVITE, params);
  }

  static async rejectGroupInvite(params: RejectInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectInviteRequest>(ApiEndpoints.GROUP.REJECT_INVITE, params);
  }

  static async leaveGroup(params: LeaveGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, LeaveGroupRequest>(ApiEndpoints.GROUP.LEAVE, params);
  }

  // ==================== 群聊消息 API ====================

  static async sendGroupMessage(params: SendGroupMessageRequest): Promise<ApiResponse<MessageSent>> {
    return HttpUtil.post<ApiResponse<MessageSent>, SendGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.SEND, params);
  }

  static async getGroupMessages(params: GetGroupMessagesHistoryRequest): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMessageItem>>, GetGroupMessagesHistoryRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.GET_HISTORY, params);
  }

  static async updateGroupMessage(params: UpdateGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.UPDATE, params);
  }

  static async recallGroupMessage(params: RecallGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.RECALL, params);
  }

  static async markGroupMessagesRead(params: MarkGroupMessagesReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkGroupMessagesReadRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.MARK_READ, params);
  }

  static async markAllGroupMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkAsReadRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.MARK_READ, { scope: 'all' });
  }

  // ==================== 举报 API ====================

  static async createReport(params: CreateReportRequest): Promise<ApiResponse<ReportCreated>> {
    return HttpUtil.post<ApiResponse<ReportCreated>, CreateReportRequest>(ApiEndpoints.REPORT.CREATE, params);
  }

  static async getMyReports(params: MyReportsRequest): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<ReportItem>>, MyReportsRequest>(ApiEndpoints.REPORT.LIST_MY, params);
  }

  // ==================== 信息流 API ====================

  static async getFeedMoments(params: GetFeedMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, GetFeedMomentsRequest>(ApiEndpoints.FEED.LIST_MOMENTS, params);
  }

  static async getFeedGroups(params: GetFeedGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, GetFeedGroupsRequest>(ApiEndpoints.FEED.LIST_GROUPS, params);
  }

  static async getFeedUsers(params: GetFeedUsersRequest): Promise<ApiResponse<PaginatedData<UserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<UserItem>>, GetFeedUsersRequest>(ApiEndpoints.FEED.LIST_USERS, params);
  }

  static async getFeedTrending(params: GetFeedTrendingRequest): Promise<ApiResponse<TrendingBundle>> {
    return HttpUtil.post<ApiResponse<TrendingBundle>, GetFeedTrendingRequest>(ApiEndpoints.FEED.LIST_TRENDING, params);
  }

  static async dismissFeedItem(params: DismissFeedItemRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DismissFeedItemRequest>(ApiEndpoints.FEED.DISMISS, params);
  }

  // ==================== 搜索 API ====================

  static async searchSuggest(params: SearchSuggestRequest): Promise<ApiResponse<SearchSuggestResponse>> {
    return HttpUtil.post<ApiResponse<SearchSuggestResponse>, SearchSuggestRequest>(ApiEndpoints.SEARCH.SUGGEST, params);
  }

  static async searchAll(params: SearchAllRequest): Promise<ApiResponse<SearchAllResult>> {
    return HttpUtil.post<ApiResponse<SearchAllResult>, SearchAllRequest>(ApiEndpoints.SEARCH.ALL, params);
  }

	// ==================== 地图 API ====================

	static async getMapCanvas(params: MapCanvasRequest): Promise<ApiResponse<MapCanvasResponse>> {
		return HttpUtil.post<ApiResponse<MapCanvasResponse>, MapCanvasRequest>(ApiEndpoints.MAP.CANVAS, params);
	}

  // ==================== 分享 API ====================

  static async createShare(params: CreateShareRequest): Promise<ApiResponse<ShareCreated>> {
    return HttpUtil.post<ApiResponse<ShareCreated>, CreateShareRequest>(ApiEndpoints.SHARE.CREATE, params);
  }

  static async getShare(params: GetShareRequest): Promise<ApiResponse<ShareDetail>> {
    return HttpUtil.post<ApiResponse<ShareDetail>, GetShareRequest>(ApiEndpoints.SHARE.GET, params);
  }

  static async revokeShare(params: RevokeShareRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeShareRequest>(ApiEndpoints.SHARE.REVOKE, params);
  }

  static async listMyShares(params?: ShareListRequest): Promise<ApiResponse<PaginatedData<ShareListItem>>> {
    const payload: ShareListRequest = params ?? {};
    return HttpUtil.post<ApiResponse<PaginatedData<ShareListItem>>, ShareListRequest>(ApiEndpoints.SHARE.LIST_MY, payload);
  }

  // ==================== 通知 API ====================

  static async listNotifications(params?: NotificationListRequest): Promise<ApiResponse<PaginatedData<NotificationItem>>> {
    const payload: NotificationListRequest = params ?? {};
    return HttpUtil.post<ApiResponse<PaginatedData<NotificationItem>>, NotificationListRequest>(ApiEndpoints.NOTIFY.LIST, payload);
  }

  static async getUnreadNotifications(): Promise<ApiResponse<UnreadCountResult>> {
    return HttpUtil.post<ApiResponse<UnreadCountResult>, EmptyRequest>(ApiEndpoints.NOTIFY.GET_UNREAD_COUNT, new EmptyRequest());
  }

  static async markNotificationsRead(params: MarkAsReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkAsReadRequest>(ApiEndpoints.NOTIFY.MARK_READ, params);
  }

  static async markAllNotificationsRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkAsReadRequest>(ApiEndpoints.NOTIFY.MARK_READ, { scope: 'all' });
  }
}
