use sqlx::PgPool;

use crate::{error::AppError, models::event_subscription::EventSubscription};

#[derive(Debug, Clone)]
pub struct EventSubscriptionRepository {
    pool: PgPool,
}

impl EventSubscriptionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建默认订阅配置（用户注册时调用）
    pub async fn create_default_subscription(
        &self,
        user_id: i64,
    ) -> Result<EventSubscription, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            INSERT INTO event_subscriptions (
                user_id, friend_request_enabled, friend_accepted_enabled,
                group_invite_enabled, group_invite_response_enabled, message_mention_enabled,
                moment_like_enabled, moment_comment_enabled, comment_reply_enabled,
                system_announcement_enabled
            )
            VALUES ($1, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE, TRUE)
            RETURNING id, user_id, friend_request_enabled, friend_accepted_enabled,
                      group_invite_enabled, group_invite_response_enabled, message_mention_enabled,
                      moment_like_enabled, moment_comment_enabled, comment_reply_enabled,
                      system_announcement_enabled, created_at, updated_at
            "#,
            user_id
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 获取用户的订阅配置
    pub async fn find_subscription_by_user(
        &self,
        user_id: i64,
    ) -> Result<Option<EventSubscription>, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            SELECT id, user_id, friend_request_enabled, friend_accepted_enabled,
                   group_invite_enabled, group_invite_response_enabled, message_mention_enabled,
                   moment_like_enabled, moment_comment_enabled, comment_reply_enabled,
                   system_announcement_enabled, created_at, updated_at
            FROM event_subscriptions
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 更新订阅配置
    pub async fn update_subscription(
        &self,
        user_id: i64,
        friend_request_enabled: Option<bool>,
        friend_accepted_enabled: Option<bool>,
        group_invite_enabled: Option<bool>,
        group_invite_response_enabled: Option<bool>,
        message_mention_enabled: Option<bool>,
        moment_like_enabled: Option<bool>,
        moment_comment_enabled: Option<bool>,
        comment_reply_enabled: Option<bool>,
        system_announcement_enabled: Option<bool>,
    ) -> Result<EventSubscription, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            UPDATE event_subscriptions
            SET 
                friend_request_enabled = COALESCE($2, friend_request_enabled),
                friend_accepted_enabled = COALESCE($3, friend_accepted_enabled),
                group_invite_enabled = COALESCE($4, group_invite_enabled),
                group_invite_response_enabled = COALESCE($5, group_invite_response_enabled),
                message_mention_enabled = COALESCE($6, message_mention_enabled),
                moment_like_enabled = COALESCE($7, moment_like_enabled),
                moment_comment_enabled = COALESCE($8, moment_comment_enabled),
                comment_reply_enabled = COALESCE($9, comment_reply_enabled),
                system_announcement_enabled = COALESCE($10, system_announcement_enabled),
                updated_at = NOW()
            WHERE user_id = $1
            RETURNING id, user_id, friend_request_enabled, friend_accepted_enabled,
                      group_invite_enabled, group_invite_response_enabled, message_mention_enabled,
                      moment_like_enabled, moment_comment_enabled, comment_reply_enabled,
                      system_announcement_enabled, created_at, updated_at
            "#,
            user_id,
            friend_request_enabled,
            friend_accepted_enabled,
            group_invite_enabled,
            group_invite_response_enabled,
            message_mention_enabled,
            moment_like_enabled,
            moment_comment_enabled,
            comment_reply_enabled,
            system_announcement_enabled
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 检查用户是否订阅了某个事件类型
    /// event_type: "friend_request", "friend_accepted", "group_invite", etc.
    pub async fn is_event_enabled(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<bool, AppError> {
        let subscription = self.find_subscription_by_user(user_id).await?;
        
        match subscription {
            None => Ok(true), // 没有配置则默认启用
            Some(sub) => {
                let enabled = match event_type {
                    "friend_request" => sub.friend_request_enabled,
                    "friend_accepted" => sub.friend_accepted_enabled,
                    "group_invite" => sub.group_invite_enabled,
                    "group_invite_response" => sub.group_invite_response_enabled,
                    "message_mention" => sub.message_mention_enabled,
                    "moment_like" => sub.moment_like_enabled,
                    "moment_comment" => sub.moment_comment_enabled,
                    "comment_reply" => sub.comment_reply_enabled,
                    "system_announcement" => sub.system_announcement_enabled,
                    _ => true, // 未知类型默认启用
                };
                Ok(enabled)
            }
        }
    }

    /// 删除用户的订阅配置
    pub async fn delete_subscription(&self, user_id: i64) -> Result<u64, AppError> {
        sqlx::query!(
            r#"
            DELETE FROM event_subscriptions
            WHERE user_id = $1
            "#,
            user_id
        )
        .execute(&self.pool)
        .await
        .map(|r| r.rows_affected())
        .map_err(AppError::from)
    }
}
