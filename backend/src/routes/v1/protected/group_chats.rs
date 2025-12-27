use axum::{routing::post, Router};

use crate::{handlers::group_chats, state::AppState};

pub fn routes() -> Router<AppState> {
    // 主路由 - 群组操作
    let main_routes = Router::new()
        // 基础操作
        .route("/create", post(group_chats::create))
        .route("/update", post(group_chats::update))
        .route("/delete", post(group_chats::delete))
        .route("/transfer_ownership", post(group_chats::transfer_ownership))
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
