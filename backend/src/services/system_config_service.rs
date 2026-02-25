// 系统配置服务
//
// 管理系统配置项，支持 Redis 缓存和配置变更订阅
//
// 设计：强类型聚合配置 (GlobalSettings)

use redis::AsyncCommands;
use std::collections::HashMap;
use std::sync::Arc;

use crate::{
    constants::{REDIS_CHANNEL_CONFIG_UPDATES, REDIS_KEY_CONFIG},
    dto::{
        request::{
            AdminUpdateResponseDefaultsRequest, FetchGlobalSettingsRequest, UpdateContactRequest,
            UpdateDefaultsRequest, UpdateFeaturesRequest, UpdateLimitsRequest,
            UpdateRateLimitsRequest, UpdateUploadConfigRequest,
        },
        response::{
            AdminGlobalSettingsData, AppLimits, ClientConfig, ClientDefaults, ContactInfo,
            ContentDefaults, DefaultLocation, FeatureSwitches, GlobalSettings, RateLimitConfig,
            UploadConfig,
        },
    },
    error::Result,
    repositories::system_config_repo::SystemConfigRepository,
    types::enums::GroupJoinMethod,
    utils::cache::{cache_keys, CacheManager},
};

#[derive(Clone)]
pub struct SystemConfigService {
    repo: SystemConfigRepository,
    redis: Arc<redis::Client>,
    cache_manager: CacheManager,
}

impl SystemConfigService {
    pub fn new(
        repo: SystemConfigRepository,
        redis: Arc<redis::Client>,
        cache_manager: CacheManager,
    ) -> Self {
        Self {
            repo,
            redis,
            cache_manager,
        }
    }

    // ==================== 客户端拉取配置 (Client Fetch) ====================

    /// 拉取客户端公开配置 (system/get_client_config)
    /// 返回 ClientConfig，仅包含 spec 规定的 features/limits/contact/defaults
    pub async fn get_client_config(
        &self,
        _req: FetchGlobalSettingsRequest,
    ) -> Result<ClientConfig> {
        let settings = self.get_full_global_settings().await?;
        Ok(ClientConfig {
            features: settings.features,
            limits: settings.limits,
            contact: settings.contact,
            defaults: ClientDefaults {
                default_location: settings.defaults.default_location,
                default_user_avatar: settings.defaults.default_user_avatar,
                default_group_avatar: settings.defaults.default_group_avatar,
            },
        })
    }

    // ==================== 管理员配置管理 (Admin Management) ====================

    /// 获取完整配置 (Admin)
    pub async fn get_admin_global_settings(&self) -> Result<AdminGlobalSettingsData> {
        let settings = self.get_full_global_settings().await?;

        // 获取最后更新元信息（不做变更历史，只记录“最后一次更新时间/操作者”）
        let updated_at = self
            .get_config_value("meta.updated_at")
            .await?
            .unwrap_or_else(|| chrono::Utc::now().to_rfc3339());
        let updated_by_user_id = self
            .get_config_value("meta.updated_by_user_id")
            .await?
            .and_then(|s| s.parse::<i64>().ok())
            .unwrap_or(0);

        Ok(AdminGlobalSettingsData {
            settings,
            updated_at,
            updated_by_user_id,
        })
    }

