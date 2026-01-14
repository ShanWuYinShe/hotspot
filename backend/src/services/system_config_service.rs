//! 系统配置服务
//!
//! 管理系统配置项，支持 Redis 缓存和配置变更订阅
//!
//! 新版设计：支持强类型聚合配置 (GlobalSettings) 和版本控制

use redis::AsyncCommands;
use std::collections::HashMap;
use std::sync::Arc;

use crate::{
    constants::{REDIS_CHANNEL_CONFIG_UPDATES, REDIS_KEY_CONFIG},
    dto::{
        request::{
            FetchGlobalSettingsRequest, UpdateContactRequest, UpdateDefaultsRequest,
            UpdateFeaturesRequest, UpdateLimitsRequest, UpdateUploadConfigRequest,
        },
        response::{
            AdminGlobalSettingsData, AppLimits, ContactInfo, ContentDefaults, FeatureSwitches,
            FetchSettingsData, GlobalSettings, UploadConfig,
        },
    },
    error::Result,
    repositories::system_config_repo::SystemConfigRepository,
};

#[derive(Clone)]
pub struct SystemConfigService {
    repo: SystemConfigRepository,
    redis: Arc<redis::Client>,
}

impl SystemConfigService {
    pub fn new(repo: SystemConfigRepository, redis: Arc<redis::Client>) -> Self {
        Self { repo, redis }
    }

    // ==================== 客户端拉取配置 (Client Fetch) ====================

    /// 拉取全局配置 (Client Side)
    /// 支持版本控制 (ETag-like) 和 上下文感知 (Context-aware)
    pub async fn fetch_global_settings(
        &self,
        req: FetchGlobalSettingsRequest,
    ) -> Result<FetchSettingsData> {
        // 1. 获取当前全局版本号
        //    我们约定 key="global.version" 存储当前配置的版本号 (hash 或 timestamp)
        //    每次更新任意配置时，都应更新此版本号
        let current_server_version = self
            .get_config_value("global.version")
            .await?
            .unwrap_or_else(|| "v1".to_string());

        // 2. 检查版本
        if let Some(client_version) = req.current_version {
            if client_version == current_server_version {
                return Ok(FetchSettingsData {
                    action: "no_change".to_string(),
                    config: None,
                    latest_version: current_server_version,
                });
            }
        }

        // 3. 拉取完整配置
        let settings = self.get_full_global_settings().await?;

        // 4. (可选) 根据 req.context 进行灰度/开关处理
        //    例如：如果 client context.os == "android" 且 version < "1.0.0"，禁用某些 feature
        //    这里暂时直接返回通用配置
        tracing::debug!(
            "Client context: os={}, version={}, device={:?}, lang={:?}",
            req.context.os,
            req.context.app_version,
            req.context.device_id,
            req.context.language
        );

        Ok(FetchSettingsData {
            action: "update".to_string(),
            config: Some(settings),
            latest_version: current_server_version,
        })
    }

    // ==================== 管理员配置管理 (Admin Management) ====================

    /// 获取完整配置 (Admin)
    pub async fn get_admin_global_settings(&self) -> Result<AdminGlobalSettingsData> {
        let settings = self.get_full_global_settings().await?;

        // 获取最后更新元数据 (取 global.version 的更新记录?)
        // 为了简单，这里暂且再查一次 global.version 的 updated_at
        // 实际场景可以优化为一次查询
        let version_config = self.repo.find_by_key("global.version").await?;
        let (updated_at, updated_by) = if let Some(c) = version_config {
            (c.updated_at.map(|t| t.to_rfc3339()).unwrap_or_default(), 0) // TODO: 记录 updated_by
        } else {
            (chrono::Utc::now().to_rfc3339(), 0)
        };

        Ok(AdminGlobalSettingsData {
            settings,
            updated_at,
            updated_by_user_id: updated_by,
        })
    }

