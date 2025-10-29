use axum::{
    routing::post,
    Router,
};

use crate::{handlers::group, state::AppState};

pub fn group_routes() -> Router<AppState> {
    Router::new()
        // 公开端点
        .route("/api/v1/public/group/search_by_name", post(group::search_by_name))
        .route("/api/v1/public/group/search_by_location", post(group::search_by_location))
        
        // 受保护端点
        .route("/api/v1/protected/group/create", post(group::create))
        .route("/api/v1/protected/group/update", post(group::update))
        .route("/api/v1/protected/group/delete", post(group::delete))
        .route("/api/v1/protected/group/join", post(group::join))
        .route("/api/v1/protected/group/leave", post(group::leave))
        .route("/api/v1/protected/group/members", post(group::list_members))
        .route("/api/v1/protected/group/remove_member", post(group::remove_member))
        .route("/api/v1/protected/group/mute_member", post(group::mute_member))
        .route("/api/v1/protected/group/unmute_member", post(group::unmute_member))
        .route("/api/v1/protected/group/set_member_role", post(group::set_member_role))
        .route("/api/v1/protected/group/transfer_ownership", post(group::transfer_ownership))
        // TODO: 添加其他路由
        // - invite_member
        // - ban_member
        // - list_my_groups
        // - list_invites
        // - accept_invite
        // - reject_invite
}