    /// 更新上传配置
    pub async fn update_upload_config(
        &self,
        req: UpdateUploadConfigRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let u = &req.upload;
        let mut updates = HashMap::new();
        if let Some(v) = u.image_max_size {
            updates.insert("upload.image_max_size", v.to_string());
        }
        if let Some(v) = u.video_max_size {
            updates.insert("upload.video_max_size", v.to_string());
        }
        if let Some(v) = u.audio_max_size {
            updates.insert("upload.audio_max_size", v.to_string());
        }
        if let Some(v) = u.file_max_size {
            updates.insert("upload.file_max_size", v.to_string());
        }
        if let Some(v) = u.allowed_image_types.clone() {
            updates.insert("upload.allowed_image_types", serde_json::to_string(&v)?);
        }
        if let Some(v) = u.allowed_video_types.clone() {
            updates.insert("upload.allowed_video_types", serde_json::to_string(&v)?);
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    /// 更新默认值配置
    pub async fn update_defaults_config(
        &self,
        req: UpdateDefaultsRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let mut updates = HashMap::new();
        if let Some(loc) = req.default_location {
            if let Some(lat) = loc.latitude {
                updates.insert("defaults.default_location_latitude", lat.to_string());
            }
            if let Some(lon) = loc.longitude {
                updates.insert("defaults.default_location_longitude", lon.to_string());
            }
            if let Some(name) = loc.location_name {
                updates.insert("defaults.default_location_name", name);
            }
        }
        if let Some(v) = req.default_group_avatar {
            updates.insert("defaults.default_group_avatar", v);
        }
        if let Some(v) = req.default_user_avatar {
            updates.insert("defaults.default_user_avatar", v);
        }
        if let Some(v) = req.default_group_settings {
            updates.insert(
                "defaults.default_group_settings",
                serde_json::to_string(&v)?,
            );
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    /// 更新功能开关
    pub async fn update_features_config(
        &self,
        req: UpdateFeaturesRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let mut updates = HashMap::new();
        if let Some(v) = req.enable_public_registration {
            updates.insert("features.enable_public_registration", v.to_string());
        }
        if let Some(v) = req.enable_guest_login {
            updates.insert("features.enable_guest_login", v.to_string());
        }
        if let Some(v) = req.enable_moments {
            updates.insert("features.enable_moments", v.to_string());
        }
        if let Some(v) = req.enable_nearby_people {
            updates.insert("features.enable_nearby_people", v.to_string());
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    /// 更新联系信息
    pub async fn update_contact_config(
        &self,
        req: UpdateContactRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let mut updates = HashMap::new();
        if let Some(v) = req.support_email {
            updates.insert("contact.support_email", v);
        }
        if let Some(v) = req.privacy_policy_url {
            updates.insert("contact.privacy_policy_url", v);
        }
        if let Some(v) = req.terms_of_service_url {
            updates.insert("contact.terms_of_service_url", v);
        }
        if let Some(v) = req.official_website {
            updates.insert("contact.official_website", v);
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    /// 更新限制配置
    pub async fn update_limits_config(
        &self,
        req: UpdateLimitsRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let mut updates = HashMap::new();
        let limits = &req.limits;
        if let Some(v) = limits.max_group_members {
            updates.insert("limits.max_group_members", v.to_string());
        }
        if let Some(v) = limits.max_moment_tags {
            updates.insert("limits.max_moment_tags", v.to_string());
        }
        if let Some(v) = limits.max_moment_images {
            updates.insert("limits.max_moment_images", v.to_string());
        }
        if let Some(v) = limits.moment_description_max_length {
            updates.insert("limits.moment_description_max_length", v.to_string());
        }
        if let Some(v) = limits.comment_max_length {
            updates.insert("limits.comment_max_length", v.to_string());
        }
        if let Some(v) = limits.search_radius_max {
            updates.insert("limits.search_radius_max", v.to_string());
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    /// 更新频率限制配置
    pub async fn update_rate_limits_config(
        &self,
        req: UpdateRateLimitsRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let mut updates = HashMap::new();
        let rl = &req.rate_limits;

        if let Some(v) = rl.global_per_hour {
            updates.insert("rate_limits.global_per_hour", v.to_string());
        }
        if let Some(v) = rl.user_per_hour {
            updates.insert("rate_limits.user_per_hour", v.to_string());
        }
        if let Some(v) = rl.auth_register_per_10minutes {
            updates.insert("rate_limits.auth_register_per_10minutes", v.to_string());
        }
        if let Some(v) = rl.auth_login_per_10minutes {
            updates.insert("rate_limits.auth_login_per_10minutes", v.to_string());
        }
        if let Some(v) = rl.auth_guest_per_10minutes {
            updates.insert("rate_limits.auth_guest_per_10minutes", v.to_string());
        }
        if let Some(v) = rl.auth_refresh_per_10minutes {
            updates.insert("rate_limits.auth_refresh_per_10minutes", v.to_string());
        }
        if let Some(v) = rl.auth_reset_per_10minutes {
            updates.insert("rate_limits.auth_reset_per_10minutes", v.to_string());
        }
        if let Some(v) = rl.moments_create_per_minute {
            updates.insert("rate_limits.moments_create_per_minute", v.to_string());
        }
        if let Some(v) = rl.comments_create_per_minute {
            updates.insert("rate_limits.comments_create_per_minute", v.to_string());
        }
        if let Some(v) = rl.search_per_minute {
            updates.insert("rate_limits.search_per_minute", v.to_string());
        }
        if let Some(v) = rl.write_per_minute {
            updates.insert("rate_limits.write_per_minute", v.to_string());
        }
        if let Some(v) = rl.reports_create_per_hour {
            updates.insert("rate_limits.reports_create_per_hour", v.to_string());
        }
        if let Some(v) = rl.upload_presign_per_hour {
            updates.insert("rate_limits.upload_presign_per_hour", v.to_string());
        }
        if let Some(v) = rl.ws_subscribe_per_second {
            updates.insert("rate_limits.ws_subscribe_per_second", v.to_string());
        }
        if let Some(v) = rl.admin_read_per_minute {
            updates.insert("rate_limits.admin_read_per_minute", v.to_string());
        }
        if let Some(v) = rl.admin_write_per_minute {
            updates.insert("rate_limits.admin_write_per_minute", v.to_string());
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    /// 更新响应默认值配置（分页默认值等）
    pub async fn update_response_defaults_config(
        &self,
        req: AdminUpdateResponseDefaultsRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let rd = &req.response_defaults;
        let mut updates = HashMap::new();
        if let Some(v) = rd.default_page_size {
            updates.insert("response_defaults.default_page_size", v.to_string());
        }
        if let Some(v) = rd.max_page_size {
            updates.insert("response_defaults.max_page_size", v.to_string());
        }
        if let Some(v) = rd.system_announcements_count {
            updates.insert(
                "response_defaults.system_announcements_count",
                v.to_string(),
            );
        }

        if !updates.is_empty() {
            self.batch_update_configs(updates, admin_id).await?;
        }

        self.get_full_global_settings().await
    }

    // ==================== 内部辅助方法 ====================

    /// 批量更新配置（不做版本控制/变更历史）
    async fn batch_update_configs(
        &self,
        updates: HashMap<&str, String>,
        admin_id: i64,
    ) -> Result<()> {
        if updates.is_empty() {
            return Ok(());
        }

        for (key, value) in updates {
            // 获取旧值用于日志
            let _old_val = self
                .repo
                .find_by_key(key)
                .await?
                .and_then(|c| config_value_to_string(c.config_value));

            self.repo
                .upsert(
                    key.to_string(),
                    serde_json::Value::String(value.clone()),
                    None,
                    false,
                    Some(admin_id),
                )
                .await?;

            // 清除单个配置项的 Redis 缓存（实时生效的关键）
            self.clear_single_config_cache(key).await?;
        }

        // 记录“最后更新时间/操作者”（用于 AdminSettings.updated_at / updated_by_user_id）
        let now = chrono::Utc::now().to_rfc3339();
        let meta_updates = [
            ("meta.updated_at", now),
            ("meta.updated_by_user_id", admin_id.to_string()),
        ];
        for (key, value) in meta_updates {
            self.repo
                .upsert(
                    key.to_string(),
                    serde_json::Value::String(value.clone()),
                    None,
                    false,
                    Some(admin_id),
                )
                .await?;
            self.clear_single_config_cache(key).await?;
        }

        // 清除完整配置缓存（只需清除一次）
        self.clear_full_settings_cache().await?;

        // 广播配置更新事件 (通知其他服务实例/客户端)
        self.publish_config_update("global.settings").await?;

        Ok(())
    }

    /// 获取单个配置值（带缓存）
    pub async fn get_config_value(&self, key: &str) -> Result<Option<String>> {
        let cache_key = format!("{}:{}", REDIS_KEY_CONFIG, key);
        let mut conn = self.redis.get_multiplexed_async_connection().await?;

        // 1. 尝试从缓存获取
        if let Ok(Some(val)) = conn.get::<_, Option<String>>(&cache_key).await {
            return Ok(Some(val));
        }

        // 2. 查 DB
        let config = self.repo.find_by_key(key).await?;

        // 3. 写入缓存 (如果有值)
        if let Some(c) = &config {
            if let Some(val) = config_value_to_string(c.config_value.clone()) {
                let _: () = conn.set_ex(&cache_key, &val, 3600).await?; // 1 hour TTL
            }
        }

        Ok(config.and_then(|c| config_value_to_string(c.config_value)))
    }

    /// 组装完整 GlobalSettings
    /// 这里可以使用 Multi-Get 优化 DB 查询，或者查出所有配置在内存 Map
    async fn get_full_global_settings(&self) -> Result<GlobalSettings> {
        let cache_key = "full_settings";

        // 1. 尝试从缓存获取
        match self
            .cache_manager
            .get::<GlobalSettings>(cache_keys::SYSTEM_CONFIG, cache_key)
            .await
        {
            Ok(Some(Some(settings))) => return Ok(settings),
            Ok(Some(None)) => {}
            Ok(None) => {}
            Err(e) => {
                tracing::warn!(error = %e, "get system config cache failed");
            }
        }

        // 2. 查询所有配置
        // 假设 list(None, 1000, None) 能拉取所有需要的配置
        let configs = self.repo.list_all().await?;
        let map: HashMap<String, String> = configs
            .into_iter()
            .filter_map(|c| config_value_to_string(c.config_value).map(|v| (c.config_key, v)))
            .collect();

        // 辅助闭包：获取值或默认值
        let get_str = |k: &str, default: &str| map.get(k).cloned().unwrap_or(default.to_string());
        let get_int =
            |k: &str, default: i64| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_i32 =
            |k: &str, default: i32| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_bool =
            |k: &str, default: bool| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_f64 =
            |k: &str, default: f64| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_json_vec = |k: &str| -> Vec<String> {
            map.get(k)
                .and_then(|v| serde_json::from_str(v).ok())
                .unwrap_or_default()
        };
        let get_json_settings = |k: &str| -> crate::dto::response::GroupSettings {
            map.get(k)
                .and_then(|v| serde_json::from_str(v).ok())
                .unwrap_or(crate::dto::response::GroupSettings {
                    join_method: GroupJoinMethod::Open,
                    is_searchable: true,
                    max_members: 500,
                    location: crate::dto::request::Location {
                        point: crate::dto::request::GeoPoint {
                            latitude: 0.0,
                            longitude: 0.0,
                        },
                        location_name: "Unknown".to_string(),
                    },
                })
        };

        let upload = UploadConfig {
            image_max_size: get_int("upload.image_max_size", 10 * 1024 * 1024),
            video_max_size: get_int("upload.video_max_size", 100 * 1024 * 1024),
            audio_max_size: get_int("upload.audio_max_size", 20 * 1024 * 1024),
            file_max_size: get_int("upload.file_max_size", 100 * 1024 * 1024),
            allowed_image_types: get_json_vec("upload.allowed_image_types"),
            allowed_video_types: get_json_vec("upload.allowed_video_types"),
        };

        let defaults = ContentDefaults {
            default_location: DefaultLocation {
                point: crate::dto::response::GeoPoint {
                    latitude: get_f64("defaults.default_location_latitude", 35.154922),
                    longitude: get_f64("defaults.default_location_longitude", 112.682524),
                },
                location_name: get_str("defaults.default_location_name", "南天门"),
            },
            default_group_avatar: get_str("defaults.default_group_avatar", ""),
            default_user_avatar: get_str("defaults.default_user_avatar", ""),
            default_group_settings: get_json_settings("defaults.default_group_settings"),
        };

        let features = FeatureSwitches {
            enable_public_registration: get_bool("features.enable_public_registration", true),
            enable_guest_login: get_bool("features.enable_guest_login", true),
            enable_moments: get_bool("features.enable_moments", true),
            enable_nearby_people: get_bool("features.enable_nearby_people", true),
        };

        let contact = ContactInfo {
            support_email: get_str("contact.support_email", "support@example.com"),
            privacy_policy_url: get_str("contact.privacy_policy_url", ""),
            terms_of_service_url: get_str("contact.terms_of_service_url", ""),
            official_website: get_str("contact.official_website", ""),
        };

        let limits = AppLimits {
            max_group_members: get_i32("limits.max_group_members", 500),
            max_moment_tags: get_i32("limits.max_moment_tags", 10),
            max_moment_images: get_i32("limits.max_moment_images", 9),
            moment_description_max_length: get_i32("limits.moment_description_max_length", 2000),
            comment_max_length: get_i32("limits.comment_max_length", 500),
            search_radius_max: get_i32(
                "limits.search_radius_max",
                crate::constants::MAX_SEARCH_RADIUS_I32,
            ),
        };

        let get_u32 =
            |k: &str, default: u32| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);

        let rate_limits = RateLimitConfig {
            global_per_hour: get_u32("rate_limits.global_per_hour", 20000),
            user_per_hour: get_u32("rate_limits.user_per_hour", 5000),
            auth_register_per_10minutes: get_u32("rate_limits.auth_register_per_10minutes", 10),
            auth_login_per_10minutes: get_u32("rate_limits.auth_login_per_10minutes", 30),
            auth_guest_per_10minutes: get_u32("rate_limits.auth_guest_per_10minutes", 60),
            auth_refresh_per_10minutes: get_u32("rate_limits.auth_refresh_per_10minutes", 120),
            auth_reset_per_10minutes: get_u32("rate_limits.auth_reset_per_10minutes", 10),
            search_per_minute: get_u32("rate_limits.search_per_minute", 60),
            write_per_minute: get_u32("rate_limits.write_per_minute", 30),
            moments_create_per_minute: get_u32("rate_limits.moments_create_per_minute", 20),
            comments_create_per_minute: get_u32("rate_limits.comments_create_per_minute", 60),
            reports_create_per_hour: get_u32("rate_limits.reports_create_per_hour", 10),
            upload_presign_per_hour: get_u32("rate_limits.upload_presign_per_hour", 100),
            ws_subscribe_per_second: get_u32("rate_limits.ws_subscribe_per_second", 10),
            admin_read_per_minute: get_u32("rate_limits.admin_read_per_minute", 100),
            admin_write_per_minute: get_u32("rate_limits.admin_write_per_minute", 30),
        };

        let settings = GlobalSettings {
            upload,
            defaults,
            features,
            contact,
            limits,
            rate_limits,
        };

        // 3. 写入缓存
        if let Err(e) = self
            .cache_manager
            .set(cache_keys::SYSTEM_CONFIG, cache_key, &settings)
            .await
        {
            tracing::warn!(error = %e, "set system config cache failed");
        }

        Ok(settings)
    }

    /// 清除单个配置项的 Redis 缓存
    async fn clear_single_config_cache(&self, config_key: &str) -> Result<()> {
        let mut conn = self.redis.get_multiplexed_async_connection().await?;
        let cache_key = format!("{}:{}", REDIS_KEY_CONFIG, config_key);
        let _: () = conn.del(&cache_key).await?;
        Ok(())
    }

    /// 清除完整配置缓存（full_settings）
    async fn clear_full_settings_cache(&self) -> Result<()> {
        let _ = self
            .cache_manager
            .delete(cache_keys::SYSTEM_CONFIG, "full_settings")
            .await;
        Ok(())
    }

    /// 发布配置更新事件（通过 Redis Pub/Sub）
    async fn publish_config_update(&self, config_key: &str) -> Result<()> {
        let mut conn = self.redis.get_multiplexed_async_connection().await?;
        let message = serde_json::json!({
            "config_key": config_key,
            "timestamp": chrono::Utc::now().to_rfc3339(),
        });
        let _: std::result::Result<(), _> = conn
            .publish(REDIS_CHANNEL_CONFIG_UPDATES, message.to_string())
            .await;
        Ok(())
    }
}

fn config_value_to_string(value: serde_json::Value) -> Option<String> {
    match value {
        serde_json::Value::String(v) => Some(v),
        serde_json::Value::Number(v) => Some(v.to_string()),
        serde_json::Value::Bool(v) => Some(v.to_string()),
        serde_json::Value::Null => None,
        other => serde_json::to_string(&other).ok(),
    }
}
