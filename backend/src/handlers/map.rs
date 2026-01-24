use axum::{extract::State, Json};

use crate::{
    dto::{
        request::MapRenderRequest,
        response::{ApiResponse, MapRenderData},
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
) -> Result<Json<ApiResponse<MapRenderData>>> {
    let only_friends = req
        .filter
        .as_ref()
        .and_then(|f| f.only_friends)
        .unwrap_or(false);

    // spec: only_friends “仅登录态生效”
    // - 未登录：忽略 only_friends（按匿名可见范围渲染）
    // - 已登录：only_friends=true 时启用社交筛选
    let data = if only_friends {
        if let Some(user_id) = user.user_id {
            state
                .map_service
                .render_map_with_social_filter(user_id, req)
                .await?
        } else {
            state.map_service.render_map(req).await?
        }
    } else {
        state.map_service.render_map(req).await?
    };
    Ok(Json(ApiResponse::success(data)))
}

// 注意：已移除 /protected/map/render 端点（与 spec 的单端点 /api/v1/map/render 对齐）
