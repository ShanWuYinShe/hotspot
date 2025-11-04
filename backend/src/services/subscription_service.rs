use sqlx::PgPool;

use crate::{
    dto::{
        request::{GetSubscriptionsRequest, UpdateSubscriptionsRequest},
        response::{SubscriptionResponse, UpdateSubscriptionsResponse},
    },
    error::{AppError, Result},
};

pub struct SubscriptionService {
    pool: PgPool,
}

impl SubscriptionService {
    pub fn new(pool: &PgPool) -> Self {
        Self {
            pool: pool.clone(),
        }
    }

    /// GET /api/v1/protected/subscriptions/get - 获取订阅配置
    pub async fn get_subscriptions(
        &self,
        user_id: i64,
        _req: &GetSubscriptionsRequest,
    ) -> Result<SubscriptionResponse> {
        // 从数据库获取用户的订阅配置
        // 使用 query_as! 确保类型安全
        let subscription = sqlx::query_as!(
            SubscriptionConfig,
            r#"
            SELECT 
                friend_request_enabled,
                friend_accepted_enabled,
                group_invite_enabled,
                group_invite_response_enabled,
                message_mention_enabled,
                moment_like_enabled,
                moment_comment_enabled,
                comment_reply_enabled,
                system_announcement_enabled
            FROM event_subscriptions
            WHERE user_id = $1
            "#,
            user_id
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::from)?;

        // 如果没有配置,返回默认值(全部启用)
        let config = subscription.unwrap_or_default();

        Ok(SubscriptionResponse {
            friend_request_enabled: config.friend_request_enabled,
            friend_accepted_enabled: config.friend_accepted_enabled,
            group_invite_enabled: config.group_invite_enabled,
            group_invite_response_enabled: config.group_invite_response_enabled,
            message_mention_enabled: config.message_mention_enabled,
            moment_like_enabled: config.moment_like_enabled,
            moment_comment_enabled: config.moment_comment_enabled,
            comment_reply_enabled: config.comment_reply_enabled,
            system_announcement_enabled: config.system_announcement_enabled,
        })
    }

    /// POST /api/v1/protected/subscriptions/update - 更新订阅配置
    pub async fn update_subscriptions(
        &self,
        user_id: i64,
        req: &UpdateSubscriptionsRequest,
    ) -> Result<UpdateSubscriptionsResponse> {
        // 使用 UPSERT (INSERT ... ON CONFLICT DO UPDATE)
        sqlx::query!(
            r#"
            INSERT INTO event_subscriptions (
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
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            ON CONFLICT (user_id) DO UPDATE SET
                friend_request_enabled = COALESCE($2, event_subscriptions.friend_request_enabled),
                friend_accepted_enabled = COALESCE($3, event_subscriptions.friend_accepted_enabled),
                group_invite_enabled = COALESCE($4, event_subscriptions.group_invite_enabled),
                group_invite_response_enabled = COALESCE($5, event_subscriptions.group_invite_response_enabled),
                message_mention_enabled = COALESCE($6, event_subscriptions.message_mention_enabled),
                moment_like_enabled = COALESCE($7, event_subscriptions.moment_like_enabled),
                moment_comment_enabled = COALESCE($8, event_subscriptions.moment_comment_enabled),
                comment_reply_enabled = COALESCE($9, event_subscriptions.comment_reply_enabled),
                system_announcement_enabled = COALESCE($10, event_subscriptions.system_announcement_enabled),
                updated_at = NOW()
            "#,
            user_id,
            req.friend_request_enabled,
            req.friend_accepted_enabled,
            req.group_invite_enabled,
            req.group_invite_response_enabled,
            req.message_mention_enabled,
            req.moment_like_enabled,
            req.moment_comment_enabled,
            req.comment_reply_enabled,
            req.system_announcement_enabled
        )
        .execute(&self.pool)
        .await
        .map_err(AppError::from)?;

        Ok(UpdateSubscriptionsResponse {
            success: true,
            message: "订阅配置已更新".to_string(),
        })
    }

    /// 检查用户是否订阅了某个事件类型(内部使用)
    pub async fn is_subscribed(&self, user_id: i64, event_type: &str) -> Result<bool> {
        let subscription = self.get_subscriptions(user_id, &GetSubscriptionsRequest {}).await?;

        let enabled = match event_type {
            "friend_request" => subscription.friend_request_enabled,
            "friend_accepted" => subscription.friend_accepted_enabled,
            "group_invite" => subscription.group_invite_enabled,
            "group_invite_response" => subscription.group_invite_response_enabled,
            "message_mention" => subscription.message_mention_enabled,
            "moment_like" => subscription.moment_like_enabled,
            "moment_comment" => subscription.moment_comment_enabled,
            "comment_reply" => subscription.comment_reply_enabled,
            "system_announcement" => subscription.system_announcement_enabled,
            _ => return Err(AppError::BadRequest(format!("未知的事件类型: {}", event_type))),
        };

        Ok(enabled)
    }

    /// 更新单个订阅（别名方法）
    pub async fn update_subscription(
        &self,
        user_id: i64,
        req: &crate::dto::request::UpdateSubscriptionsRequest,
    ) -> Result<crate::dto::response::UpdateSubscriptionsResponse> {
        self.update_subscriptions(user_id, req).await
    }

    /// 批量更新订阅（别名方法）
    pub async fn update_subscriptions_batch(
        &self,
        user_id: i64,
        req: &crate::dto::request::UpdateSubscriptionsRequest,
    ) -> Result<crate::dto::response::UpdateSubscriptionsResponse> {
        self.update_subscriptions(user_id, req).await
    }
}

/// 数据库查询结果结构
#[derive(Debug)]
struct SubscriptionConfig {
    friend_request_enabled: bool,
    friend_accepted_enabled: bool,
    group_invite_enabled: bool,
    group_invite_response_enabled: bool,
    message_mention_enabled: bool,
    moment_like_enabled: bool,
    moment_comment_enabled: bool,
    comment_reply_enabled: bool,
    system_announcement_enabled: bool,
}

impl Default for SubscriptionConfig {
    fn default() -> Self {
        Self {
            friend_request_enabled: true,
            friend_accepted_enabled: true,
            group_invite_enabled: true,
            group_invite_response_enabled: true,
            message_mention_enabled: true,
            moment_like_enabled: true,
            moment_comment_enabled: true,
            comment_reply_enabled: true,
            system_announcement_enabled: true,
        }
    }
}
