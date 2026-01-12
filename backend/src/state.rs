use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;
use aws_sdk_s3::{Client as S3Client, config::Region};

use crate::{
    config::Config,
    events::event_bus::EventBus,
    repositories::{
        admin_rbac_repo::AdminRbacRepository, user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository,
    },
    services::{
        admin_rbac_service::AdminRbacService,
        announcement_service::AnnouncementService,
        auth_service::AuthService, comment_service::CommentService,
        device_tokens_service::DeviceTokensService, feed_service::FeedService,
        group_message_service::GroupMessageService, group_service::GroupService,
        map_service::MapService,
        media_service::MediaService, message_common_service::MessageCommonService,
        moment_service::MomentService, presence_service::PresenceService,
        private_chat_service::PrivateChatService, private_message_service::PrivateMessageService,
        report_service::ReportService, subscription_service::SubscriptionService,
        system_config_service::SystemConfigService, tag_service::TagService,
        user_service::UserService,
    },
    websocket::{
        manager::ConnectionManager as WsConnectionManager, subscription::SubscriptionManager,
    },
};

#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    pub redis: ConnectionManager,
    // redis_client removed as it is internal to SystemConfigService
    pub config: Config,
    pub ws_manager: WsConnectionManager,
    pub ws_subscription: SubscriptionManager,
    // event_bus removed as it is internal to Services
    // Services
    pub auth_service: AuthService,
    pub user_service: UserService,
    pub moment_service: MomentService,
    pub comment_service: CommentService,
    pub group_service: GroupService,
    pub group_message_service: GroupMessageService,
    pub map_service: MapService,
    pub private_chat_service: PrivateChatService,
    pub private_message_service: PrivateMessageService,
    pub message_common_service: MessageCommonService,
    pub media_service: MediaService,
    pub tag_service: TagService,
    pub feed_service: FeedService,
    pub device_tokens_service: DeviceTokensService,
    pub subscription_service: SubscriptionService,
    pub report_service: ReportService,
    pub system_config_service: SystemConfigService,
    pub presence_service: PresenceService,
    pub admin_rbac_service: AdminRbacService,
    pub notification_service: crate::services::notification_service::NotificationService,
    pub announcement_service: AnnouncementService,
}

