// ==================== 举报API类型 ====================
import { GeoPoint, Location, MessageMedia, MessageType, PaginationParams, RelativeLocation, ReportAction, ReportReason, ReportStatus, ReportTargetType } from './common';
import { GroupItem, MomentSummary, UserInfo, UserItem } from './entities';


export interface CreateReportRequest {
  target_type: ReportTargetType;
  target_id: string;
  reason: ReportReason;
  description?: string;
}

export interface ReportCreated {
  report_id: string;
  is_duplicate: boolean;
}

export interface FetchReportsRequest {
  status?: ReportStatus;
  target_type?: ReportTargetType;
  pagination?: PaginationParams;
}

// 会话类型（spec types.md → ReportSnapshot.chat_type）
export type ReportChatType = 'private' | 'group';

// 消息举报快照（spec types.md：由服务端在举报受理时基于消息真值生成并冻结，客户端不提交）
export interface ReportSnapshot {
  chat_type: ReportChatType;           // 会话类型
  target_chat_id: string;              // 会话目标 ID：private 会话 = 对方 user_id，group 会话 = group_id
  message_type: MessageType;
  snapshot_content: string | null;     // 文本内容（text 类型）；非文本类型为 null
  snapshot_media: MessageMedia | null; // 媒体引用（图片/视频等）；文本类型为 null
  sent_at: string;
}

// 处理结果摘要（脱敏，不含处理者身份；spec types.md → ReportResolution）
export interface ReportResolution {
  action: ReportAction;
  resolved_at: string;
  note: string | null;   // 可选的处理说明（脱敏后）
}

// 举报项（report + admin 两模块共用：spec types.md → ReportItem）
export interface ReportItem {
  report_id: string;
  target_type: ReportTargetType;
  target_id: string;
  reason: ReportReason;
  description: string | null;
  status: ReportStatus;
  created_at: string;
  handled_at: string | null;
  // 消息举报快照（仅 target_type='message'）
  message_snapshot: ReportSnapshot | null;

  // 处理结果摘要（恒定 nullable：null = 尚未处理）
  resolution: ReportResolution | null;
}

// 标签项（搜索、发现等模块共用）
export interface TagItem {
  name: string;
  usage_count: number;
}

// ==================== 搜索API类型 ====================

export type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

export interface SearchSuggestRequest {
  prefix: string;
  reference_point?: GeoPoint;
  types?: SearchSuggestType[];
  radius?: number;
}

// 瞬间建议项（spec types.md → MomentSuggestItem = MomentSummary + 作者与位置）
export interface MomentSuggestItem {
  moment_id: string;
  description_preview: string;
  cover_url: string | null;     // 首张图片缩略图 URL；null = 无图片
  author: UserInfo;
  location: RelativeLocation | null;
}

// spec discovery.md → SearchSuggestItem：tag 分支复用 TagItem，moment 分支为 MomentSuggestItem
export interface SearchSuggestItemTag {
  type: 'tag';
  payload: TagItem;
}

// 登录视角建议项（spec discovery.md → SearchSuggestItem）：
// user/group 分支为完整 UserItem/GroupItem（含查看者状态），不复用精简投影
export interface SearchSuggestItemUser {
  type: 'user';
  payload: UserItem;
}

export interface SearchSuggestItemGroup {
  type: 'group';
  payload: GroupItem;
}

export interface SearchSuggestItemMoment {
  type: 'moment';
  payload: MomentSuggestItem;
}

export type SearchSuggestItem =
  | SearchSuggestItemTag
  | SearchSuggestItemUser
  | SearchSuggestItemGroup
  | SearchSuggestItemMoment;

export interface SearchSuggestResponse {
  items: SearchSuggestItem[];
}

// ==================== 系统配置API类型 ====================

export interface FeatureSwitches {
  enable_public_registration: boolean;
  enable_guest_register: boolean;
  enable_moments: boolean;
  enable_nearby_people: boolean;
}

