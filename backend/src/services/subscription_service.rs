use sqlx::PgPool;

use crate::{
    dto::{
        request::{GetSubscriptionsRequest, UpdateSubscriptionRequest, UpdateSubscriptionsBatchRequest},
        response::SubscriptionResponse,
    },
    error::AppError,
    models::event_subscription::EventSubscription,
    repositories::event_subscription_repo::EventSubscriptionRepository,
};

#[derive(Debug, Clone)]
pub struct SubscriptionService {
    subscription_repo: EventSubscriptionRepository,
}

impl SubscriptionService {
    pub fn new(pool: PgPool) -> Self {
        Self {
            subscription_repo: EventSubscriptionRepository::new(pool),
        }
    }

    /// 获取用户的所有订阅配置
    pub async fn get_subscriptions(
        &self,
        user_id: i64,
        _req: GetSubscriptionsRequest,
    ) -> Result<Vec<SubscriptionResponse>, AppError> {
        let subscriptions = self
            .subscription_repo
            .find_subscriptions_by_user(user_id)
            .await?;

        // 如果用户没有订阅配置，创建默认配置
        let subscriptions = if subscriptions.is_empty() {
            let defaults = EventSubscription::create_defaults(user_id);
            self.subscription_repo
                .create_subscriptions_batch(defaults)
                .await?
        } else {
            subscriptions
        };

        Ok(subscriptions
            .into_iter()
            .map(|s| SubscriptionResponse {
                event_type: s.event_type,
                enabled: s.enabled,
                push_enabled: s.push_enabled,
                email_enabled: s.email_enabled,
            })
            .collect())
    }

    /// 更新单个订阅配置
    pub async fn update_subscription(
        &self,
        user_id: i64,
        req: UpdateSubscriptionRequest,
    ) -> Result<SubscriptionResponse, AppError> {
        // 验证event_type
        let valid_types = vec![
            "private_message",
            "group_message",
            "friend_request",
            "group_invite",
            "moment",
            "system_notification",
        ];

        if !valid_types.contains(&req.event_type.as_str()) {
            return Err(AppError::InvalidInput(format!(
                "无效的事件类型: {}",
                req.event_type
            )));
        }

        // 检查订阅是否存在
        let existing = self
            .subscription_repo
            .find_subscription_by_type(user_id, &req.event_type)
            .await?;

        if existing.is_none() {
            // 如果不存在，创建新的订阅配置
            let subscription = EventSubscription::new(user_id, req.event_type.clone());
            self.subscription_repo
                .create_subscription(&subscription)
                .await?;
        }

        // 更新订阅配置
        let updated = self
            .subscription_repo
            .update_subscription(
                user_id,
                &req.event_type,
                req.enabled,
                req.push_enabled,
                req.email_enabled,
            )
            .await?;

        Ok(SubscriptionResponse {
            event_type: updated.event_type,
            enabled: updated.enabled,
            push_enabled: updated.push_enabled,
            email_enabled: updated.email_enabled,
        })
    }

    /// 批量更新订阅配置
    pub async fn update_subscriptions_batch(
        &self,
        user_id: i64,
        req: UpdateSubscriptionsBatchRequest,
    ) -> Result<Vec<SubscriptionResponse>, AppError> {
        // 验证所有event_type
        let valid_types = vec![
            "private_message",
            "group_message",
            "friend_request",
            "group_invite",
            "moment",
            "system_notification",
        ];

        for update in &req.subscriptions {
            if !valid_types.contains(&update.event_type.as_str()) {
                return Err(AppError::InvalidInput(format!(
                    "无效的事件类型: {}",
                    update.event_type
                )));
            }
        }

        // 确保所有订阅配置存在
        for update in &req.subscriptions {
            let existing = self
                .subscription_repo
                .find_subscription_by_type(user_id, &update.event_type)
                .await?;

            if existing.is_none() {
                let subscription = EventSubscription::new(user_id, update.event_type.clone());
                self.subscription_repo
                    .create_subscription(&subscription)
                    .await?;
            }
        }

        // 批量更新
        let updates: Vec<(String, Option<bool>, Option<bool>, Option<bool>)> = req
            .subscriptions
            .into_iter()
            .map(|u| (u.event_type, u.enabled, u.push_enabled, u.email_enabled))
            .collect();

        let updated_subscriptions = self
            .subscription_repo
            .update_subscriptions_batch(user_id, updates)
            .await?;

        Ok(updated_subscriptions
            .into_iter()
            .map(|s| SubscriptionResponse {
                event_type: s.event_type,
                enabled: s.enabled,
                push_enabled: s.push_enabled,
                email_enabled: s.email_enabled,
            })
            .collect())
    }

    /// 检查用户是否订阅了某个事件类型（内部使用）
    pub async fn is_subscribed(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<bool, AppError> {
        self.subscription_repo
            .is_subscribed(user_id, event_type)
            .await
    }

    /// 检查用户是否启用了推送通知（内部使用）
    pub async fn is_push_enabled(
        &self,
        user_id: i64,
        event_type: &str,
    ) -> Result<bool, AppError> {
        self.subscription_repo
            .is_push_enabled(user_id, event_type)
            .await
    }
}

