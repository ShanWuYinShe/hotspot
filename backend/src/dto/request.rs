//! API 请求类型定义
//! 
//! 部分字段暂未使用，但已按 API 规范定义

#![allow(dead_code)]

use serde::Deserialize;

// ==================== 认证模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct RegisterRequest {
    pub login_id: String,
    pub nickname: String,
    pub password_hash: String,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub login_id: String,
    pub password_hash: String,
}

#[derive(Debug, Deserialize)]
pub struct LogoutRequest {}

#[derive(Debug, Deserialize)]
pub struct CreateTempRequest {
    pub nickname: String,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckStatusRequest {}

#[derive(Debug, Deserialize)]
pub struct RequestPasswordResetRequest {
    pub login_id: String,
}

#[derive(Debug, Deserialize)]
pub struct VerifyResetCodeRequest {
    pub reset_id: String,
    pub code: String,
}

#[derive(Debug, Deserialize)]
pub struct ResetPasswordRequest {
    pub reset_token: String,
    pub password_hash: String,
}

#[derive(Debug, Deserialize)]
pub struct GetSessionsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RevokeSessionRequest {
    pub id: String,
}

// ==================== 用户模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct GetMyProfileRequest {}

#[derive(Debug, Deserialize)]
pub struct GetUserStatsRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct GetMyStatsRequest {}

#[derive(Debug, Deserialize)]
pub struct GetUserFriendsListRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUserFollowingListRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUserFollowersListRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUserGroupsListRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUserLikedMomentsListRequest {
    pub id: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: Option<String>,
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
    pub password_hash: String,
}

#[derive(Debug, Deserialize)]
pub struct GetSettingsRequest {}

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
pub struct FollowUserRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct UnfollowUserRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BlockUserRequest {
    pub id: i64,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnblockUserRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct SetFriendRemarkRequest {
    pub id: i64,
    pub remark: Option<String>,
}

// ==================== 生活瞬间 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateMomentRequest {
    pub description: String,
    pub location: LocationInfo,
    pub tags: Option<Vec<String>>,
    pub images: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct LocationInfo {
    pub latitude: f64,
    pub longitude: f64,
    pub name: String,
}

#[derive(Debug, Deserialize)]
pub struct EditMomentRequest {
    pub id: String,
    pub description: Option<String>,
    pub images: Option<Vec<String>>,
    pub tags: Option<Vec<String>>,
    pub location: Option<LocationInfo>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateFavoriteMomentRequest {
    pub id: String,
    pub action: String, // 'add' | 'remove'
}

#[derive(Debug, Deserialize)]
pub struct ListFavoritesRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByLocationRequest {
    pub kw: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByTagsRequest {
    pub tags: Vec<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListMomentsRequest {
    pub user_id: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteMomentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct LikeMomentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct GetMomentLikesRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 评论与点赞 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateCommentRequest {
    pub moment_id: String,
    pub content: String,
    pub parent_id: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteCommentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct ListCommentsRequest {
    pub moment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListRepliesRequest {
    pub id: String,
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
pub struct GetCommentsRequest {
    pub moment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetRepliesRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetCommentLikesListRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetCommentLikesRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 私聊管理 API ====================

#[derive(Debug, Deserialize)]
pub struct CreatePrivateChatRequest {
    pub target_user_id: i64,
    pub initial_message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetPrivateChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_archived: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivateChatSettingsRequest {
    pub id: String,
    pub is_muted: Option<bool>,
    pub is_pinned: Option<bool>,
    pub is_archived: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeletePrivateChatRequest {
    pub id: String,
}

// ==================== 群组管理 API ====================

#[derive(Debug, Deserialize, Default)]
pub struct GroupSettings {
    #[serde(default = "default_join_method")]
    pub join_method: String,
    #[serde(default = "default_true")]
    pub is_searchable: bool,
    pub max_members: Option<i32>,
    pub password_hash: Option<String>,
}

fn default_join_method() -> String {
    "open".to_string()
}

fn default_true() -> bool {
    true
}

#[derive(Debug, Deserialize)]
pub struct GroupSettingsUpdate {
    pub join_method: Option<String>,
    pub is_searchable: Option<bool>,
    pub max_members: Option<i32>,
    pub password_hash: Option<String>,
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
pub struct DiscoverGroupChatsRequest {
    pub keyword: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<f64>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetJoinedGroupChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupChatRequest {
    pub id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    pub avatar_file_id: Option<String>,
    pub tags: Option<Vec<String>>,
    pub settings: Option<GroupSettingsUpdate>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteGroupChatRequest {
    pub id: String,
}

// ==================== 群组成员管理 API ====================

#[derive(Debug, Deserialize)]
pub struct GetMembersRequest {
    pub id: String,
    pub role: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct InviteMemberRequest {
    pub id: String,
    pub user_id: i64,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct JoinGroupRequest {
    pub id: String,
    pub password_hash: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ApplyToGroupRequest {
    pub id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SetMemberRoleRequest {
    pub id: String,
    pub user_id: i64,
    pub role: String,
}

#[derive(Debug, Deserialize)]
pub struct MuteMemberRequest {
    pub id: String,
    pub user_id: i64,
    pub duration_minutes: i32,
}

#[derive(Debug, Deserialize)]
pub struct UnmuteMemberRequest {
    pub id: String,
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BanMemberRequest {
    pub id: String,
    pub user_id: i64,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RemoveMemberRequest {
    pub id: String,
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct LeaveGroupRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupApplicationsRequest {
    pub id: String,
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
    pub id: String,
    pub new_owner_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupInvitesRequest {
    pub status: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AcceptGroupInviteRequest {
    pub invite_id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectGroupInviteRequest {
    pub invite_id: String,
}

// ==================== 私聊消息 API ====================

#[derive(Debug, Deserialize)]
pub struct SendPrivateMessageRequest {
    pub id: String,
    pub r#type: String,
    pub content: Option<String>,  // 文本消息必填，媒体消息可选
    pub file_id: Option<String>,  // 媒体消息必填，服务端转换为实际 url
    pub reply_to: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetPrivateMessagesRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivateMessageRequest {
    pub id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallPrivateMessageRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct DeletePrivateMessageRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct MarkPrivateMessagesReadRequest {
    pub id: String,
    pub up_to: Option<String>,
}

// ==================== 群聊消息 API ====================

#[derive(Debug, Deserialize)]
pub struct SendGroupMessageRequest {
    pub id: String,
    pub r#type: String,
    pub content: Option<String>,
    pub file_id: Option<String>,
    pub reply_to: Option<String>,
    pub mentions: Option<Vec<i64>>,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupMessagesRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_system: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupMessageRequest {
    pub id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallGroupMessageRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct DeleteGroupMessageRequest {
    pub id: String,
    pub for_everyone: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct MarkGroupMessagesReadRequest {
    pub id: String,
    pub up_to: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetMessageReadStatusRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AddGroupMessageReactionRequest {
    pub id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupMessageReactionsRequest {
    pub id: String,
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
pub struct ListUserReportsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListReportsRequest {
    pub status: Option<String>,
    pub target_type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct HandleReportRequest {
    pub id: String,
    pub action: String,
    pub note: Option<String>,
}

// ==================== 通知订阅设置 API ====================

#[derive(Debug, Deserialize)]
pub struct GetSubscriptionsRequest {}

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

// ==================== 通知中心 API ====================

#[derive(Debug, Deserialize)]
pub struct ListInboxItemsRequest {
    pub category: Option<String>,
    pub types: Option<Vec<String>>,
    pub is_read: Option<bool>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUnreadInboxSummaryRequest {}

#[derive(Debug, Deserialize)]
pub struct MarkInboxItemsReadRequest {
    pub ids: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteInboxItemsRequest {
    pub ids: Vec<String>,
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

// ==================== 系统健康检查 API ====================

#[derive(Debug, Deserialize)]
pub struct HealthCheckRequest {}


// ==================== 文件与媒体上传 API ====================

// UploadMediaRequest usually involves multipart/form-data which isn't typically defined as a JSON struct in Rust handlers 
// unless using a specific crate. 
// But let's define it as a struct for type checking, though handling might differ.
// Note: file field is usually handled separately.
#[derive(Debug, Deserialize)]
pub struct UploadMediaRequest {
    pub category: String,
}

#[derive(Debug, Deserialize)]
pub struct InitiateUploadRequest {
    pub file_name: String,
    pub file_size: i64,
    pub mime_type: String,
    pub category: String,      // 文件分类
    pub chunk_size: Option<i64>,
}

#[derive(Debug, Deserialize)]
pub struct PartInfo {
    pub part_number: i32,
    pub etag: String,
}

#[derive(Debug, Deserialize)]
pub struct CompleteUploadRequest {
    pub upload_id: String,
    pub parts: Vec<PartInfo>,
}

#[derive(Debug, Deserialize)]
pub struct GetUploadStatusRequest {
    pub upload_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AbortUploadRequest {
    pub upload_id: String,
}

// ==================== 信息流 API ====================

#[derive(Debug, Deserialize)]
pub struct GetFeedMomentsRequest {
    pub source: Option<String>,  // 'following' | 'nearby' | 'all'，默认 'all'
    pub radius: Option<i32>,     // 仅 source='nearby' 时有效，默认 5000 米
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetFeedGroupsRequest {
    pub radius: Option<i32>,     // 搜索半径（米），默认 10000
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetFeedUsersRequest {
    pub radius: Option<i32>,     // 搜索半径（米），默认 5000
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetFeedTrendingRequest {
    pub limit: Option<i32>,      // 默认 20，最大 50
}

#[derive(Debug, Deserialize)]
pub struct FeedFeedbackRequest {
    pub content_type: String,    // 'moment' | 'group' | 'user'
    pub content_id: String,
    pub action: String,          // 'hide' | 'not_interested' | 'report'
    pub reason: Option<String>,
}

// ==================== 标签管理 API ====================

#[derive(Debug, Deserialize)]
pub struct GetTrendingTagsRequest {
    pub limit: Option<i32>,
    pub timeframe: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SuggestTagsRequest {
    pub prefix: String,
    pub limit: Option<i32>,
}

// ==================== 系统配置管理 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateConfigRequest {
    pub key: String,
    pub value: String,
    pub r#type: String,
    pub category: String,
    pub description: String,
}

#[derive(Debug, Deserialize)]
pub struct GetConfigRequest {
    pub key: String,
}

#[derive(Debug, Deserialize)]
pub struct GetConfigListRequest {
    pub category: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateConfigRequest {
    pub key: String,
    pub value: String,
}

#[derive(Debug, Deserialize)]
pub struct DeleteConfigRequest {
    pub key: String,
}

#[derive(Debug, Deserialize)]
pub struct GetConfigHistoryRequest {
    pub key: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListAllConfigChangesRequest {
    pub start_time: Option<String>,
    pub end_time: Option<String>,
    pub changed_by: Option<i64>,  // 操作者用户ID
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 别名和补充类型 ====================

// Group 相关
pub type UnbanMemberRequest = RemoveMemberRequest; // 复用结构
pub type GetBannedMembersListRequest = GetMembersRequest; // 复用结构

/// 按关键字搜索群组请求
#[derive(Debug, Deserialize)]
pub struct SearchGroupsByKeywordRequest {
    pub kw: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 按标签搜索群组请求
#[derive(Debug, Deserialize)]
pub struct SearchGroupsByTagRequest {
    pub tag: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

/// 按位置搜索群组请求
#[derive(Debug, Deserialize)]
pub struct SearchGroupsByLocationRequest {
    pub kw: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

pub type GetGroupSettingsRequest = DeleteGroupChatRequest; // 只需要 group_chat_id
pub type GetGroupInvitesListRequest = GetGroupInvitesRequest;
pub type CancelGroupApplicationRequest = ApproveGroupApplicationRequest; // 只需要 application_id
pub type GetGroupApplicationsListRequest = GetGroupApplicationsRequest;

#[derive(Debug, Deserialize)]
pub struct GroupOpenTheDoorRequest {
    pub id: String,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByKeywordRequest {
    pub kw: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByTagRequest {
    pub tag: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetMomentCommentsRequest {
    pub moment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetUserMomentsListRequest {
    pub id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetMyMomentsListRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MomentOpenTheDoorRequest {
    pub id: String,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Deserialize)]
pub struct UnlikeMomentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckMomentLikedRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct FavoriteMomentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnfavoriteMomentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckMomentFavoritedRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct GetFavoritesListRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetMomentLikesListRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AddPrivateMessageReactionRequest {
    pub id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct RemovePrivateMessageReactionRequest {
    pub id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct GetPrivateMessageReactionsListRequest {
    pub id: String,
    pub r#type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RemoveGroupMessageReactionRequest {
    pub id: String,
    pub r#type: String,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupMessageReactionsListRequest {
    pub id: String,
    pub r#type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// Device Tokens
#[derive(Debug, Deserialize)]
pub struct GetDeviceTokensRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// Media Upload
#[derive(Debug, Deserialize)]
pub struct UploadChunkRequest {
    pub upload_id: String,
    pub part_number: i32,
}

// Private Chat
#[derive(Debug, Deserialize)]
pub struct PrivateChatOpenTheDoorRequest {
    pub id: String,
}

// Reports
#[derive(Debug, Deserialize)]
pub struct GetMyReportsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

pub type GetReportsRequest = ListReportsRequest;
pub type GetConfigChangesRequest = ListAllConfigChangesRequest;
/// 修改密码请求
#[derive(Debug, Deserialize)]
pub struct ChangePasswordRequest {
    pub old_password_hash: String,
    pub new_password_hash: String,
}

// User 相关
#[derive(Debug, Deserialize)]
pub struct SearchUsersByKeywordRequest {
    pub kw: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchUsersByTagRequest {
    pub tag: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchUsersByLocationRequest {
    pub kw: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetStatusRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct UserOpenTheDoorRequest {
    pub id: i64,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Deserialize)]
pub struct GetRelationshipRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct GetFriendsListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetFollowingListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetFollowersListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetBlockedUsersListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

