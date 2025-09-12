/// HotSpot HTTP 路由系统
/// 
/// 功能职责：
/// - 定义所有 HTTP API 路由和端点
/// - 组织路由层次结构和中间件应用
/// - 管理公开路由和受保护路由的分离
/// - 集成静态文件服务和WebSocket路由
/// 
/// 架构特点：
/// - 零重复路由，每个功能只有一套路由定义
/// - 完全符合 hotspot.md API 规范
/// - 统一的认证中间件应用策略
/// - 清晰的模块化路由组织结构
/// 
/// 路由组织：
/// - /api/v1/auth/*     - 认证系统（公开）
/// - /api/v1/user/*     - 用户数据管理（受保护）
/// - /api/v1/group/*    - 群组管理（受保护）
/// - /api/v1/chat/*     - 消息通讯（受保护）
/// - /api/v1/moments/*  - 生活瞬间（受保护）
/// - /api/v1/admin/*    - 管理接口（受保护）
use axum::{Router, middleware, routing::post};
use tower_http::services::ServeDir;
use std::sync::Arc;

use super::controllers::{
    AdminController, GroupController, HealthController, /* MessageController, */ MomentController,
    UserController, MonitoringController, SearchController, ActivityController, /* FriendController, */
    CommentController, UploadController, metrics_controller, unified_monitoring_controller,
};
use super::middleware::metrics::metrics_middleware;
use super::websocket_routes::{create_websocket_info_routes, create_websocket_routes};
use super::auth::{JwtConfig, require_auth};

use crate::application::{UserApplicationService, MomentApplicationService};
use crate::ports::UserRepository;

/// 服务容器 - 统一管理所有控制器
#[derive(Clone)]
pub struct ServiceContainer {
    pub user_controller: UserController<UserApplicationService>,
    pub group_controller: GroupController<crate::application::GroupApplicationService>,
    // pub message_controller: MessageController, // 暂时禁用
    pub moment_controller: MomentController<MomentApplicationService>,
    pub search_controller: SearchController,
    pub activity_controller: ActivityController<crate::adapters::PostgresUserRepository, crate::adapters::PgGroupRepository, crate::adapters::PgMomentRepository>,
    // pub friend_controller: FriendController<crate::application::FriendApplicationServiceImpl<crate::adapters::PgFriendRepository, crate::adapters::PostgresUserRepository>>, // 暂时禁用
    pub comment_controller: CommentController<crate::adapters::PgCommentRepository>,
    pub upload_controller: UploadController,
    pub admin_controller: AdminController,
    pub metrics_service: Option<std::sync::Arc<dyn crate::services::MetricsService>>,
}

/// 创建应用路由 - 新架构
pub fn create_application_routes(
    services: ServiceContainer,
    jwt_config: JwtConfig,
    user_repo: Arc<dyn UserRepository>,
    pg_pool: sqlx::PgPool,
    enable_websocket: bool,
) -> Router {
    let router = Router::new()
        // 静态文件服务
        .nest_service("/uploads", ServeDir::new("uploads"))
        
        // 系统健康检查路由（公开）
        .merge(create_health_routes())
        
        // 监控仪表板路由（需要管理员权限）
        .merge(create_monitoring_routes(jwt_config.clone()))
        
        // API v1 路由组（添加监控中间件）
        .nest("/api/v1", create_api_v1_routes(services, jwt_config.clone())
            .layer(middleware::from_fn(metrics_middleware)));
    
    // WebSocket路由（如果启用）
    let router = if enable_websocket {
        router
            .merge(create_websocket_info_routes())
            .merge(create_websocket_routes(user_repo, pg_pool))
    } else {
        router
    };
    
    router
}

