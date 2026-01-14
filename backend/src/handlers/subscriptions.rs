use axum::{extract::State, Json};

use crate::{
    dto::{
        request::{MySubscriptionsRequest, UpdateSubscriptionsRequest},
        response::{ApiResponse, SubscriptionsData},
    },
    error::Result,
    middleware::auth::AuthenticatedUser,
    state::AppState,
};

fn is_update_subscriptions_empty(req: &UpdateSubscriptionsRequest) -> bool {
    req.new_follower_enabled.is_none()
        && req.mutual_follow_enabled.is_none()
        && req.group_invite_enabled.is_none()
        && req.group_invite_response_enabled.is_none()
        && req.group_application_enabled.is_none()
        && req.group_application_response_enabled.is_none()
        && req.message_mention_enabled.is_none()
        && req.moment_like_enabled.is_none()
        && req.moment_comment_enabled.is_none()
        && req.comment_reply_enabled.is_none()
        && req.system_announcement_enabled.is_none()
}

/// 获取订阅配置（当前用户）
pub async fn get_me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(_req): Json<MySubscriptionsRequest>,
) -> Result<Json<ApiResponse<SubscriptionsData>>> {
    let data = state.subscription_service.get_config(user.user_id).await?;
    Ok(Json(ApiResponse::success(data)))
}

/// 更新订阅配置（当前用户）
pub async fn update_me(
    State(state): State<AppState>,
    user: AuthenticatedUser,
    Json(req): Json<UpdateSubscriptionsRequest>,
) -> Result<Json<ApiResponse<()>>> {
    if is_update_subscriptions_empty(&req) {
        return Ok(Json(ApiResponse::success(())));
    }

    state
        .subscription_service
        .update_config(user.user_id, &req)
        .await?;

    Ok(Json(ApiResponse::success(())))
}
