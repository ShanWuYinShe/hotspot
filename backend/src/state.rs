use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;

use crate::{
    config::Config,
    events::event_bus::EventBus,
    websocket::{
        manager::ConnectionManager as WsConnectionManager,
        subscription::SubscriptionManager,
    },
};

#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    pub redis: ConnectionManager,
    pub redis_client: Arc<redis::Client>,
    pub config: Config,
    pub ws_manager: WsConnectionManager,
    pub ws_subscription: SubscriptionManager,
    pub event_bus: Arc<EventBus>,
}

impl AppState {
    pub async fn new(config: Config, event_bus: Arc<EventBus>) -> Result<Self, Box<dyn std::error::Error>> {
        // 初始化数据库连接池
        let db = PgPool::connect(&config.database_url).await?;
        
        tracing::info!("Database connected successfully");

        // 初始化 Redis 连接
        let redis_client = redis::Client::open(config.redis_url.as_str())?;
        let redis = ConnectionManager::new(redis_client.clone()).await?;
        let redis_client = Arc::new(redis_client);
        
        tracing::info!("Redis connected successfully");

        // 初始化 WebSocket 管理器
        let ws_manager = WsConnectionManager::new();
        let ws_subscription = SubscriptionManager::new();
        
        tracing::info!("WebSocket managers initialized");

        Ok(Self {
            db,
            redis,
            redis_client,
            config,
            ws_manager,
            ws_subscription,
            event_bus,
        })
    }
}

