// src/services/subscription_service.rs - 通知订阅设置服务

use sqlx::PgPool;

use crate::{
    dto::{request::UpdateSubscriptionsRequest, response::SubscriptionsData},
    error::Result,
};

/// 通知订阅设置服务
#[derive(Clone)]
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
                new_follower_enabled,
                mutual_follow_enabled,
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
                new_follower_enabled: r.new_follower_enabled,
                mutual_follow_enabled: r.mutual_follow_enabled,
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
                    new_follower_enabled: true,
                    mutual_follow_enabled: true,
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
    pub async fn update_config(
        &self,
        user_id: i64,
        req: &UpdateSubscriptionsRequest,
    ) -> Result<()> {
        // 使用 UPSERT 模式
        // 注意：传入 Option<bool>，在 SQL 中使用 COALESCE 处理
        // INSERT 时：如果为 NULL 则使用默认值 TRUE
        // UPDATE 时：如果为 NULL 则保持原值
        sqlx::query!(
            r#"
            INSERT INTO notification_subscriptions (
                user_id,
                new_follower_enabled,
                mutual_follow_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                group_application_enabled,
                group_application_response_enabled,
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
                COALESCE($12, TRUE)
            )
            ON CONFLICT (user_id) DO UPDATE SET
                new_follower_enabled = COALESCE($2, notification_subscriptions.new_follower_enabled),
                mutual_follow_enabled = COALESCE($3, notification_subscriptions.mutual_follow_enabled),
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
            req.new_follower_enabled,
            req.mutual_follow_enabled,
            req.group_invite_enabled,
            req.group_invite_response_enabled,
            req.group_application_enabled,
            req.group_application_response_enabled,
            req.message_mention_enabled,
            req.moment_like_enabled,
            req.moment_comment_enabled,
            req.comment_reply_enabled,
            req.system_announcement_enabled
        )
        .execute(&self.db)
        .await?;

        Ok(())
    }
}
