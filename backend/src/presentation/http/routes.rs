use crate::application::UserApplicationService;
/// HTTP路由定义
///
/// 定义应用的路由结构和中间件
use axum::{Router, middleware, routing::post};
use tower_http::services::ServeDir;

use super::controllers::{
    AdminController, GroupController, HealthController, MessageController, MomentController,
    UserController, MonitoringController, SearchController, ActivityController, FriendController,
    UploadController,
};
use super::websocket_routes::{create_websocket_info_routes, create_websocket_routes};

use crate::application::services::moment_application_service::MomentApplicationService;
// 移除未使用导入

/// 创建应用路由
pub fn create_routes(
    user_controller: UserController<UserApplicationService>,
    group_controller: GroupController<crate::application::GroupApplicationService>,
    message_controller: MessageController,
    moment_controller: MomentController<MomentApplicationService>,
    search_controller: SearchController,
    activity_controller: ActivityController<crate::adapters::PgUserRepository, crate::adapters::PgGroupRepository, crate::adapters::PgMomentRepository>,
    friend_controller: FriendController<crate::adapters::PgFriendRepository, crate::adapters::PgUserRepository>,
    upload_controller: UploadController,
    jwt: crate::presentation::http::auth::JwtConfig,
    user_repo: std::sync::Arc<dyn crate::ports::UserRepositoryPort>,
    pg_pool: sqlx::PgPool,
    enable_websocket: bool,
) -> Router {
    let router = Router::new()
        // 健康检查路由（统一POST）
        .route("/health", post(HealthController::health_check))
        .route("/health/detailed", post(HealthController::detailed_health_check))
        .route("/ready", post(HealthController::readiness_check))
        .route("/alive", post(HealthController::liveness_check))
        
        // 监控仪表板路由（需要管理员权限）
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
        // API v1 路由组
        .nest(
            "/api/v1",
            create_api_v1_routes(
                user_controller,
                group_controller,
                message_controller,
                moment_controller,
                search_controller,
                activity_controller,
                friend_controller,
                upload_controller,
                jwt.clone(),
                user_repo.clone(),
            ),
        );
    
    // WebSocket信息路由（简化版本）
    let router = if enable_websocket {
        router.merge(create_websocket_info_routes())
    } else {
        router
    };
    
    // 集成WebSocket连接路由
    let router = if enable_websocket {
        router.merge(create_websocket_routes(user_repo.clone(), pg_pool.clone()))
    } else {
        router
    };
    
    router
}

