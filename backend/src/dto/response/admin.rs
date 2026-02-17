//! 管理员 RBAC 及管理相关响应类型

use serde::Serialize;
use serde_json::Value;

use super::{
    common::{id_as_string, RelativeLocation},
    group::GroupItem,
    user::{UserInfo, UserStats},
};

/// 管理员权限项 - API 文档: AdminPermissionItem
#[derive(Debug, Serialize)]
pub struct AdminPermissionItem {
    pub key: String,
    pub description: String,
}

/// 管理员权限数据
#[derive(Debug, Serialize)]
pub struct AdminPermissionsData {
    pub items: Vec<AdminPermissionItem>,
}

/// 管理员角色项 - API 文档: AdminRoleItem
#[derive(Debug, Serialize)]
pub struct AdminRoleItem {
    pub role_id: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
    pub permissions: Vec<String>,
}

/// 管理员角色数据
#[derive(Debug, Serialize)]
pub struct AdminRolesData {
    pub items: Vec<AdminRoleItem>,
}

/// 管理员角色基础信息 - API 文档: AdminRoleBasic
#[derive(Debug, Serialize)]
pub struct AdminRoleBasic {
    pub role_id: String,
    pub name: String,
}

/// 管理员项 - API 文档: AdminAdminItem
#[derive(Debug, Serialize)]
pub struct AdminAdminItem {
    pub user: UserInfo,
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

/// 当前管理员数据 - API 文档: AdminMeData
#[derive(Debug, Serialize)]
pub struct AdminMeData {
    pub permissions: Vec<String>,
    pub roles: Vec<AdminRoleBasic>,
}

/// 管理员用户项 - API 文档: AdminUserSearchItem
#[derive(Debug, Serialize)]
pub struct AdminUserSearchItem {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: crate::types::enums::UserType,
    pub status: crate::types::enums::UserAccountStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    pub login_id: String,
    pub presence_status: crate::types::enums::UserPresenceStatus,
}

/// 管理员用户详情 - API 文档: AdminUserDetail
#[derive(Debug, Serialize)]
pub struct AdminUserDetail {
    #[serde(serialize_with = "id_as_string::serialize")]
    pub user_id: i64,
    pub nickname: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avatar_url: Option<String>,
    pub user_type: crate::types::enums::UserType,
    pub status: crate::types::enums::UserAccountStatus,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_active_at: Option<String>,
    pub login_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub phone: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bio: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tags: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub location: Option<RelativeLocation>,
    pub presence_status: crate::types::enums::UserPresenceStatus,
    pub stats: UserStats,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ban_history: Option<Vec<AdminBanRecord>>,
}

/// 管理员封禁记录 - API 文档: AdminBanRecord
#[derive(Debug, Serialize, Clone)]
pub struct AdminBanRecord {
    pub banned_at: String,
    pub banned_by: UserInfo,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub duration_days: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unbanned_at: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unbanned_by: Option<UserInfo>,
}

#[derive(Debug, Serialize)]
pub struct AdminGroupStats {
    pub messages_count: i64,
    pub active_members_today: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reports_count: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_reported_at: Option<String>,
}

/// 管理员群组项 - API 文档: AdminGroupItem
#[derive(Debug, Serialize)]
pub struct AdminGroupItem {
    #[serde(flatten)]
    pub group: GroupItem,
    pub owner: UserInfo,
    pub admins: Vec<UserInfo>,
    pub stats: AdminGroupStats,
}

/// 审计日志项 - API 文档: AuditLogItem
#[derive(Debug, Serialize, Clone)]
pub struct AuditLogItem {
    pub log_id: String,
    pub action: String,
    pub operator: UserInfo,
    pub target_type: String,
    pub target_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    pub created_at: String,
}

/// 审计日志详情 - API 文档: AuditLogDetail
#[derive(Debug, Serialize)]
pub struct AuditLogDetail {
    pub log_id: String,
    pub action: String,
    pub operator: UserInfo,
    pub target_type: String,
    pub target_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub target_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub details: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ip_address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_agent: Option<String>,
    pub created_at: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub before_state: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub after_state: Option<Value>,
}

/// 创建公告返回数据 — spec: { announcement_id }
#[derive(Debug, Serialize)]
pub struct CreateAnnouncementData {
    pub announcement_id: String,
}

/// 系统公告项 - API 文档: SystemAnnouncementItem
#[derive(Debug, Serialize)]
pub struct SystemAnnouncementItem {
    pub announcement_id: String,
    pub title: String,
    pub content: String,
    pub severity: crate::types::enums::AnnouncementSeverity,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub action_url: Option<String>,
    pub created_at: String,
}

/// 公开系统公告列表响应（非分页，返回条数由服务端 ResponseDefaults 控制）
#[derive(Debug, Serialize)]
pub struct SystemAnnouncementListData {
    pub items: Vec<SystemAnnouncementItem>,
}

/// 管理员公告列表项 - API 文档: AdminAnnouncementItem
#[derive(Debug, Serialize)]
pub struct AdminAnnouncementItem {
    #[serde(flatten)]
    pub announcement: SystemAnnouncementItem,
    pub created_by: UserInfo,
    pub send_push: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub push_sent_at: Option<String>,
    pub view_count: i32,
}
