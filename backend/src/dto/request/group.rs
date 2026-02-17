//! 群组管理 + 群组成员管理 + 群公告 API

use serde::{Deserialize, Serialize};

use crate::dto::request::common::Location;
use crate::types::enums::{ApplicationStatusFilter, GroupJoinMethod, GroupRole, SetableGroupRole};

#[derive(Debug, Deserialize, Serialize)]
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
    pub location: Location,
}

#[derive(Debug, Deserialize)]
pub struct GroupOpenTheDoorRequest {
    pub group_id: String,
}

#[derive(Debug, Deserialize)]
pub struct MyGroupChatsRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupChatRequest {
    pub group_id: String,
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
    pub group_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupMySettingsRequest {
    pub group_id: String,
    pub is_muted: Option<bool>,
    pub is_pinned: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct GroupMembersRequest {
    pub group_id: String,
    pub role: Option<GroupRole>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct InviteMemberRequest {
    pub group_id: String,
    pub user_id: String,
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct JoinGroupRequest {
    pub group_id: String,
    pub password: Option<String>,
    /// 申请加入时的附言（approval 群组可选）
    pub message: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct SetMemberRoleRequest {
    pub group_id: String,
    pub user_id: String,
    pub role: SetableGroupRole,
}

#[derive(Debug, Deserialize)]
pub struct MuteMemberRequest {
    pub group_id: String,
    pub user_id: String,
    pub duration_minutes: i32,
}

#[derive(Debug, Deserialize)]
pub struct UnmuteMemberRequest {
    pub group_id: String,
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct BanMemberRequest {
    pub group_id: String,
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct RemoveMemberRequest {
    pub group_id: String,
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct LeaveGroupRequest {
    pub group_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UnbanMemberRequest {
    pub group_id: String,
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct BannedMembersRequest {
    pub group_id: String,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct GroupApplicationsRequest {
    pub group_id: String,
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
    pub group_id: String,
    pub new_owner_id: String,
}

#[derive(Debug, Deserialize)]
pub struct CreateGroupAnnouncementRequest {
    pub group_id: String,
    pub title: String,
    pub content: String,
    pub is_pinned: Option<bool>,
}

#[derive(Debug, Deserialize)]
pub struct GroupAnnouncementsRequest {
    pub group_id: String,
}

#[derive(Debug, Deserialize)]
pub struct UpdateGroupAnnouncementRequest {
    pub group_id: String,
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

/// 我的入群申请列表请求 - POST /api/v1/groups/me/applications/list
#[derive(Debug, Deserialize)]
pub struct MyApplicationsRequest {
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
