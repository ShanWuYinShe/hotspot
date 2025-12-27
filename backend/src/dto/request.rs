//! API 请求类型定义
//!
//! 部分字段暂未使用，但已按 API 规范定义

use serde::Deserialize;

// ==================== 认证模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct RegisterRequest {
    pub login_id: String,
    pub nickname: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub login_id: String,
    pub password: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LogoutRequest {}

#[derive(Debug, Deserialize)]
pub struct GuestRequest {
    pub nickname: String,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    /// app：在 body 传递；web：可不传（从 HttpOnly cookie 读取）
    pub refresh_token: Option<String>,
    /// 客户端类型：web/app（用于 refresh_token 投递方式）
    pub client_type: Option<String>,
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
    pub id: String,
}

// ==================== 用户模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct MyProfileRequest {}

#[derive(Debug, Deserialize)]
pub struct UserStatsRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct MyStatsRequest {}

#[derive(Debug, Deserialize)]
pub struct UserRelationshipRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct UserFriendsRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserFollowingRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserFollowersRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserGroupsRequest {
    pub id: i64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UserLikedMomentsRequest {
    pub id: i64,
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
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct UnfollowRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BlockRequest {
    pub id: i64,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnblockRequest {
    pub id: i64,
}

#[derive(Debug, Deserialize)]
pub struct SetRemarkRequest {
    pub id: i64,
    pub remark: Option<String>,
}

// ==================== 生活瞬间 API ====================

#[derive(Debug, Deserialize)]
pub struct CreateMomentRequest {
    pub description: String,
    pub location: LocationInfo,
    pub tags: Option<Vec<String>>,
    pub image_file_ids: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct LocationInfo {
    pub latitude: f64,
    pub longitude: f64,
    pub name: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateMomentRequest {
    pub id: String,
    pub description: Option<String>,
    pub image_file_ids: Option<Vec<String>>,
    pub tags: Option<Vec<String>>,
    pub location: Option<LocationInfo>,
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
pub struct MyFavoritesRequest {
    pub latitude: f64,
    pub longitude: f64,
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
pub struct MomentOpenTheDoorRequest {
    pub id: String,
    pub latitude: f64,
    pub longitude: f64,
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
pub struct UserMomentsRequest {
    pub id: i64,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct MyMomentsRequest {
    pub latitude: f64,
    pub longitude: f64,
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
pub struct UnlikeMomentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct MomentLikesRequest {
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
pub struct MomentCommentsRequest {
    pub moment_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CommentRepliesRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LikeCommentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnlikeCommentRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct CommentLikesRequest {
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
pub struct MyPrivateChatsRequest {
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
    pub password: Option<String>,
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
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupsByTagRequest {
    pub tag: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupsByLocationRequest {
    pub kw: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub radius: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupOpenTheDoorRequest {
    pub id: String,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Deserialize)]
pub struct MyGroupChatsRequest {
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
pub struct GroupMembersRequest {
    pub id: String,
    pub role: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupSettingsRequest {
    pub id: String,
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
    pub password: Option<String>,
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
pub struct UnbanMemberRequest {
    pub id: String,
    pub user_id: i64,
}

#[derive(Debug, Deserialize)]
pub struct BannedMembersRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupApplicationsRequest {
    pub id: String,
    pub status: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ApproveApplicationRequest {
    pub application_id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectApplicationRequest {
    pub application_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct TransferOwnershipRequest {
    pub id: String,
    pub new_owner_id: i64,
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
    pub id: String,
    pub r#type: String,
    pub content: Option<String>, // 文本消息必填，媒体消息可选
    pub file_id: Option<String>, // 媒体消息必填，服务端转换为实际 url
    pub reply_to: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct PrivateMessagesHistoryRequest {
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
pub struct GroupMessagesHistoryRequest {
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
pub struct MessageReadStatusRequest {
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
pub struct RemoveGroupMessageReactionRequest {
    pub id: String,
    pub r#type: String,
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
pub struct GroupMessageReactionsRequest {
    pub id: String,
    pub r#type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct PrivateMessageReactionsRequest {
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
    pub id: String,
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

// ==================== 系统健康检查 API ====================

#[derive(Debug, Deserialize)]
pub struct SystemHealthRequest {}

// ==================== 文件与媒体上传 API ====================

#[derive(Debug, Deserialize)]
pub struct InitiateUploadRequest {
    pub file_name: String,
    pub file_size: i64,
    pub mime_type: String,
    pub category: String, // 文件分类
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
pub struct UploadStatusRequest {
    pub upload_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AbortUploadRequest {
    pub upload_id: String,
}

// Media Upload (辅助：按需获取分块上传地址)
#[derive(Debug, Deserialize)]
pub struct UploadPartUrlRequest {
    pub upload_id: String,
    pub part_number: i32,
}

// ==================== 信息流 API ====================

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
pub struct ConfigDetailRequest {
    pub key: String,
}

#[derive(Debug, Deserialize)]
pub struct ConfigListRequest {
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
pub struct ConfigHistoryRequest {
    pub key: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ConfigChangesRequest {
    pub start_time: Option<String>,
    pub end_time: Option<String>,
    pub changed_by: Option<i64>, // 操作者用户ID
    pub limit: Option<i32>,
    pub cursor: Option<String>,
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
    pub id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    /// true 表示将 description 清空（置 NULL）
    pub clear_description: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRoleDeleteRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRoleSetPermissionsRequest {
    pub id: String,
    pub permission_keys: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacAdminsListRequest {}

#[derive(Debug, Deserialize)]
pub struct AdminRbacGrantAdminRequest {
    pub user_id: i64,
    pub is_super: Option<bool>,
    pub role_ids: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRevokeAdminRequest {
    pub user_id: i64,
}

// ==================== 补充类型（如确有需要再添加，避免重复定义） ====================

#[derive(Debug, Deserialize)]
pub struct SearchMomentsByKeywordRequest {
    pub kw: String,
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CheckLikedRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct CheckFavoritedRequest {
    pub id: String,
}

// Device Tokens
#[derive(Debug, Deserialize)]
pub struct MyDeviceTokensRequest {
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

// ==================== Map 聚合 / 视口查询（public） ====================

#[derive(Debug, Deserialize, Clone, Copy)]
pub struct Bounds {
    pub north: f64,
    pub south: f64,
    pub east: f64,
    pub west: f64,
}

#[derive(Debug, Deserialize)]
pub struct MapClusterRequest {
    pub bounds: Bounds,
    pub zoom: i32,
    pub grid_size: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct MapSearchByBoundsRequest {
    pub bounds: Bounds,
    /// 用于计算 GeoData(distance/degrees) 的参考点（通常是“浏览锚点”）
    pub latitude: f64,
    pub longitude: f64,
    pub limit: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct UserOpenTheDoorRequest {
    pub id: i64,
    pub latitude: f64,
    pub longitude: f64,
}

#[derive(Debug, Deserialize)]
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
