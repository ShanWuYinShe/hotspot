//! 系统配置服务
//!
//! 管理系统配置项，支持 Redis 缓存和配置变更订阅

use redis::AsyncCommands;
use sqlx::PgPool;
use std::sync::Arc;

use crate::{
    constants::{REDIS_CHANNEL_CONFIG_UPDATES, REDIS_KEY_CONFIG},
    dto::{
        request::{
            ConfigChangesRequest, ConfigHistoryRequest, ConfigListRequest, CreateConfigRequest,
            UpdateConfigRequest,
        },
        response::{
            ConfigHistoryItem, ConfigItem, CreateConfigData, PaginatedData, PaginationMeta,
        },
    },
    error::{AppError, Result},
    repositories::system_config_repo::{
        ConfigChangeLogRow, SystemConfigRepository, SystemConfigRow,
    },
    utils::validation::Validator,
};

#[derive(Clone)]
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

    /// 获取单个配置（管理员）
    pub async fn get_single_config(&self, key: &str) -> Result<ConfigItem> {
        let config = self
            .repo
            .find_by_key(key)
            .await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;
        Ok(Self::to_config_item(config))
    }

    /// 列出所有配置项
    pub async fn get_list(&self, req: ConfigListRequest) -> Result<PaginatedData<ConfigItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (configs, next_cursor) = self.repo.list(req.category, limit as i32, cursor).await?;

        let items: Vec<ConfigItem> = configs.into_iter().map(Self::to_config_item).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
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

        // DB 强一致：create + log_change 必须在同一事务内
        let mut tx = self.repo.pool().begin().await?;

        let config = self
            .repo
            .create_tx(
                &mut tx,
                req.key.clone(),
                req.value.clone(),
                Some(req.description.clone()),
                req.r#type.clone(),
                req.category.clone(),
            )
            .await?;

        self.repo
            .log_change_tx(&mut tx, &req.key, None, req.value.clone(), admin_id)
            .await?;

        tx.commit().await?;

        // 发布配置更新事件
        self.publish_config_update(&req.key).await?;

        Ok(CreateConfigData { key: config.key })
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

        // DB 强一致：update + log_change 必须在同一事务内
        let mut tx = self.repo.pool().begin().await?;

        self.repo.update_tx(&mut tx, key, req.value.clone()).await?;
        self.repo
            .log_change_tx(
                &mut tx,
                key,
                Some(old_config.value.clone()),
                req.value.clone(),
                admin_id,
            )
            .await?;

        tx.commit().await?;

        // 清除 Redis 缓存
        self.clear_config_cache(key).await?;

        // 发布配置更新事件
        self.publish_config_update(key).await?;

        Ok(())
    }

    /// 删除配置项
    pub async fn delete_config(&self, key: &str, admin_id: i64) -> Result<()> {
        // 获取旧值
        let old_config = self
            .repo
            .find_by_key(key)
            .await?
            .ok_or_else(|| AppError::NotFound("配置项不存在".to_string()))?;

        // DB 强一致：delete + log_change 必须在同一事务内
        let mut tx = self.repo.pool().begin().await?;

        self.repo.delete_tx(&mut tx, key).await?;
        self.repo
            .log_change_tx(
                &mut tx,
                key,
                Some(old_config.value),
                "DELETED".to_string(),
                admin_id,
            )
            .await?;

        tx.commit().await?;

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
        req: ConfigHistoryRequest,
    ) -> Result<PaginatedData<ConfigHistoryItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let (logs, next_cursor) = self
            .repo
            .get_change_history(config_key.to_string(), limit as i32, cursor)
            .await?;

        let items: Vec<ConfigHistoryItem> = logs.into_iter().map(Self::to_history_item).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
                next_cursor,
            },
        })
    }

    /// 获取所有变更历史
    pub async fn get_changes(
        &self,
        req: ConfigChangesRequest,
    ) -> Result<PaginatedData<ConfigHistoryItem>> {
        let limit = Validator::validate_limit(req.limit, 100)?;
        let cursor = Validator::parse_cursor(req.cursor.as_deref())?;

        let changed_by = req.changed_by;
        let start_time = Validator::parse_datetime(req.start_time.as_deref())?;
        let end_time = Validator::parse_datetime(req.end_time.as_deref())?;

        let (logs, next_cursor) = self
            .repo
            .list_all_changes(start_time, end_time, changed_by, limit as i32, cursor)
            .await?;

        let items: Vec<ConfigHistoryItem> = logs.into_iter().map(Self::to_history_item).collect();

        Ok(PaginatedData {
            items,
            pagination: PaginationMeta {
                has_more: next_cursor.is_some(),
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
            changed_by: log.changed_by,
            changed_at: log.changed_at.to_rfc3339(),
        }
    }
}
