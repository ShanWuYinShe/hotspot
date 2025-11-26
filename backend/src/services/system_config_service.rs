use futures_util::StreamExt;
use redis::AsyncCommands;
use sqlx::PgPool;
use std::sync::Arc;
use uuid::Uuid;

use crate::{
    constants::{REDIS_CHANNEL_CONFIG_UPDATES, REDIS_KEY_CONFIG, REDIS_TTL_CONFIG},
    dto::{
        request::{
            CreateConfigRequest, UpdateConfigRequest, GetConfigListRequest, 
            GetConfigHistoryRequest, ListAllConfigChangesRequest
        },
        response::{
            ConfigItem, ConfigHistoryItem, CreateConfigData, 
            PaginatedData, PaginationMeta
        },
    },
    error::{AppError, Result},
    repositories::system_config_repo::{SystemConfigRepository, SystemConfigRow, ConfigChangeLogRow},
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
        let mut conn = self.redis.get_multiplexed_async_connection().await?;

        if let Ok(Some(cached_value)) = conn.get::<_, Option<String>>(&cache_key).await {
            return Ok(Some(cached_value));
        }

        // 2. 从数据库读取
        if let Some(config) = self.repo.find_by_key(config_key).await? {
            // 写入 Redis 缓存
            let _: std::result::Result<(), _> = conn
                .set_ex(&cache_key, &config.value, REDIS_TTL_CONFIG as u64)
                .await;

            return Ok(Some(config.value));
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

    /// 获取单个配置（管理员）
    pub async fn get_single_config(&self, key: &str) -> Result<ConfigItem> {
        let config = self.repo.find_by_key(key).await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;
        Ok(Self::to_config_item(config))
    }

    /// 解析 cursor 字符串为 UUID
    fn parse_cursor(cursor: Option<String>) -> Result<Option<Uuid>> {
        match cursor {
            Some(c) => {
                let uuid = Uuid::parse_str(&c)
                    .map_err(|_| AppError::BadRequest("无效的 cursor 格式".to_string()))?;
                Ok(Some(uuid))
            }
            None => Ok(None),
        }
    }

    /// 列出所有配置项
    pub async fn get_list(&self, req: GetConfigListRequest) -> Result<PaginatedData<ConfigItem>> {
        let limit = req.limit.unwrap_or(20).min(100) as i32;
        let cursor = Self::parse_cursor(req.cursor)?;

        let configs = self.repo.list(req.category, limit + 1, cursor).await?;

        let has_more = configs.len() > limit as usize;
        let configs_to_return: Vec<SystemConfigRow> = configs.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的cursor_id
        let next_cursor = if has_more {
            configs_to_return.last().map(|c| c.cursor_id.to_string())
        } else {
            None
        };

        let items: Vec<ConfigItem> = configs_to_return
            .into_iter()
            .map(Self::to_config_item)
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 创建配置项
    pub async fn create_config(
        &self,
        req: CreateConfigRequest,
        admin_id: i64,
    ) -> Result<CreateConfigData> {
        // 验证key唯一性
        if self.repo.find_by_key(&req.key).await?.is_some() {
            return Err(AppError::Conflict("配置项已存在".to_string()));
        }

        let config = self.repo.create(
            req.key.clone(),
            req.value.clone(),
            Some(req.description.clone()),
            req.r#type.clone(),
            req.category.clone(),
        ).await?;

        // 记录变更日志
        self.repo.log_change(
            &req.key,
            None,
            req.value.clone(),
            admin_id
        ).await?;

        // 发布配置更新事件
        self.publish_config_update(&req.key).await?;

        Ok(CreateConfigData {
            key: config.key,
        })
    }

    /// 更新配置项
    pub async fn update_config(
        &self,
        key: &str,
        req: UpdateConfigRequest,
        admin_id: i64,
    ) -> Result<()> {
        // 获取旧值
        let old_config = self
            .repo
            .find_by_key(key)
            .await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;

        // 更新配置
        self.repo.update(key, req.value.clone()).await?;

        // 记录变更日志
        self.repo.log_change(
            key,
            Some(old_config.value.clone()),
            req.value.clone(),
            admin_id
        ).await?;

        // 清除 Redis 缓存
        self.clear_config_cache(key).await?;

        // 发布配置更新事件
        self.publish_config_update(key).await?;

        Ok(())
    }

    /// 删除配置项
    pub async fn delete_config(
        &self,
        key: &str,
        admin_id: i64,
    ) -> Result<()> {
        // 获取旧值
        let old_config = self
            .repo
            .find_by_key(key)
            .await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;

        // 删除配置
        self.repo.delete(key).await?;

        // 记录变更日志
        self.repo.log_change(
            key,
            Some(old_config.value),
            "DELETED".to_string(),
            admin_id
        ).await?;

        // 清除 Redis 缓存
        self.clear_config_cache(key).await?;

        // 发布配置更新事件
        self.publish_config_update(key).await?;

        Ok(())
    }

    /// 获取配置变更历史
    pub async fn get_config_history(
        &self,
        config_key: &str,
        req: GetConfigHistoryRequest,
    ) -> Result<PaginatedData<ConfigHistoryItem>> {
        let limit = req.limit.unwrap_or(20).min(100) as i32;
        let cursor = Self::parse_cursor(req.cursor)?;

        let logs = self.repo.get_change_history(config_key, limit + 1, cursor).await?;

        let has_more = logs.len() > limit as usize;
        let logs_to_return: Vec<ConfigChangeLogRow> = logs.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的cursor_id
        let next_cursor = if has_more {
            logs_to_return.last().map(|l| l.cursor_id.to_string())
        } else {
            None
        };

        let items: Vec<ConfigHistoryItem> = logs_to_return
            .into_iter()
            .map(Self::to_history_item)
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 获取所有变更历史
    pub async fn list_all_changes(
        &self,
        req: ListAllConfigChangesRequest,
    ) -> Result<PaginatedData<ConfigHistoryItem>> {
        let limit = req.limit.unwrap_or(20).min(100) as i32;
        let cursor = Self::parse_cursor(req.cursor)?;

        let changed_by = match req.changed_by.as_deref() {
            Some(s) => Some(s.parse::<i64>().map_err(|_| AppError::BadRequest("无效的操作者ID".to_string()))?),
            None => None,
        };

        let logs = self.repo.list_all_changes(
            req.start_time.as_deref(),
            req.end_time.as_deref(),
            changed_by,
            limit + 1,
            cursor
        ).await?;

        let has_more = logs.len() > limit as usize;
        let logs_to_return: Vec<ConfigChangeLogRow> = logs.into_iter().take(limit as usize).collect();
        
        // next_cursor是最后一条记录的cursor_id
        let next_cursor = if has_more {
            logs_to_return.last().map(|l| l.cursor_id.to_string())
        } else {
            None
        };

        let items: Vec<ConfigHistoryItem> = logs_to_return
            .into_iter()
            .map(Self::to_history_item)
            .collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor,
            },
        })
    }

    /// 清除配置缓存
    async fn clear_config_cache(&self, config_key: &str) -> Result<()> {
        let cache_key = format!("{}:{}", REDIS_KEY_CONFIG, config_key);
        let mut conn = self.redis.get_multiplexed_async_connection().await?;
        let _: std::result::Result<(), _> = conn.del(&cache_key).await;
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

    /// 订阅配置更新事件（后台任务）
    pub async fn subscribe_config_updates(&self) -> Result<()> {
        let mut pubsub = self.redis.get_async_pubsub().await?;
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

    fn to_config_item(config: SystemConfigRow) -> ConfigItem {
        ConfigItem {
            key: config.key,
            value: config.value,
            r#type: config.config_type.unwrap_or_default(),
            category: config.category.unwrap_or_default(),
            description: config.description.unwrap_or_default(),
            updated_at: config.updated_at.map_or("".to_string(), |t| t.to_rfc3339()),
        }
    }

    fn to_history_item(log: ConfigChangeLogRow) -> ConfigHistoryItem {
        ConfigHistoryItem {
            id: log.id.to_string(),
            key: log.key,
            old_value: log.old_value.unwrap_or_default(),
            new_value: log.new_value,
            changed_by: log.changed_by.to_string(),
            changed_at: log.changed_at.to_rfc3339(),
        }
    }
}
