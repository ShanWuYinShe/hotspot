use axum::{extract::State, Json};

use crate::{
    dto::{
        request::SearchSuggestRequest,
        response::{ApiResponse, SearchSuggestions},
    },
    error::Result,
    middleware::auth::OptionalAuthenticatedUser,
    state::AppState,
};

/// 综合搜索建议
/// POST /api/v1/search/suggest
pub async fn suggest(
    State(state): State<AppState>,
    user: OptionalAuthenticatedUser,
    Json(req): Json<SearchSuggestRequest>,
) -> Result<Json<ApiResponse<SearchSuggestions>>> {
    let started_at = std::time::Instant::now();
    tracing::info!(
        viewer_id = user.user_id,
        prefix_len = req.prefix.chars().count(),
        types_count = req.types.as_ref().map(|v| v.len()).unwrap_or(0),
        radius = req.radius.unwrap_or(5000),
        "search.suggest request received"
    );
    let result = state.search_service.suggest(user.user_id, req).await?;
    tracing::info!(
        viewer_id = user.user_id,
        items = result.items.len(),
        elapsed_ms = started_at.elapsed().as_millis(),
        "search.suggest request completed"
    );
    Ok(Json(ApiResponse::success(result)))
}
