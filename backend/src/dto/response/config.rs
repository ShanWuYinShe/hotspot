//! 系统配置相关响应类型

use serde::{Deserialize, Serialize};

use super::common::{id_as_string, GeoPoint};
use super::group::GroupSettings;

/// 上传配置 - API 文档: UploadConfig
#[derive(Debug, Serialize, Deserialize)]
pub struct UploadConfig {
    pub image_max_size: i64,
    pub video_max_size: i64,
    pub audio_max_size: i64,
    pub file_max_size: i64,
    pub allowed_image_types: Vec<String>,
    pub allowed_video_types: Vec<String>,
}

/// 默认位置 - API 文档: DefaultLocation (对应 spec 中的 Location 类型)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DefaultLocation {
    pub point: GeoPoint,
    pub location_name: String,
}

/// 内容默认值 - API 文档: ContentDefaults
#[derive(Debug, Serialize, Deserialize)]
pub struct ContentDefaults {
    pub default_location: DefaultLocation,
    pub default_group_avatar: String,
    pub default_user_avatar: String,
    pub default_group_settings: GroupSettings,
}

/// 客户端可见的默认值 - API 文档: ClientDefaults
#[derive(Debug, Serialize, Deserialize)]
pub struct ClientDefaults {
    pub default_location: DefaultLocation,
    pub default_user_avatar: String,
    pub default_group_avatar: String,
}

/// 客户端公开配置 - API 文档: ClientConfig
#[derive(Debug, Serialize, Deserialize)]
pub struct ClientConfig {
    pub features: FeatureSwitches,
    pub limits: AppLimits,
    pub contact: ContactInfo,
    pub defaults: ClientDefaults,
}

/// 功能开关 - API 文档: FeatureSwitches
#[derive(Debug, Serialize, Deserialize)]
pub struct FeatureSwitches {
    pub enable_public_registration: bool,
    pub enable_guest_login: bool,
    pub enable_moments: bool,
    pub enable_nearby_people: bool,
}

/// 联系信息 - API 文档: ContactInfo
#[derive(Debug, Serialize, Deserialize)]
pub struct ContactInfo {
    pub support_email: String,
    pub privacy_policy_url: String,
    pub terms_of_service_url: String,
    pub official_website: String,
}

/// 应用限制 - API 文档: AppLimits
#[derive(Debug, Serialize, Deserialize)]
pub struct AppLimits {
    pub max_group_members: i32,
    pub max_moment_tags: i32,
    pub max_moment_images: i32,
    pub moment_description_max_length: i32,
    pub comment_max_length: i32,
    pub search_radius_max: i32,
}

/// 频率限制配置 - API 文档: RateLimitConfig
#[derive(Debug, Serialize, Deserialize)]
pub struct RateLimitConfig {
    // IP级限速（单IP限制，考虑多用户共享同一IP的场景）
    pub ip_per_hour: u32,
    // 用户级限流
    pub user_per_5minutes: u32,
    pub user_per_hour: u32,
    // 认证相关限速
    pub auth_send_code_per_minute: u32,
    pub auth_verify_code_per_10minutes: u32,
    pub auth_register_per_10minutes: u32,
    pub auth_login_per_10minutes: u32,
    pub auth_guest_per_10minutes: u32,
    pub auth_refresh_per_10minutes: u32,
    pub auth_reset_request_per_minute: u32,
    pub auth_reset_verify_per_10minutes: u32,
    pub auth_reset_per_10minutes: u32,
    // 业务相关限速
    pub moments_create_per_minute: u32,
    pub comments_create_per_minute: u32,
    pub reports_create_per_hour: u32,
}

/// 全局设置 - API 文档: GlobalSettings
#[derive(Debug, Serialize, Deserialize)]
pub struct GlobalSettings {
    pub upload: UploadConfig,
    pub defaults: ContentDefaults,
    pub features: FeatureSwitches,
    pub contact: ContactInfo,
    pub limits: AppLimits,
    pub rate_limits: RateLimitConfig,
}

/// 管理员全局设置数据 - API 文档: AdminGlobalSettingsData
#[derive(Debug, Serialize)]
pub struct AdminGlobalSettingsData {
    #[serde(flatten)]
    pub settings: GlobalSettings,
    pub updated_at: String,
    #[serde(serialize_with = "id_as_string::serialize")]
    pub updated_by_user_id: i64,
}

/// 更新上传策略响应
#[derive(Debug, Serialize)]
pub struct UpdateUploadConfigData {
    pub upload: UploadConfig,
    pub updated_at: String,
}

/// 更新内容默认值响应
#[derive(Debug, Serialize)]
pub struct UpdateDefaultsConfigData {
    pub defaults: ContentDefaults,
    pub updated_at: String,
}

/// 更新功能开关响应
#[derive(Debug, Serialize)]
pub struct UpdateFeaturesConfigData {
    pub features: FeatureSwitches,
    pub updated_at: String,
}

/// 更新联系信息响应
#[derive(Debug, Serialize)]
pub struct UpdateContactConfigData {
    pub contact: ContactInfo,
    pub updated_at: String,
}

/// 更新系统限制响应
#[derive(Debug, Serialize)]
pub struct UpdateLimitsConfigData {
    pub limits: AppLimits,
    pub updated_at: String,
}

/// 更新频率限制响应
#[derive(Debug, Serialize)]
pub struct UpdateRateLimitsConfigData {
    pub rate_limits: RateLimitConfig,
    pub updated_at: String,
}
