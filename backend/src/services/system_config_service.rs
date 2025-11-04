use futures_util::StreamExt;
use redis::AsyncCommands;
use sqlx::PgPool;
use std::sync::Arc;

use crate::{
    constants::{REDIS_CHANNEL_CONFIG_UPDATES, REDIS_KEY_CONFIG, REDIS_TTL_CONFIG},
    error::{AppError, Result},
    models::{
        config_change_log::{ConfigChangeLogResponse, CreateConfigChangeLogRequest},
        system_config::{
            CreateSystemConfigRequest, SystemConfig, SystemConfigResponse,
            UpdateSystemConfigRequest,
        },
    },
    repositories::system_config_repo::SystemConfigRepository,
};

pub struct SystemConfigService {
    repo: SystemConfigRepository,
    redis: Arc<redis::Client>,
}

impl SystemConfigService {
    pub fn new(pool: PgPool, redis: Arc<redis::Client>) -> Self {
        Self {
            repo: SystemConfigRepository::new(pool),
            redis,
        }
    }

    /// 获取配置项（按优先级：Redis 缓存 > 数据库 > 环境变量 > 默认值）
    pub async fn get_config(&self, config_key: &str) -> Result<Option<String>> {
        // 1. 尝试从 Redis 缓存读取
        let cache_key = format!("{}:{}", REDIS_KEY_CONFIG, config_key);
        let mut conn = self.redis.get_async_connection().await?;

        if let Ok(Some(cached_value)) = conn.get::<_, Option<String>>(&cache_key).await {
            return Ok(Some(cached_value));
        }

        // 2. 从数据库读取
        if let Some(config) = self.repo.get(config_key).await? {
            // 写入 Redis 缓存
            let _ = conn
                .set_ex(&cache_key, &config.config_value, REDIS_TTL_CONFIG as u64)
                .await;

            return Ok(Some(config.config_value));
        }

        // 3. 尝试从环境变量读取
        if let Ok(env_value) = std::env::var(config_key.to_uppercase()) {
            return Ok(Some(env_value));
        }

        // 4. 返回 None，调用方应使用默认值
        Ok(None)
    }

    /// 获取配置项（带类型转换）
    pub async fn get_config_as<T>(&self, config_key: &str, default: T) -> Result<T>
    where
        T: std::str::FromStr,
    {
        if let Some(value) = self.get_config(config_key).await? {
            value
                .parse::<T>()
                .map_err(|_| AppError::BadRequest("配置值类型转换失败".to_string()))
        } else {
            Ok(default)
        }
    }

    /// 列出所有配置项
    pub async fn list_configs(&self) -> Result<Vec<SystemConfigResponse>> {
        let configs = self.repo.list().await?;
        Ok(configs.into_iter().map(Into::into).collect())
    }

    /// 创建配置项
    pub async fn create_config(
        &self,
        req: CreateSystemConfigRequest,
        admin_id: i64,
        ip_address: Option<String>,
        user_agent: Option<String>,
    ) -> Result<SystemConfigResponse> {
        let config = self.repo.create(&req).await?;

        // 记录变更日志
        let log_req = CreateConfigChangeLogRequest {
            config_key: req.config_key.clone(),
            old_value: None,
            new_value: req.config_value.clone(),
            changed_by: admin_id,
            ip_address,
            user_agent,
        };
        self.repo.log_change(&log_req).await?;

        // 发布配置更新事件（通知其他服务实例清除缓存）
        self.publish_config_update(&req.config_key).await?;

        Ok(config.into())
    }

    /// 更新配置项
    pub async fn update_config(
        &self,
        config_key: &str,
        req: UpdateSystemConfigRequest,
        admin_id: i64,
        ip_address: Option<String>,
        user_agent: Option<String>,
    ) -> Result<SystemConfigResponse> {
        // 获取旧值
        let old_config = self
            .repo
            .get(config_key)
            .await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;

        // 更新配置
        let config = self.repo.update(config_key, &req).await?;

        // 记录变更日志
        let log_req = CreateConfigChangeLogRequest {
            config_key: config_key.to_string(),
            old_value: Some(old_config.config_value),
            new_value: req.config_value.clone(),
            changed_by: admin_id,
            ip_address,
            user_agent,
        };
        self.repo.log_change(&log_req).await?;

        // 清除 Redis 缓存
        self.clear_config_cache(config_key).await?;

        // 发布配置更新事件
        self.publish_config_update(config_key).await?;

        Ok(config.into())
    }

    /// 删除配置项
    pub async fn delete_config(
        &self,
        config_key: &str,
        admin_id: i64,
        ip_address: Option<String>,
        user_agent: Option<String>,
    ) -> Result<()> {
        // 获取旧值
        let old_config = self
            .repo
            .get(config_key)
            .await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;

        // 删除配置
        self.repo.delete(config_key).await?;

        // 记录变更日志
        let log_req = CreateConfigChangeLogRequest {
            config_key: config_key.to_string(),
            old_value: Some(old_config.config_value),
            new_value: "DELETED".to_string(),
            changed_by: admin_id,
            ip_address,
            user_agent,
        };
        self.repo.log_change(&log_req).await?;

        // 清除 Redis 缓存
        self.clear_config_cache(config_key).await?;

        // 发布配置更新事件
        self.publish_config_update(config_key).await?;

        Ok(())
    }

    /// 获取配置变更历史
    pub async fn get_config_history(
        &self,
        config_key: &str,
        limit: i64,
    ) -> Result<Vec<ConfigChangeLogResponse>> {
        let logs = self.repo.get_change_history(config_key, limit).await?;
        Ok(logs.into_iter().map(Into::into).collect())
    }

    /// 获取所有变更历史
    pub async fn list_all_changes(
        &self,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<ConfigChangeLogResponse>> {
        let logs = self.repo.list_all_changes(limit, offset).await?;
        Ok(logs.into_iter().map(Into::into).collect())
    }

    /// 清除配置缓存
    async fn clear_config_cache(&self, config_key: &str) -> Result<()> {
        let cache_key = format!("{}:{}", REDIS_KEY_CONFIG, config_key);
        let mut conn = self.redis.get_async_connection().await?;
        let _ = conn.del(&cache_key).await;
        Ok(())
    }

    /// 发布配置更新事件（通过 Redis Pub/Sub）
    async fn publish_config_update(&self, config_key: &str) -> Result<()> {
        let mut conn = self.redis.get_async_connection().await?;
        let message = serde_json::json!({
            "config_key": config_key,
            "timestamp": chrono::Utc::now().to_rfc3339(),
        });
        let _ = conn
            .publish(REDIS_CHANNEL_CONFIG_UPDATES, message.to_string())
            .await;
        Ok(())
    }

    /// 订阅配置更新事件（后台任务）
    pub async fn subscribe_config_updates(&self) -> Result<()> {
        let mut pubsub = self.redis.get_async_connection().await?.into_pubsub();
        pubsub.subscribe(REDIS_CHANNEL_CONFIG_UPDATES).await?;

        let mut stream = pubsub.on_message();
        while let Some(msg) = stream.next().await {
            if let Ok(payload) = msg.get_payload::<String>() {
                if let Ok(data) = serde_json::from_str::<serde_json::Value>(&payload) {
                    if let Some(config_key) = data.get("config_key").and_then(|v| v.as_str()) {
                        tracing::info!("收到配置更新通知: {}", config_key);
                        // 清除本地缓存（如果有的话）
                        let _ = self.clear_config_cache(config_key).await;
                    }
                }
            }
        }

        Ok(())
    }
}

