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
pub struct SetPasswordRequest {
    pub password_hash: String,
}

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
pub struct ListSessionsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RevokeSessionRequest {
    pub id: String,
}

// ==================== 用户模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct DiscoverUsersRequest {
    pub keyword: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<f64>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetMyProfileRequest {}

#[derive(Debug, Deserialize)]
pub struct OpenTheDoorRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct GetStatsRequest {
    pub id: Option<String>,
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
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
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
    pub is_location_sharing: Option<bool>,
    pub can_be_followed: Option<bool>,
    pub allow_private_chat: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteAccountRequest {
    pub confirmation: String,
}

#[derive(Debug, Deserialize)]
pub struct SendFriendRequestRequest {
    pub to_user_id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AcceptFriendRequestRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectFriendRequestRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct RemoveFriendRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct FollowUserRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnfollowUserRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct BlockUserRequest {
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnblockUserRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct ListFriendsRequest {
    pub status: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListFollowingRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListFollowersRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetBlockedUsersRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetFriendRequestsRequest {
    pub status: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CancelFriendRequestRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct SetFriendRemarkRequest {
    pub user_id: String,
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
    pub latitude: f64,
    pub longitude: f64,
    pub radius: Option<f64>,
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
    pub action: String, // 'like' | 'unlike'
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
    pub id: String,
    pub action: String, // 'like' | 'unlike'
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
    pub target_user_id: String,
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
    pub private_chat_id: String,
    pub is_muted: Option<bool>,
    pub is_pinned: Option<bool>,
    pub is_archived: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeletePrivateChatRequest {
    pub private_chat_id: String,
}

// ==================== 群组管理 API ====================

#[derive(Debug, Deserialize)]
pub struct GroupSettings {
    pub join_method: String, // 'open' | 'approval' | 'invite_only'
    pub is_searchable: bool,
    pub max_members: Option<i32>,
    pub password_hash: Option<String>,
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
    pub avatar_url: Option<String>,
    pub settings: GroupSettings,
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
    pub group_chat_id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub settings: Option<GroupSettingsUpdate>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteGroupChatRequest {
    pub group_chat_id: String,
}

// ==================== 群组成员管理 API ====================

#[derive(Debug, Deserialize)]
pub struct GetMembersRequest {
    pub group_chat_id: String,
    pub role: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct InviteMemberRequest {
    pub group_chat_id: String,
    pub user_id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct JoinGroupRequest {
    pub group_chat_id: String,
    pub password_hash: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ApplyToGroupRequest {
    pub group_chat_id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SetMemberRoleRequest {
    pub group_chat_id: String,
    pub user_id: String,
    pub role: String, // 'admin' | 'member'
}

#[derive(Debug, Deserialize)]
pub struct MuteMemberRequest {
    pub group_chat_id: String,
    pub user_id: String,
    pub duration_minutes: i32,
}

#[derive(Debug, Deserialize)]
pub struct UnmuteMemberRequest {
    pub group_chat_id: String,
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct BanMemberRequest {
    pub group_chat_id: String,
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RemoveMemberRequest {
    pub group_chat_id: String,
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct LeaveGroupRequest {
    pub group_chat_id: String,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupApplicationsRequest {
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
    pub new_owner_id: String,
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
    pub private_chat_id: String,
    pub r#type: String,
    pub content: String,
    pub reply_to: Option<String>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GetPrivateMessagesRequest {
    pub private_chat_id: String,
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
    pub private_chat_id: String,
    pub up_to: Option<String>,
}

// ==================== 群聊消息 API ====================

#[derive(Debug, Deserialize)]
pub struct SendGroupMessageRequest {
    pub group_chat_id: String,
    pub r#type: String,
    pub content: String,
    pub reply_to: Option<String>,
    pub mentions: Option<Vec<String>>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GetGroupMessagesRequest {
    pub group_chat_id: String,
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
    pub group_chat_id: String,
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
    pub action: String,
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
    pub friend_request_enabled: Option<bool>,
    pub friend_accepted_enabled: Option<bool>,
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

// ==================== 排行榜 API ====================

#[derive(Debug, Deserialize)]
pub struct GetCreatorLeaderboardRequest {
    pub timeframe: String,
    pub scope: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<f64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GetMomentLeaderboardRequest {
    pub timeframe: String,
    pub scope: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<f64>,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct GetMyCreatorRankRequest {
    pub timeframe: String,
    pub scope: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<f64>,
}

#[derive(Debug, Deserialize)]
pub struct GetMyMomentRankRequest {
    pub timeframe: String,
    pub scope: Option<String>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub radius: Option<f64>,
}

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
pub struct ListFeedRequest {
    pub channel: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FeedFeedbackRequest {
    pub item_id: String,
    pub action: String,
}

// ==================== 批量操作 API ====================

#[derive(Debug, Deserialize)]
pub struct MarkAsReadBatchItem {
    pub chat_type: String,
    pub chat_id: String,
    pub up_to: String,
}

#[derive(Debug, Deserialize)]
pub struct MarkAsReadBatchRequest {
    pub items: Vec<MarkAsReadBatchItem>,
}

#[derive(Debug, Deserialize)]
pub struct RemoveInboxBatchRequest {
    pub ids: Vec<String>,
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
    pub changed_by: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}