export interface AppLimits {
  max_group_members: number;
  max_moment_tags: number;
  max_moment_images: number;
  moment_description_max_length: number;
  comment_max_length: number;
  search_radius_max: number;
  max_chat_files: number; // 单条聊天消息可附带的最大文件数（spec types.md → AppLimits）
  message_recall_window_seconds: number;
  stranger_message_daily_quota: number;
  reaction_emojis: string[];
  group_invite_ttl_hours: number;
  group_application_ttl_hours: number;
}

export interface ContactInfo {
  support_email: string;
  privacy_policy_url: string;
  terms_of_service_url: string;
  official_website_url: string;
}

export interface ClientDefaults {
  default_location: Location;
  default_user_avatar_url: string;
  default_group_avatar_url: string;
}

export interface ClientConfig {
  features: FeatureSwitches;
  limits: AppLimits;
  contact: ContactInfo;
  defaults: ClientDefaults;
}

export type SystemHealthStatus = 'healthy' | 'degraded' | 'unhealthy';

export interface HealthStatus {
  status: SystemHealthStatus;
  timestamp: string;
}

export type ApiVersionStatus = 'active' | 'deprecated' | 'sunset';

export interface ApiVersionMeta {
  version: string;
  status: ApiVersionStatus;
  deprecated_at: string | null;
  sunset_at: string | null;
  message: string | null;
  recommended_version: string | null;
}

export interface AnnouncementListRequest {}

// ==================== 文件与媒体上传API类型 ====================

export type MediaCategory = 'avatar' | 'group_avatar' | 'moment_image' | 'chat_image' | 'chat_video' | 'chat_audio' | 'chat_file';

export interface PresignedUpload {
  upload_id: string;
  upload_url: string;
  headers: Record<string, string>;
  expires_at: string;
}

export interface PresignFileInput {
  file_name: string;
  file_size: number;
  mime_type: string;
  category: MediaCategory;
  md5?: string;
}

export interface PresignUploadRequest {
  files: PresignFileInput[];
}

export interface PresignedUploads {
  uploads: PresignedUpload[];
}

export interface CompleteUploadRequest {
  upload_ids: string[];
}

export interface CompletedUploads {
  uploads: UploadedMedia[];
}

export interface UploadedMedia {
  file_id: string;
  thumbnail_url?: string;
  file_size: number;
  mime_type: string;
  duration?: number;
  width?: number;
  height?: number;
}

// ==================== 分享API类型 ====================

export type ShareType = 'user' | 'group' | 'moment' | 'location';

export interface UserSharePayload {
  shared_user_id: string;
}

export interface GroupSharePayload {
  shared_group_id: string;
}

export interface MomentSharePayload {
  shared_moment_id: string;
}

export interface LocationSharePayload {
  shared_location: Location;
}

export type SharePayload =
  | UserSharePayload
  | GroupSharePayload
  | MomentSharePayload
  | LocationSharePayload;

export interface ShareBase {
  share_id: string;
  share_code: string;
  share_type: ShareType;
  creator: UserInfo;
  created_at: string;
  expires_at: string | null;
}

export interface ShareCreated {
  share_id: string;
  share_code: string;
  expires_at: string | null;
}

export interface SharePreview {
  title: string;
  subtitle: string | null;
  cover_url: string | null;
}

export interface ShareDetail {
  share: ShareBase;
  payload: SharePayload;
  preview?: SharePreview;
}

export interface ShareListItem {
  share_id: string;
  share_code: string;
  share_type: ShareType;
  created_at: string;
  expires_at: string | null;
  is_revoked: boolean;
  preview: SharePreview | null;
}

export interface CreateShareRequest {
  share_type: ShareType;
  payload: SharePayload;
  expires_in_seconds?: number;
}

export interface GetShareRequest {
  share_code: string;
}

export interface FetchCreatedSharesRequest {
  share_type?: ShareType;
  pagination?: PaginationParams;
}

export interface RevokeShareRequest {
  share_id?: string;
  share_code?: string;
}
