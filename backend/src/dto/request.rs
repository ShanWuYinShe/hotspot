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
pub struct CreateTempRequest {
    pub nickname: String,
}

#[derive(Debug, Deserialize)]
pub struct RefreshTokenRequest {
    pub refresh_token: String,
}

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
pub struct SearchUserByNameRequest {
    pub name: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchUserByIdRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct SearchUserByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: Option<i32>,
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
    pub avatar_url: Option<String>,
    pub bio: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpgradeAccountRequest {
    pub login_id: String,
    pub password_hash: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateSettingsRequest {
    pub privacy_level: Option<String>,
    pub location_sharing: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteAccountRequest {
    pub confirmation: String,
}

#[derive(Debug, Deserialize)]
pub struct AddFriendRequest {
    pub to_user_id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AcceptFriendRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectFriendRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct RemoveFriendRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct FollowRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnfollowRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct BlockRequest {
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UnblockRequest {
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
pub struct ListBlockedRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SetFriendRemarkRequest {
    pub user_id: String,
    pub remark: String,
}

// ==================== 群组模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct SearchGroupByLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub radius: Option<i32>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupByNameRequest {
    pub name: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SearchGroupByIdRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct ListMembersRequest {
    pub id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct CreateGroupRequest {
    pub group_type: String,
    pub name: String,
    pub description: Option<String>,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub password_hash: Option<String>,
    pub avatar_url: Option<String>,
    pub max_members: Option<i32>,
    pub initial_members: Option<Vec<String>>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupProfileRequest {
    pub id: String,
    pub name: Option<String>,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupLocationRequest {
    pub id: String,
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdatePrivacyRequest {
    pub id: String,
    pub group_type: Option<String>,
    pub password_hash: Option<String>,
    pub max_members: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct SetMemberRoleRequest {
    pub id: String,
    pub user_id: String,
    pub role: String,
}

#[derive(Debug, Deserialize)]
pub struct MuteMemberRequest {
    pub id: String,
    pub user_id: String,
    pub duration: i64,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct BanMemberRequest {
    pub id: String,
    pub user_id: String,
    pub action: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct InviteRequest {
    pub id: String,
    pub user_ids: Vec<String>,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AcceptInviteRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct RejectInviteRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct JoinRequest {
    pub id: String,
    pub password_hash: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct LeaveRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct RemoveMemberRequest {
    pub id: String,
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct TransferOwnershipRequest {
    pub id: String,
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct ListGroupsRequest {
    pub group_type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListInvitesRequest {
    pub status: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateNotificationsRequest {
    pub id: String,
    pub enabled: Option<bool>,
    pub mention_only: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct ApplyToGroupRequest {
    pub id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ReviewApplicationRequest {
    pub id: String,
    pub action: String,
    pub reason: Option<String>,
}

// ==================== 消息模块请求 ====================

#[derive(Debug, Deserialize)]
pub struct SendMessageRequest {
    pub target_id: String,
    pub target_type: String,
    pub message_type: String,
    pub content: String,
    pub reply_to: Option<String>,
    pub mentions: Option<Vec<String>>,
    pub latitude: Option<f64>,
    pub longitude: Option<f64>,
    pub location_name: Option<String>,
    pub file_name: Option<String>,
    pub file_size: Option<i64>,
    pub thumbnail_url: Option<String>,
    pub duration: Option<i32>,
}

#[derive(Debug, Deserialize)]
pub struct EditMessageRequest {
    pub id: String,
    pub content: String,
}

#[derive(Debug, Deserialize)]
pub struct RecallMessageRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct ListMessagesRequest {
    pub target_id: String,
    pub target_type: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
    pub include_system: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteMessageRequest {
    pub id: String,
    pub for_everyone: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct MarkReadRequest {
    pub target_id: String,
    pub target_type: String,
    pub up_to: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ReactToMessageRequest {
    pub id: String,
    pub reaction_type: String,
    pub action: String,
}

#[derive(Debug, Deserialize)]
pub struct ListReactionsRequest {
    pub id: String,
    pub reaction_type: Option<String>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

// ==================== 私聊会话请求 ====================

#[derive(Debug, Deserialize)]
pub struct StartChatRequest {
    pub user_id: String,
    pub initial_message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ListChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GetChatSettingsRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateChatSettingsRequest {
    pub id: String,
    pub muted: Option<bool>,
    pub pinned: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteChatRequest {
    pub id: String,
}

#[derive(Debug, Deserialize)]
pub struct UpgradeToGroupRequest {
    pub id: String,
    pub group_type: String,
    pub name: String,
    pub description: Option<String>,
    pub avatar_url: Option<String>,
    pub password_hash: Option<String>,
    pub invited_users: Option<Vec<String>>,
}

