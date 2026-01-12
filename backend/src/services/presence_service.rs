//! 用户在线状态服务
//!
//! 使用 Redis 管理用户实时在线状态和位置
//! 提供带有隐私控制的状态查询

use chrono::Utc;
use redis::aio::ConnectionManager as RedisConnectionManager;
use redis::AsyncCommands;
use sqlx::PgPool;
use std::collections::HashMap;

use crate::constants::{REDIS_KEY_PRESENCE_LOCATION, REDIS_KEY_PRESENCE_ONLINE};
use crate::error::{AppError, Result};
use crate::repositories::{
    user_online_status_repo::UserOnlineStatusRepository, user_settings_repo::UserSettingsRepository,
};

#[derive(Clone)]
pub struct PresenceService {
    redis: RedisConnectionManager,
    online_status_repo: UserOnlineStatusRepository,
    user_settings_repo: UserSettingsRepository,
}

impl PresenceService {
    pub fn new(redis: RedisConnectionManager, pool: PgPool) -> Self {
        Self {
            redis,
            online_status_repo: UserOnlineStatusRepository::new(pool.clone()),
            user_settings_repo: UserSettingsRepository::new(pool.clone()),
        }
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

    /// 更新用户位置缓存
    pub async fn update_location_cache(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
    ) -> Result<()> {
        let mut conn = self.redis.clone();
        let key = format!("{}{}", REDIS_KEY_PRESENCE_LOCATION, user_id);
        let value = format!("{},{}", latitude, longitude);

        // 设置位置缓存，过期时间 1 小时 (或者更短，取决于实时性要求)
        let _: () = conn
            .set_ex(&key, value, 3600)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;

        Ok(())
    }

    /// 获取用户当前位置 (Redis -> DB fallback)
    /// 返回 (latitude, longitude)
    pub async fn get_user_location(&self, user_id: i64) -> Result<Option<(f64, f64)>> {
        // 1. 尝试从 Redis 获取
        let mut conn = self.redis.clone();
        let key = format!("{}{}", REDIS_KEY_PRESENCE_LOCATION, user_id);
        let cached: Option<String> = conn.get(&key).await.ok();

        if let Some(s) = cached {
            let parts: Vec<&str> = s.split(',').collect();
            if parts.len() == 2 {
                if let (Ok(lat), Ok(lng)) = (parts[0].parse::<f64>(), parts[1].parse::<f64>()) {
                    return Ok(Some((lat, lng)));
                }
            }
        }

        // 2. Redis 未命中，从 DB 获取 (UserRepo 需要在这里被调用，或者我们直接查表？)
        // PresenceService 并没有直接持有 UserRepository，但是持有 UserSettingsRepository 和 UserOnlineStatusRepository
        // 这里为了避免循环依赖 (UserRepo 依赖 PresenceService?), 我们可能需要直接查库
        // 或者让调用方处理 Fallback。
        // 为了方便 FeedService 调用，我们在 PresenceService 内部直接查库比较好。
        // 但是 PresenceService 没有 UserRepo。
        // 我们可以直接用 sqlx 查询，或者添加 UserRepo 字段。
        // 添加 UserRepo 可能会导致循环引用 (UserRepo -> ... -> PresenceService -> UserRepo?)
        // UserRepo struct 本身没有引用 PresenceService。是 UserService 引用了两者。
        // 所以给 PresenceService 添加 UserRepo 是安全的。
        
        // 暂时直接查库，避免修改 struct 依赖
        let row = sqlx::query!(
            r#"
            SELECT current_latitude, current_longitude
            FROM users
            WHERE user_id = $1 AND is_deleted = FALSE
            "#,
            user_id
        )
        .fetch_optional(self.online_status_repo.pool()) // Use existing pool from online_status_repo
        .await
        .map_err(AppError::Database)?;

        if let Some(r) = row {
            if let (Some(lat), Some(lng)) = (r.current_latitude, r.current_longitude) {
                // 回填 Redis
                let _ = self.update_location_cache(user_id, lat, lng).await;
                return Ok(Some((lat, lng)));
            }
        }

        Ok(None)
    }

    /// 获取用户在线状态（受隐私设置控制，返回 Option 值）
    pub async fn get_online_status_with_privacy(
        &self,
        user_id: i64,
    ) -> (Option<bool>, Option<String>) {
        // 获取用户隐私设置
        let settings = self
            .user_settings_repo
            .find_by_user_id(user_id)
            .await
            .ok()
            .flatten();

        // 默认值：true（如果没有设置记录）
        let online_visible = settings.as_ref().map(|s| s.online_visible).unwrap_or(true);
        let last_online_visible = settings
            .as_ref()
            .map(|s| s.last_online_visible)
            .unwrap_or(true);

        // 如果用户不允许显示在线状态，返回 None
        if !online_visible {
            return (None, None);
        }

        // 获取在线状态
        let (is_online, last_online) = self.get_online_status(user_id).await;

        (
            Some(is_online),
            // 如果用户不允许显示最后在线时间，不返回该字段
            if last_online_visible {
                last_online
            } else {
                None
            },
        )
    }

    /// 批量获取用户在线状态（受隐私设置控制）
    pub async fn batch_get_online_status_with_privacy(
        &self,
        user_ids: &[i64],
    ) -> HashMap<i64, (Option<bool>, Option<String>)> {
        if user_ids.is_empty() {
            return HashMap::new();
        }

        // 1) 批量拿隐私设置（避免 N+1）
        let mut settings_map: HashMap<i64, crate::repositories::user_settings_repo::UserSettings> =
            HashMap::new();
        if let Ok(settings) = self.user_settings_repo.get_batch(user_ids).await {
            settings_map = settings.into_iter().map(|s| (s.user_id, s)).collect();
        }

        // 2) 批量拿在线状态（Redis pipeline + DB ANY）
        let online_status_map = self.batch_get_online_status(user_ids).await;

        // 3) 预生成需要 last_online 的用户列表（离线 + 允许显示）
        let mut need_last_online_ids: Vec<i64> = Vec::new();
        for &user_id in user_ids {
            let settings = settings_map.get(&user_id);
            let online_visible = settings.map(|s| s.online_visible).unwrap_or(true);
            let last_online_visible = settings.map(|s| s.last_online_visible).unwrap_or(true);
            if !online_visible {
                continue;
            }
            let is_online = online_status_map.get(&user_id).copied().unwrap_or(false);
            if !is_online && last_online_visible {
                need_last_online_ids.push(user_id);
            }
        }

        // 4) Redis 批量 MGET last_online
        let mut last_online_map: HashMap<i64, Option<String>> = HashMap::new();
        if !need_last_online_ids.is_empty() {
            let keys: Vec<String> = need_last_online_ids
                .iter()
                .map(|id| format!("presence:last_online:{}", id))
                .collect();

            let mut conn = self.redis.clone();
            let values: redis::RedisResult<Vec<Option<String>>> =
                redis::cmd("MGET").arg(keys).query_async(&mut conn).await;

            if let Ok(vs) = values {
                for (idx, v) in vs.into_iter().enumerate() {
                    last_online_map.insert(need_last_online_ids[idx], v);
                }
            }
        }

        // 5) DB 批量补齐 last_seen_at（仅对 redis miss 的用户）
        let mut miss_ids: Vec<i64> = Vec::new();
        for &user_id in &need_last_online_ids {
            if !last_online_map.contains_key(&user_id)
                || last_online_map
                    .get(&user_id)
                    .and_then(|v| v.as_ref())
                    .is_none()
            {
                miss_ids.push(user_id);
            }
        }
        let mut db_last_seen_map: HashMap<i64, Option<String>> = HashMap::new();
        if !miss_ids.is_empty() {
            if let Ok(rows) = self.online_status_repo.get_batch_status(&miss_ids).await {
                for r in rows {
                    db_last_seen_map.insert(r.user_id, r.last_seen_at.map(|t| t.to_rfc3339()));
                }
            }
        }

        // 6) 组装结果
        let mut result: HashMap<i64, (Option<bool>, Option<String>)> = HashMap::new();
        for &user_id in user_ids {
            let settings = settings_map.get(&user_id);
            let online_visible = settings.map(|s| s.online_visible).unwrap_or(true);
            let last_online_visible = settings.map(|s| s.last_online_visible).unwrap_or(true);

            if !online_visible {
                result.insert(user_id, (None, None));
                continue;
            }

            let is_online = online_status_map.get(&user_id).copied().unwrap_or(false);
            let last_online = if !is_online && last_online_visible {
                last_online_map
                    .get(&user_id)
                    .and_then(|v| v.clone())
                    .or_else(|| db_last_seen_map.get(&user_id).and_then(|v| v.clone()))
            } else {
                None
            };

            result.insert(user_id, (Some(is_online), last_online));
        }

        result
    }

    /// 批量获取用户位置可见性设置
    pub async fn batch_get_location_visible(&self, user_ids: &[i64]) -> HashMap<i64, bool> {
        // 默认值：true（如果没有 settings 记录则视为可见，符合“有默认值”的语义）
        let mut result: HashMap<i64, bool> =
            user_ids.iter().copied().map(|id| (id, true)).collect();

        // 真正的批量查询：避免 N+1
        if let Ok(settings) = self.user_settings_repo.get_batch(user_ids).await {
            for s in settings {
                result.insert(s.user_id, s.location_visible);
            }
        }

        result
    }

    /// 批量获取用户在线状态（仅 bool，用于 FriendsList 等，不处理 privacy）
    /// 这里的命名应该区分一下，或者直接 expose public
    pub async fn batch_get_online_status_raw(&self, user_ids: &[i64]) -> HashMap<i64, bool> {
        self.batch_get_online_status(user_ids).await
    }

    /// 获取用户在线状态（内部方法，不检查隐私设置）
    pub async fn get_online_status(&self, user_id: i64) -> (bool, Option<String>) {
        // 1. 从 Redis 查询实时在线状态
        let redis_online: redis::RedisResult<Option<f64>> = redis::cmd("ZSCORE")
            .arg(crate::constants::REDIS_KEY_PRESENCE_ONLINE)
            .arg(user_id)
            .query_async(&mut self.redis.clone())
            .await;

        if redis_online.ok().flatten().is_some() {
            return (true, None);
        }

        // 2. 从 Redis 查询最后上线时间
        let key = format!("presence:last_online:{}", user_id);
        let redis_last_online: redis::RedisResult<Option<String>> = redis::cmd("GET")
            .arg(&key)
            .query_async(&mut self.redis.clone())
            .await;

        if let Some(last_online) = redis_last_online.ok().flatten() {
            return (false, Some(last_online));
        }

        // 3. 从数据库查询
        match self.online_status_repo.get_status(user_id).await {
            Ok(Some(status)) => {
                let last_online = if !status.is_online {
                    status.last_seen_at.map(|t| t.to_rfc3339())
                } else {
                    None
                };
                (status.is_online, last_online)
            }
            _ => (false, None),
        }
    }

    /// 批量获取用户在线状态
    async fn batch_get_online_status(&self, user_ids: &[i64]) -> HashMap<i64, bool> {
        if user_ids.is_empty() {
            return HashMap::new();
        }

        let mut result: HashMap<i64, bool> = HashMap::new();

        // 1) Redis：使用 pipeline 批量 ZSCORE（减少 RTT）
        let mut conn = self.redis.clone();
        let mut pipe = redis::pipe();
        for &user_id in user_ids {
            pipe.cmd("ZSCORE")
                .arg(crate::constants::REDIS_KEY_PRESENCE_ONLINE)
                .arg(user_id);
        }
        let redis_scores: redis::RedisResult<Vec<Option<f64>>> = pipe.query_async(&mut conn).await;
        if let Ok(scores) = redis_scores {
            for (idx, score) in scores.into_iter().enumerate() {
                if score.is_some() {
                    result.insert(user_ids[idx], true);
                }
            }
        }

        // 2. 从数据库查询 Redis 中不在线的用户
        let not_online_ids: Vec<i64> = user_ids
            .iter()
            .filter(|id| !result.contains_key(id))
            .copied()
            .collect();

        if !not_online_ids.is_empty() {
            if let Ok(db_statuses) = self
                .online_status_repo
                .get_batch_status(&not_online_ids)
                .await
            {
                result.extend(db_statuses.into_iter().map(|s| (s.user_id, s.is_online)));
            }
        }

        // 3. 默认离线
        for &user_id in user_ids {
            result.entry(user_id).or_insert(false);
        }

        result
    }
}