    /// 更新上传配置
    pub async fn update_upload_config(
        &self,
        req: UpdateUploadConfigRequest,
        admin_id: i64,
    ) -> Result<GlobalSettings> {
        let mut updates = HashMap::new();
        if let Some(v) = req.image_max_size {
            updates.insert("upload.image_max_size", v.to_string());
        }
        if let Some(v) = req.video_max_size {
            updates.insert("upload.video_max_size", v.to_string());
        }
        if let Some(v) = req.audio_max_size {
            updates.insert("upload.audio_max_size", v.to_string());
        }
        if let Some(v) = req.file_max_size {
            updates.insert("upload.file_max_size", v.to_string());
        }
        if let Some(v) = req.allowed_image_types {
            updates.insert("upload.allowed_image_types", serde_json::to_string(&v)?);
        }
        if let Some(v) = req.allowed_video_types {
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
        if let Some(v) = req.default_location_name {
            updates.insert("defaults.default_location_name", v);
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
        if let Some(v) = req.max_group_members {
            updates.insert("limits.max_group_members", v.to_string());
        }
        if let Some(v) = req.max_moment_tags {
            updates.insert("limits.max_moment_tags", v.to_string());
        }
        if let Some(v) = req.max_moment_images {
            updates.insert("limits.max_moment_images", v.to_string());
        }
        if let Some(v) = req.moment_description_max_length {
            updates.insert("limits.moment_description_max_length", v.to_string());
        }
        if let Some(v) = req.comment_max_length {
            updates.insert("limits.comment_max_length", v.to_string());
        }
        if let Some(v) = req.search_radius_max {
            updates.insert("limits.search_radius_max", v.to_string());
        }

        self.batch_update_configs(updates, admin_id).await?;
        self.get_full_global_settings().await
    }

    // ==================== 内部辅助方法 ====================

    /// 批量更新配置并刷新版本号
    async fn batch_update_configs(
        &self,
        updates: HashMap<&str, String>,
        admin_id: i64,
    ) -> Result<()> {
        if updates.is_empty() {
            return Ok(());
        }

        let mut tx = self.repo.pool().begin().await?;

        for (key, value) in updates {
            // 获取旧值用于日志
            let old_val = self.repo.find_by_key(key).await?.map(|c| c.value);

            // Upsert (Create or Update)
            // 这里简化为：如果不存在则 create，存在则 update
            // 实际上 repo 可能没有 upsert，需手动判断
            if old_val.is_none() {
                self.repo
                    .create_tx(
                        &mut tx,
                        key.to_string(),
                        value.clone(),
                        None,
                        crate::types::enums::ConfigValueType::String
                            .as_str()
                            .to_string(),
                        "system".to_string(),
                    )
                    .await?;
            } else {
                self.repo.update_tx(&mut tx, key, value.clone()).await?;
            }

            // 记录日志
            self.repo
                .log_change_tx(&mut tx, key, old_val, value.clone(), admin_id)
                .await?;

            // 清除缓存
            self.clear_config_cache(key).await?;
        }

        // 更新全局版本号
        let new_version = uuid::Uuid::new_v4().to_string(); // 简单生成新版本号
        self.repo
            .update_tx(&mut tx, "global.version", new_version.clone())
            .await?;
        // 也可以记录 version 的变更日志...

        tx.commit().await?;

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
            let _: () = conn.set_ex(&cache_key, &c.value, 3600).await?; // 1 hour TTL
        }

        Ok(config.map(|c| c.value))
    }

    /// 组装完整 GlobalSettings
    /// 这里可以使用 Multi-Get 优化 DB 查询，或者查出所有配置在内存 Map
    async fn get_full_global_settings(&self) -> Result<GlobalSettings> {
        let cache_key = format!("{}:full_settings", REDIS_KEY_CONFIG);
        let mut conn = self.redis.get_multiplexed_async_connection().await?;

        // 1. 尝试从缓存获取
        if let Ok(Some(json)) = conn.get::<_, Option<String>>(&cache_key).await {
            if let Ok(settings) = serde_json::from_str(&json) {
                return Ok(settings);
            }
        }

        // 2. 查询所有配置
        // 假设 list(None, 1000, None) 能拉取所有需要的配置
        let (configs, _) = self.repo.list(None, 1000, None).await?;
        let map: HashMap<String, String> = configs.into_iter().map(|c| (c.key, c.value)).collect();

        // 辅助闭包：获取值或默认值
        let get_str =
            |k: &str, default: &str| map.get(k).cloned().unwrap_or_else(|| default.to_string());
        let get_int =
            |k: &str, default: i64| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_i32 =
            |k: &str, default: i32| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_bool =
            |k: &str, default: bool| map.get(k).and_then(|v| v.parse().ok()).unwrap_or(default);
        let get_json_vec = |k: &str| -> Vec<String> {
            map.get(k)
                .and_then(|v| serde_json::from_str(v).ok())
                .unwrap_or_default()
        };
        let get_json_settings = |k: &str| -> crate::dto::response::GroupSettings {
            map.get(k)
                .and_then(|v| serde_json::from_str(v).ok())
                .unwrap_or(crate::dto::response::GroupSettings {
                    // Default empty
                    join_method: Some("open".to_string()),
                    is_searchable: Some(true),
                    max_members: Some(500),
                    longitude: None,
                    latitude: None,
                    location_name: None,
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
            default_location_name: get_str("defaults.default_location_name", "Unknown Location"),
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
            search_radius_max: get_i32("limits.search_radius_max", 50000),
        };

        let version = get_str("global.version", "v1");

        let settings = GlobalSettings {
            upload,
            defaults,
            features,
            contact,
            limits,
            version,
        };

        // 3. 写入缓存
        if let Ok(json) = serde_json::to_string(&settings) {
            let _: () = conn.set_ex(&cache_key, json, 3600).await?;
        }

        Ok(settings)
    }

    /// 清除配置缓存
    async fn clear_config_cache(&self, config_key: &str) -> Result<()> {
        let mut conn = self.redis.get_multiplexed_async_connection().await?;

        // 1. 清除单个 key 缓存
        let cache_key = format!("{}:{}", REDIS_KEY_CONFIG, config_key);
        let _: () = conn.del(&cache_key).await?;

        // 2. 清除 full settings 缓存
        let full_key = format!("{}:full_settings", REDIS_KEY_CONFIG);
        let _: () = conn.del(&full_key).await?;

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
