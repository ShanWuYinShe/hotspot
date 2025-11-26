use axum::{routing::post, Router};

use crate::{handlers::group, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        // 3.6 群组管理
        .route("/create", post(group::create))
        .route("/update", post(group::update))
        .route("/delete", post(group::delete))
        .route("/get_joined", post(group::get_joined))
        .route("/transfer_ownership", post(group::transfer_ownership))

        // 3.7 群组成员管理
        .route("/members/get_list", post(group::get_members))
        .route("/members/invite", post(group::invite))
        .route("/members/join", post(group::join))
        .route("/members/apply", post(group::apply))
        .route("/members/set_role", post(group::set_role))
        .route("/members/mute", post(group::mute))
        .route("/members/unmute", post(group::unmute))
        .route("/members/ban", post(group::ban))
        .route("/members/remove", post(group::remove_member))
        .route("/members/leave", post(group::leave))
        
        // 3.7.11 - 3.7.13 群组申请
        .route("/applications/get_list", post(group::get_applications))
        .route("/applications/approve", post(group::approve_application))
        .route("/applications/reject", post(group::reject_application))
        
        // 3.7.15 - 3.7.17 群组邀请
        .route("/invites/get_list", post(group::get_invites))
        .route("/invites/accept", post(group::accept_invite))
        .route("/invites/reject", post(group::reject_invite))
}
