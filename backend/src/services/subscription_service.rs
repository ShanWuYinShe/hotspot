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
        use sqlx::Row;
        let mut conn = self.redis.clone();
        let key = format!("user:subscription_config:{}", user_id);

        if let Ok(Some(json)) = conn.get::<_, Option<String>>(&key).await {
            if let Ok(config) = serde_json::from_str(&json) {
                return Ok(config);
            }
        }

        let row = sqlx::query(
            r#"
            SELECT
                new_follower_enabled,
                mutual_follow_enabled,
                follow_request_enabled,
                follow_request_handled_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                group_application_enabled,
                group_application_response_enabled,
                group_announcement_enabled,
                message_mention_enabled,
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                system_announcement_enabled
            FROM notification_subscriptions
            WHERE user_id = $1
            "#,
        )
        .bind(user_id)
        .fetch_optional(&self.db)
        .await?;

        let data = if let Some(r) = row {
            SubscriptionsData {
                new_follower_enabled: r.get("new_follower_enabled"),
                mutual_follow_enabled: r.get("mutual_follow_enabled"),
                follow_request_enabled: r.get("follow_request_enabled"),
                follow_request_handled_enabled: r.get("follow_request_handled_enabled"),
                group_invite_enabled: r.get("group_invite_enabled"),
                group_invite_response_enabled: r.get("group_invite_response_enabled"),
                group_application_enabled: r.get("group_application_enabled"),
                group_application_response_enabled: r.get("group_application_response_enabled"),
                group_announcement_enabled: r.get("group_announcement_enabled"),
                message_mention_enabled: r.get("message_mention_enabled"),
                moment_like_enabled: r.get("moment_like_enabled"),
                moment_comment_enabled: r.get("moment_comment_enabled"),
                comment_reply_enabled: r.get("comment_reply_enabled"),
                system_announcement_enabled: r.get("system_announcement_enabled"),
            }
        } else {
            // 如果没有记录，返回默认值（全部启用）
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
        // 使用 UPSERT 模式
        // 注意：传入 Option<bool>，在 SQL 中使用 COALESCE 处理
        // INSERT 时：如果为 NULL 则使用默认值 TRUE
        // UPDATE 时：如果为 NULL 则保持原值
        sqlx::query(
            r#"
            INSERT INTO notification_subscriptions (
                user_id,
                new_follower_enabled,
                mutual_follow_enabled,
                follow_request_enabled,
                follow_request_handled_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                group_application_enabled,
                group_application_response_enabled,
                group_announcement_enabled,
                message_mention_enabled,
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                system_announcement_enabled
            ) VALUES (
                $1, 
                COALESCE($2, TRUE), 
                COALESCE($3, TRUE), 
                COALESCE($4, TRUE), 
                COALESCE($5, TRUE), 
                COALESCE($6, TRUE), 
                COALESCE($7, TRUE), 
                COALESCE($8, TRUE), 
                COALESCE($9, TRUE), 
                COALESCE($10, TRUE), 
                COALESCE($11, TRUE), 
                COALESCE($12, TRUE), 
                COALESCE($13, TRUE), 
                COALESCE($14, TRUE), 
                COALESCE($15, TRUE)
            )
            ON CONFLICT (user_id) DO UPDATE SET
                new_follower_enabled = COALESCE($2, notification_subscriptions.new_follower_enabled),
                mutual_follow_enabled = COALESCE($3, notification_subscriptions.mutual_follow_enabled),
                follow_request_enabled = COALESCE($4, notification_subscriptions.follow_request_enabled),
                follow_request_handled_enabled = COALESCE($5, notification_subscriptions.follow_request_handled_enabled),
                group_invite_enabled = COALESCE($6, notification_subscriptions.group_invite_enabled),
                group_invite_response_enabled = COALESCE($7, notification_subscriptions.group_invite_response_enabled),
                group_application_enabled = COALESCE($8, notification_subscriptions.group_application_enabled),
                group_application_response_enabled = COALESCE($9, notification_subscriptions.group_application_response_enabled),
                group_announcement_enabled = COALESCE($10, notification_subscriptions.group_announcement_enabled),
                message_mention_enabled = COALESCE($11, notification_subscriptions.message_mention_enabled),
                moment_like_enabled = COALESCE($12, notification_subscriptions.moment_like_enabled),
                moment_comment_enabled = COALESCE($13, notification_subscriptions.moment_comment_enabled),
                comment_reply_enabled = COALESCE($14, notification_subscriptions.comment_reply_enabled),
                system_announcement_enabled = COALESCE($15, notification_subscriptions.system_announcement_enabled),
                updated_at = NOW()
            "#,
        )
        .bind(user_id)
        .bind(req.new_follower_enabled)
        .bind(req.mutual_follow_enabled)
        .bind(req.follow_request_enabled)
        .bind(req.follow_request_handled_enabled)
        .bind(req.group_invite_enabled)
        .bind(req.group_invite_response_enabled)
        .bind(req.group_application_enabled)
        .bind(req.group_application_response_enabled)
        .bind(req.group_announcement_enabled)
        .bind(req.message_mention_enabled)
        .bind(req.moment_like_enabled)
        .bind(req.moment_comment_enabled)
        .bind(req.comment_reply_enabled)
        .bind(req.system_announcement_enabled)
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