/// 创建健康检查路由（公开）
fn create_health_routes() -> Router {
    Router::new()
        .route("/health", post(HealthController::health_check))
        .route("/health/detailed", post(HealthController::detailed_health_check))
        .route("/ready", post(HealthController::readiness_check))
        .route("/alive", post(HealthController::liveness_check))
        // Prometheus 监控指标端点
        .route("/metrics", post(metrics_controller::get_metrics))
        // TODO: 查询统计端点（暂时禁用）
        // .route("/query-stats/report", get(query_stats_controller::get_query_performance_report))
        // .route("/query-stats/slow", get(query_stats_controller::get_slow_queries))
        // .route("/query-stats/slowest", get(query_stats_controller::get_slowest_queries))
        // .route("/query-stats/summary", get(query_stats_controller::get_query_stats_summary))
        // .route("/query-stats/reset", post(query_stats_controller::reset_query_stats))
        // .route("/query-stats/cleanup", post(query_stats_controller::cleanup_old_stats))
        // TODO: 缓存监控端点（暂时禁用）
        // .route("/cache-monitor/report", get(cache_monitor_controller::get_cache_performance_report))
        // .route("/cache-monitor/stats", get(cache_monitor_controller::get_all_cache_stats))
        // .route("/cache-monitor/summary", get(cache_monitor_controller::get_cache_summary))
        // .route("/cache-monitor/reset", post(cache_monitor_controller::reset_cache_stats))
        // .route("/cache-monitor/cleanup", post(cache_monitor_controller::cleanup_old_cache_stats))
        // 统一监控端点（全部改为POST）
        .route("/monitoring/health", post(unified_monitoring_controller::get_system_health))
        .route("/monitoring/performance", post(unified_monitoring_controller::get_performance_summary))
        .route("/monitoring/report", post(unified_monitoring_controller::get_detailed_monitoring_report))
        .route("/monitoring/realtime", post(unified_monitoring_controller::get_realtime_metrics))
        .route("/monitoring/config", post(unified_monitoring_controller::get_monitoring_config))
        .route("/monitoring/reset", post(unified_monitoring_controller::reset_monitoring_stats))
}

/// 创建监控路由（需要管理员权限）
fn create_monitoring_routes(jwt_config: JwtConfig) -> Router {
    Router::new()
        .route("/monitoring/overview", post(MonitoringController::get_system_overview))
        .route("/monitoring/business-metrics", post(MonitoringController::get_business_metrics))
        .route("/monitoring/user-activity", post(MonitoringController::get_user_activity_metrics))
        .route("/monitoring/group-activity", post(MonitoringController::get_group_activity_metrics))
        .route("/monitoring/message-activity", post(MonitoringController::get_message_activity_metrics))
        .route("/monitoring/websocket-activity", post(MonitoringController::get_websocket_activity_metrics))
        .route("/monitoring/api-performance", post(MonitoringController::get_api_performance_metrics))
        .route("/monitoring/database-performance", post(MonitoringController::get_database_performance_metrics))
        .route("/monitoring/recent-activity", post(MonitoringController::get_recent_activity))
        .route("/monitoring/system-health", post(MonitoringController::get_system_health))
        // 应用管理员认证中间件
        .route_layer(middleware::from_fn_with_state(jwt_config, require_auth))
}

/// 创建API v1路由 - 使用 public/protected 前缀明确区分认证要求
fn create_api_v1_routes(services: ServiceContainer, jwt_config: JwtConfig) -> Router {
    Router::new()
        // 认证路由（公开） - 7个端点
        .nest("/auth", create_auth_routes(services.user_controller.clone()))
        
        // 公开API路由（无需认证）
        .nest("/public", create_public_routes(services.clone()))
        
        // 受保护API路由（需要认证）
        .nest("/protected", create_protected_routes(services, jwt_config))
}

