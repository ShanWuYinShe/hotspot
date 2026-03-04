use axum::{extract::State, Json};
use tracing::info;

use crate::{
    dto::{
        request::MapRenderRequest,
        response::{ApiResponse, MapCanvas},
    },
    error::Result,
    middleware::auth::OptionalAuthenticatedUser,
    state::AppState,
};

/// Public map render (no social filtering)
pub async fn render(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<MapRenderRequest>,
) -> Result<Json<ApiResponse<MapCanvas>>> {
    // spec: only_friends "仅登录态生效"
    // - 未登录：忽略 only_friends（按匿名可见范围渲染）
    // - 已登录：only_friends=true 时启用社交筛选
    let social_filter_user_id = req
        .filter
        .as_ref()
        .and_then(|f| f.only_friends)
        .filter(|&b| b)
        .and_then(|_| user.user_id);

    let user_label = user
        .user_id
        .map(|id| id.to_string())
        .unwrap_or_else(|| "guest".to_string());
    let bounds = &req.viewport.bounds;
    info!(
        target = "map::render",
        user_id = %user_label,
        zoom = req.viewport.zoom,
        north = bounds.north,
        south = bounds.south,
        east = bounds.east,
        west = bounds.west,
        filter = ?req.filter,
        "map render request"
    );

    let data = state
        .map_service
        .render_map(req, social_filter_user_id)
        .await?;
    info!(
        target = "map::render",
        user_id = %user_label,
        item_count = data.items.len(),
        has_toast = data.toast_message.as_ref().map(|s| !s.is_empty()).unwrap_or(false),
        "map render response"
    );
    Ok(Json(ApiResponse::success(data)))
}

// 注意：已移除 /protected/map/render 端点（与 spec 的单端点 /api/v1/map/render 对齐）
