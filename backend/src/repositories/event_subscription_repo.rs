use sqlx::PgPool;
use uuid::Uuid;

use crate::{error::AppError, models::event_subscription::EventSubscription};

#[derive(Debug, Clone)]
pub struct EventSubscriptionRepository {
    pool: PgPool,
}

impl EventSubscriptionRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }

    /// 创建订阅配置
    pub async fn create_subscription(
        &self,
        subscription: &EventSubscription,
    ) -> Result<EventSubscription, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            INSERT INTO event_subscriptions (id, user_id, event_type, enabled, push_enabled, email_enabled, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            RETURNING id, user_id, event_type, enabled, push_enabled, email_enabled, created_at, updated_at
            "#,
            subscription.id,
            subscription.user_id,
            subscription.event_type,
            subscription.enabled,
            subscription.push_enabled,
            subscription.email_enabled,
            subscription.created_at,
            subscription.updated_at
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 批量创建订阅配置
    pub async fn create_subscriptions_batch(
        &self,
        subscriptions: Vec<EventSubscription>,
    ) -> Result<Vec<EventSubscription>, AppError> {
        let mut results = Vec::new();
        for subscription in subscriptions {
            let result = self.create_subscription(&subscription).await?;
            results.push(result);
        }
        Ok(results)
    }

    /// 查找用户的所有订阅配置
    pub async fn find_subscriptions_by_user(
        &self,
        user_id: i64,
    ) -> Result<Vec<EventSubscription>, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            SELECT id, user_id, event_type, enabled, push_enabled, email_enabled, created_at, updated_at
            FROM event_subscriptions
            WHERE user_id = $1
            ORDER BY event_type
            "#,
            user_id
        )
        .fetch_all(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 查找特定事件类型的订阅配置
    pub async fn find_subscription_by_type(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<Option<EventSubscription>, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            SELECT id, user_id, event_type, enabled, push_enabled, email_enabled, created_at, updated_at
            FROM event_subscriptions
            WHERE user_id = $1 AND event_type = $2
            "#,
            user_id,
            event_type
        )
        .fetch_optional(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 更新订阅配置
    pub async fn update_subscription(
        &self,
        user_id: i64,
        event_type: &str,
        enabled: Option<bool>,
        push_enabled: Option<bool>,
        email_enabled: Option<bool>,
    ) -> Result<EventSubscription, AppError> {
        sqlx::query_as!(
            EventSubscription,
            r#"
            UPDATE event_subscriptions
            SET 
                enabled = COALESCE($3, enabled),
                push_enabled = COALESCE($4, push_enabled),
                email_enabled = COALESCE($5, email_enabled),
                updated_at = NOW()
            WHERE user_id = $1 AND event_type = $2
            RETURNING id, user_id, event_type, enabled, push_enabled, email_enabled, created_at, updated_at
            "#,
            user_id,
            event_type,
            enabled,
            push_enabled,
            email_enabled
        )
        .fetch_one(&self.pool)
        .await
        .map_err(AppError::from)
    }

    /// 批量更新订阅配置
    pub async fn update_subscriptions_batch(
        &self,
        user_id: i64,
        updates: Vec<(String, Option<bool>, Option<bool>, Option<bool>)>,
    ) -> Result<Vec<EventSubscription>, AppError> {
        let mut results = Vec::new();
        for (event_type, enabled, push_enabled, email_enabled) in updates {
            let result = self
                .update_subscription(user_id, &event_type, enabled, push_enabled, email_enabled)
                .await?;
            results.push(result);
        }
        Ok(results)
    }

    /// 删除订阅配置
    pub async fn delete_subscription(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<u64, AppError> {
        sqlx::query!(
            r#"
            DELETE FROM event_subscriptions
            WHERE user_id = $1 AND event_type = $2
            "#,
            user_id,
            event_type
        )
        .execute(&self.pool)
        .await
        .map(|r| r.rows_affected())
        .map_err(AppError::from)
    }

    /// 检查用户是否订阅了某个事件类型
    pub async fn is_subscribed(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<bool, AppError> {
        let subscription = self.find_subscription_by_type(user_id, event_type).await?;
        Ok(subscription.map(|s| s.enabled).unwrap_or(false))
    }

    /// 检查用户是否启用了推送通知
    pub async fn is_push_enabled(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<bool, AppError> {
        let subscription = self.find_subscription_by_type(user_id, event_type).await?;
        Ok(subscription
            .map(|s| s.enabled && s.push_enabled)
            .unwrap_or(false))
    }
}