/// 创建公开API路由（无需认证）
/// 路径：/api/v1/public/*
fn create_public_routes(services: ServiceContainer) -> Router {
    Router::new()
        // 用户查询子路由
        .nest("/user", Router::new()
            .route("/search_by_name", post(UserController::<UserApplicationService>::search_by_name))
            .route("/search_by_id", post(UserController::<UserApplicationService>::search_by_id))
            .route("/search_by_location", post(UserController::<UserApplicationService>::search_by_location))
            .with_state(services.user_controller.clone())
        )
        
        // 群组查询子路由
        .nest("/group", Router::new()
            .route("/search_by_location", post(GroupController::<crate::application::GroupApplicationService>::search_by_location))
            .route("/search_by_name", post(GroupController::<crate::application::GroupApplicationService>::search_by_name))
            .route("/search_by_id", post(GroupController::<crate::application::GroupApplicationService>::search_by_id))
            .route("/members", post(GroupController::<crate::application::GroupApplicationService>::group_members))
            .with_state(services.group_controller.clone())
        )
        
        // 生活瞬间查询子路由
        .nest("/moments", Router::new()
            .route("/history", post(MomentController::<MomentApplicationService>::history))
            .route("/search_by_location", post(MomentController::<MomentApplicationService>::search_by_location))
            .route("/search_by_tags", post(MomentController::<MomentApplicationService>::search_by_tags))
            .route("/search_by_id", post(MomentController::<MomentApplicationService>::search_by_id))
            .with_state(services.moment_controller.clone())
        )
        
        // 搜索功能子路由
        .nest("/search", Router::new()
            .route("/search_by_global", post(SearchController::search_by_global))
            .with_state(services.search_controller.clone())
        )
        
        // 活动查询子路由
        .nest("/activities", Router::new()
            .route("/recent", post(ActivityController::get_recent_activities))
            .with_state(services.activity_controller.clone())
        )
        
        // 评论查询子路由
        .nest("/comments", Router::new()
            .route("/moment/{moment_id}", post(CommentController::get_moment_comments))
            .route("/user", post(CommentController::get_user_comments))
            .with_state(services.comment_controller)
        )
}

