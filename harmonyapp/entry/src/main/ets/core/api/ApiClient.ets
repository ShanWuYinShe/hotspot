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
  PresignedUploads,
  CompleteUploadRequest,
  CompletedUploads,
  UploadedMedia,
  MediaCategory,

  // 认证相关
  AuthData,
  RegisterRequest,
  SendRegisterCodeRequest,
  LoginRequest,
  GuestRequest,
  RefreshTokenRequest,
  SessionStatus,
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
  OpenUserDoorRequest,
  AccountInfo,
  UpdateAccountRequest,
  UpdateLocationRequest,
  UpgradeAccountRequest,
  UserSettings,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  FollowRequest,
  BlockRequest,
  RelatedUserItem,
  BlockedUserItem,
  SetRemarkRequest,
  FetchFriendsRequest,
  FetchFollowingRequest,
  FetchFollowersRequest,
  FetchBlockedRequest,
  FetchUserGroupsRequest,
  FetchUserMomentsRequest,
  FetchUserReactedMomentsRequest,
  GroupItem,
  UserProfileBundle,

  // 生活瞬间
  MomentItem,
  SearchMomentsRequest,
  OpenMomentDoorRequest,
  CreateMomentRequest,
  EditMomentRequest,
  DeleteMomentRequest,
  FavoriteMomentRequest,
  ReactionItem,
  FetchCreatedMomentsRequest,
  FetchFollowingMomentsRequest,
  FetchNearbyMomentsRequest,
  FetchFavoritesRequest,

  // 评论
  CreateCommentRequest,
  DeleteCommentRequest,
  FetchCommentsRequest,
  CommentItem,
  FetchCommentRepliesRequest,

  // Reactions
  SetReactionRequest,
  FetchReactionsRequest,

  // 私聊管理
  FetchPrivateChatsRequest,
  PrivateChatItem,
  OpenPrivateChatDoorRequest,
  PrivateChatOpenResult,
  UpdatePrivateChatPreferencesRequest,
  DeletePrivateChatRequest,

  // 私聊消息
  SendPrivateMessageRequest,
  MessageSentResult,
  PrivateMessageItem,
  FetchPrivateMessagesRequest,
  UpdatePrivateMessageRequest,
  RecallPrivateMessageRequest,
  MarkPrivateReadRequest,

  // 群组管理
  SearchGroupsRequest,
  OpenGroupDoorRequest,
  CreateGroupRequest,
  UpdateGroupRequest,
  DismissGroupRequest,
  UpdateGroupPreferencesRequest,
  FetchGroupMembersRequest,
  GroupMemberItem,
  CreateGroupInviteRequest,
  Invitation,
  EnterGroupRequest,
  JoinGroupResult,
  SetMemberRoleRequest,
  SetMemberMuteRequest,
  SetMemberBanRequest,
  FetchBannedMembersRequest,
  AdminBannedMemberItem,
  RemoveMemberRequest,
  FetchGroupApplicationsRequest,
  GroupApplicationItem,
  ApproveGroupApplicationRequest,
  RejectGroupApplicationRequest,
  TransferOwnershipRequest,
  FetchJoinedGroupsRequest,
  FetchSubmittedApplicationsRequest,
  CancelApplicationRequest,
  ListSentInvitesRequest,
  GroupSentInviteItem,
  RevokeSentInviteRequest,
  FetchReceivedInvitesRequest,
  GroupInviteItem,
  AcceptInviteRequest,
  InviteAcceptedResult,
  RejectInviteRequest,
  ExitGroupRequest,

  // 群公告
  CreateGroupAnnouncementRequest,
  FetchGroupAnnouncementsRequest,
  GroupAnnouncementItem,
  UpdateGroupAnnouncementRequest,
  DeleteGroupAnnouncementRequest,

  // 群聊消息
  SendGroupMessageRequest,
  GroupMessageItem,
  FetchGroupMessagesRequest,
  UpdateGroupMessageRequest,
  RecallGroupMessageRequest,
  RecallAdminGroupMessageRequest,
  MarkGroupReadRequest,

  // 举报
  CreateReportRequest,
  ReportCreated,
  ReportItem,
  FetchReportsRequest,

  TagItem,

  // 搜索
	SearchSuggestRequest,
	SearchSuggestResponse,

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
  FetchCreatedSharesRequest,
  ShareListItem,

  // Response 包装类型
  SystemAnnouncementListData,
  SessionListData,
  GroupAnnouncementListData,

  // 通知中心
  NotifyItem,
  FetchNotifyRequest,
  UnreadSummaryResult,
  NotifyMarkReadRequest,
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

  static async presignUpload(params: PresignUploadRequest): Promise<ApiResponse<PresignedUploads>> {
    return HttpUtil.post<ApiResponse<PresignedUploads>, PresignUploadRequest>(ApiEndpoints.UPLOAD.PRESIGN_UPLOAD, params);
  }

  static async completeUpload(params: CompleteUploadRequest): Promise<ApiResponse<CompletedUploads>> {
    return HttpUtil.post<ApiResponse<CompletedUploads>, CompleteUploadRequest>(ApiEndpoints.UPLOAD.COMPLETE_UPLOAD, params);
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

    // spec 契约为批量（files 1..N），单文件上传包一层单元素数组
    const presignReq: PresignUploadRequest = {
      files: [
        {
          file_name: fileName,
          file_size: size,
          mime_type: mimeType,
          category: category
        }
      ]
    };

    const presignRes = await ApiClient.presignUpload(presignReq);
    const presignData = presignRes.data?.uploads[0];
    if (presignRes.code !== 0 || !presignData) {
      throw new Error(presignRes.message || 'Presign failed');
    }
    const upload_url = presignData.upload_url;
    const upload_id = presignData.upload_id;
    const presignHeaders = presignData.headers;

    const headers: Record<string, string> = {};
    if (presignHeaders) {
      const headerKeys = Object.keys(presignHeaders);
      for (let i = 0; i < headerKeys.length; i += 1) {
        const key = headerKeys[i];
        const value: string | undefined = presignHeaders[key];
        if (typeof value === 'string') {
          headers[key] = value;
        }
      }
    }

    if (!buffer) {
      throw new Error('Failed to read file buffer');
    }
    await HttpUtil.uploadToUrl(upload_url, buffer, headers);

    const completeReq: CompleteUploadRequest = { upload_ids: [upload_id] };
    const completeRes = await ApiClient.completeUpload(completeReq);
    const completed = completeRes.data?.uploads[0];
    if (completeRes.code !== 0 || !completed) {
      throw new Error(completeRes.message || 'Complete failed');
    }
    return completed;
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

  // 注册验证码（手机号/邮箱所有权验证，spec public.md → auth/send_register_code）
  static async sendRegisterCode(params: SendRegisterCodeRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SendRegisterCodeRequest>(ApiEndpoints.AUTH.SEND_REGISTER_CODE, params);
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

  static async checkAuthStatus(): Promise<ApiResponse<SessionStatus>> {
    return HttpUtil.post<ApiResponse<SessionStatus>, EmptyRequest>(ApiEndpoints.AUTH.GET_STATUS, new EmptyRequest());
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

  // spec user.md → open_the_door：响应为 UserProfileBundle（include_recent_moments 可选段）
  static async getUserProfile(params: OpenUserDoorRequest): Promise<ApiResponse<UserProfileBundle>> {
    return HttpUtil.post<ApiResponse<UserProfileBundle>, OpenUserDoorRequest>(ApiEndpoints.USER.GET_DETAIL, params);
  }

  static async getUserGroups(params: FetchUserGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, FetchUserGroupsRequest>(ApiEndpoints.USER.LIST_GROUPS, params);
  }

  static async getUserMoments(params: FetchUserMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, FetchUserMomentsRequest>(ApiEndpoints.USER.LIST_MOMENTS, params);
  }

  static async getUserReactedMoments(params: FetchUserReactedMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, FetchUserReactedMomentsRequest>(ApiEndpoints.USER.LIST_REACTED_MOMENTS, params);
  }

  // ==================== 当前用户 API（/me/）====================

  static async getAccountInfo(): Promise<ApiResponse<AccountInfo>> {
    return HttpUtil.post<ApiResponse<AccountInfo>, EmptyRequest>(ApiEndpoints.USER.GET_ACCOUNT, new EmptyRequest());
  }

  static async updateProfile(params: UpdateAccountRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateAccountRequest>(ApiEndpoints.USER.UPDATE_ACCOUNT, params);
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

  static async unfollowUser(params: FollowRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, FollowRequest>(ApiEndpoints.USER.FOLLOW, params);
  }

  static async blockUser(params: BlockRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, BlockRequest>(ApiEndpoints.USER.BLOCK, params);
  }

  static async unblockUser(params: BlockRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, BlockRequest>(ApiEndpoints.USER.BLOCK, params);
  }

  static async getFriendsList(params: FetchFriendsRequest): Promise<ApiResponse<PaginatedData<RelatedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<RelatedUserItem>>, FetchFriendsRequest>(ApiEndpoints.USER.LIST_FRIENDS, params);
  }

  static async getFollowingList(params: FetchFollowingRequest): Promise<ApiResponse<PaginatedData<RelatedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<RelatedUserItem>>, FetchFollowingRequest>(ApiEndpoints.USER.LIST_FOLLOWING, params);
  }

  static async getFollowersList(params: FetchFollowersRequest): Promise<ApiResponse<PaginatedData<RelatedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<RelatedUserItem>>, FetchFollowersRequest>(ApiEndpoints.USER.LIST_FOLLOWERS, params);
  }

  static async getBlockedList(params: FetchBlockedRequest): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<BlockedUserItem>>, FetchBlockedRequest>(ApiEndpoints.USER.LIST_BLOCKED, params);
  }

  static async setFriendRemark(params: SetRemarkRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetRemarkRequest>(ApiEndpoints.USER.SET_REMARK, params);
  }

  // ==================== 生活瞬间 API ====================

  static async searchMoments(params: SearchMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, SearchMomentsRequest>(ApiEndpoints.MOMENT.SEARCH, params);
  }

  static async getMomentDetail(params: OpenMomentDoorRequest): Promise<ApiResponse<MomentItem>> {
    return HttpUtil.post<ApiResponse<MomentItem>, OpenMomentDoorRequest>(ApiEndpoints.MOMENT.GET_DETAIL, params);
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

  static async unfavoriteMoment(params: FavoriteMomentRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, FavoriteMomentRequest>(ApiEndpoints.MOMENT.FAVORITE, params);
  }

  static async setMomentReaction(targetId: string, emoji: string, active: boolean): Promise<ApiResponse<void>> {
    const req: SetReactionRequest = { target_type: 'moment', target_id: targetId, emoji: emoji, active: active };
    return HttpUtil.post<ApiResponse<void>, SetReactionRequest>(ApiEndpoints.REACTIONS.SET, req);
  }

  /**
   * 「点赞」使用的默认 emoji。spec 要求 emoji 必须在 AppConfig.limits.reaction_emojis
   * 白名单内，否则后端返回 10000。优先取白名单首项；配置未加载时退化为 '👍'（规范默认白名单含此项）。
   */
  static getDefaultLikeEmoji(): string {
    const emojis = GlobalStateManager.getInstance().getLimits()?.reaction_emojis;
    return (emojis && emojis.length > 0) ? emojis[0] : '👍';
  }

  static async getMyMoments(params: FetchCreatedMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, FetchCreatedMomentsRequest>(ApiEndpoints.MOMENT.LIST_CREATED, params);
  }

  // 关注动态（原 feed/following 迁移，spec moment.md → moments/fetch_following）
  static async fetchFollowingMoments(params: FetchFollowingMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, FetchFollowingMomentsRequest>(ApiEndpoints.MOMENT.FETCH_FOLLOWING, params);
  }

  // 附近瞬间发现流（原 feed/nearby_moments 迁移，spec moment.md → moments/fetch_nearby）
  static async fetchNearbyMoments(params: FetchNearbyMomentsRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, FetchNearbyMomentsRequest>(ApiEndpoints.MOMENT.FETCH_NEARBY, params);
  }

  static async getMyFavorites(params: FetchFavoritesRequest): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<MomentItem>>, FetchFavoritesRequest>(ApiEndpoints.MOMENT.LIST_FAVORITES, params);
  }

  // ==================== 评论 API ====================

  static async getComments(params: FetchCommentsRequest): Promise<ApiResponse<PaginatedData<CommentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentItem>>, FetchCommentsRequest>(ApiEndpoints.COMMENT.LIST, params);
  }

  static async getReplies(params: FetchCommentRepliesRequest): Promise<ApiResponse<PaginatedData<CommentItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<CommentItem>>, FetchCommentRepliesRequest>(ApiEndpoints.COMMENT.LIST_REPLIES, params);
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

  // ==================== 私聊管理 API ====================

  static async getPrivateChats(params: FetchPrivateChatsRequest): Promise<ApiResponse<PaginatedData<PrivateChatItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateChatItem>>, FetchPrivateChatsRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.LIST, params);
  }

  static async privateChatOpenTheDoor(params: OpenPrivateChatDoorRequest): Promise<ApiResponse<PrivateChatOpenResult>> {
    return HttpUtil.post<ApiResponse<PrivateChatOpenResult>, OpenPrivateChatDoorRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.GET_DETAIL, params);
  }

  static async updatePrivateChatSettings(params: UpdatePrivateChatPreferencesRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateChatPreferencesRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.UPDATE_PREFERENCES, params);
  }

  static async deletePrivateChat(params: DeletePrivateChatRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeletePrivateChatRequest>(ApiEndpoints.CHAT.PRIVATE_CHATS.DELETE, params);
  }

  // ==================== 私聊消息 API ====================

  static async sendPrivateMessage(params: SendPrivateMessageRequest): Promise<ApiResponse<MessageSentResult>> {
    return HttpUtil.post<ApiResponse<MessageSentResult>, SendPrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.SEND, params);
  }

  static async getPrivateMessages(params: FetchPrivateMessagesRequest): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<PrivateMessageItem>>, FetchPrivateMessagesRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.LIST, params);
  }

  static async updatePrivateMessage(params: UpdatePrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdatePrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.UPDATE, params);
  }

  static async recallPrivateMessage(params: RecallPrivateMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallPrivateMessageRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.RECALL, params);
  }

  static async markPrivateMessagesRead(params: MarkPrivateReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkPrivateReadRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.MARK_READ, params);
  }

  static async markAllPrivateMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkPrivateReadRequest>(ApiEndpoints.CHAT.PRIVATE_MESSAGES.MARK_READ, { scope: 'all' });
  }

  // ==================== 群组管理 API ====================

  static async searchGroups(params: SearchGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, SearchGroupsRequest>(ApiEndpoints.GROUP.SEARCH, params);
  }

  static async getGroupDetail(params: OpenGroupDoorRequest): Promise<ApiResponse<GroupItem>> {
    return HttpUtil.post<ApiResponse<GroupItem>, OpenGroupDoorRequest>(ApiEndpoints.GROUP.GET_DETAIL, params);
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

  static async updateGroupMySettings(params: UpdateGroupPreferencesRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupPreferencesRequest>(ApiEndpoints.GROUP.UPDATE_PREFERENCES, params);
  }

  static async getGroupMembers(params: FetchGroupMembersRequest): Promise<ApiResponse<PaginatedData<GroupMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMemberItem>>, FetchGroupMembersRequest>(ApiEndpoints.GROUP.MEMBERS.LIST, params);
  }

  static async inviteMember(params: CreateGroupInviteRequest): Promise<ApiResponse<Invitation>> {
    return HttpUtil.post<ApiResponse<Invitation>, CreateGroupInviteRequest>(ApiEndpoints.GROUP.MEMBERS.INVITE, params);
  }

  static async getSentInvites(params: ListSentInvitesRequest): Promise<ApiResponse<PaginatedData<GroupSentInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupSentInviteItem>>, ListSentInvitesRequest>(ApiEndpoints.GROUP.MEMBERS.INVITES_LIST, params);
  }

  static async revokeSentInvite(params: RevokeSentInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RevokeSentInviteRequest>(ApiEndpoints.GROUP.MEMBERS.INVITES_REVOKE, params);
  }

  static async joinGroup(params: EnterGroupRequest): Promise<ApiResponse<JoinGroupResult>> {
    return HttpUtil.post<ApiResponse<JoinGroupResult>, EnterGroupRequest>(ApiEndpoints.GROUP.JOIN, params);
  }

  static async setMemberRole(params: SetMemberRoleRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, SetMemberRoleRequest>(ApiEndpoints.GROUP.MEMBERS.SET_ROLE, params);
  }

  static async muteMember(params: SetMemberMuteRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, SetMemberMuteRequest>(ApiEndpoints.GROUP.MEMBERS.SET_MUTE, params);
  }

  static async unmuteMember(params: SetMemberMuteRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, SetMemberMuteRequest>(ApiEndpoints.GROUP.MEMBERS.SET_MUTE, params);
  }

  static async banMember(params: SetMemberBanRequest): Promise<ApiResponse<void>> {
    params.active = true;
    return HttpUtil.post<ApiResponse<void>, SetMemberBanRequest>(ApiEndpoints.GROUP.MEMBERS.BAN, params);
  }

  static async unbanMember(params: SetMemberBanRequest): Promise<ApiResponse<void>> {
    params.active = false;
    return HttpUtil.post<ApiResponse<void>, SetMemberBanRequest>(ApiEndpoints.GROUP.MEMBERS.UNBAN, params);
  }

  static async getBannedMembers(params: FetchBannedMembersRequest): Promise<ApiResponse<PaginatedData<AdminBannedMemberItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<AdminBannedMemberItem>>, FetchBannedMembersRequest>(ApiEndpoints.GROUP.MEMBERS.LIST_BANNED, params);
  }

  static async removeMember(params: RemoveMemberRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RemoveMemberRequest>(ApiEndpoints.GROUP.MEMBERS.REMOVE, params);
  }

  static async getGroupApplications(params: FetchGroupApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, FetchGroupApplicationsRequest>(ApiEndpoints.GROUP.APPLICATIONS.LIST, params);
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

  static async getGroupAnnouncements(params: FetchGroupAnnouncementsRequest): Promise<ApiResponse<GroupAnnouncementListData>> {
    return HttpUtil.post<ApiResponse<GroupAnnouncementListData>, FetchGroupAnnouncementsRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.LIST, params);
  }

  static async updateGroupAnnouncement(params: UpdateGroupAnnouncementRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupAnnouncementRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.UPDATE, params);
  }

  static async deleteGroupAnnouncement(params: DeleteGroupAnnouncementRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, DeleteGroupAnnouncementRequest>(ApiEndpoints.GROUP.ANNOUNCEMENTS.DELETE, params);
  }

  // ==================== 当前用户群组 API（/me/）====================

  static async getJoinedGroups(params: FetchJoinedGroupsRequest): Promise<ApiResponse<PaginatedData<GroupItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupItem>>, FetchJoinedGroupsRequest>(ApiEndpoints.GROUP.LIST_JOINED, params);
  }

  static async getMyGroupApplications(params: FetchSubmittedApplicationsRequest): Promise<ApiResponse<PaginatedData<GroupApplicationItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupApplicationItem>>, FetchSubmittedApplicationsRequest>(ApiEndpoints.GROUP.LIST_SUBMITTED_APPLICATIONS, params);
  }

  static async cancelGroupApplication(params: CancelApplicationRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, CancelApplicationRequest>(ApiEndpoints.GROUP.CANCEL_APPLICATION, params);
  }

  static async getGroupInvites(params: FetchReceivedInvitesRequest): Promise<ApiResponse<PaginatedData<GroupInviteItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupInviteItem>>, FetchReceivedInvitesRequest>(ApiEndpoints.GROUP.LIST_INVITES, params);
  }

  static async acceptGroupInvite(params: AcceptInviteRequest): Promise<ApiResponse<InviteAcceptedResult>> {
    return HttpUtil.post<ApiResponse<InviteAcceptedResult>, AcceptInviteRequest>(ApiEndpoints.GROUP.ACCEPT_INVITE, params);
  }

  static async rejectGroupInvite(params: RejectInviteRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RejectInviteRequest>(ApiEndpoints.GROUP.REJECT_INVITE, params);
  }

  static async leaveGroup(params: ExitGroupRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, ExitGroupRequest>(ApiEndpoints.GROUP.LEAVE, params);
  }

  // ==================== 群聊消息 API ====================

  static async sendGroupMessage(params: SendGroupMessageRequest): Promise<ApiResponse<MessageSentResult>> {
    return HttpUtil.post<ApiResponse<MessageSentResult>, SendGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.SEND, params);
  }

  static async getGroupMessages(params: FetchGroupMessagesRequest): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<GroupMessageItem>>, FetchGroupMessagesRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.LIST, params);
  }

  static async updateGroupMessage(params: UpdateGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, UpdateGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.UPDATE, params);
  }

  static async recallGroupMessage(params: RecallGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallGroupMessageRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.RECALL, params);
  }

  // 群主/管理员撤回他人消息（spec groups/admin/messages/recall，无时间窗，对齐前端 recallGroupMessageAsAdmin）
  static async recallGroupMessageAsAdmin(params: RecallAdminGroupMessageRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, RecallAdminGroupMessageRequest>(ApiEndpoints.GROUP.ADMIN_RECALL_MESSAGE, params);
  }

  static async markGroupMessagesRead(params: MarkGroupReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkGroupReadRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.MARK_READ, params);
  }

  static async markAllGroupMessagesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, MarkGroupReadRequest>(ApiEndpoints.CHAT.GROUP_MESSAGES.MARK_READ, { scope: 'all' });
  }

  // ==================== 举报 API ====================

  static async createReport(params: CreateReportRequest): Promise<ApiResponse<ReportCreated>> {
    return HttpUtil.post<ApiResponse<ReportCreated>, CreateReportRequest>(ApiEndpoints.REPORT.CREATE, params);
  }

  static async getMyReports(params: FetchReportsRequest): Promise<ApiResponse<PaginatedData<ReportItem>>> {
    return HttpUtil.post<ApiResponse<PaginatedData<ReportItem>>, FetchReportsRequest>(ApiEndpoints.REPORT.LIST, params);
  }

  // ==================== 搜索 API ====================

  static async searchSuggest(params: SearchSuggestRequest): Promise<ApiResponse<SearchSuggestResponse>> {
    return HttpUtil.post<ApiResponse<SearchSuggestResponse>, SearchSuggestRequest>(ApiEndpoints.SEARCH.SUGGEST, params);
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

  static async listMyShares(params?: FetchCreatedSharesRequest): Promise<ApiResponse<PaginatedData<ShareListItem>>> {
    const payload: FetchCreatedSharesRequest = params ?? {};
    return HttpUtil.post<ApiResponse<PaginatedData<ShareListItem>>, FetchCreatedSharesRequest>(ApiEndpoints.SHARE.LIST, payload);
  }

  // ==================== Notify API ====================

  static async listNotifies(params?: FetchNotifyRequest): Promise<ApiResponse<PaginatedData<NotifyItem>>> {
    const payload: FetchNotifyRequest = params ?? {};
    return HttpUtil.post<ApiResponse<PaginatedData<NotifyItem>>, FetchNotifyRequest>(ApiEndpoints.NOTIFY.LIST, payload);
  }

  static async getUnreadCount(): Promise<ApiResponse<UnreadSummaryResult>> {
    return HttpUtil.post<ApiResponse<UnreadSummaryResult>, EmptyRequest>(ApiEndpoints.NOTIFY.GET_UNREAD_COUNT, new EmptyRequest());
  }

  static async markNotifiesRead(params: NotifyMarkReadRequest): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, NotifyMarkReadRequest>(ApiEndpoints.NOTIFY.MARK_READ, params);
  }

  static async markAllNotifiesRead(): Promise<ApiResponse<void>> {
    return HttpUtil.post<ApiResponse<void>, NotifyMarkReadRequest>(ApiEndpoints.NOTIFY.MARK_READ, { scope: 'all' });
  }
}
