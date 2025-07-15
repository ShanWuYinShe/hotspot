use std::sync::Arc;
use tracing::info;

use crate::services::notification::{NotificationPolicy, NotificationPolicyService, WebSocketNotificationService, GroupNotificationPolicy, GroupNotificationService};
use crate::repositories::{PgUserRepository, group_repository::PgGroupRepository};
use crate::ws::message_router::MessageRouter;
use crate::cache::NotificationCache;

/// 通知管理器 - 统一管理所有通知相关服务
pub struct NotificationManager {
    websocket_service: Arc<WebSocketNotificationService>,
    #[allow(dead_code)]
    group_service: Arc<GroupNotificationService>,
    policy_service: Arc<NotificationPolicyService>,
}

impl NotificationManager {
    pub fn new(
        message_router: Arc<MessageRouter>,
        user_repository: Arc<PgUserRepository>,
        group_repository: Arc<PgGroupRepository>,
        initial_policy: Option<NotificationPolicy>,
        initial_group_policy: Option<GroupNotificationPolicy>,
    ) -> Self {
        let policy_service = Arc::new(NotificationPolicyService::new(
            user_repository.clone(),
            initial_policy,
            initial_group_policy,
        ));

        let websocket_service = Arc::new(WebSocketNotificationService::new(
            message_router.clone(),
            user_repository.clone(),
            Some(policy_service.get_policy().clone()),
        ));

        let group_service = Arc::new(GroupNotificationService::new(
            message_router,
            user_repository,
            group_repository,
            Some(policy_service.get_group_policy().clone()),
        ));

        Self {
            websocket_service,
            group_service,
            policy_service,
        }
    }

    /// 创建带有缓存的通知管理器
    pub fn with_cache(
        message_router: Arc<MessageRouter>,
        user_repository: Arc<PgUserRepository>,
        group_repository: Arc<PgGroupRepository>,
        initial_policy: Option<NotificationPolicy>,
        initial_group_policy: Option<GroupNotificationPolicy>,
        notification_cache: Arc<dyn NotificationCache>,
    ) -> Self {
        let policy_service = Arc::new(NotificationPolicyService::new(
            user_repository.clone(),
            initial_policy,
            initial_group_policy,
        ));

        let websocket_service = Arc::new(WebSocketNotificationService::with_cache(
            message_router.clone(),
            user_repository.clone(),
            Some(policy_service.get_policy().clone()),
            notification_cache,
        ));

        let group_service = Arc::new(GroupNotificationService::new(
            message_router,
            user_repository,
            group_repository,
            Some(policy_service.get_group_policy().clone()),
        ));

        Self {
            websocket_service,
            group_service,
            policy_service,
        }
    }

    /// 获取 WebSocket 通知服务
    pub fn websocket_service(&self) -> Arc<WebSocketNotificationService> {
        self.websocket_service.clone()
    }

    /// 获取群组通知服务
    #[allow(dead_code)]
    pub fn group_service(&self) -> Arc<GroupNotificationService> {
        self.group_service.clone()
    }

    /// 获取策略服务
    #[allow(dead_code)]
    pub fn policy_service(&self) -> Arc<NotificationPolicyService> {
        self.policy_service.clone()
    }

    /// 更新全局通知策略
    #[allow(dead_code)]
    pub async fn update_policy(&self, new_policy: NotificationPolicy) {
        info!("更新用户状态通知策略: {:?}", new_policy);
        // 注意：这里需要重新创建服务以应用新策略
        // 在实际应用中，你可能需要更复杂的策略更新机制
    }

    /// 更新群组事件通知策略
    #[allow(dead_code)]
    pub async fn update_group_policy(&self, new_group_policy: GroupNotificationPolicy) {
        info!("更新群组事件通知策略: {:?}", new_group_policy);
    }

    /// 获取当前策略摘要
    pub fn get_policy_summary(&self) -> String {
        let policy = self.policy_service.get_policy();
        let group_policy = self.policy_service.get_group_policy();
        format!(
            "用户状态通知策略: 好友通知={}, 附近用户通知={}({}km), 全局广播={}; 群组事件通知策略: 成员加入/离开={}, 群组消息={}, 群组设置={}",
            policy.enable_friend_notifications,
            policy.enable_nearby_notifications,
            policy.nearby_radius_km,
            policy.enable_global_broadcast,
            group_policy.enable_member_join_leave_notifications,
            group_policy.enable_group_message_notifications,
            group_policy.enable_group_settings_notifications
        )
    }

    /// 创建生产环境推荐的策略
    pub fn production_policy() -> NotificationPolicy {
        NotificationPolicy {
            enable_friend_notifications: true,
            enable_nearby_notifications: true,
            nearby_radius_km: 3.0, // 生产环境使用较小的半径
            enable_global_broadcast: false, // 生产环境禁用全局广播
        }
    }

    /// 创建开发环境推荐的策略
    pub fn development_policy() -> NotificationPolicy {
        NotificationPolicy {
            enable_friend_notifications: true,
            enable_nearby_notifications: true,
            nearby_radius_km: 10.0, // 开发环境使用较大的半径便于测试
            enable_global_broadcast: true, // 开发环境可以启用全局广播便于调试
        }
    }

    /// 创建生产环境推荐的群组策略
    pub fn production_group_policy() -> GroupNotificationPolicy {
        GroupNotificationPolicy {
            enable_member_join_leave_notifications: true,
            enable_group_message_notifications: true,
            enable_group_settings_notifications: true,
        }
    }

    /// 创建开发环境推荐的群组策略
    pub fn development_group_policy() -> GroupNotificationPolicy {
        GroupNotificationPolicy {
            enable_member_join_leave_notifications: true,
            enable_group_message_notifications: true,
            enable_group_settings_notifications: true,
        }
    }
}
