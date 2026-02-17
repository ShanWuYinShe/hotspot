//! 群组成员相关响应类型

use crate::types::enums::{ApplicationStatus, GroupRole, UserPresenceStatus};
use serde::Serialize;

use super::group::GroupInfo;
use super::user::UserInfo;

/// 群组成员项 - API 文档: GroupMemberItem
#[derive(Debug, Serialize)]
pub struct GroupMemberItem {
    pub user: UserInfo,
    pub role: GroupRole,
    pub joined_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub muted_until: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presence_status: Option<UserPresenceStatus>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_online: Option<String>,
}

/// 被封禁成员项 - API 文档: BannedMemberItem
#[derive(Debug, Serialize, Clone)]
pub struct BannedMemberItem {
    pub user: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    pub banned_at: String,
    /// spec: banned_by?: UserInfo (@group-admin-only)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub banned_by: Option<UserInfo>,
}

/// 邀请成员返回数据
#[derive(Debug, Serialize)]
pub struct InviteMemberData {
    pub invite_id: String,
    pub expires_at: String,
}

/// 我的入群申请列表项（含群组信息）- 用于 groups/me/applications/list
#[derive(Debug, Serialize)]
pub struct MyApplicationItem {
    pub application_id: String,
    pub group: GroupInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_by: Option<UserInfo>,
}

/// 群组申请项 - API 文档: GroupApplicationItem
#[derive(Debug, Serialize)]
pub struct GroupApplicationItem {
    pub application_id: String,
    pub applicant: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub handled_by: Option<UserInfo>,
}

/// 加入群组状态
#[derive(Debug, Serialize, Clone, Copy, PartialEq, Eq)]
#[serde(rename_all = "snake_case")]
pub enum JoinGroupStatus {
    Joined,
    PendingApproval,
}

/// 申请响应 - API 文档: Application（groups/me/join 的 pending_approval 时返回）
#[derive(Debug, Serialize)]
pub struct JoinGroupApplication {
    pub application_id: String,
}

/// 加入群组响应 - API 文档: JoinGroupResult
#[derive(Debug, Serialize)]
pub struct JoinGroupResult {
    pub status: JoinGroupStatus,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub application: Option<JoinGroupApplication>,
}

/// 群组邀请项 - API 文档: GroupInviteItem
#[derive(Debug, Serialize)]
pub struct GroupInviteItem {
    pub invite_id: String,
    pub group: GroupInfo,
    pub inviter: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
    pub status: ApplicationStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub expires_at: Option<String>,
}
