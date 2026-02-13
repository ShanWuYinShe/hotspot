//! API 请求类型定义
//!
//! 所有请求类型与 API 规范 (spec/api_spec_*.mdc) 一一对应
//! 枚举类型字段使用 crate::types::enums 中定义的类型，确保类型安全

use crate::types::enums::{
    AnnouncementSeverity, ApplicationStatusFilter, ClientType, FeedSource, GroupJoinMethod,
    GroupRole, MapDataType, MediaCategory, NotificationType, PrivacyLevel, PrivateChatPermission,
    ReportHandleAction, ReportReason, ReportStatus, ReportTargetType, SendMessageType,
    SetableGroupRole, ShareType, UserAccountStatus, UserType,
};
use serde::{Deserialize, Serialize};

// ==================== 认证模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct RegisterRequest {
    pub login_id: String,
    pub nickname: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub login_id: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct LogoutRequest {}

#[derive(Debug, Deserialize)]
pub struct GuestRequest {
    pub nickname: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    /// app：在 body 传递；web：必须不传（从 HttpOnly cookie 读取）
    pub refresh_token: Option<String>,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct CheckStatusRequest {}

#[derive(Debug, Deserialize)]
pub struct ChangePasswordRequest {
    pub old_password: String,
    pub new_password: String,
}

#[derive(Debug, Deserialize)]
pub struct RequestPasswordResetRequest {
    /// 注册时使用的邮箱或手机号
    pub login_id: String,
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct ResetPasswordRequest {
    pub login_id: String,
    pub verification_code: String,
    pub new_password: String,
    pub client_type: ClientType,
}

#[derive(Debug, Deserialize)]
pub struct SessionsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RevokeSessionRequest {
    /// 一次性撤销令牌（来自 SessionItem.revoke_token）
    pub revoke_token: String,
}

// ==================== 用户模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct MyProfileRequest {}

#[derive(Debug, Deserialize)]
pub struct GetRelationshipRequest {
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct UserFriendsRequest {
    pub user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserFollowingRequest {
    pub user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserFollowersRequest {
    pub user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserGroupsRequest {
    pub user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserLikedMomentsRequest {
    pub user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserOpenTheDoorRequest {
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct UpdateLocationRequest {
    pub location: Location,
}

#[derive(Debug, Deserialize)]
pub struct UpdateProfileRequest {
    pub nickname: Option<String>,
    pub avatar_file_id: Option<String>,
    pub bio: Option<String>,
    pub tags: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct UpgradeAccountRequest {
    pub login_id: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct MySettingsRequest {}

#[derive(Debug, Deserialize)]
pub struct UpdateSettingsRequest {
    pub privacy_level: Option<PrivacyLevel>,
    pub location_visible: Option<bool>,
    pub online_visible: Option<bool>,
    pub last_online_visible: Option<bool>,
    pub friends_visible: Option<bool>,
    pub following_visible: Option<bool>,
    pub followers_visible: Option<bool>,
    pub groups_visible: Option<bool>,
    pub moments_visible: Option<bool>,
    pub liked_moments_visible: Option<bool>,
    pub stats_visible: Option<bool>,
    pub allow_follow: Option<bool>,
    pub allow_private_chat: Option<PrivateChatPermission>,
    pub searchable: Option<bool>,
    pub phone_searchable: Option<bool>,
    pub nickname_searchable: Option<bool>,
    pub location_searchable: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteAccountRequest {
    pub confirmation: String,
}

#[derive(Debug, Deserialize)]
pub struct FollowRequest {
    pub user_id: i64,
    pub source: Option<crate::types::enums::FollowSource>, // 可选：关注来源
    pub source_moment_id: Option<String>,                  // 可选：如果是通过瞬间关注，提供瞬间 ID
}

#[derive(Debug, Deserialize)]
pub struct UnfollowRequest {
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BlockRequest {
    pub user_id: i64,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnblockRequest {
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct SetRemarkRequest {
    pub user_id: i64,
    pub remark: Option<String>,
}

// ==================== 生活瞬间 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateMomentRequest {
    pub description: String,
    pub location: Location,
    pub tags: Option<Vec<String>>,
    pub image_file_ids: Option<Vec<String>>,
    pub privacy_level: Option<PrivacyLevel>,
    pub mentions: Option<Vec<i64>>,
}

#[derive(Debug, Clone, Deserialize, serde::Serialize)]
pub struct Location {
    pub point: GeoPoint,
    pub location_name: String,
}

#[derive(Debug, Deserialize)]
pub struct EditMomentRequest {
    pub moment_id: String,
    pub description: Option<String>,
    pub image_file_ids: Option<Vec<String>>,
    pub tags: Option<Vec<String>>,
    pub location: Option<Location>,
    pub privacy_level: Option<PrivacyLevel>,
    pub mentions: Option<Vec<i64>>,
}

#[derive(Debug, Deserialize)]
pub struct FavoriteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnfavoriteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MyFavoritesRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByLocationRequest {
    pub kw: Option<String>,
    pub reference_point: GeoPoint,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MomentOpenTheDoorRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByTagRequest {
    pub tag: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserMomentsRequest {
    pub user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyMomentsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct LikeMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnlikeMomentRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MomentLikesRequest {
    pub moment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 评论与点赞 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateCommentRequest {
    pub moment_id: String,
    pub content: String,
    pub parent_id: Option<String>,
    pub mentions: Option<Vec<i64>>, // @ 提及的用户 ID 列表（最多 10 个）
}

#[derive(Debug, Deserialize)]
pub struct DeleteCommentRequest {
    pub comment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MomentCommentsRequest {
    pub moment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CommentRepliesRequest {
    pub comment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LikeCommentRequest {
    pub comment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnlikeCommentRequest {
    pub comment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct CommentLikesRequest {
    pub comment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 私聊管理 API ====================

#[derive(Debug, Deserialize)]
pub struct MyPrivateChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_archived: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivateChatSettingsRequest {
    pub target_user_id: i64,
    pub is_muted: Option<bool>,
    pub is_pinned: Option<bool>,
    pub is_archived: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeletePrivateChatRequest {
    pub target_user_id: i64,
}

// ==================== 群组管理 API ====================

#[derive(Debug, Deserialize, serde::Serialize)]
pub struct GroupSettingsUpdate {
    pub join_method: Option<GroupJoinMethod>,
    pub is_searchable: Option<bool>,
    pub max_members: Option<i32>,
    pub location: Option<Location>,
}

#[derive(Debug, Deserialize)]
pub struct CreateGroupChatRequest {
    pub name: String,
    pub description: Option<String>,
    pub avatar_file_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub password: Option<String>, // 群组密码（设置后加入需要验证）
    pub join_method: Option<GroupJoinMethod>,
    pub is_searchable: Option<bool>,
    pub max_members: Option<i32>,
    pub location: Option<Location>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupsByKeywordRequest {
    pub kw: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupsByTagRequest {
    pub tag: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupsByLocationRequest {
    pub kw: Option<String>,
    pub reference_point: GeoPoint,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupOpenTheDoorRequest {
    pub group_chat_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MyGroupChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupChatRequest {
    pub group_chat_id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    pub avatar_file_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub password: Option<String>, // 设置/更新群组密码（传空字符串清除密码）
    pub join_method: Option<GroupJoinMethod>,
    pub is_searchable: Option<bool>,
    pub max_members: Option<i32>,
    pub location: Option<Location>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteGroupChatRequest {
    pub group_chat_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupMySettingsRequest {
    pub group_chat_id: String,
    pub is_muted: Option<bool>,
    pub is_pinned: Option<bool>,
}

// ==================== 群组成员管理 API ====================

#[derive(Debug, Deserialize)]
pub struct GroupMembersRequest {
    pub group_chat_id: String,
    pub role: Option<GroupRole>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct InviteMemberRequest {
    pub group_chat_id: String,
    pub user_id: i64,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct JoinGroupRequest {
    pub group_chat_id: String,
    pub password: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SetMemberRoleRequest {
    pub group_chat_id: String,
    pub user_id: i64,
    pub role: SetableGroupRole,
}

#[derive(Debug, Deserialize)]
pub struct MuteMemberRequest {
    pub group_chat_id: String,
    pub user_id: i64,
    pub duration_minutes: i32,
}

#[derive(Debug, Deserialize)]
pub struct UnmuteMemberRequest {
    pub group_chat_id: String,
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BanMemberRequest {
    pub group_chat_id: String,
    pub user_id: i64,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RemoveMemberRequest {
    pub group_chat_id: String,
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct LeaveGroupRequest {
    pub group_chat_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnbanMemberRequest {
    pub group_chat_id: String,
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BannedMembersRequest {
    pub group_chat_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupApplicationsRequest {
    pub group_chat_id: String,
    pub status: Option<ApplicationStatusFilter>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ApproveGroupApplicationRequest {
    pub application_id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectGroupApplicationRequest {
    pub application_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct TransferOwnershipRequest {
    pub group_chat_id: String,
    pub new_owner_id: i64,
}

// ==================== 群公告 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateGroupAnnouncementRequest {
    pub group_chat_id: String,
    pub title: String,
    pub content: String,
    pub is_pinned: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct GroupAnnouncementsRequest {
    pub group_chat_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupAnnouncementRequest {
    pub announcement_id: String,
    pub title: Option<String>,
    pub content: Option<String>,
    pub is_pinned: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteGroupAnnouncementRequest {
    pub announcement_id: String,
}

#[derive(Debug, Deserialize)]
pub struct CancelApplicationRequest {
    pub application_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MyInvitesRequest {
    pub status: Option<ApplicationStatusFilter>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AcceptInviteRequest {
    pub invite_id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectInviteRequest {
    pub invite_id: String,
}

// ==================== 私聊消息 API ====================

#[derive(Debug, Deserialize)]
pub struct SendPrivateMessageRequest {
    pub target_user_id: i64,
    pub r#type: SendMessageType,
    pub content: Option<String>,
    pub file_id: Option<String>,
    pub reply_to_message_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct PrivateMessagesHistoryRequest {
    pub target_user_id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivateMessageRequest {
    pub message_id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallPrivateMessageRequest {
    pub message_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MarkPrivateMessagesReadRequest {
    pub target_user_id: i64,
    pub up_to_message_id: Option<String>,
}

// ==================== 群聊消息 API ====================

#[derive(Debug, Deserialize)]
pub struct SendGroupMessageRequest {
    pub group_chat_id: String,
    pub r#type: SendMessageType,
    pub content: Option<String>,
    pub file_id: Option<String>,
    pub reply_to_message_id: Option<String>,
    pub mentions: Option<Vec<i64>>,
}

#[derive(Debug, Deserialize)]
pub struct GroupMessagesHistoryRequest {
    pub group_chat_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_system: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupMessageRequest {
    pub message_id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallGroupMessageRequest {
    pub message_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MarkGroupMessagesReadRequest {
    pub group_chat_id: String,
    pub up_to_message_id: Option<String>,
}

// ==================== 举报审核 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateReportRequest {
    pub target_type: ReportTargetType,
    pub target_id: String, // 统一为 string（user_id 也转为字符串格式）
    pub reason: ReportReason,
    pub description: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyReportsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ReportsRequest {
    pub status: Option<ReportStatus>,
    pub target_type: Option<ReportTargetType>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct HandleReportRequest {
    pub report_id: String,
    pub action: ReportHandleAction,
    pub note: Option<String>,
}

// ==================== 系统健康检查 API ====================

#[derive(Debug, Deserialize)]
pub struct SystemHealthRequest {}

// ==================== 文件与媒体上传 API ====================

#[derive(Debug, Deserialize)]
pub struct PresignUploadRequest {
    pub file_name: String,
    pub file_size: i64,
    pub mime_type: String,
    pub category: MediaCategory,
    pub md5: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CompleteUploadRequest {
    pub upload_id: String,
}

// ==================== Map 聚合 / 视口查询（public） ====================

#[derive(Debug, Deserialize, Serialize, Clone, Copy)]
pub struct GeoPoint {
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Deserialize, Clone, Copy)]
pub struct Bounds {
    pub north: f64,
    pub south: f64,
    pub east: f64,
    pub west: f64,
}

#[derive(Debug, Deserialize, Clone)]
pub struct MapViewport {
    pub bounds: Bounds,
    pub zoom: i32,
}

#[derive(Debug, Deserialize)]
pub struct TimeRangeParams {
    pub start_time: Option<String>,
    pub end_time: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MapFilter {
    pub data_type: Option<MapDataType>,
    pub only_friends: Option<bool>,
    pub time_range: Option<TimeRangeParams>,
}

#[derive(Debug, Deserialize)]
pub struct MapRenderRequest {
    pub viewport: MapViewport,
    pub filter: Option<MapFilter>,
}

// ==================== 系统全局配置 API (Client Fetch) ====================

#[derive(Debug, Deserialize)]
pub struct FetchGlobalSettingsRequest {}

// ==================== 管理员系统配置更新 API ====================

#[derive(Debug, Deserialize)]
pub struct AdminGetGlobalSettingsRequest {}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateUploadConfigRequest {
    pub image_max_size: Option<i64>,
    pub video_max_size: Option<i64>,
    pub audio_max_size: Option<i64>,
    pub file_max_size: Option<i64>,
    pub allowed_image_types: Option<Vec<String>>,
    pub allowed_video_types: Option<Vec<String>>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateDefaultLocationRequest {
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateDefaultsRequest {
    pub default_location: Option<UpdateDefaultLocationRequest>,
    pub default_group_avatar: Option<String>,
    pub default_user_avatar: Option<String>,
    pub default_group_settings: Option<GroupSettingsUpdate>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateFeaturesRequest {
    pub enable_public_registration: Option<bool>,
    pub enable_guest_login: Option<bool>,
    pub enable_moments: Option<bool>,
    pub enable_nearby_people: Option<bool>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateContactRequest {
    pub support_email: Option<String>,
    pub privacy_policy_url: Option<String>,
    pub terms_of_service_url: Option<String>,
    pub official_website: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateLimitsRequest {
    pub max_group_members: Option<i32>,
    pub max_moment_tags: Option<i32>,
    pub max_moment_images: Option<i32>,
    pub moment_description_max_length: Option<i32>,
    pub comment_max_length: Option<i32>,
    pub search_radius_max: Option<i32>,
}

/// 更新频率限制配置请求 - API 文档: UpdateRateLimitsRequest
#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateRateLimitsRequest {
    // IP级限速（单IP限制，考虑多用户共享同一IP的场景）
    pub ip_per_hour: Option<u32>,
    // 用户级限流
    pub user_per_5minutes: Option<u32>,
    pub user_per_hour: Option<u32>,
    // 认证相关限速
    pub auth_send_code_per_minute: Option<u32>,
    pub auth_verify_code_per_10minutes: Option<u32>,
    pub auth_register_per_10minutes: Option<u32>,
    pub auth_login_per_10minutes: Option<u32>,
    pub auth_guest_per_10minutes: Option<u32>,
    pub auth_refresh_per_10minutes: Option<u32>,
    pub auth_reset_request_per_minute: Option<u32>,
    pub auth_reset_verify_per_10minutes: Option<u32>,
    pub auth_reset_per_10minutes: Option<u32>,
    // 业务相关限速
    pub moments_create_per_minute: Option<u32>,
    pub comments_create_per_minute: Option<u32>,
    pub reports_create_per_hour: Option<u32>,
}

#[derive(Debug, Deserialize)]
pub struct FeedMomentsRequest {
    pub source: Option<FeedSource>,
    pub reference_point: Option<GeoPoint>,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedGroupsRequest {
    pub reference_point: GeoPoint,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedUsersRequest {
    pub reference_point: GeoPoint,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedTrendingRequest {
    pub limit: Option<i32>, // 默认 20，最大 50
}

#[derive(Debug, Deserialize)]
pub struct SearchSuggestRequest {
    pub prefix: String,
    pub limit: Option<i32>,
    pub location: Option<GeoPoint>,
}

// ==================== 标签管理 API ====================

#[derive(Debug, Deserialize)]
pub struct SuggestTagsRequest {
    pub prefix: String,
    pub limit: Option<i32>,
}

// ==================== 管理员 RBAC（admin/rbac） ====================

#[derive(Debug, Deserialize)]
pub struct AdminRbacPermissionsRequest {}

#[derive(Debug, Deserialize)]
pub struct AdminRbacMeRequest {}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRolesListRequest {}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRoleCreateRequest {
    pub name: String,
    pub description: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRoleUpdateRequest {
    pub role_id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    /// true 表示将 description 清空（置 NULL）
    pub clear_description: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRoleDeleteRequest {
    pub role_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRoleSetPermissionsRequest {
    pub role_id: String,
    pub permission_keys: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacAdminsListRequest {}

#[derive(Debug, Deserialize)]
pub struct AdminRbacGrantAdminRequest {
    pub user_id: i64,
    pub grant_all_permissions: Option<bool>,
    pub role_ids: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRevokeAdminRequest {
    pub user_id: i64,
}

// ==================== 管理员标签管理 ====================

#[derive(Debug, Deserialize)]
pub struct AdminDeleteTagRequest {
    pub tag_name: String,
}

// ==================== 管理员用户管理 ====================

#[derive(Debug, Deserialize)]
pub struct AdminUserSearchRequest {
    pub user_id: Option<i64>,
    pub nickname: Option<String>,
    pub login_id: Option<String>,
    pub status: Option<UserAccountStatus>,
    pub user_type: Option<UserType>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminUserDetailRequest {
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct AdminBanUserRequest {
    pub user_id: i64,
    pub reason: Option<String>,
    pub duration_days: Option<i32>, // 0 or null = permanent
}

#[derive(Debug, Deserialize)]
pub struct AdminUnbanUserRequest {
    pub user_id: i64,
    pub reason: Option<String>,
}

// ==================== 管理员群组管理 ====================

#[derive(Debug, Deserialize)]
pub struct AdminGroupSearchRequest {
    pub group_chat_id: Option<String>,
    pub name: Option<String>,
    pub owner_id: Option<i64>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminGroupDetailRequest {
    pub group_chat_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AdminDismissGroupRequest {
    pub group_chat_id: String,
    pub reason: Option<String>,
}

// ==================== 管理员审计日志 ====================

#[derive(Debug, Deserialize)]
pub struct AuditLogListRequest {
    pub action: Option<String>,
    pub operator_id: Option<i64>,
    pub target_type: Option<String>,
    pub target_id: Option<String>,
    pub start_time: Option<String>,
    pub end_time: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AuditLogDetailRequest {
    pub log_id: String,
}

// ==================== 管理员系统公告管理 ====================

#[derive(Debug, Deserialize)]
pub struct CreateAnnouncementRequest {
    pub title: String,
    pub content: String,
    pub severity: AnnouncementSeverity,
    pub action_url: Option<String>,
    pub send_push: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct AdminAnnouncementListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteAnnouncementRequest {
    pub announcement_id: String,
}

// ==================== 公开系统公告查询 ====================

/// 公开系统公告列表请求（空结构体，返回条数由服务端 ResponseDefaults 控制）
#[derive(Debug, Deserialize)]
pub struct PublicAnnouncementListRequest {}

// ==================== 补充类型（如确有需要再添加，避免重复定义） ====================

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByKeywordRequest {
    pub kw: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// Private Chat
#[derive(Debug, Deserialize)]
pub struct PrivateChatOpenTheDoorRequest {
    pub target_user_id: i64,
    pub recent_messages_limit: Option<i32>,
}

// User 相关
#[derive(Debug, Deserialize)]
pub struct SearchUsersByKeywordRequest {
    pub kw: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchUsersByTagRequest {
    pub tag: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchUsersByLocationRequest {
    pub kw: Option<String>,
    pub reference_point: GeoPoint,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
// NOTE: Duplicate removed. Use the canonical `UserOpenTheDoorRequest` with `user_id` defined earlier in this file.
pub struct MyFriendsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyFollowingRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyFollowersRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyBlockedRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 位置分享模块请求 ====================

// ==================== 分享模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct CreateShareRequest {
    pub share_type: ShareType,
    #[serde(flatten)]
    pub payload: serde_json::Value,
}

#[derive(Debug, Deserialize)]
pub struct GetShareRequest {
    pub share_code: String,
}

// ==================== 统一搜索请求（与 spec 对齐） ====================

/// 统一用户搜索请求 - POST /api/v1/users/search
#[derive(Debug, Deserialize)]
pub struct SearchUsersRequest {
    pub keyword: Option<String>,
    pub tag: Option<String>,
    pub radius: Option<f64>,
    pub sort_by: Option<String>,
    pub reference_point: Option<GeoPoint>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 统一群组搜索请求 - POST /api/v1/groups/search
#[derive(Debug, Deserialize)]
pub struct SearchGroupsRequest {
    pub keyword: Option<String>,
    pub tag: Option<String>,
    pub radius: Option<f64>,
    pub sort_by: Option<String>,
    pub reference_point: Option<GeoPoint>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 统一瞬间搜索请求 - POST /api/v1/moments/search
#[derive(Debug, Deserialize)]
pub struct SearchMomentsRequest {
    pub keyword: Option<String>,
    pub tag: Option<String>,
    pub radius: Option<f64>,
    pub sort_by: Option<String>,
    pub reference_point: Option<GeoPoint>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 群组成员搜索请求 - POST /api/v1/groups/members/search
#[derive(Debug, Deserialize)]
pub struct SearchGroupMembersRequest {
    pub group_chat_id: String,
    pub keyword: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 撤销所有会话请求 - POST /api/v1/auth/me/sessions/revoke_all
#[derive(Debug, Deserialize)]
pub struct RevokeAllSessionsRequest {}

/// 系统元信息请求 - POST /api/v1/system/get_meta
#[derive(Debug, Deserialize)]
pub struct SystemGetMetaRequest {}

/// 分享撤销请求 - POST /api/v1/share/revoke
#[derive(Debug, Deserialize)]
pub struct RevokeShareRequest {
    pub share_id: String,
}

/// 我的分享列表请求 - POST /api/v1/share/me/list
#[derive(Debug, Deserialize)]
pub struct MySharesRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 批量预签名上传请求 - POST /api/v1/media/batch_presign_upload
#[derive(Debug, Deserialize)]
pub struct BatchPresignUploadRequest {
    pub items: Vec<PresignUploadRequest>,
}

/// 管理员标签搜索请求 - POST /api/v1/admin/tags/search
#[derive(Debug, Deserialize)]
pub struct AdminTagSearchRequest {
    pub keyword: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 管理员更新公告请求 - POST /api/v1/admin/announcements/update
#[derive(Debug, Deserialize)]
pub struct AdminUpdateAnnouncementRequest {
    pub announcement_id: String,
    pub title: Option<String>,
    pub content: Option<String>,
    pub severity: Option<AnnouncementSeverity>,
    pub is_active: Option<bool>,
}

/// 管理员更新响应默认值请求
#[derive(Debug, Deserialize)]
pub struct AdminUpdateResponseDefaultsRequest {
    pub default_page_size: Option<i32>,
    pub max_page_size: Option<i32>,
    pub system_announcements_count: Option<i32>,
}

// ==================== 通知模块请求 ====================

/// 通知列表请求
#[derive(Debug, Deserialize)]
pub struct NotificationListRequest {
    #[serde(rename = "type")]
    pub notification_type: Option<NotificationType>,
    pub cursor: Option<String>,
    pub limit: Option<i64>,
}

/// 标记通知已读请求
#[derive(Debug, Deserialize)]
pub struct MarkNotificationsReadRequest {
    pub notification_ids: Vec<String>,
}
