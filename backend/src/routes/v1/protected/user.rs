use axum::{routing::post, Router};

use crate::{handlers::user, state::AppState};

pub fn routes() -> Router<AppState> {
    Router::new()
        .route("/me", post(user::me))
        .route("/open_the_door", post(user::open_the_door))
        .route("/get_stats", post(user::get_stats))
        .route("/update_location", post(user::update_location))
        .route("/update_profile", post(user::update_profile))
        .route("/upgrade_account", post(user::upgrade_account))
        .route("/get_settings", post(user::get_settings))
        .route("/update_settings", post(user::update_settings))
        .route("/delete_account", post(user::delete_account))
        .route("/send_friend_request", post(user::send_friend_request))
        .route("/accept_friend_request", post(user::accept_friend_request))
        .route("/reject_friend_request", post(user::reject_friend_request))
        .route("/remove_friend", post(user::remove_friend))
        .route("/follow", post(user::follow))
        .route("/unfollow", post(user::unfollow))
        .route("/block_user", post(user::block_user))
        .route("/get_friends", post(user::get_friends))
        .route("/get_following", post(user::get_following))
        .route("/get_followers", post(user::get_followers))
        .route("/get_blocked_users", post(user::get_blocked_users))
        .route("/get_friend_requests", post(user::get_friend_requests))
        .route("/cancel_friend_request", post(user::cancel_friend_request))
        .route("/set_friend_remark", post(user::set_friend_remark))
        .route("/unblock_user", post(user::unblock_user))
}




