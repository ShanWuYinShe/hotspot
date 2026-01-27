use aws_sdk_s3::{config::Region, Client as S3Client};
use redis::aio::ConnectionManager;
use sqlx::PgPool;
use std::sync::Arc;

use crate::{
    config::Config,
    events::event_bus::EventBus,
    repositories::{
        admin_audit_repo::AdminAuditRepository, admin_rbac_repo::AdminRbacRepository,
        announcement_repo::AnnouncementRepository, chat_unread_repo::ChatUnreadRepository,
        comment_repo::CommentRepository, device_token_repo::DeviceTokenRepository,
        feed_repo::FeedRepository, follow_request_repo::FollowRequestRepository,
        group_announcement_repo::GroupAnnouncementRepository,
        group_application_repo::GroupApplicationRepository, group_bans_repo::GroupBansRepository,
        group_invite_repo::GroupInviteRepository,
        group_member_history_repo::GroupMemberHistoryRepository,
        group_member_repo::GroupMemberRepository, group_message_repo::GroupMessageRepository,
        group_repo::GroupRepository, group_settings_repo::GroupSettingsRepository,
        map_repo::MapRepository, moment_repo::MomentRepository,
        notification_repo::NotificationRepository, password_reset_repo::PasswordResetRepository,
        private_chat_repo::PrivateChatRepository, private_message_repo::PrivateMessageRepository,
        report_repo::ReportRepository, session_repo::SessionRepository,
        system_config_repo::SystemConfigRepository, tag_repo::TagRepository,
        user_blocks_repo::UserBlocksRepository, user_follows_repo::UserFollowsRepository,
        user_friend_remarks_repo::UserFriendRemarksRepository,
        user_location_history_repo::UserLocationHistoryRepository,
        user_online_status_repo::UserOnlineStatusRepository, user_repo::UserRepository,
        user_settings_repo::UserSettingsRepository, verification_repo::VerificationRepository,
    },
    services::{
        admin_audit_service::AdminAuditService, admin_rbac_service::AdminRbacService,
        announcement_service::AnnouncementService, auth_service::AuthService,
        comment_service::CommentService, device_tokens_service::DeviceTokensService,
        feed_service::FeedService, group_message_service::GroupMessageService,
        group_service::GroupService, map_service::MapService, media_service::MediaService,
        message_common_service::MessageCommonService, moment_service::MomentService,
        presence_service::PresenceService, private_chat_service::PrivateChatService,
        private_message_service::PrivateMessageService, report_service::ReportService,
        subscription_service::SubscriptionService, system_config_service::SystemConfigService,
        tag_service::TagService, user_service::UserService,
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
    // 缓存和防护工具
    pub resource_checker: crate::utils::bloom_filter::ResourceExistenceChecker,
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
    pub admin_audit_service: AdminAuditService,
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

        // 初始化 Redis 连接
        let redis_client = redis::Client::open(config.redis_url.as_str())?;
        let redis = ConnectionManager::new(redis_client.clone()).await?;
        let redis_client = Arc::new(redis_client);

        tracing::debug!("Redis connected");

        // 初始化缓存和防护工具
        let resource_checker =
            crate::utils::bloom_filter::ResourceExistenceChecker::new(redis.clone());
        let cache_manager = crate::utils::cache::CacheManager::new(redis.clone());
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
        let device_token_repo = DeviceTokenRepository::new(db.clone());
        let feed_repo = FeedRepository::new(db.clone());
        // file_repo might be needed for MediaService if refactored
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
        let notification_repo = NotificationRepository::new(db.clone());
        let password_reset_repo = PasswordResetRepository::new(db.clone());
        let private_chat_repo = PrivateChatRepository::new(db.clone());
        let private_message_repo = PrivateMessageRepository::new(db.clone());
        let report_repo = ReportRepository::new(db.clone());
        let session_repo = SessionRepository::new(db.clone());
        // subscription_repo not yet implemented
        let system_config_repo = SystemConfigRepository::new(db.clone());
        let tag_repo = TagRepository::new(db.clone());
        let user_blocks_repo = UserBlocksRepository::new(db.clone());
        let user_follows_repo = UserFollowsRepository::new(db.clone());
        let follow_request_repo = FollowRequestRepository::new(db.clone());
        let user_friend_remarks_repo = UserFriendRemarksRepository::new(db.clone());
        let user_location_history_repo = UserLocationHistoryRepository::new(db.clone());
        let user_online_status_repo = UserOnlineStatusRepository::new(db.clone());
        let user_repo = UserRepository::new(db.clone());
        let user_settings_repo = UserSettingsRepository::new(db.clone());
        let verification_repo = VerificationRepository::new(db.clone());
        let message_reactions_repo =
            crate::repositories::message_reactions_repo::MessageReactionsRepository::new(
                db.clone(),
            );

        // --- Services Initialization ---

        let media_service = MediaService::new(
            &db, // Assuming MediaService still takes db directly as I didn't refactor its constructor in this session
            s3_client,
            config.s3_bucket.clone(),
            config.s3_public_base.clone(),
            redis.clone(),
        );

        let presence_service = PresenceService::new(
            redis.clone(),
            user_online_status_repo.clone(),
            user_settings_repo.clone(),
        );

        let auth_service = AuthService::new(
            &db, // Still takes db? Yes, signature in grep output showed &db
            &redis,
            config.jwt_secret.clone(),
            config.jwt_access_token_expiration,
            config.jwt_refresh_token_expiration,
            user_repo.clone(),
            session_repo.clone(),
            password_reset_repo.clone(),
            verification_repo.clone(),
        );

        let user_service = UserService::new(
            event_bus.clone(),
            user_repo.clone(),
            user_settings_repo.clone(),
            user_location_history_repo.clone(),
            user_follows_repo.clone(),
            follow_request_repo.clone(),
            user_blocks_repo.clone(),
            user_friend_remarks_repo.clone(),
            device_token_repo.clone(),
            session_repo.clone(),
            tag_repo.clone(),
            group_member_repo.clone(),
            moment_repo.clone(),
            presence_service.clone(),
            media_service.clone(),
            cache_manager.clone(),
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
        );

        let comment_service = CommentService::new(
            event_bus.clone(),
            redis.clone(),
            comment_repo.clone(),
            moment_repo.clone(),
            user_repo.clone(),
            cache_manager.clone(),
        );

        let group_service = GroupService::new(
            event_bus.clone(),
            group_repo.clone(),
            group_message_repo.clone(),
            group_member_repo.clone(),
            group_member_history_repo.clone(),
            group_application_repo.clone(),
            group_bans_repo.clone(),
            group_invite_repo.clone(),
            group_settings_repo.clone(),
            group_announcement_repo.clone(),
            tag_repo.clone(),
            user_repo.clone(),
            chat_unread_repo.clone(),
            redis.clone(),
            presence_service.clone(),
            media_service.clone(),
            cache_manager.clone(),
        );

        let group_message_service = GroupMessageService::new(
            db.clone(),
            redis.clone(),
            event_bus.clone(),
            group_message_repo.clone(),
            group_member_repo.clone(),
            user_repo.clone(),
            chat_unread_repo.clone(),
            message_reactions_repo.clone(),
            media_service.clone(),
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
            cache_manager.clone(),
        );

        let private_message_service = PrivateMessageService::new(
            event_bus.clone(),
            redis.clone(),
            private_message_repo.clone(),
            private_chat_repo.clone(),
            user_repo.clone(),
            chat_unread_repo.clone(),
            user_follows_repo.clone(),
            user_blocks_repo.clone(),
            user_settings_repo.clone(),
            message_reactions_repo.clone(),
            media_service.clone(),
        );

        let message_common_service = MessageCommonService::new(chat_unread_repo.clone());

        let tag_service = TagService::new(tag_repo.clone(), cache_manager.clone());

        let feed_service = FeedService::new(
            redis.clone(),
            feed_repo.clone(),
            moment_repo.clone(),
            user_repo.clone(),
            group_repo.clone(),
            tag_repo.clone(),
            presence_service.clone(),
            media_service.clone(),
        );

        let device_tokens_service = DeviceTokensService::new(device_token_repo.clone());

        let subscription_service = SubscriptionService::new(db.clone(), redis.clone());

        let report_service = ReportService::new(
            report_repo.clone(),
            user_repo.clone(),
            moment_repo.clone(),
            comment_repo.clone(),
            group_repo.clone(),
            private_message_repo.clone(),
            group_message_repo.clone(),
        );

        let system_config_service = SystemConfigService::new(
            system_config_repo.clone(),
            redis_client.clone(),
            cache_manager.clone(),
        );

        let admin_rbac_service = AdminRbacService::new(
            admin_rbac_repo.clone(),
            user_repo.clone(),
            user_settings_repo.clone(),
        );

        let admin_audit_service = AdminAuditService::new(
            admin_audit_repo,
            user_repo.clone(),
            media_service.clone(),
        );

        let map_service = MapService::new(map_repo.clone());

        let notification_service = crate::services::notification_service::NotificationService::new(
            notification_repo.clone(),
            user_repo.clone(),
            event_bus.clone(),
            media_service.clone(),
        );

        let announcement_service = AnnouncementService::new(
            announcement_repo.clone(),
            event_bus.clone(),
            media_service.clone(),
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

        // Return AppState
        Ok(Self {
            db,
            redis,
            config,
            ws_manager,
            ws_subscription,
            resource_checker,
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
            admin_audit_service,
            notification_service,
            announcement_service,
        })
    }
}

async fn warmup_bloom_filters(
    db: PgPool,
    checker: crate::utils::bloom_filter::ResourceExistenceChecker,
) -> Result<(), Box<dyn std::error::Error>> {
    if checker.is_ready().await.unwrap_or(false) {
        return Ok(());
    }

    // 清空旧过滤器，避免脏数据影响初始化
    checker.clear_all().await?;

    const BATCH_SIZE: i64 = 1000;

    let user_repo = UserRepository::new(db.clone());
    let group_repo = GroupRepository::new(db.clone());
    let moment_repo = MomentRepository::new(db.clone());

    // 用户
    let mut last_user_id = 0;
    loop {
        let ids = user_repo.list_user_ids_after(last_user_id, BATCH_SIZE).await?;
        if ids.is_empty() {
            break;
        }
        checker.sync_users(&ids).await?;
        if let Some(last) = ids.last() {
            last_user_id = *last;
        }
    }

    // 群组
    let mut last_group_row_id = 0;
    loop {
        let rows = group_repo
            .list_group_ids_after(last_group_row_id, BATCH_SIZE)
            .await?;
        if rows.is_empty() {
            break;
        }
        let group_ids: Vec<String> = rows.iter().map(|(_, gid)| gid.to_string()).collect();
        checker.sync_groups(&group_ids).await?;
        if let Some((last_id, _)) = rows.last() {
            last_group_row_id = *last_id;
        }
    }

    // 瞬间
    let mut last_moment_row_id = 0;
    loop {
        let rows = moment_repo
            .list_moment_ids_after(last_moment_row_id, BATCH_SIZE)
            .await?;
        if rows.is_empty() {
            break;
        }
        let moment_ids: Vec<String> = rows.iter().map(|(_, mid)| mid.to_string()).collect();
        checker.sync_moments(&moment_ids).await?;
        if let Some((last_id, _)) = rows.last() {
            last_moment_row_id = *last_id;
        }
    }

    // 评论
    let comment_repo = CommentRepository::new(db.clone());
    let mut last_comment_row_id = 0;
    loop {
        let rows = comment_repo
            .list_comment_ids_after(last_comment_row_id, BATCH_SIZE)
            .await?;
        if rows.is_empty() {
            break;
        }
        let comment_ids: Vec<String> = rows.iter().map(|(_, cid)| cid.to_string()).collect();
        checker.sync_comments(&comment_ids).await?;
        if let Some((last_id, _)) = rows.last() {
            last_comment_row_id = *last_id;
        }
    }

    // 消息（群聊消息 + 私聊消息）
    let group_message_repo = GroupMessageRepository::new(db.clone());
    let private_message_repo = PrivateMessageRepository::new(db.clone());

    let mut last_group_msg_id = 0;
    loop {
        let rows = group_message_repo
            .list_message_ids_after(last_group_msg_id, BATCH_SIZE)
            .await?;
        if rows.is_empty() {
            break;
        }
        let message_ids: Vec<String> = rows.iter().map(|(_, mid)| mid.to_string()).collect();
        checker.sync_messages(&message_ids).await?;
        if let Some((last_id, _)) = rows.last() {
            last_group_msg_id = *last_id;
        }
    }

    let mut last_private_msg_id = 0;
    loop {
        let rows = private_message_repo
            .list_message_ids_after(last_private_msg_id, BATCH_SIZE)
            .await?;
        if rows.is_empty() {
            break;
        }
        let message_ids: Vec<String> = rows.iter().map(|(_, mid)| mid.to_string()).collect();
        checker.sync_messages(&message_ids).await?;
        if let Some((last_id, _)) = rows.last() {
            last_private_msg_id = *last_id;
        }
    }

    checker.set_ready().await?;
    checker.log_filter_sizes().await;
    tracing::info!("Bloom filter warmup completed");
    Ok(())
}