impl AppState {
    pub async fn new(
        config: Config,
        event_bus: Arc<EventBus>,
    ) -> Result<Self, Box<dyn std::error::Error>> {
        // 初始化数据库连接池
        let db = PgPool::connect(&config.database_url).await?;

        tracing::debug!("Database connected");

        // 自动运行数据库迁移
        tracing::info!("Running database migrations...");
        sqlx::migrate!("./migrations")
            .run(&db)
            .await
            .map_err(|e| {
                tracing::error!("Failed to run database migrations: {}", e);
                e
            })?;
        tracing::info!("Database migrations applied successfully");

        // 初始化 Redis 连接
        let redis_client = redis::Client::open(config.redis_url.as_str())?;
        let redis = ConnectionManager::new(redis_client.clone()).await?;
        let redis_client = Arc::new(redis_client);

        tracing::debug!("Redis connected");

        // 初始化 WebSocket 管理器
        let ws_manager = WsConnectionManager::new();
        let ws_subscription = SubscriptionManager::new();

        tracing::debug!("WebSocket managers initialized");

        // 初始化所有 Service
        let auth_service = AuthService::new(
            &db,
            &redis,
            config.jwt_secret.clone(),
            config.jwt_access_token_expiration,
            config.jwt_refresh_token_expiration,
        );
        let user_service = UserService::new(&db, &redis, event_bus.clone());
        let moment_service = MomentService::new(&db, &redis, event_bus.clone());
        let comment_service = CommentService::new(&db, &redis, event_bus.clone());
        let group_service = GroupService::new(&db, &redis, event_bus.clone());
        let group_message_service = GroupMessageService::new(&db, &redis, event_bus.clone());
        let private_chat_service = PrivateChatService::new(db.clone(), &redis);
        let private_message_service = PrivateMessageService::new(db.clone(), event_bus.clone());
        let message_common_service = MessageCommonService::new(db.clone());

        // Init S3 Clients
        // 直接使用配置的区域，避免在本地开发环境中尝试从 IMDS 加载区域（会产生警告）
        let region = Region::new(config.s3_region.clone());
        let shared_config = aws_config::defaults(aws_config::BehaviorVersion::latest())
            .region(region)
            .load()
            .await;

        let s3_internal_config = aws_sdk_s3::config::Builder::from(&shared_config)
            .endpoint_url(config.s3_endpoint.clone())
            .force_path_style(true)
            .credentials_provider(aws_credential_types::Credentials::new(
                config.s3_access_key.clone(),
                config.s3_secret_key.clone(),
                None,
                None,
                "Static",
            ))
            .build();
        let s3_internal = S3Client::from_conf(s3_internal_config);

        // Public Endpoint Handling: strip bucket name from S3_PUBLIC_URL if present to get the base endpoint
        // e.g. http://localhost:9000/hotspot -> http://localhost:9000
        let s3_bucket = config.s3_bucket.clone();
        let public_url = config.s3_public_url.clone();
        let public_endpoint = if public_url.ends_with(&format!("/{}", s3_bucket)) {
            public_url.trim_end_matches(&format!("/{}", s3_bucket)).to_string()
        } else {
            public_url
        };

        let s3_public_config = aws_sdk_s3::config::Builder::from(&shared_config)
            .endpoint_url(public_endpoint)
            .force_path_style(true)
            .credentials_provider(aws_credential_types::Credentials::new(
                config.s3_access_key.clone(),
                config.s3_secret_key.clone(),
                None,
                None,
                "Static",
            ))
            .build();
        let s3_public = S3Client::from_conf(s3_public_config);

        let media_service = MediaService::new(
            &db,
            s3_internal,
            s3_public,
            s3_bucket,
        );
        let tag_service = TagService::new(crate::repositories::tag_repo::TagRepository::new(
            db.clone(),
        ));
        let feed_service = FeedService::new(db.clone(), redis.clone());
        let device_tokens_service = DeviceTokensService::new(db.clone());
        let subscription_service = SubscriptionService::new(db.clone(), redis.clone());
        let report_service = ReportService::new(db.clone());
        let system_config_service = SystemConfigService::new(db.clone(), redis_client.clone());
        let presence_service = PresenceService::new(redis.clone(), db.clone());
        let admin_rbac_service = AdminRbacService::new(
            AdminRbacRepository::new(db.clone()),
            UserRepository::new(db.clone()),
            UserSettingsRepository::new(db.clone()),
        );
        let map_service = MapService::new(crate::repositories::map_repo::MapRepository::new(
            db.clone(),
        ));
        let notification_service =
            crate::services::notification_service::NotificationService::new(&db, &redis, event_bus.clone());
        let announcement_service = AnnouncementService::new(
            crate::repositories::announcement_repo::AnnouncementRepository::new(db.clone()),
            event_bus.clone(),
        );

        tracing::debug!("All services initialized");

        // 启动时确保超级管理员存在
        // 规则：
        // 1. 如果已存在超级管理员，直接通过
        // 2. 如果不存在超级管理员，必须通过 SUPER_ADMIN_PASSWORD 环境变量创建，否则阻塞启动
        // 这个逻辑是幂等的：仅在没有任何超级管理员时才会创建/升级
        admin_rbac_service
            .ensure_super_admin_bootstrap(&config)
            .await
            .map_err(|e| {
                tracing::error!("Failed to bootstrap super admin: {}", e);
                e
            })?;

        // Return AppState
        Ok(Self {
            db,
            redis,
            config,
            ws_manager,
            ws_subscription,
            auth_service,
            user_service,
            moment_service,
            comment_service,
            group_service,
            group_message_service,
            map_service,
            private_chat_service,
            private_message_service,
            message_common_service,
            media_service,
            tag_service,
            feed_service,
            device_tokens_service,
            subscription_service,
            report_service,
            system_config_service,
            presence_service,
            admin_rbac_service,
            notification_service,
            announcement_service,
        })
    }
}
