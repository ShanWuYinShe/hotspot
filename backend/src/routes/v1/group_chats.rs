//! 群组路由模块
//!
//! 统一的群组 API 路由，根据 Token 要求分为两类：
//! - token_optional_routes: Token 可选的路由（搜索、详情、成员列表等）
//! - token_required_routes: Token 必须的路由（创建、加入、管理等）

use axum::{routing::post, Router};

use crate::{handlers::group_chats, state::AppState};

/// Token 可选的群组路由
/// 带 Token 时返回 @auth-only 字段（如 my_state）
pub fn token_optional_routes() -> Router<AppState> {
    Router::new()
        // 搜索 API
        .route("/search_by_keyword", post(group_chats::search_by_keyword))
        .route("/search_by_tag", post(group_chats::search_by_tag))
        .route("/search_by_location", post(group_chats::search_by_location))
        // 获取指定群组信息
        .route("/open_the_door", post(group_chats::open_the_door))
        .route("/settings", post(group_chats::settings))
        // 成员列表
        .route("/members", post(group_chats::members))
        // 群公告
        .route("/announcements", post(group_chats::list_announcements))
}

/// Token 必须的群组路由
pub fn token_required_routes() -> Router<AppState> {
    // 主路由 - 群组操作
    let main_routes = Router::new()
        // 基础操作
        .route("/create", post(group_chats::create))
        .route("/update", post(group_chats::update))
        .route("/delete", post(group_chats::delete))
        .route("/transfer_ownership", post(group_chats::transfer_ownership))
        // 我在群内的状态
        .route("/my_state", post(group_chats::get_my_state))
        .route("/update_my_settings", post(group_chats::update_my_settings))
        // 成员操作
        .route("/members/invite", post(group_chats::invite))
        .route("/members/join", post(group_chats::join))
        .route("/members/apply", post(group_chats::apply))
        .route("/members/set_role", post(group_chats::set_role))
        .route("/members/mute", post(group_chats::mute))
        .route("/members/unmute", post(group_chats::unmute))
        .route("/members/ban", post(group_chats::ban))
        .route("/members/unban", post(group_chats::unban))
        .route("/members/banned", post(group_chats::banned))
        .route("/members/remove_member", post(group_chats::remove_member))
        // 申请管理（群管理员操作）
        .route("/applications", post(group_chats::applications))
        .route(
            "/applications/approve",
            post(group_chats::approve_application),
        )
        .route(
            "/applications/reject",
            post(group_chats::reject_application),
        )
        // 群公告管理
        .route(
            "/announcements/create",
            post(group_chats::create_announcement),
        )
        .route(
            "/announcements/update",
            post(group_chats::update_announcement),
        )
        .route(
            "/announcements/delete",
            post(group_chats::delete_announcement),
        );

    // /me 子路由 - 当前用户的群组相关操作
    let me_routes = Router::new()
        .route("/", post(group_chats::me))
        .route(
            "/applications/cancel",
            post(group_chats::cancel_application),
        )
        .route("/invites", post(group_chats::my_invites))
        .route("/invites/accept", post(group_chats::accept_invite))
        .route("/invites/reject", post(group_chats::reject_invite))
        .route("/leave", post(group_chats::leave_group));

    main_routes.nest("/me", me_routes)
}
