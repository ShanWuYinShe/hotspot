// src/services/subscription_service.rs - 通知订阅设置服务

use sqlx::PgPool;

use crate::{
    dto::{
        request::UpdateSubscriptionsRequest,
        response::SubscriptionsData,
    },
    error::Result,
};

/// 通知订阅设置服务
pub struct SubscriptionService {
    db: PgPool,
}

impl SubscriptionService {
    pub fn new(db: PgPool) -> Self {
        Self { db }
    }

    /// 获取订阅配置
    pub async fn get_config(&self, user_id: i64) -> Result<SubscriptionsData> {
        let row = sqlx::query!(
            r#"
            SELECT
                friend_request_enabled,
                friend_accepted_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                group_application_enabled,
                group_application_response_enabled,
                message_mention_enabled,
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                system_announcement_enabled
            FROM notification_subscriptions
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.db)
        .await?;

        match row {
            Some(r) => Ok(SubscriptionsData {
                friend_request_enabled: r.friend_request_enabled,
                friend_accepted_enabled: r.friend_accepted_enabled,
                group_invite_enabled: r.group_invite_enabled,
                group_invite_response_enabled: r.group_invite_response_enabled,
                group_application_enabled: r.group_application_enabled,
                group_application_response_enabled: r.group_application_response_enabled,
                message_mention_enabled: r.message_mention_enabled,
                moment_like_enabled: r.moment_like_enabled,
                moment_comment_enabled: r.moment_comment_enabled,
                comment_reply_enabled: r.comment_reply_enabled,
                system_announcement_enabled: r.system_announcement_enabled,
            }),
            None => {
                // 如果没有记录，返回默认值（全部启用）
                Ok(SubscriptionsData {
                    friend_request_enabled: true,
                    friend_accepted_enabled: true,
                    group_invite_enabled: true,
                    group_invite_response_enabled: true,
                    group_application_enabled: true,
                    group_application_response_enabled: true,
                    message_mention_enabled: true,
                    moment_like_enabled: true,
                    moment_comment_enabled: true,
                    comment_reply_enabled: true,
                    system_announcement_enabled: true,
                })
            }
        }
    }

    /// 更新订阅配置
    pub async fn update_config(&self, user_id: i64, req: &UpdateSubscriptionsRequest) -> Result<()> {
        // 使用 UPSERT 模式
        sqlx::query!(
            r#"
            INSERT INTO notification_subscriptions (
                user_id,
                friend_request_enabled,
                friend_accepted_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                group_application_enabled,
                group_application_response_enabled,
                message_mention_enabled,
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                system_announcement_enabled
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (user_id) DO UPDATE SET
                friend_request_enabled = COALESCE($2, notification_subscriptions.friend_request_enabled),
                friend_accepted_enabled = COALESCE($3, notification_subscriptions.friend_accepted_enabled),
                group_invite_enabled = COALESCE($4, notification_subscriptions.group_invite_enabled),
                group_invite_response_enabled = COALESCE($5, notification_subscriptions.group_invite_response_enabled),
                group_application_enabled = COALESCE($6, notification_subscriptions.group_application_enabled),
                group_application_response_enabled = COALESCE($7, notification_subscriptions.group_application_response_enabled),
                message_mention_enabled = COALESCE($8, notification_subscriptions.message_mention_enabled),
                moment_like_enabled = COALESCE($9, notification_subscriptions.moment_like_enabled),
                moment_comment_enabled = COALESCE($10, notification_subscriptions.moment_comment_enabled),
                comment_reply_enabled = COALESCE($11, notification_subscriptions.comment_reply_enabled),
                system_announcement_enabled = COALESCE($12, notification_subscriptions.system_announcement_enabled),
                updated_at = NOW()
            "#,
            user_id,
            req.friend_request_enabled.unwrap_or(true),
            req.friend_accepted_enabled.unwrap_or(true),
            req.group_invite_enabled.unwrap_or(true),
            req.group_invite_response_enabled.unwrap_or(true),
            req.group_application_enabled.unwrap_or(true),
            req.group_application_response_enabled.unwrap_or(true),
            req.message_mention_enabled.unwrap_or(true),
            req.moment_like_enabled.unwrap_or(true),
            req.moment_comment_enabled.unwrap_or(true),
            req.comment_reply_enabled.unwrap_or(true),
            req.system_announcement_enabled.unwrap_or(true)
        )
        .execute(&self.db)
        .await?;

        Ok(())
    }
}
