use std::sync::Arc;

use crate::services::{ConfigService, GroupService, HealthService, MessageService, MomentService, UserService};
use crate::repositories::PgUserRepository;
use crate::utils::metrics::AppMetrics;
use crate::utils::events::EventBus;
use crate::ws::manager::WebSocketManager;
use crate::services::notification::WebSocketNotificationService;
use super::auth::AppState;

/// AppState 构建器，便于后续扩展注入新能力
pub struct AppStateBuilder {
    config_service: Arc<ConfigService>,
    user_service: Arc<UserService>,
    user_repository: Arc<PgUserRepository>,
    group_service: Arc<GroupService>,
    message_service: Arc<MessageService>,
    moment_service: Arc<MomentService>,
    health_service: Arc<HealthService>,
    websocket_notification_service: Arc<WebSocketNotificationService>,
    ws_manager: Arc<WebSocketManager>,
    metrics: Arc<AppMetrics>,
    event_bus: Arc<EventBus>,
    #[cfg(feature = "jobs")]
    job_service: Option<Arc<crate::jobs::JobService>>,
}

impl AppStateBuilder {
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        config_service: Arc<ConfigService>,
        user_service: Arc<UserService>,
        user_repository: Arc<PgUserRepository>,
        group_service: Arc<GroupService>,
        message_service: Arc<MessageService>,
        moment_service: Arc<MomentService>,
        health_service: Arc<HealthService>,
        websocket_notification_service: Arc<WebSocketNotificationService>,
        ws_manager: Arc<WebSocketManager>,
        metrics: Arc<AppMetrics>,
    ) -> Self {
        Self {
            config_service,
            user_service,
            user_repository,
            group_service,
            message_service,
            moment_service,
            health_service,
            websocket_notification_service,
            ws_manager,
            metrics,
            event_bus: Arc::new(EventBus::new()),
            #[cfg(feature = "jobs")]
            job_service: None,
        }
    }

    #[cfg(feature = "jobs")]
    pub fn with_job_service(mut self, job_service: Option<Arc<crate::jobs::JobService>>) -> Self {
        self.job_service = job_service;
        self
    }

    pub fn build(self) -> Arc<AppState> {
        Arc::new(AppState {
            config_service: self.config_service,
            user_service: self.user_service,
            user_repository: self.user_repository,
            group_service: self.group_service,
            message_service: self.message_service,
            moment_service: self.moment_service,
            health_service: self.health_service,
            websocket_notification_service: self.websocket_notification_service,
            ws_manager: self.ws_manager,
            #[cfg(feature = "jobs")]
            job_service: self.job_service,
            metrics: self.metrics,
            event_bus: self.event_bus,
        })
    }
}


