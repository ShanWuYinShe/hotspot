use chrono::{Duration, NaiveDate, Utc};
use redis::{aio::ConnectionManager, AsyncCommands};

use crate::error::Result;

/// 控制陌生人消息速率（按发送者-接收者维度 + 按日限制）
#[derive(Clone)]
pub struct StrangerMessageLimiter {
    redis: ConnectionManager,
    limit_per_pair_per_day: i64,
}

impl StrangerMessageLimiter {
    pub fn new(redis: ConnectionManager, limit_per_pair_per_day: i64) -> Self {
        Self {
            redis,
            limit_per_pair_per_day,
        }
    }

    fn today_suffix() -> NaiveDate {
        Utc::now().date_naive()
    }

    fn key(&self, sender_id: i64, target_user_id: i64) -> String {
        let date = Self::today_suffix();
        format!("stranger_msg:{}:{}:{}", sender_id, target_user_id, date)
    }

    fn seconds_until_next_day() -> i64 {
        let now = Utc::now();
        let tomorrow = now.date_naive() + Duration::days(1);
        let midnight = tomorrow.and_hms_opt(0, 0, 0).unwrap().and_utc();
        (midnight - now).num_seconds().max(1)
    }

    /// 判断 sender→target 的陌生人消息是否已达上限
    pub async fn is_over_limit(&self, sender_id: i64, target_user_id: i64) -> Result<bool> {
        let mut conn = self.redis.clone();
        let key = self.key(sender_id, target_user_id);
        let current: Option<i64> = conn.get(&key).await?;
        Ok(current.unwrap_or(0) >= self.limit_per_pair_per_day)
    }

    /// 自增计数器，超过限制则返回 false 并回滚增量
    pub async fn increment_if_allowed(&self, sender_id: i64, target_user_id: i64) -> Result<bool> {
        let key = self.key(sender_id, target_user_id);
        let ttl = Self::seconds_until_next_day();
        let mut conn = self.redis.clone();

        let count: i64 = redis::cmd("INCR").arg(&key).query_async(&mut conn).await?;

        if count == 1 {
            let _: () = conn.expire(&key, ttl).await?;
        }

        if count > self.limit_per_pair_per_day {
            let _: i64 = redis::cmd("DECR").arg(&key).query_async(&mut conn).await?;
            return Ok(false);
        }

        Ok(true)
    }
}
