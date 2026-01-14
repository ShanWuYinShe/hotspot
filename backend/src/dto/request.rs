//! API 请求类型定义
//!
//! 部分字段暂未使用，但已按 API 规范定义

use crate::types::enums::NotificationApiType;
use serde::Deserialize;

// ==================== 认证模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct RegisterRequest {
    pub login_id: String,
    pub nickname: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    /// 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
    pub client_type: String,
    /// 可选：如果通过手机验证码注册，提供此 token
    pub verify_token: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub login_id: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    /// 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
    pub client_type: String,
}

#[derive(Debug, Deserialize)]
pub struct LogoutRequest {}

#[derive(Debug, Deserialize)]
pub struct GuestRequest {
    pub nickname: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    /// 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
    pub client_type: String,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    /// app：在 body 传递；web：必须不传（从 HttpOnly cookie 读取），若传入则视为参数错误
    pub refresh_token: Option<String>,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    /// 强制：客户端必须显式传递，避免服务端推断导致的行为漂移
    pub client_type: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckStatusRequest {}

#[derive(Debug, Deserialize)]
pub struct SendVerificationCodeRequest {
    pub phone: String,
    pub r#type: String, // 'register' | 'reset_password' | 'change_phone'
    pub captcha_token: Option<String>, // 可选：图形验证码 token（防机器刷）
}

#[derive(Debug, Deserialize)]
pub struct VerifyCodeRequest {
    pub code_id: String,
    pub code: String,
}

#[derive(Debug, Deserialize)]
pub struct RequestPasswordResetRequest {
    pub login_id: String,
    pub verify_token: Option<String>, // 可选：如果通过验证码验证，提供此 token
}

#[derive(Debug, Deserialize)]
pub struct VerifyResetCodeRequest {
    pub reset_id: String,
    pub code: String,
}

#[derive(Debug, Deserialize)]
pub struct ResetPasswordRequest {
    pub reset_token: String,
    pub password: String,
}

#[derive(Debug, Deserialize)]
pub struct ChangePasswordRequest {
    pub old_password: String,
    pub new_password: String,
}

#[derive(Debug, Deserialize)]
pub struct SessionsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RevokeSessionRequest {
    pub session_id: String,
}

// ==================== 用户模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct MyProfileRequest {}

#[derive(Debug, Deserialize)]
pub struct UserStatsRequest {
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct MyStatsRequest {}

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
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserOpenTheDoorRequest {
    pub user_id: i64,
    pub reference_point: GeoPoint,
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
    pub privacy_level: Option<String>,
    pub location_visible: Option<bool>,
    pub online_visible: Option<bool>,
    pub last_online_visible: Option<bool>,
    pub friends_visible: Option<String>,
    pub following_visible: Option<String>,
    pub followers_visible: Option<String>,
    pub moments_visible: Option<String>,
    pub stats_visible: Option<String>,
    pub allow_follow: Option<bool>,
    pub allow_private_chat: Option<String>,
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
    pub privacy_level: Option<String>, // 'public' | 'friends' | 'private'，未提供时使用用户全局设置
    pub mentions: Option<Vec<i64>>,    // @ 提及的用户 ID 列表（最多 10 个）
}

#[derive(Debug, Deserialize)]
pub struct Location {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
}

#[derive(Debug, Deserialize)]
pub struct MessageLocation {
    pub location: Location,
    pub address: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateMomentRequest {
    pub moment_id: String,
    pub description: Option<String>,
    pub image_file_ids: Option<Vec<String>>,
    pub tags: Option<Vec<String>>,
    pub location: Option<Location>,
    pub privacy_level: Option<String>, // 'public' | 'friends' | 'private'
    pub mentions: Option<Vec<i64>>,    // @ 提及的用户 ID 列表（最多 10 个）
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
    pub reference_point: GeoPoint,
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
    pub reference_point: GeoPoint,
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
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyMomentsRequest {
    pub reference_point: GeoPoint,
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

#[derive(Debug, Deserialize, Default)]
pub struct GroupSettings {
    #[serde(default = "default_join_method")]
    pub join_method: String,
    #[serde(default = "default_true")]
    pub is_searchable: bool,
    pub max_members: Option<i32>,
    pub password: Option<String>,
}

fn default_join_method() -> String {
    "open".to_string()
}

fn default_true() -> bool {
    true
}

#[derive(Debug, Deserialize, serde::Serialize)]
pub struct GroupSettingsUpdate {
    pub join_method: Option<String>,
    pub is_searchable: Option<bool>,
    pub max_members: Option<i32>,
    pub password: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CreateGroupChatRequest {
    pub name: String,
    pub description: Option<String>,
    pub avatar_file_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub settings: Option<GroupSettings>,
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
    pub reference_point: GeoPoint,
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
    pub settings: Option<GroupSettingsUpdate>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteGroupChatRequest {
    pub group_chat_id: String,
}

// ==================== 群组成员管理 API ====================

#[derive(Debug, Deserialize)]
pub struct GroupMembersRequest {
    pub group_chat_id: String,
    pub role: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupSettingsRequest {
    pub group_chat_id: String,
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
pub struct ApplyToGroupRequest {
    pub group_chat_id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SetMemberRoleRequest {
    pub group_chat_id: String,
    pub user_id: i64,
    pub role: String,
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
    pub status: Option<String>,
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
    pub status: Option<String>,
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
    pub r#type: String,
    pub content: Option<String>,           // 文本消息必填，媒体消息可选
    pub file_id: Option<String>,           // 媒体消息必填，服务端转换为实际 url
    pub location: Option<MessageLocation>, // 位置消息必填（当 type='location' 时）
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
pub struct DeletePrivateMessageRequest {
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
    pub r#type: String,
    pub content: Option<String>,
    pub file_id: Option<String>,
    pub location: Option<MessageLocation>, // 位置消息必填（当 type='location' 时）
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
pub struct DeleteGroupMessageRequest {
    pub message_id: String,
    pub for_everyone: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct MarkGroupMessagesReadRequest {
    pub group_chat_id: String,
    pub up_to_message_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MessageReadStatusRequest {
    pub message_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AddGroupMessageReactionRequest {
    pub message_id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct RemoveGroupMessageReactionRequest {
    pub message_id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct AddPrivateMessageReactionRequest {
    pub message_id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct RemovePrivateMessageReactionRequest {
    pub message_id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct GroupMessageReactionsRequest {
    pub message_id: String,
    pub r#type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct PrivateMessageReactionsRequest {
    pub message_id: String,
    pub r#type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 消息通用 API ====================

#[derive(Debug, Deserialize)]
pub struct MarkAllMessagesReadRequest {}

// ==================== 举报审核 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateReportRequest {
    pub target_type: String,
    pub target_id: String,
    pub reason: String,
    pub description: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyReportsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ReportsRequest {
    pub status: Option<String>,
    pub target_type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct HandleReportRequest {
    pub report_id: String,
    pub action: String,
    pub note: Option<String>,
}

// ==================== 通知订阅设置 API ====================

#[derive(Debug, Deserialize)]
pub struct MySubscriptionsRequest {}

#[derive(Debug, Deserialize)]
pub struct UpdateSubscriptionsRequest {
    pub new_follower_enabled: Option<bool>,
    pub mutual_follow_enabled: Option<bool>,
    pub group_invite_enabled: Option<bool>,
    pub group_invite_response_enabled: Option<bool>,
    pub group_application_enabled: Option<bool>,
    pub group_application_response_enabled: Option<bool>,
    pub message_mention_enabled: Option<bool>,
    pub moment_like_enabled: Option<bool>,
    pub moment_comment_enabled: Option<bool>,
    pub comment_reply_enabled: Option<bool>,
    pub system_announcement_enabled: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct RegisterDeviceTokenRequest {
    pub platform: String,
    pub device_token: String,
}

#[derive(Debug, Deserialize)]
pub struct UnregisterDeviceTokenRequest {
    pub platform: String,
    pub device_token: String,
}

// ==================== 通知 API ====================

#[derive(Debug, Deserialize)]
pub struct GetNotificationsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUnreadNotificationCountRequest {}

#[derive(Debug, Deserialize)]
pub struct MarkNotificationsReadRequest {
    pub notification_ids: Option<Vec<String>>,
    pub r#type: Option<NotificationApiType>,
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
    pub category: crate::types::enums::FileCategory,
    pub md5: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CompleteUploadRequest {
    pub upload_id: String,
}

// Chunked Upload (Optional/Enhanced)
#[derive(Debug, Deserialize)]
pub struct InitChunkedUploadRequest {
    pub file_name: String,
    pub file_size: i64,
    pub mime_type: String,
    pub category: crate::types::enums::FileCategory,
    pub chunk_count: i32,
}

#[derive(Debug, Deserialize)]
pub struct FinishChunkedUploadRequest {
    pub upload_id: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckChunkedUploadRequest {
    pub upload_id: String,
}

// ==================== Map 聚合 / 视口查询（public） ====================

#[derive(Debug, Deserialize, Clone, Copy)]
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
    pub data_type: Option<String>, // 'user' | 'group' | 'moment' | 'all'
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
pub struct ClientContext {
    pub os: String, // 'ios' | 'android' | 'web' | 'harmony'
    pub app_version: String,
    pub device_id: Option<String>,
    pub language: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FetchGlobalSettingsRequest {
    pub context: ClientContext,
    pub current_version: Option<String>,
}

// ==================== 管理员系统配置更新 API ====================

#[derive(Debug, Deserialize)]
pub struct AdminGetGlobalSettingsRequest {}

#[derive(Debug, Deserialize)]
pub struct UpdateUploadConfigRequest {
    pub image_max_size: Option<i64>,
    pub video_max_size: Option<i64>,
    pub audio_max_size: Option<i64>,
    pub file_max_size: Option<i64>,
    pub allowed_image_types: Option<Vec<String>>,
    pub allowed_video_types: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateDefaultsRequest {
    pub default_location_name: Option<String>,
    pub default_group_avatar: Option<String>,
    pub default_user_avatar: Option<String>,
    pub default_group_settings: Option<GroupSettingsUpdate>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateFeaturesRequest {
    pub enable_public_registration: Option<bool>,
    pub enable_guest_login: Option<bool>,
    pub enable_moments: Option<bool>,
    pub enable_nearby_people: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateContactRequest {
    pub support_email: Option<String>,
    pub privacy_policy_url: Option<String>,
    pub terms_of_service_url: Option<String>,
    pub official_website: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateLimitsRequest {
    pub max_group_members: Option<i32>,
    pub max_moment_tags: Option<i32>,
    pub max_moment_images: Option<i32>,
    pub moment_description_max_length: Option<i32>,
    pub comment_max_length: Option<i32>,
    pub search_radius_max: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct FeedMomentsRequest {
    pub source: Option<String>, // 'following' | 'nearby' | 'all'，默认 'all'
    pub radius: Option<i32>,    // 仅 source='nearby' 时有效，默认 5000 米
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedGroupsRequest {
    pub radius: Option<i32>, // 搜索半径（米），默认 10000
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedUsersRequest {
    pub radius: Option<i32>, // 搜索半径（米），默认 5000
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedTrendingRequest {
    pub limit: Option<i32>, // 默认 20，最大 50
}

#[derive(Debug, Deserialize)]
pub struct FeedFeedbackRequest {
    pub content_type: String, // 'moment' | 'group' | 'user'
    pub content_id: String,
    pub action: String, // 'hide' | 'not_interested' | 'report'
    pub reason: Option<String>,
}

// ==================== 标签管理 API ====================

#[derive(Debug, Deserialize)]
pub struct TrendingTagsRequest {
    pub limit: Option<i32>,
    pub timeframe: Option<String>,
}

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
    pub status: Option<String>, // 'normal' | 'banned'
    pub user_type: Option<crate::types::enums::UserType>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
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
pub struct AdminDismissGroupRequest {
    pub group_chat_id: String,
    pub reason: Option<String>,
}

// ==================== 管理员系统公告管理 ====================

#[derive(Debug, Deserialize)]
pub struct CreateAnnouncementRequest {
    pub title: String,
    pub content: String,
    pub severity: String, // 'info' | 'warning' | 'error'
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

#[derive(Debug, Deserialize)]
pub struct PublicAnnouncementListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 补充类型（如确有需要再添加，避免重复定义） ====================

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByKeywordRequest {
    pub kw: String,
    pub reference_point: GeoPoint,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CheckLikedRequest {
    pub moment_id: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckFavoritedRequest {
    pub moment_id: String,
}

// Device Tokens
#[derive(Debug, Deserialize)]
pub struct MyDeviceTokensRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// Media Upload

// Private Chat
#[derive(Debug, Deserialize)]
pub struct PrivateChatOpenTheDoorRequest {
    pub target_user_id: i64,
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