/// 创建受保护API路由（需要认证）
/// 路径：/api/v1/protected/*
fn create_protected_routes(services: ServiceContainer, jwt_config: JwtConfig) -> Router {
    Router::new()
        // 用户管理子路由
        .nest("/user", Router::new()
            .route("/update_location", post(UserController::<UserApplicationService>::update_location))
            .route("/update_profile", post(UserController::<UserApplicationService>::update_profile))
            .route("/upgrade_account", post(UserController::<UserApplicationService>::upgrade_account))
            .route("/get_settings", post(UserController::<UserApplicationService>::get_settings))
            .route("/update_settings", post(UserController::<UserApplicationService>::update_settings))
            .route("/delete_account", post(UserController::<UserApplicationService>::delete_account))
            .with_state(services.user_controller.clone())
        )
        
        // 群组管理子路由
        .nest("/group", Router::new()
            .route("/create", post(GroupController::<crate::application::GroupApplicationService>::create_group))
            .route("/join", post(GroupController::<crate::application::GroupApplicationService>::join_group))
            .route("/leave", post(GroupController::<crate::application::GroupApplicationService>::leave_group))
            .route("/kick", post(GroupController::<crate::application::GroupApplicationService>::kick_member))
            .route("/transfer_ownership", post(GroupController::<crate::application::GroupApplicationService>::transfer_ownership))
            .with_state(services.group_controller.clone())
        )
        
        // // 消息通讯子路由 - 暂时禁用
        // .nest("/chat", Router::new()
        //     .route("/group/send", post(MessageController::send_group_message))
        //     .route("/group/history", post(MessageController::get_group_messages))
        //     .route("/group/delete", post(MessageController::delete_group_message))
        //     .route("/group/mark-read", post(MessageController::mark_group_messages_read))
        //     .route("/group/unread_count", post(MessageController::get_group_unread_count))
        //     .route("/user/send", post(MessageController::send_direct_message))
        //     .route("/user/history", post(MessageController::get_direct_messages))
        //     .route("/user/conversations", post(MessageController::get_conversations))
        //     .route("/user/delete", post(MessageController::delete_direct_message))
        //     .route("/user/mark-read", post(MessageController::mark_user_messages_read))
        //     .route("/user/unread_count", post(MessageController::get_user_unread_count))
        //     .with_state(services.message_controller.clone())
        // )
        
        // 生活瞬间管理子路由
        .nest("/moments", Router::new()
            .route("/create", post(MomentController::<MomentApplicationService>::create))
            .route("/delete", post(MomentController::<MomentApplicationService>::delete))
            .route("/like", post(MomentController::<MomentApplicationService>::like_moment))
            .route("/unlike", post(MomentController::<MomentApplicationService>::unlike_moment))
            .with_state(services.moment_controller.clone())
        )
        
        // // 好友系统子路由 - 暂时禁用
        // .nest("/friends", Router::new()
        //     .route("/send_request", post(FriendController::send_friend_request))
        //     .route("/accept_request", post(FriendController::accept_friend_request))
        //     .route("/reject_request", post(FriendController::reject_friend_request))
        //     .route("/remove_friendship", post(FriendController::remove_friendship))
        //     .route("/list", post(FriendController::get_friends))
        //     .route("/received_requests", post(FriendController::get_received_friend_requests))
        //     .route("/sent_requests", post(FriendController::get_sent_friend_requests))
        //     .route("/check_friendship", post(FriendController::check_friendship))
        //     .route("/pending_count", post(FriendController::get_pending_requests_count))
        //     .with_state(services.friend_controller.clone())
        // )
        
        // 评论管理子路由
        .nest("/comments", Router::new()
            .route("/create", post(CommentController::create_comment))
            .route("/{comment_id}", post(CommentController::delete_comment))
            .route("/{comment_id}/update", post(CommentController::update_comment))
            .with_state(services.comment_controller.clone())
        )
        
        // 文件上传子路由
        .nest("/upload", Router::new()
            .route("/avatar", post(UploadController::upload_avatar))
            .route("/images", post(UploadController::upload_images))
            .route("/delete", post(UploadController::delete_file))
            .with_state(services.upload_controller.clone())
        )
        
        // 管理接口子路由
        .nest("/admin", Router::new()
            .route("/user-id/stats", post(AdminController::get_partition_stats))
            .route("/user-id/validate", post(AdminController::validate_user_id))
            .route("/user-id/info", post(AdminController::get_user_id_info))
            .route("/user-id/ranges", post(AdminController::get_user_type_ranges))
            .route("/user-id/generate", post(AdminController::generate_user_ids))
            .route("/user-id/format", post(AdminController::format_user_id))
            .with_state(services.admin_controller)
        )
        
        // TODO: 指标监控子路由 - 暂时注释等待修复Handler问题
        // .nest("/metrics", Router::new()
        //     .route("/aggregated", post(crate::presentation::http::controllers::metrics_controller::get_aggregated_metrics))
        //     .route("/realtime", post(crate::presentation::http::controllers::metrics_controller::get_realtime_metrics))
        //     .route("/events/statistics", post(crate::presentation::http::controllers::metrics_controller::get_event_statistics))
        //     .route("/top", post(crate::presentation::http::controllers::metrics_controller::get_top_metrics))
        //     .route("/cleanup", post(crate::presentation::http::controllers::metrics_controller::cleanup_expired_metrics))
        //     .with_state(services.clone())
        // )
        
        // 应用认证中间件到所有受保护路由
        .route_layer(middleware::from_fn_with_state(jwt_config, require_auth))
}

/// 1. 认证路由（公开） - 7个端点
/// 路径：/api/v1/auth/*
fn create_auth_routes(user_controller: UserController<UserApplicationService>) -> Router {
    Router::new()
        .route("/register", post(UserController::<UserApplicationService>::register_user))
        .route("/login", post(UserController::<UserApplicationService>::login))
        .route("/create_temp", post(UserController::<UserApplicationService>::create_temp_user))
        .route("/refresh_token", post(UserController::<UserApplicationService>::refresh_token))
        .route("/check_status", post(UserController::<UserApplicationService>::check_auth_status))
        .route("/set_password", post(UserController::<UserApplicationService>::set_password))
        .route("/password_status", post(UserController::<UserApplicationService>::password_status))
        .with_state(user_controller)
}
