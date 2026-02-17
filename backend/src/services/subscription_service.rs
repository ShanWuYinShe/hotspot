// src/services/subscription_service.rs - 通知订阅设置服务

use redis::aio::ConnectionManager;
use sqlx::PgPool;

use crate::{dto::response::SubscriptionsData, error::Result};

/// 通知订阅设置服务
#[derive(Clone)]
pub struct SubscriptionService {
    db: PgPool,
    redis: ConnectionManager,
}

impl SubscriptionService {
    pub fn new(db: PgPool, redis: ConnectionManager) -> Self {
        Self { db, redis }
    }

    /// 获取订阅配置
    pub async fn get_config(&self, user_id: i64) -> Result<SubscriptionsData> {
        // 1. 尝试从缓存获取
        use redis::AsyncCommands;
        let mut conn = self.redis.clone();
        let key = format!("user:subscription_config:{}", user_id);

        if let Ok(Some(json)) = conn.get::<_, Option<String>>(&key).await {
            if let Ok(config) = serde_json::from_str(&json) {
                return Ok(config);
            }
        }

        let row = sqlx::query!(
            r#"
            SELECT
                new_follower_enabled,
                mutual_follow_enabled,
                follow_request_enabled,
                follow_request_handled_enabled,
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                group_apply_enabled,
                group_apply_result_enabled,
                group_announcement_enabled,
                message_mention_enabled,
                system_announcement_enabled
            FROM notification_subscriptions
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.db)
        .await?;

        let data = if let Some(r) = row {
            SubscriptionsData {
                new_follower_enabled: r.new_follower_enabled,
                mutual_follow_enabled: r.mutual_follow_enabled,
                follow_request_enabled: r.follow_request_enabled,
                follow_request_handled_enabled: r.follow_request_handled_enabled,
                group_invite_enabled: r.group_invite_enabled,
                group_invite_response_enabled: r.group_invite_response_enabled,
                group_application_enabled: r.group_apply_enabled,
                group_application_response_enabled: r.group_apply_result_enabled,
                group_announcement_enabled: r.group_announcement_enabled,
                message_mention_enabled: r.message_mention_enabled,
                moment_like_enabled: r.moment_like_enabled,
                moment_comment_enabled: r.moment_comment_enabled,
                comment_reply_enabled: r.comment_reply_enabled,
                system_announcement_enabled: r.system_announcement_enabled,
            }
        } else {
            // 如果没有记录，返回默认值
            SubscriptionsData {
                new_follower_enabled: true,
                mutual_follow_enabled: true,
                follow_request_enabled: true,
                follow_request_handled_enabled: true,
                group_invite_enabled: true,
                group_invite_response_enabled: true,
                group_application_enabled: true,
                group_application_response_enabled: true,
                group_announcement_enabled: true,
                message_mention_enabled: true,
                moment_like_enabled: true,
                moment_comment_enabled: true,
                comment_reply_enabled: true,
                system_announcement_enabled: true,
            }
        };

        // 写入缓存 (1小时)
        if let Ok(json) = serde_json::to_string(&data) {
            let _: () = conn.set_ex(&key, json, 3600).await.unwrap_or(());
        }

        Ok(data)
    }
}
