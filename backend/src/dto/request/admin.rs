//! 管理员系统配置 + RBAC + 标签管理 + 用户管理 + 群组管理 + 审计日志 + 系统公告

use serde::{Deserialize, Serialize};

use super::{common::PaginationParams, group::GroupSettingsUpdate};
use crate::types::enums::{AnnouncementSeverity, UserAccountStatus, UserType};

#[derive(Debug, Deserialize)]
pub struct AdminGetGlobalSettingsRequest {}

#[derive(Debug, Deserialize, Serialize)]
pub struct UpdateUploadConfigRequest {
    pub upload: UploadConfigUpdate,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct UploadConfigUpdate {
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
    pub limits: LimitsUpdate,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct LimitsUpdate {
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
pub struct AdminRbacAdminsListRequest {
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacGrantAdminRequest {
    pub user_id: String,
    pub grant_all_permissions: Option<bool>,
    pub role_ids: Vec<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminRbacRevokeAdminRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AdminDeleteTagRequest {
    pub name: String,
}

/// 管理员标签搜索请求 - POST /api/v1/admin/tags/search
#[derive(Debug, Deserialize, Clone, Copy)]
#[serde(rename_all = "snake_case")]
pub enum AdminTagSortBy {
    UsageCountDesc,
    UsageCountAsc,
    CreatedAtDesc,
    LastUsedAtDesc,
}

#[derive(Debug, Deserialize)]
pub struct AdminTagSearchRequest {
    pub keyword: Option<String>,
    pub sort_by: Option<AdminTagSortBy>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct AdminUserSearchRequest {
    pub user_id: Option<String>,
    pub nickname: Option<String>,
    pub login_id: Option<String>,
    pub status: Option<UserAccountStatus>,
    pub user_type: Option<UserType>,
    pub limit: Option<i32>,
    pub cursor: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminUserDetailRequest {
    pub user_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AdminBanUserRequest {
    pub user_id: String,
    pub reason: Option<String>,
    pub duration_days: Option<i32>, // 0 or null = permanent
}

#[derive(Debug, Deserialize)]
pub struct AdminUnbanUserRequest {
    pub user_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AdminGroupSearchRequest {
    pub group_id: Option<String>,
    pub name: Option<String>,
    pub owner_id: Option<String>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct AdminGroupDetailRequest {
    pub group_id: String,
}

#[derive(Debug, Deserialize)]
pub struct AdminDismissGroupRequest {
    pub group_id: String,
    pub reason: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AuditLogListRequest {
    pub action: Option<String>,
    pub operator_id: Option<String>,
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
    pub severity: Option<AnnouncementSeverity>,
    pub pagination: Option<PaginationParams>,
}

#[derive(Debug, Deserialize)]
pub struct DeleteAnnouncementRequest {
    pub announcement_id: String,
}

/// 管理员更新公告请求 - POST /api/v1/admin/announcements/update
#[derive(Debug, Deserialize)]
pub struct AdminUpdateAnnouncementRequest {
    pub announcement_id: String,
    pub title: Option<String>,
    pub content: Option<String>,
    pub severity: Option<AnnouncementSeverity>,
    pub action_url: Option<String>,
}

/// 管理员更新响应默认值请求
#[derive(Debug, Deserialize)]
pub struct AdminUpdateResponseDefaultsRequest {
    pub response_defaults: ResponseDefaultsUpdate,
}

#[derive(Debug, Deserialize)]
pub struct ResponseDefaultsUpdate {
    pub default_page_size: Option<i32>,
    pub max_page_size: Option<i32>,
    pub system_announcements_count: Option<i32>,
}
