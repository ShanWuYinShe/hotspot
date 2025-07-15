//! 健康检查仓库实现

use sqlx::PgPool;
use tracing::error;

use crate::repositories::{Repository, RepositoryResult};

/// PostgreSQL健康检查仓库实现
#[derive(Debug)]
pub struct PgHealthRepository {
    pg_pool: PgPool,
    redis_manager: Option<redis::Client>,
}

impl PgHealthRepository {
    /// 创建新的PostgreSQL健康检查仓库
    pub fn new(pg_pool: PgPool, redis_manager: Option<redis::Client>) -> Self {
        Self { pg_pool, redis_manager }
    }

    /// 检查数据库连接
    pub async fn check_database(&self) -> RepositoryResult<bool> {
        match sqlx::query!("SELECT 1 as one").fetch_one(&self.pg_pool).await {
            Ok(_) => Ok(true),
            Err(e) => {
                error!("数据库健康检查失败: {:?}", e);
                Ok(false)
            }
        }
    }
    
    /// 检查Redis连接
    pub async fn check_redis(&self) -> RepositoryResult<bool> {
        if let Some(redis_manager) = &self.redis_manager {
            let mut conn = match redis_manager.get_multiplexed_async_connection().await {
                Ok(conn) => conn,
                Err(e) => {
                    error!("获取Redis连接失败: {:?}", e);
                    return Ok(false);
                }
            };
            
            match redis::cmd("PING").query_async::<String>(&mut conn).await {
                Ok(_) => Ok(true),
                Err(e) => {
                    error!("Redis健康检查失败: {:?}", e);
                    Ok(false)
                }
            }
        } else {
            // Redis未配置，视为禁用
            Ok(false)
        }
    }
}

impl Repository for PgHealthRepository {
    fn name(&self) -> &str {
        "PostgreSQL健康检查仓库"
    }
}
