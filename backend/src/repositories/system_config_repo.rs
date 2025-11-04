use sqlx::PgPool;

use crate::{
    error::{AppError, Result},
    models::{
        config_change_log::{ConfigChangeLog, CreateConfigChangeLogRequest},
        system_config::{CreateSystemConfigRequest, SystemConfig, UpdateSystemConfigRequest},
    },
};

pub struct SystemConfigRepository {
    pool: PgPool,
}

impl SystemConfigRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 获取配置项
    pub async fn get(&self, config_key: &str) -> Result<Option<SystemConfig>> {
        let config = sqlx::query_as!(
            SystemConfig,
            r#"
            SELECT id, config_key, config_value, value_type, description, created_at, updated_at
            FROM system_configs
            WHERE config_key = $1
            "#,
            config_key
        )
        .fetch_optional(&self.pool)
        .await?;

        Ok(config)
    }

    /// 列出所有配置项
    pub async fn list(&self) -> Result<Vec<SystemConfig>> {
        let configs = sqlx::query_as!(
            SystemConfig,
            r#"
            SELECT id, config_key, config_value, value_type, description, created_at, updated_at
            FROM system_configs
            ORDER BY config_key
            "#
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(configs)
    }

    /// 创建配置项
    pub async fn create(&self, req: &CreateSystemConfigRequest) -> Result<SystemConfig> {
        let config = sqlx::query_as!(
            SystemConfig,
            r#"
            INSERT INTO system_configs (config_key, config_value, value_type, description)
            VALUES ($1, $2, $3, $4)
            RETURNING id, config_key, config_value, value_type, description, created_at, updated_at
            "#,
            req.config_key,
            req.config_value,
            req.value_type,
            req.description
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(config)
    }

    /// 更新配置项
    pub async fn update(
        &self,
        config_key: &str,
        req: &UpdateSystemConfigRequest,
    ) -> Result<SystemConfig> {
        let config = sqlx::query_as!(
            SystemConfig,
            r#"
            UPDATE system_configs
            SET config_value = $2, updated_at = NOW()
            WHERE config_key = $1
            RETURNING id, config_key, config_value, value_type, description, created_at, updated_at
            "#,
            config_key,
            req.config_value
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(config)
    }

    /// 删除配置项
    pub async fn delete(&self, config_key: &str) -> Result<()> {
        sqlx::query!(
            r#"
            DELETE FROM system_configs
            WHERE config_key = $1
            "#,
            config_key
        )
        .execute(&self.pool)
        .await?;

        Ok(())
    }

    /// 记录配置变更日志
    pub async fn log_change(&self, req: &CreateConfigChangeLogRequest) -> Result<ConfigChangeLog> {
        let log = sqlx::query_as!(
            ConfigChangeLog,
            r#"
            INSERT INTO config_change_logs (config_key, old_value, new_value, changed_by, ip_address, user_agent)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id, config_key, old_value, new_value, changed_by, changed_at, ip_address, user_agent
            "#,
            req.config_key,
            req.old_value,
            req.new_value,
            req.changed_by,
            req.ip_address,
            req.user_agent
        )
        .fetch_one(&self.pool)
        .await?;

        Ok(log)
    }

    /// 获取配置变更历史
    pub async fn get_change_history(
        &self,
        config_key: &str,
        limit: i64,
    ) -> Result<Vec<ConfigChangeLog>> {
        let logs = sqlx::query_as!(
            ConfigChangeLog,
            r#"
            SELECT id, config_key, old_value, new_value, changed_by, changed_at, ip_address, user_agent
            FROM config_change_logs
            WHERE config_key = $1
            ORDER BY changed_at DESC
            LIMIT $2
            "#,
            config_key,
            limit
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(logs)
    }

    /// 获取所有变更历史（带分页）
    pub async fn list_all_changes(&self, limit: i64, offset: i64) -> Result<Vec<ConfigChangeLog>> {
        let logs = sqlx::query_as!(
            ConfigChangeLog,
            r#"
            SELECT id, config_key, old_value, new_value, changed_by, changed_at, ip_address, user_agent
            FROM config_change_logs
            ORDER BY changed_at DESC
            LIMIT $1 OFFSET $2
            "#,
            limit,
            offset
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(logs)
    }
}











