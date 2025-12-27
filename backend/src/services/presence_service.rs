//! 用户在线状态服务
//!
//! 使用 Redis 管理用户实时在线状态和位置

use chrono::Utc;
use redis::aio::ConnectionManager as RedisConnectionManager;
use redis::AsyncCommands;

use crate::constants::{REDIS_KEY_PRESENCE_LOCATION, REDIS_KEY_PRESENCE_ONLINE};
use crate::error::{AppError, Result};

#[derive(Clone)]
pub struct PresenceService {
    redis: RedisConnectionManager,
}

impl PresenceService {
    pub fn new(redis: RedisConnectionManager) -> Self {
        Self { redis }
    }

    /// 标记用户上线
    pub async fn mark_user_online(&self, user_id: i64) -> Result<()> {
        let mut conn = self.redis.clone();
        let timestamp = Utc::now().timestamp();

        let _: () = conn
            .zadd(REDIS_KEY_PRESENCE_ONLINE, user_id, timestamp)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        tracing::debug!("用户上线: user_id={}", user_id);
        Ok(())
    }

    /// 标记用户离线
    pub async fn mark_user_offline(&self, user_id: i64) -> Result<()> {
        let mut conn = self.redis.clone();

        let _: () = conn
            .zrem(REDIS_KEY_PRESENCE_ONLINE, user_id)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        // 删除位置信息
        let location_key = format!("{}{}", REDIS_KEY_PRESENCE_LOCATION, user_id);
        let _: () = conn
            .del(&location_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        tracing::debug!("用户离线: user_id={}", user_id);
        Ok(())
    }
}
