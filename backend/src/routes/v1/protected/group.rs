use axum::{routing::post, Router};

use crate::{handlers::group, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        // 基础操作
        .route("/create", post(group::create))
        .route("/update", post(group::update))
        .route("/delete", post(group::delete))
        .route("/join", post(group::join))
        .route("/leave", post(group::leave))
        
        // 群组管理
        .route("/update_profile", post(group::update_profile))
        .route("/update_location", post(group::update_location))
        .route("/update_privacy", post(group::update_privacy))
        .route("/update_notifications", post(group::update_notifications))
        
        // 成员管理
        .route("/members", post(group::list_members))
        .route("/remove_member", post(group::remove_member))
        .route("/mute_member", post(group::mute_member))
        .route("/unmute_member", post(group::unmute_member))
        .route("/ban_member", post(group::ban_member))
        .route("/set_member_role", post(group::set_member_role))
        .route("/transfer_ownership", post(group::transfer_ownership))
        
        // 邀请系统
        .route("/invite", post(group::invite))
        .route("/accept_invite", post(group::accept_invite))
        .route("/reject_invite", post(group::reject_invite))
        .route("/list_invites", post(group::list_invites))
        
        // 申请系统
        .route("/apply", post(group::apply))
        .route("/review_application", post(group::review_application))
        
        // 群组列表
        .route("/list", post(group::list))
}

