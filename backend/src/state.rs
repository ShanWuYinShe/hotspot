use aws_sdk_s3::{config::Region, Client as S3Client};
use moka::future::Cache;
use redis::aio::ConnectionManager;
use sqlx::{postgres::PgPoolOptions, PgPool};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::broadcast;

use crate::{
	config::Config,
	events::event_bus::EventBus,
	repositories::{
		admin::audit_repo::AdminAuditRepository,
		admin::rbac_repo::AdminRbacRepository,
		announcement_repo::AnnouncementRepository,
		chat_unread_repo::ChatUnreadRepository,
		comment_repo::CommentRepository,
		dismiss_repo::FeedDismissRepository,
		group::announcement_repo::GroupAnnouncementRepository,
		group::application_repo::GroupApplicationRepository,
		group::bans_repo::GroupBansRepository,
		group::invite_repo::GroupInviteRepository,
		group::member_history_repo::GroupMemberHistoryRepository,
		group::member_repo::GroupMemberRepository,
		group::message_repo::GroupMessageRepository,
		group::settings_repo::GroupSettingsRepository,
		group_repo::GroupRepository,
		map_repo::MapRepository,
		moment_repo::MomentRepository,
		notification_repo::NotificationRepository,
		private_chat_repo::PrivateChatRepository,
		private_message_repo::PrivateMessageRepository,
		report_repo::ReportRepository,
		session_repo::SessionRepository,
		system_config_repo::SystemConfigRepository,
		tag_repo::TagRepository,
		user::blocks_repo::UserBlocksRepository,
		user::follows_repo::UserFollowsRepository,
		user::friend_remarks_repo::UserFriendRemarksRepository,
		user::location_history_repo::UserLocationHistoryRepository,
			user::online_status_repo::UserOnlineStatusRepository,
			user::settings_repo::UserSettingsRepository,
			user_repo::UserRepository,
			verification_repo::VerificationRepository,
			region_boundary_repo::RegionBoundaryRepo,
			region_boundary_stats_repo::RegionBoundaryStatsRepository,
	},
	services::{
		admin_audit_service::AdminAuditService,
			admin_rbac_service::AdminRbacService,
			announcement_service::AnnouncementService,
			region_boundary_service::RegionBoundaryService,
		auth_service::AuthService,
		comment_service::CommentService,
		feed_service::FeedService,
		group_message_service::GroupMessageService,
		group_service::GroupService,
		map_service::MapService,
		media_service::MediaService,
		moment_service::MomentService,
		notification_service::NotificationService,
		presence_service::PresenceService,
		private_chat_service::PrivateChatService,
		private_message_service::PrivateMessageService,
		report_service::ReportService,
		search_service::SearchService,
		share_service::ShareService,
		stranger_message_limiter::StrangerMessageLimiter,
		system_config_service::SystemConfigService,
		tag_service::TagService,
		user_service::UserService,
	},
	utils::cache_invalidation::{start_invalidation_listener, InvalidationEvent},
	websocket::{
		manager::ConnectionManager as WsConnectionManager,
		subscription::SubscriptionManager,
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
    // 缓存和防护工具
    pub resource_checker: crate::utils::bloom_filter::ResourceExistenceChecker,
    pub cache_manager: crate::utils::cache::CacheManager,
    /// L1 本地缓存：Token -> (UserId, SessionId)
    /// TTL: 60s (短 TTL 配合 Redis Pub/Sub 实现分布式一致性)
    pub token_cache: Cache<String, (i64, uuid::Uuid)>,
    /// L1 本地黑名单：SessionId -> ()
    /// 用于处理分布式环境下的会话撤销，TTL 略大于 token_cache (e.g. 70s)
    pub revoked_sessions: Cache<String, ()>,

    // event_bus removed as it is internal to Services
    // Services
    pub auth_service: AuthService,
    pub user_service: UserService,
    pub moment_service: MomentService,
    pub comment_service: CommentService,
    pub group_service: GroupService,
    pub group_message_service: GroupMessageService,
    pub map_service: MapService,
    pub notification_service: NotificationService,
    pub private_chat_service: PrivateChatService,
    pub private_message_service: PrivateMessageService,
    pub media_service: MediaService,
    pub tag_service: TagService,
    pub feed_service: FeedService,
    pub report_service: ReportService,
    pub share_service: ShareService,
    pub system_config_service: SystemConfigService,
    pub presence_service: PresenceService,
    pub admin_rbac_service: AdminRbacService,
    pub admin_audit_service: AdminAuditService,
    pub announcement_service: AnnouncementService,
    pub search_service: SearchService,
}

impl AppState {
    pub async fn new(
        config: Config,
        event_bus: Arc<EventBus>,
    ) -> Result<Self, Box<dyn std::error::Error>> {
        // 初始化数据库连接池（使用配置参数，适配分布式部署）
        let db = PgPoolOptions::new()
            .max_connections(config.db_pool_max_connections)
            .min_connections(config.db_pool_min_connections)
            .acquire_timeout(Duration::from_secs(config.db_pool_acquire_timeout_secs))
            .idle_timeout(Duration::from_secs(config.db_pool_idle_timeout_secs))
            .max_lifetime(Duration::from_secs(config.db_pool_max_lifetime_secs))
            .test_before_acquire(config.db_pool_test_before_acquire)
            .connect(&config.database_url)
            .await?;

        tracing::info!(
            max_connections = config.db_pool_max_connections,
            min_connections = config.db_pool_min_connections,
            "Database connection pool initialized"
        );

        // 初始化 Redis 连接
        let redis_client = redis::Client::open(config.redis_url.as_str())?;
        let redis = ConnectionManager::new(redis_client.clone()).await?;
        let redis_client = Arc::new(redis_client);

        tracing::debug!("Redis connected");

        // 初始化缓存和防护工具
        let resource_checker =
            crate::utils::bloom_filter::ResourceExistenceChecker::new(redis.clone());
        let cache_manager = crate::utils::cache::CacheManager::new(redis.clone());

        // 初始化 L1 Token 缓存 (TTL 60s)
        let token_cache = Cache::builder()
            .time_to_live(std::time::Duration::from_secs(60))
            .max_capacity(100_000)
            .build();

        // 初始化 L1 会话黑名单 (TTL 70s)
        let revoked_sessions = Cache::builder()
            .time_to_live(std::time::Duration::from_secs(70))
            .max_capacity(10_000)
            .build();

        let media_signed_url_cache = Cache::builder()
            .time_to_live(std::time::Duration::from_secs(300))
            .max_capacity(20_000)
            .build();

        // 启动 Redis 缓存失效监听器（增大容量以防消息丢失）
        let (tx, mut rx) = broadcast::channel::<InvalidationEvent>(1000);
        let _listener_handle = start_invalidation_listener(config.redis_url.clone(), tx);

        // 启动本地缓存清理任务
        let revoked_sessions_clone = revoked_sessions.clone();
        tokio::spawn(async move {
            tracing::info!("Local cache invalidation handler started");
            while let Ok(event) = rx.recv().await {
                match event {
                    InvalidationEvent::SessionRevoked {
                        user_id,
                        session_id,
                    } => {
                        tracing::debug!(user_id, %session_id, "Received session revocation event, adding to local blacklist");
                        // 将撤销的 Session ID 加入本地黑名单
                        revoked_sessions_clone.insert(session_id, ()).await;
                    }
                    InvalidationEvent::UserProfileUpdated { user_id } => {
                        tracing::debug!(user_id, "Received user profile update event");
                        // 用户资料更新事件（L1 缓存暂时只处理会话，用户资料走 L2 Redis）
                    }
                    InvalidationEvent::GroupInfoUpdated { group_id } => {
                        tracing::debug!(%group_id, "Received group info update event");
                        // 群组信息更新事件
                    }
                    InvalidationEvent::GroupMemberChanged { group_id } => {
                        tracing::debug!(%group_id, "Received group member change event");
                        // 群组成员变更事件
                    }
                    InvalidationEvent::MomentUpdated { moment_id } => {
                        tracing::debug!(%moment_id, "Received moment update event");
                        // 瞬间内容更新事件
                    }
                    InvalidationEvent::MomentDeleted { moment_id } => {
                        tracing::debug!(%moment_id, "Received moment delete event");
                        // 瞬间删除事件
                    }
                    InvalidationEvent::CommentChanged {
                        moment_id,
                        comment_id,
                    } => {
                        tracing::debug!(%moment_id, %comment_id, "Received comment change event");
                        // 评论变更事件
                    }
                }
            }
        });

        tracing::debug!("Cache and bloom filter initialized");

        // 初始化 WebSocket 管理器
        let ws_manager = WsConnectionManager::new();
        let ws_subscription = SubscriptionManager::new();

        tracing::debug!("WebSocket managers initialized");

        // Init S3 Client
        let region = Region::new(config.s3_region.clone());
        let shared_config = aws_config::defaults(aws_config::BehaviorVersion::latest())
            .region(region)
            .load()
            .await;

        let s3_config = aws_sdk_s3::config::Builder::from(&shared_config)
            .endpoint_url(config.s3_endpoint.clone())
            .force_path_style(config.s3_force_path_style)
            .credentials_provider(aws_credential_types::Credentials::new(
                config.s3_access_key.clone(),
                config.s3_secret_key.clone(),
                None,
                None,
                "Static",
            ))
            .build();
        let s3_client = S3Client::from_conf(s3_config);

        // --- Repositories Initialization ---
        let admin_audit_repo = AdminAuditRepository::new(db.clone());
        let admin_rbac_repo = AdminRbacRepository::new(db.clone());
        let announcement_repo = AnnouncementRepository::new(db.clone());
        let chat_unread_repo = ChatUnreadRepository::new(db.clone());
        let comment_repo = CommentRepository::new(db.clone());
        let file_repo = crate::repositories::file_repo::FileRepository::new(db.clone());
        let group_announcement_repo = GroupAnnouncementRepository::new(db.clone());
        let group_application_repo = GroupApplicationRepository::new(db.clone());
        let group_bans_repo = GroupBansRepository::new(db.clone());
        let group_invite_repo = GroupInviteRepository::new(db.clone());
        let group_member_history_repo = GroupMemberHistoryRepository::new(db.clone());
        let group_member_repo = GroupMemberRepository::new(db.clone(), redis.clone());
		let group_message_repo = GroupMessageRepository::new(db.clone());
		let group_repo = GroupRepository::new(db.clone());
		let group_settings_repo = GroupSettingsRepository::new(db.clone());
		let map_repo = MapRepository::new(db.clone());
		let moment_repo = MomentRepository::new(db.clone());
		let region_boundary_repo = RegionBoundaryRepo::new(db.clone());
		let region_boundary_stats_repo = RegionBoundaryStatsRepository::new(db.clone());
		let notification_repo = NotificationRepository::new(db.clone());
        let private_chat_repo = PrivateChatRepository::new(db.clone());
        let private_message_repo = PrivateMessageRepository::new(db.clone());
        let report_repo = ReportRepository::new(db.clone());
        let feed_dismiss_repo = FeedDismissRepository::new(db.clone());
        let session_repo = SessionRepository::new(db.clone());
        // subscription_repo not yet implemented
        let system_config_repo = SystemConfigRepository::new(db.clone());
        let tag_repo = TagRepository::new(db.clone());
        let user_blocks_repo = UserBlocksRepository::new(db.clone());
        let user_follows_repo = UserFollowsRepository::new(db.clone());
        let user_friend_remarks_repo = UserFriendRemarksRepository::new(db.clone());
        let user_location_history_repo = UserLocationHistoryRepository::new(db.clone());
        let user_online_status_repo = UserOnlineStatusRepository::new(db.clone());
        let user_repo = UserRepository::new(db.clone());
        let user_settings_repo = UserSettingsRepository::new(db.clone());
        let verification_repo = VerificationRepository::new(db.clone());

        // --- Services Initialization ---

        let media_service = MediaService::new(
            file_repo.clone(),
            s3_client,
            config.s3_bucket.clone(),
            config.s3_public_base.clone(),
            redis.clone(),
            media_signed_url_cache.clone(),
        );

        let presence_service = PresenceService::new(
            redis.clone(),
            user_online_status_repo.clone(),
            user_settings_repo.clone(),
        );

		let auth_service = AuthService::new(
			&redis,
			config.jwt_secret.clone(),
            config.jwt_access_token_expiration,
            config.jwt_refresh_token_expiration,
            user_repo.clone(),
            session_repo.clone(),
            user_settings_repo.clone(),
            verification_repo.clone(),
            config.auth_reset_delivery_webhook_url.clone(),
			config.auth_reset_delivery_webhook_token.clone(),
			config.auth_reset_delivery_timeout_ms,
		);

		let region_boundary_service = RegionBoundaryService::new(region_boundary_repo.clone());

		let user_service = UserService::new(
				event_bus.clone(),
				user_repo.clone(),
            user_settings_repo.clone(),
            user_location_history_repo.clone(),
            user_follows_repo.clone(),
            user_blocks_repo.clone(),
            user_friend_remarks_repo.clone(),
            session_repo.clone(),
            tag_repo.clone(),
            group_member_repo.clone(),
				moment_repo.clone(),
				presence_service.clone(),
				media_service.clone(),
				cache_manager.clone(),
				resource_checker.clone(),
				region_boundary_service.clone(),
		);

		let moment_service = MomentService::new(
			redis.clone(),
			event_bus.clone(),
			moment_repo.clone(),
			tag_repo.clone(),
			user_repo.clone(),
			user_settings_repo.clone(),
			media_service.clone(),
			cache_manager.clone(),
			region_boundary_service.clone(),
		);

        let comment_service = CommentService::new(
            event_bus.clone(),
            comment_repo.clone(),
            moment_repo.clone(),
            user_repo.clone(),
            media_service.clone(),
            resource_checker.clone(),
        );

		let group_service = GroupService::new(
			event_bus.clone(),
			group_repo.clone(),
			group_member_repo.clone(),
			group_member_history_repo.clone(),
			group_application_repo.clone(),
			group_bans_repo.clone(),
			group_invite_repo.clone(),
			group_settings_repo.clone(),
			group_announcement_repo.clone(),
			tag_repo.clone(),
			report_repo.clone(),
			user_repo.clone(),
			chat_unread_repo.clone(),
			admin_rbac_repo.clone(),
			media_service.clone(),
			cache_manager.clone(),
			resource_checker.clone(),
			region_boundary_service.clone(),
		);

        let group_message_service = GroupMessageService::new(
            redis.clone(),
            event_bus.clone(),
            group_message_repo.clone(),
            group_member_repo.clone(),
            group_repo.clone(),
            user_repo.clone(),
            chat_unread_repo.clone(),
            media_service.clone(),
        );

        let stranger_limiter = StrangerMessageLimiter::new(
            redis.clone(),
            crate::constants::STRANGER_MESSAGE_LIMIT_PER_DAY,
        );

        let private_chat_service = PrivateChatService::new(
            private_chat_repo.clone(),
            user_settings_repo.clone(),
            user_blocks_repo.clone(),
            user_follows_repo.clone(),
            user_repo.clone(),
            private_message_repo.clone(),
            chat_unread_repo.clone(),
            presence_service.clone(),
            media_service.clone(),
            stranger_limiter.clone(),
            cache_manager.clone(),
        );

        let private_message_service = PrivateMessageService::new(
            event_bus.clone(),
            private_message_repo.clone(),
            private_chat_repo.clone(),
            user_repo.clone(),
            chat_unread_repo.clone(),
            user_blocks_repo.clone(),
            user_settings_repo.clone(),
            user_follows_repo.clone(),
            media_service.clone(),
            stranger_limiter.clone(),
            resource_checker.clone(),
        );

        let tag_service = TagService::new(tag_repo.clone());

        let feed_service = FeedService::new(
            moment_repo.clone(),
            user_repo.clone(),
            group_repo.clone(),
            tag_repo.clone(),
            user_follows_repo.clone(),
            group_member_repo.clone(),
            presence_service.clone(),
            media_service.clone(),
            feed_dismiss_repo.clone(),
        );

        let system_config_service = SystemConfigService::new(
            system_config_repo.clone(),
            redis_client.clone(),
            cache_manager.clone(),
        );

        let admin_rbac_service = AdminRbacService::new(
            redis.clone(),
            admin_rbac_repo.clone(),
            user_repo.clone(),
            user_settings_repo.clone(),
        );

        let admin_audit_service =
            AdminAuditService::new(admin_audit_repo, user_repo.clone(), media_service.clone());

		let report_service = ReportService::new(
			report_repo.clone(),
			user_repo.clone(),
			moment_repo.clone(),
			comment_repo.clone(),
			group_repo.clone(),
			private_message_repo.clone(),
			group_message_repo.clone(),
			media_service.clone(),
			user_service.clone(),
			admin_rbac_service.clone(),
			event_bus.clone(),
		);

		let map_service = MapService::new(
			region_boundary_service.clone(),
			region_boundary_stats_repo.clone(),
			map_repo.clone(),
		);

        let notification_service =
            NotificationService::new(notification_repo.clone(), user_repo.clone());

        let announcement_service = AnnouncementService::new(
            announcement_repo.clone(),
            user_repo.clone(),
            media_service.clone(),
            event_bus.clone(),
        );

        let search_service = SearchService::new(
            user_repo.clone(),
            group_repo.clone(),
            tag_repo.clone(),
            moment_repo.clone(),
            media_service.clone(),
        );

        let share_service = ShareService::new(
            db.clone(),
            redis.clone(),
            std::sync::Arc::new(media_service.clone()),
            cache_manager.clone(),
            moment_repo.clone(),
            user_repo.clone(),
            group_repo.clone(),
        );

        tracing::debug!("All services initialized");

        // 启动时确保超级管理员存在
        admin_rbac_service
            .ensure_super_admin_bootstrap(&config)
            .await
            .map_err(|e| {
                tracing::error!("Failed to bootstrap super admin: {}", e);
                e
            })?;

        // 后台初始化布隆过滤器（避免阻塞启动）
        {
            let db = db.clone();
            let checker = resource_checker.clone();
            tokio::spawn(async move {
                if let Err(e) = warmup_bloom_filters(db, checker).await {
                    tracing::warn!("Bloom filter warmup failed: {}", e);
                }
            });
        }

        {
            let db = db.clone();
            let checker = resource_checker.clone();
            tokio::spawn(async move {
                let mut interval = tokio::time::interval(Duration::from_secs(60));
                interval.tick().await;
                loop {
                    interval.tick().await;
                    match checker.is_ready().await {
                        Ok(true) => {}
                        Ok(false) => {
                            tracing::warn!("Bloom filter marked unready, attempting rebuild");
                            if let Err(e) = warmup_bloom_filters(db.clone(), checker.clone()).await {
                                tracing::warn!("Bloom filter rebuild failed: {}", e);
                            }
                        }
                        Err(e) => {
                            tracing::warn!("Bloom filter readiness check failed: {}", e);
                        }
                    }
                }
            });
        }

        // Return AppState
        Ok(Self {
            db,
            redis,
            config,
            ws_manager,
            ws_subscription,
            resource_checker,
            cache_manager,
            auth_service,
            user_service,
            moment_service,
            comment_service,
            group_service,
            group_message_service,
            map_service,
            notification_service,
            private_chat_service,
            private_message_service,
            media_service,
            tag_service,
            feed_service,
            report_service,
            share_service,
            system_config_service,
            presence_service,
            admin_rbac_service,
            admin_audit_service,
            announcement_service,
            search_service,
            token_cache,
            revoked_sessions,
        })
    }
}

macro_rules! warmup_ids_loop {
    ($last_var:ident, $repo:expr, $list_fn:ident, $checker:expr, $sync_fn:ident, $batch:expr) => {
        let mut $last_var = 0i64;
        loop {
            let rows = $repo.$list_fn($last_var, $batch).await?;
            if rows.is_empty() {
                break;
            }
            let ids: Vec<String> = rows.iter().map(|(_, x)| x.to_string()).collect();
            $checker.$sync_fn(&ids).await?;
            if let Some((last_id, _)) = rows.last() {
                $last_var = *last_id;
            }
        }
    };
}

async fn warmup_bloom_filters(
    db: PgPool,
    checker: crate::utils::bloom_filter::ResourceExistenceChecker,
) -> crate::error::Result<()> {
    if checker.is_ready().await.unwrap_or(false) {
        return Ok(());
    }

    // 清空旧过滤器，避免脏数据影响初始化
    checker.clear_all().await?;

    const BATCH_SIZE: i64 = 1000;

    let user_repo = UserRepository::new(db.clone());
    let group_repo = GroupRepository::new(db.clone());
    let moment_repo = MomentRepository::new(db.clone());
    let comment_repo = CommentRepository::new(db.clone());
    let group_message_repo = GroupMessageRepository::new(db.clone());
    let private_message_repo = PrivateMessageRepository::new(db.clone());

    // 用户（sync_users 接受 &[i64]，与其他实体不同）
    let mut last_user_id = 0;
    loop {
        let ids = user_repo
            .list_user_ids_after(last_user_id, BATCH_SIZE)
            .await?;
        if ids.is_empty() {
            break;
        }
        checker.sync_users(&ids).await?;
        if let Some(last) = ids.last() {
            last_user_id = *last;
        }
    }

    // 群组 / 瞬间 / 评论 / 消息（复用同一循环模式）
    warmup_ids_loop!(
        last_group_row_id,
        group_repo,
        list_group_ids_after,
        checker,
        sync_groups,
        BATCH_SIZE
    );
    warmup_ids_loop!(
        last_moment_row_id,
        moment_repo,
        list_moment_ids_after,
        checker,
        sync_moments,
        BATCH_SIZE
    );
    warmup_ids_loop!(
        last_comment_row_id,
        comment_repo,
        list_comment_ids_after,
        checker,
        sync_comments,
        BATCH_SIZE
    );
    warmup_ids_loop!(
        last_group_msg_id,
        group_message_repo,
        list_message_ids_after,
        checker,
        sync_messages,
        BATCH_SIZE
    );
    warmup_ids_loop!(
        last_private_msg_id,
        private_message_repo,
        list_message_ids_after,
        checker,
        sync_messages,
        BATCH_SIZE
    );

    checker.set_ready().await?;
    checker.log_filter_sizes().await;
    tracing::info!("Bloom filter warmup completed");
    Ok(())
}