/// 创建API v1路由
fn create_api_v1_routes(
    user_controller: UserController<UserApplicationService>,
    group_controller: GroupController<crate::application::GroupApplicationService>,
    message_controller: MessageController,
    moment_controller: MomentController<MomentApplicationService>,
    search_controller: SearchController,
    activity_controller: ActivityController<crate::adapters::PgUserRepository, crate::adapters::PgGroupRepository, crate::adapters::PgMomentRepository>,
    friend_controller: FriendController<crate::adapters::PgFriendRepository, crate::adapters::PgUserRepository>,
    upload_controller: UploadController,
    jwt: crate::presentation::http::auth::JwtConfig,
    user_repo: std::sync::Arc<dyn crate::ports::UserRepositoryPort>,
) -> Router {
    use crate::presentation::http::auth::require_auth;
    Router::new()
        .nest(
            "/users",
            create_user_routes().with_state(user_controller.clone()),
        )
        // 与 hotspot.md 对齐的路由别名（新增）：
        // 认证（公开）
        .nest(
            "/auth",
            create_auth_routes().with_state(user_controller.clone()),
        )
        // 用户（受保护）
        .nest(
            "/user",
            create_user_routes_doc()
                .with_state(user_controller.clone())
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 群组相关路由
        .nest(
            "/groups",
            create_group_routes()
                .with_state(group_controller.clone())
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 群组（文档命名风格）
        .nest(
            "/group",
            create_group_routes_doc()
                .with_state(group_controller.clone())
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 消息相关路由
        .nest(
            "/messages",
            create_message_routes()
                .with_state(message_controller.clone())
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 聊天（文档命名风格）
        .nest(
            "/chat",
            create_chat_routes_doc()
                .with_state(message_controller.clone())
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 管理接口（需管理员权限）
        .nest(
            "/admin",
            create_admin_routes()
                .with_state(AdminController::with_user_repo(user_repo.clone()))
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 瞬间相关路由
        .nest(
            "/moments",
            create_moment_routes()
                .with_state(moment_controller)
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 搜索相关路由
        .nest(
            "/search",
            create_search_routes()
                .with_state(search_controller)
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 活动相关路由
        .nest(
            "/activities",
            create_activity_routes()
                .with_state(activity_controller)
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 好友相关路由
        .nest(
            "/friends",
            create_friend_routes()
                .with_state(friend_controller)
                .route_layer(middleware::from_fn_with_state(jwt.clone(), require_auth)),
        )
        // 文件上传路由
        .nest(
            "/upload",
            create_upload_routes()
                .with_state(upload_controller)
                .route_layer(middleware::from_fn_with_state(jwt, require_auth)),
        )
}

/// 创建认证路由（公开）- 7个端点
fn create_auth_routes() -> Router<UserController<UserApplicationService>> {
    Router::new()
        // 公开端点（无需认证）
        .route("/register", post(UserController::<UserApplicationService>::register_user))
        .route("/login", post(UserController::<UserApplicationService>::login))
        .route("/create_temp", post(UserController::<UserApplicationService>::create_temp_user))
        .route("/refresh_token", post(UserController::<UserApplicationService>::refresh_token))
        // 需要认证的端点（通过中间件添加认证）
        .route("/check_status", post(UserController::<UserApplicationService>::check_auth_status))
        .route("/set_password", post(UserController::<UserApplicationService>::set_password))
        .route("/password_status", post(UserController::<UserApplicationService>::password_status))
}

/// 创建用户路由（传统路由，保持兼容）
fn create_user_routes() -> Router<UserController<UserApplicationService>> {
    Router::new()
        .route(
            "/register",
            post(UserController::<UserApplicationService>::register_user),
        )
        .route(
            "/temp",
            post(UserController::<UserApplicationService>::create_temp_user),
        )
        // 其他路由已迁移到新的API结构中
}

/// 创建群组路由
fn create_group_routes() -> Router<GroupController<crate::application::GroupApplicationService>> {
    use crate::application::GroupApplicationService;
    Router::new()
        .route(
            "/",
            post(GroupController::<GroupApplicationService>::create_group),
        )
        .route(
            "/{group_id}",
            post(GroupController::<GroupApplicationService>::update_group),
        )
        .route(
            "/{group_id}/delete",
            post(GroupController::<GroupApplicationService>::delete_group),
        )
        .route(
            "/nearby",
            post(GroupController::<GroupApplicationService>::find_nearby_groups),
        )
        .route(
            "/search",
            post(GroupController::<GroupApplicationService>::search_groups),
        )
}

/// 创建管理路由（hotspot.md 对齐命名）- 6个核心端点
fn create_admin_routes() -> Router<AdminController> {
    Router::new()
        // hotspot.md 要求的6个核心端点
        .route("/user-id/stats", post(AdminController::get_partition_stats))
        .route("/user-id/validate", post(AdminController::validate_user_id))
        .route("/user-id/info", post(AdminController::get_user_id_info))
        .route("/user-id/ranges", post(AdminController::get_user_type_ranges))
        .route("/user-id/generate", post(AdminController::generate_user_ids))
        .route("/user-id/format", post(AdminController::format_user_id))
        // 扩展端点（工具功能）
        .route("/user-id/next/regular", post(AdminController::next_regular_sequence))
        .route("/user-id/next/temporary", post(AdminController::next_temporary_sequence))
        .route("/user-id/next/admin", post(AdminController::next_admin_sequence))
        .route("/user-id/next/system", post(AdminController::next_system_sequence))
        .route("/user-id/preview-allocate", post(AdminController::preview_allocate_user_ids))
}

/// 创建消息路由（新架构）
fn create_message_routes() -> Router<MessageController> {
    Router::new()
        .route(
            "/list",
            post(MessageController::get_direct_messages),
        )
        .route(
            "/send",
            post(MessageController::send_direct_message),
        )
        .route(
            "/delete",
            post(MessageController::delete_direct_message),
        )
        .route(
            "/conversations",
            post(MessageController::get_conversations),
        )
}

/// 创建生活瞬间路由（hotspot.md 对齐命名）- 8个端点
fn create_moment_routes() -> Router<MomentController<MomentApplicationService>> {
    Router::new()
        // 生活瞬间管理
        .route(
            "/create",
            post(MomentController::<MomentApplicationService>::create),
        )
        .route(
            "/delete",
            post(MomentController::<MomentApplicationService>::delete),
        )
        .route(
            "/history",
            post(MomentController::<MomentApplicationService>::history),
        )
        // 生活瞬间搜索
        .route(
            "/search_by_location",
            post(MomentController::<MomentApplicationService>::search_by_location),
        )
        .route(
            "/search_by_tags",
            post(MomentController::<MomentApplicationService>::search_by_tags),
        )
        .route(
            "/search_by_id",
            post(MomentController::<MomentApplicationService>::search_by_id),
        )
        // 生活瞬间互动
        .route(
            "/like",
            post(MomentController::<MomentApplicationService>::like_moment),
        )
        .route(
            "/unlike",
            post(MomentController::<MomentApplicationService>::unlike_moment),
        )
}

// 已替换为真实控制器

// 认证路由已合并到前面的create_auth_routes函数中

/// 用户路由（hotspot.md 对齐命名）- 9个端点
fn create_user_routes_doc() -> Router<UserController<UserApplicationService>> {
    Router::new()
        // 搜索相关
        .route("/search_by_name", post(UserController::<UserApplicationService>::search_by_name))
        .route("/search_by_id", post(UserController::<UserApplicationService>::search_by_id))
        .route("/search_by_location", post(UserController::<UserApplicationService>::search_by_location))
        // 用户数据管理
        .route("/update_location", post(UserController::<UserApplicationService>::update_location))
        .route("/update_profile", post(UserController::<UserApplicationService>::update_profile))
        .route("/upgrade_account", post(UserController::<UserApplicationService>::upgrade_account))
        // 设置管理
        .route("/get_settings", post(UserController::<UserApplicationService>::get_settings))
        .route("/update_settings", post(UserController::<UserApplicationService>::update_settings))
        // 账户管理
        .route("/delete_account", post(UserController::<UserApplicationService>::delete_account))
}

/// 群组路由（hotspot.md 对齐命名）
fn create_group_routes_doc() -> Router<GroupController<crate::application::GroupApplicationService>>
{
    use crate::application::GroupApplicationService;
    Router::new()
        .route(
            "/create",
            post(GroupController::<GroupApplicationService>::create_group),
        )
        .route(
            "/update/{group_id}",
            post(GroupController::<GroupApplicationService>::update_group),
        )
        .route(
            "/delete/{group_id}",
            post(GroupController::<GroupApplicationService>::delete_group),
        )
        .route(
            "/search_by_location",
            post(GroupController::<GroupApplicationService>::find_nearby_groups),
        )
        .route(
            "/search_by_name",
            post(GroupController::<GroupApplicationService>::search_groups),
        )
        .route(
            "/search_by_id",
            post(GroupController::<GroupApplicationService>::search_by_id),
        )
        .route(
            "/join",
            post(GroupController::<GroupApplicationService>::join_group),
        )
        .route(
            "/leave",
            post(GroupController::<GroupApplicationService>::leave_group),
        )
        .route(
            "/members",
            post(GroupController::<GroupApplicationService>::group_members),
        )
        .route(
            "/transfer_ownership",
            post(GroupController::<GroupApplicationService>::transfer_ownership),
        )
        .route(
            "/kick",
            post(GroupController::<GroupApplicationService>::kick_member),
        )
        .route(
            "/user_groups",
            post(GroupController::<GroupApplicationService>::user_groups),
        )
        .route(
            "/owned_groups",
            post(GroupController::<GroupApplicationService>::owned_groups),
        )
}

/// 聊天路由（hotspot.md 对齐命名 /chat/user/*）- 新架构
fn create_chat_routes_doc() -> Router<MessageController> {
    Router::new()
        // user-scope chat endpoints
        .route(
            "/user/history",
            post(MessageController::get_direct_messages),
        )
        .route(
            "/user/send",
            post(MessageController::send_direct_message),
        )
        .route(
            "/user/delete",
            post(MessageController::delete_direct_message),
        )
        .route(
            "/user/conversations",
            post(MessageController::get_conversations),
        )
        // group-scope chat endpoints
        .route(
            "/group/send",
            post(MessageController::send_group_message),
        )
        .route(
            "/group/history",
            post(MessageController::get_group_messages),
        )
        .route(
            "/group/delete",
            post(MessageController::delete_group_message),
        )
        .route(
            "/group/mark-read",
            post(MessageController::mark_group_messages_read),
        )
        .route(
            "/group/unread_count",
            post(MessageController::get_group_unread_count),
        )
        // user-scope chat endpoints - 添加缺失的端点
        .route(
            "/user/mark-read",
            post(MessageController::mark_user_messages_read),
        )
        .route(
            "/user/unread_count",
            post(MessageController::get_user_unread_count),
        )
}

/// 创建搜索路由
fn create_search_routes() -> Router<SearchController> {
    Router::new()
        .route(
            "/global",
            post(SearchController::global_search),
        )
}

/// 创建活动路由
fn create_activity_routes() -> Router<ActivityController<crate::adapters::PgUserRepository, crate::adapters::PgGroupRepository, crate::adapters::PgMomentRepository>> {
    Router::new()
        .route(
            "/recent",
            post(ActivityController::get_recent_activities),
        )
}

/// 创建好友路由
fn create_friend_routes() -> Router<FriendController<crate::adapters::PgFriendRepository, crate::adapters::PgUserRepository>> {
    Router::new()
        .route(
            "/send_request",
            post(FriendController::send_friend_request),
        )
        .route(
            "/accept_request",
            post(FriendController::accept_friend_request),
        )
        .route(
            "/reject_request",
            post(FriendController::reject_friend_request),
        )
        .route(
            "/remove_friendship",
            post(FriendController::remove_friendship),
        )
        .route(
            "/list",
            post(FriendController::get_friends),
        )
        .route(
            "/received_requests",
            post(FriendController::get_received_friend_requests),
        )
        .route(
            "/sent_requests",
            post(FriendController::get_sent_friend_requests),
        )
        .route(
            "/check_friendship",
            post(FriendController::check_friendship),
        )
        .route(
            "/pending_count",
            post(FriendController::get_pending_requests_count),
        )
}

/// 创建文件上传路由
fn create_upload_routes() -> Router<UploadController> {
    Router::new()
        .route(
            "/avatar",
            post(UploadController::upload_avatar),
        )
        .route(
            "/images",
            post(UploadController::upload_images),
        )
        .route(
            "/delete",
            post(UploadController::delete_file),
        )
}
