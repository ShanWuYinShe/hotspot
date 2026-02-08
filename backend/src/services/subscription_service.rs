// src/services/subscription_service.rs - 通知订阅设置服务

use redis::aio::ConnectionManager;
use sqlx::PgPool;

use crate::{
    dto::{request::UpdateSubscriptionsRequest, response::SubscriptionsData},
    error::Result,
};

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
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                group_invite_enabled,
                group_apply_enabled,
                group_apply_result_enabled,
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
                moment_like_enabled: r.moment_like_enabled,
                moment_comment_enabled: r.moment_comment_enabled,
                comment_reply_enabled: r.comment_reply_enabled,
                group_invite_enabled: r.group_invite_enabled,
                system_announcement_enabled: r.system_announcement_enabled,
                // 以下字段在 Schema 中缺失，使用默认值或待补充
                mutual_follow_enabled: true,
                follow_request_enabled: true,
                follow_request_handled_enabled: true,
                group_invite_response_enabled: true,
                group_application_enabled: r.group_apply_enabled,
                group_application_response_enabled: r.group_apply_result_enabled,
                group_announcement_enabled: true,
                message_mention_enabled: true,
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

    /// 更新订阅配置
    pub async fn update_config(
        &self,
        user_id: i64,
        req: &UpdateSubscriptionsRequest,
    ) -> Result<()> {
        sqlx::query!(
            r#"
            INSERT INTO notification_subscriptions (
                user_id, new_follower_enabled, moment_like_enabled, moment_comment_enabled,
                comment_reply_enabled, group_invite_enabled, group_apply_enabled,
                group_apply_result_enabled, system_announcement_enabled
            ) VALUES (
                $1, COALESCE($2, TRUE), COALESCE($3, TRUE), COALESCE($4, TRUE),
                COALESCE($5, TRUE), COALESCE($6, TRUE), COALESCE($7, TRUE),
                COALESCE($8, TRUE), COALESCE($9, TRUE)
            )
            ON CONFLICT (user_id) DO UPDATE SET
                new_follower_enabled = COALESCE($2, notification_subscriptions.new_follower_enabled),
                moment_like_enabled = COALESCE($3, notification_subscriptions.moment_like_enabled),
                moment_comment_enabled = COALESCE($4, notification_subscriptions.moment_comment_enabled),
                comment_reply_enabled = COALESCE($5, notification_subscriptions.comment_reply_enabled),
                group_invite_enabled = COALESCE($6, notification_subscriptions.group_invite_enabled),
                group_apply_enabled = COALESCE($7, notification_subscriptions.group_apply_enabled),
                group_apply_result_enabled = COALESCE($8, notification_subscriptions.group_apply_result_enabled),
                system_announcement_enabled = COALESCE($9, notification_subscriptions.system_announcement_enabled),
                updated_at = NOW()
            "#,
            user_id,
            req.new_follower_enabled,
            req.moment_like_enabled,
            req.moment_comment_enabled,
            req.comment_reply_enabled,
            req.group_invite_enabled,
            req.group_application_enabled, // Map to group_apply_enabled?
            req.group_application_response_enabled, // Map to group_apply_result_enabled?
            req.system_announcement_enabled
        )
        .execute(&self.db)
        .await?;

        // 清除缓存
        use redis::AsyncCommands;
        let mut conn = self.redis.clone();
        let key = format!("user:subscription_config:{}", user_id);
        let _: () = conn.del(&key).await.unwrap_or(());

        Ok(())
    }
}
